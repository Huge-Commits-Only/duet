// Package game is the Duet game itself: the state, the modes and the rules.
// It knows nothing about HTTP; see internal/server for that.
//
// This file: the state, who is seated, and what each viewer gets to see.
package game

import (
	"encoding/json"
	"errors"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type player struct {
	ID     string `json:"-"` // secret: identifies the phone, never broadcast
	Name   string `json:"name"`
	Color  string `json:"color"`
	Form   string `json:"form"` // "m" or "f": how to address the player; "" = not chosen
	Online bool   `json:"online"`
	Bot    bool   `json:"bot,omitempty"` // a stand-in for trying the game out alone
}

// maxPairs is how many couples can sit at one projector.
const maxPairs = 4

// pair is one couple and everything that is scored per couple.
// Arrays are indexed by the seat within the couple (0 or 1), in the order they joined.
type pair struct {
	players []*player // at most 2
	answers [2]int    // -1 = not answered yet; an option index, or 0-100 in "skala"
	texts   [2]string // the word typed in "telepatia"
	points  [2]int    // individual points, only in subjectModes
	stakes  [2]string // what the loser does for this player; secret until the end
	matches int       // rounds that counted as a match
	score   int       // what couples are ranked by when several play
	gain    int       // points the guesser got in the round being revealed
	hit     bool      // the round being revealed counted as a match
	found   []string  // "tak": cards both said yes to in this game
}

type Game struct {
	mu    sync.Mutex
	conns map[string]int         // open event streams per player id
	subs  map[chan []byte]string // stream -> viewer id ("" = screen)
	store *store
	key   string // join code: phones need it (from the QR link) to take a seat

	// The couple the single-couple modes are about: the first one playing (in the lobby, the first one seated).
	// Embedded, so g.players, g.answers, g.points… mean that couple's.
	*pair
	pairs   [maxPairs]*pair
	playing []*pair // the couples in the current game: every complete one at the time it started

	phase     string // lobby | question | reveal | end
	cfg       map[string]string
	deck      []card
	mode      string // mode of the current round; differs from cfg["mode"] only in "mix"
	cur       question
	subj      int      // seat in the spotlight this round (in every couple), -1 = none
	turn      int      // subject rounds played so far, to alternate the spotlight
	stmts     []string // "blef": the three statements, once the subject wrote them
	pending   *card    // "wyzwania": truth or dare not chosen yet
	strokes   [][]int  // "rysowanie": the drawing, each stroke as x,y,x,y… on a 0-1000 square
	ink       int      // numbers in strokes
	guesses   []string // "rysowanie": the latest wrong guesses
	deadline  time.Time
	seen      map[string]bool // modes already played in this game
	intro     bool            // the current round is the first of its mode: explain the rules
	epoch     int             // bumped on every start, so a timer from an abandoned game does nothing
	round     int             // 1-based number of the current question
	rounds    int
	showList  bool              // lobby: the projector shows the saved list of mutual yeses
	showStats bool              // lobby: the projector shows the statistics
	botStage  map[string]string // per bot: the situation its pending move was scheduled for
}

func New(dataDir string) *Game {
	g := &Game{
		conns: map[string]int{}, subs: map[chan []byte]string{}, store: openStore(dataDir), botStage: map[string]string{},
		phase: "lobby", cfg: map[string]string{"mode": "wolisz", "tone": "lekko", "length": "krotka"},
	}
	for i := range g.pairs {
		g.pairs[i] = &pair{}
	}
	g.pair = g.pairs[0]
	g.load()
	return g
}

var (
	ErrBad         = errors.New("niepoprawne dane")
	ErrFull        = errors.New("ta para jest już w komplecie")
	ErrColor       = errors.New("ten kolor ma już ktoś inny, wybierz inny")
	ErrKey         = errors.New("zeskanuj kod QR z ekranu, żeby dołączyć")
	errNoQuestions = errors.New("brak pytań dla tego trybu i tonu")
	errAlone       = errors.New("ktoś czeka jeszcze na swoją drugą połowę")
	errSolo        = errors.New("ten tryb jest tylko dla jednej pary")
	colorRe        = regexp.MustCompile(`^#[0-9a-f]{6}$`)
)

// Key is the join code phones need to take a seat; it goes into the QR link.
func (g *Game) Key() string { return g.key }

// find returns the couple a phone belongs to and its seat there, or nil.
func (g *Game) find(id string) (*pair, int) {
	for _, p := range g.pairs {
		for i, pl := range p.players {
			if id != "" && pl.ID == id {
				return p, i
			}
		}
	}
	return nil, -1
}

func (g *Game) seat(id string) int {
	_, i := g.find(id)
	return i
}

// seated is the couples with at least one player, in order.
func (g *Game) seated() []*pair {
	var out []*pair
	for _, p := range g.pairs {
		if len(p.players) > 0 {
			out = append(out, p)
		}
	}
	return out
}

// here reports whether a player's phone is connected (a bot always is).
func (g *Game) here(pl *player) bool { return pl.Bot || g.conns[pl.ID] > 0 }

// live is the seated couples with at least one player here. Couples whose phones are all
// gone (guests who left) keep their seats but neither play nor block anyone.
func (g *Game) live() []*pair {
	var out []*pair
	for _, p := range g.seated() {
		if slices.ContainsFunc(p.players, g.here) {
			out = append(out, p)
		}
	}
	return out
}

// multi reports whether several couples are in play (in the lobby: here to play).
func (g *Game) multi() bool {
	if g.phase == "lobby" {
		return len(g.live()) > 1
	}
	return len(g.playing) > 1
}

// lead points the embedded couple at the one the single-couple code should talk about.
func (g *Game) lead() {
	switch {
	case g.phase != "lobby" && len(g.playing) > 0:
		g.pair = g.playing[0]
	case len(g.live()) > 0:
		g.pair = g.live()[0]
	case len(g.seated()) > 0:
		g.pair = g.seated()[0]
	default:
		g.pair = g.pairs[0]
	}
}

// viewPair is one couple as shown in the lobby and in the ranking.
type viewPair struct {
	N       int      `json:"n"` // 1-based number picked when joining
	Players []player `json:"players"`
	Score   int      `json:"score"`
	Done    bool     `json:"done"`             // both answered this round
	Hit     bool     `json:"hit"`              // reveal: this round counted
	Gain    int      `json:"gain"`             // reveal: points for this round
	Detail  string   `json:"detail,omitempty"` // reveal: what they answered
}

func (g *Game) viewPlayers(p *pair) []player {
	out := []player{}
	for _, pl := range p.players {
		c := *pl
		c.Online = g.here(pl)
		out = append(out, c)
	}
	return out
}

// view renders the state as seen by one viewer, so secrets (a partner's answer
// before the reveal) are only ever added for the viewer allowed to see them. Caller holds mu.
//
// Everything about "the couple" (players, me, answers, points…) is the viewer's own couple;
// the projector gets the leading one. All couples are listed in Pairs.
func (g *Game) view(id string) []byte {
	p, me := g.find(id)
	if p == nil {
		p = g.pair
	}
	multi := g.multi()
	v := struct {
		Phase    string            `json:"phase"`
		Players  []player          `json:"players"`
		Me       int               `json:"me"`
		Pair     int               `json:"pair"`  // number of the viewer's couple, 1-based
		Pairs    []viewPair        `json:"pairs"` // every couple seated (lobby) or playing
		Multi    bool              `json:"multi"` // several couples
		Solo     []string          `json:"solo,omitempty"`
		Cfg      map[string]string `json:"cfg"`
		Menu     map[string][]opt  `json:"menu"`
		Round    int               `json:"round"`
		Rounds   int               `json:"rounds"`
		Matches  int               `json:"matches"`
		Mode     string            `json:"mode"` // mode of the current round
		Rule     string            `json:"rule,omitempty"`
		Key      string            `json:"key,omitempty"` // join code, only for the projector
		Scored   bool              `json:"scored"`        // this game gives individual points and rewards
		Subject  int               `json:"subject"`       // seat the question is about, -1 = both
		Points   [2]int            `json:"points"`
		Gain     int               `json:"gain"`
		Hit      bool              `json:"hit"`
		ListLen  int               `json:"listLen"`           // saved mutual yeses
		List     []yesItem         `json:"list,omitempty"`    // only while shown on the projector
		HasStats bool              `json:"hasStats"`          // at least one game was recorded
		Stats    *stats            `json:"stats,omitempty"`   // only while shown on the projector
		MyText   string            `json:"myText,omitempty"`  // own word in "telepatia"
		Texts    *[2]string        `json:"texts,omitempty"`   // both words, in the reveal
		Strokes  [][]int           `json:"strokes,omitempty"` // "rysowanie": only sent to the projector
		Guesses  []string          `json:"guesses,omitempty"`
		Left     int64             `json:"left,omitempty"`  // ms until the round times out
		Found    []string          `json:"found,omitempty"` // "tak": mutual yeses of this game (never the non-matches)
		StakeSet [2]bool           `json:"stakeSet"`
		MyStake  string            `json:"myStake"`
		Q        *viewQ            `json:"q,omitempty"`
		Answered *[2]bool          `json:"answered,omitempty"`
		MyAnswer *int              `json:"myAnswer,omitempty"`
		Answers  *[2]int           `json:"answers,omitempty"`
		Winner   *int              `json:"winner,omitempty"` // at the end; -1 = draw
		Stakes   *[2]string        `json:"stakes,omitempty"` // at the end; only the revealed ones
	}{Phase: g.phase, Players: g.viewPlayers(p), Me: me, Multi: multi, Pairs: []viewPair{}, Cfg: g.cfg, Menu: menu, Round: g.round, Rounds: g.rounds, Matches: p.matches,
		// rewards are between the two of a couple, so only when one couple plays
		Scored:  (subjectModes[g.cfg["mode"]] || g.cfg["mode"] == "mix") && !multi,
		Subject: -1, Mode: g.cfg["mode"], Points: p.points, StakeSet: [2]bool{p.stakes[0] != "", p.stakes[1] != ""},
		ListLen: len(g.store.yes), HasStats: len(g.store.history) > 0}
	shown := g.live() // couples whose phones are all gone are not listed
	if g.phase != "lobby" {
		shown = g.playing
	}
	for _, sp := range shown {
		vp := viewPair{N: slices.Index(g.pairs[:], sp) + 1, Players: g.viewPlayers(sp), Score: sp.score}
		switch g.phase {
		case "question":
			vp.Done = sp.answers[0] >= 0 && sp.answers[1] >= 0
		case "reveal":
			vp.Hit, vp.Gain, vp.Detail = sp.hit, sp.gain, g.detail(sp)
		}
		v.Pairs = append(v.Pairs, vp)
	}
	v.Pair = slices.Index(g.pairs[:], p) + 1
	if multi {
		for _, o := range menu["mode"] {
			if !multiModes[o.ID] {
				v.Solo = append(v.Solo, o.ID)
			}
		}
	}
	if me >= 0 {
		v.MyStake = p.stakes[me]
	}
	if g.phase == "question" || g.phase == "reveal" {
		v.Mode = g.mode
	}
	if id == "" {
		v.Key = g.key
	}
	if id == "" && g.mode == "rysowanie" && (g.phase == "question" || g.phase == "reveal") {
		v.Strokes = g.strokes
	}
	// With several couples the projector shows the ranking, not one couple's answers.
	own := id != "" || !multi
	if g.phase != "lobby" && own {
		v.Found = p.found // in the lobby the saved list is shown only on request
	}
	switch g.phase {
	case "lobby":
		v.Rule = rules[g.cfg["mode"]]
		if g.showList {
			v.List = g.store.yes
		}
		if g.showStats {
			v.Stats = g.store.stats()
		}
	case "question":
		v.Q, v.Subject, v.Guesses = g.viewQ(p, me), g.subject(), g.guesses
		if g.intro {
			v.Rule = rules[g.mode]
		}
		if g.mode == "rysowanie" {
			v.Left = max(0, time.Until(g.deadline).Milliseconds())
		}
		v.Answered = &[2]bool{p.answers[0] >= 0, p.answers[1] >= 0}
		if me >= 0 && p.answers[me] >= 0 {
			v.MyAnswer, v.MyText = &p.answers[me], p.texts[me]
		}
	case "reveal":
		v.Q, v.Subject = g.viewQ(p, me), g.subject()
		if own {
			v.Gain, v.Hit, v.Answers = p.gain, p.hit, &p.answers
			if g.mode == "telepatia" {
				v.Texts = &p.texts
			}
		}
	case "end":
		if !v.Scored {
			break // nobody wins a game without points, so there is nothing to reward
		}
		// A draw reveals both rewards; otherwise only the winner's.
		w, st := -1, p.stakes
		if p.points[0] != p.points[1] {
			w = 0
			if p.points[1] > p.points[0] {
				w = 1
			}
			st[1-w] = ""
		}
		v.Winner, v.Stakes = &w, &st
	}
	b, _ := json.Marshal(v)
	return b
}

// broadcast sends everyone a fresh snapshot. Caller holds mu.
func (g *Game) broadcast() {
	for ch, id := range g.subs {
		select { // snapshots are complete, so a stale unread one can be dropped
		case <-ch:
		default:
		}
		ch <- g.view(id)
	}
}

func (g *Game) join(id, name, color string) error { return g.JoinAs(id, name, color, "", g.key, 0) }

// JoinAs seats a phone in couple number n (0-based). A phone that is not seated yet
// needs the join code from the QR link. A phone already seated keeps its seat while a
// game is on; in the lobby it may move to another couple.
func (g *Game) JoinAs(id, name, color, form, key string, n int) error {
	name = strings.TrimSpace(name)
	if id == "" || len(id) > 64 || name == "" || utf8.RuneCountInString(name) > 20 || !colorRe.MatchString(color) ||
		form != "" && form != "m" && form != "f" || n < 0 || n >= maxPairs {
		return ErrBad
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	old, at := g.find(id)
	if key != g.key && old == nil {
		return ErrKey
	}
	for _, p := range g.pairs {
		for _, pl := range p.players {
			if pl.ID != id && pl.Color == color && g.conns[pl.ID] > 0 {
				return ErrColor // the projector tells the players apart by color
			}
		}
	}
	np := &player{ID: id, Name: name, Color: color, Form: form}
	target := g.pairs[n]
	if old != nil && (old == target || g.phase != "lobby") {
		old.players[at] = np // same seat, new name, color or form
	} else {
		slot := -1
		for i, pl := range target.players {
			if g.conns[pl.ID] == 0 { // also a bot: a real partner simply takes its place
				slot = i // a phone that went away frees its seat when the couple is complete
				break
			}
		}
		switch {
		case len(target.players) < 2:
			target.players = append(target.players, np)
		case slot >= 0:
			target.players[slot] = np
			target.stakes[slot] = "" // a different phone took the seat: the old secret is not theirs
		default:
			return ErrFull
		}
		if old != nil { // moved over from another couple
			old.players = slices.Delete(old.players, at, at+1)
			old.stakes = [2]string{}
		}
	}
	g.lead()
	g.save()
	g.broadcast()
	return nil
}

func (g *Game) Subscribe(id string) chan []byte {
	ch := make(chan []byte, 1)
	g.mu.Lock()
	defer g.mu.Unlock()
	g.subs[ch] = id
	if id != "" {
		g.conns[id]++
	}
	g.broadcast()
	return ch
}

func (g *Game) Unsubscribe(ch chan []byte) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if id := g.subs[ch]; id != "" {
		if g.conns[id]--; g.conns[id] <= 0 {
			delete(g.conns, id)
		}
	}
	delete(g.subs, ch)
	g.broadcast()
}
