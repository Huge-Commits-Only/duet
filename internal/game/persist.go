package game

import (
	"crypto/rand"
	"encoding/hex"
	"slices"
)

// saved is the game as written to game.json after every change, so a restarted
// container picks the evening up where it was. Phones reconnect by their id.
type saved struct {
	Key       string // join code in the QR link
	Pairs     []savedPair
	Playing   []int // indices into Pairs
	Active    int   // index of the couple whose turn it is
	Cfg       map[string]string
	Phase     string
	Deck      []savedCard
	Mode      string
	Cur       question
	Round     int
	Rounds    int
	Subj      int
	Turn      int
	Intro     bool
	ShowList  bool
	ShowStats bool
	Pending   bool
	Stmts     []string
	Guesses   []string
	Seen      map[string]bool
}

type savedPair struct {
	Players []savedPlayer
	Answers [2]int
	Texts   [2]string
	Points  [2]int
	Stakes  [2]string
	Matches int
	Score   int
	Gain    int
	Hit     bool
	Found   []string
}

type savedPlayer struct {
	ID, Name, Color, Form string
	Bot                   bool
}

type savedCard struct {
	Mode string
	Q    question
	Dare *question
}

// save writes the current state. Caller holds mu. The drawing itself is not kept:
// a drawing round cut short by a restart simply counts as out of time.
func (g *Game) save() {
	s := saved{
		Key: g.key, Cfg: g.cfg, Phase: g.phase, Mode: g.mode, Cur: g.cur, Round: g.round, Rounds: g.rounds,
		Subj: g.subj, Turn: g.turn, Intro: g.intro, ShowList: g.showList, ShowStats: g.showStats,
		Pending: g.pending != nil, Stmts: g.stmts, Guesses: g.guesses, Seen: g.seen,
	}
	for _, p := range g.pairs {
		sp := savedPair{Answers: p.answers, Texts: p.texts, Points: p.points, Stakes: p.stakes,
			Matches: p.matches, Score: p.score, Gain: p.gain, Hit: p.hit, Found: p.found}
		for _, pl := range p.players {
			sp.Players = append(sp.Players, savedPlayer{pl.ID, pl.Name, pl.Color, pl.Form, pl.Bot})
		}
		s.Pairs = append(s.Pairs, sp)
	}
	for _, p := range g.playing {
		s.Playing = append(s.Playing, slices.Index(g.pairs[:], p))
	}
	s.Active = slices.Index(g.pairs[:], g.pair)
	for _, c := range g.deck {
		s.Deck = append(s.Deck, savedCard{c.mode, c.q, c.dare})
	}
	g.store.write("game.json", s)
}

// load restores what save wrote, or leaves the fresh lobby when there is nothing usable.
func (g *Game) load() {
	var s saved
	g.store.read("game.json", &s)
	if g.key = s.Key; g.key == "" {
		b := make([]byte, 3)
		_, _ = rand.Read(b) // never fails on supported platforms
		g.key = hex.EncodeToString(b)
	}
	for i, sp := range s.Pairs {
		if i >= maxPairs {
			break
		}
		p := g.pairs[i]
		for _, pl := range sp.Players[:min(2, len(sp.Players))] {
			p.players = append(p.players, &player{ID: pl.ID, Name: pl.Name, Color: pl.Color, Form: pl.Form, Bot: pl.Bot})
		}
		p.stakes = sp.Stakes
	}
	for k := range g.cfg { // only known options; anything else keeps its default
		for _, o := range menu[k] {
			if o.ID == s.Cfg[k] {
				g.cfg[k] = o.ID
			}
		}
	}
	defer g.lead()
	// A game in progress is only resumed when everything it needs is there.
	var playing []*pair
	for _, i := range s.Playing {
		if i >= 0 && i < maxPairs && len(g.pairs[i].players) == 2 {
			playing = append(playing, g.pairs[i])
		}
	}
	if s.Phase == "" || s.Phase == "lobby" || len(playing) == 0 || len(playing) != len(s.Playing) ||
		s.Round < 1 || s.Round > len(s.Deck) || s.Rounds != len(s.Deck) {
		g.showList = s.Phase == "lobby" && s.ShowList
		g.showStats = s.Phase == "lobby" && s.ShowStats
		return
	}
	for i, sp := range s.Pairs[:min(maxPairs, len(s.Pairs))] {
		p := g.pairs[i]
		p.answers, p.texts, p.points = sp.Answers, sp.Texts, sp.Points
		p.matches, p.score, p.gain, p.hit, p.found = sp.Matches, sp.Score, sp.Gain, sp.Hit, sp.Found
	}
	for _, c := range s.Deck {
		g.deck = append(g.deck, card{c.Mode, c.Q, c.Dare})
	}
	g.playing = playing
	g.phase, g.mode, g.cur, g.round, g.rounds = s.Phase, s.Mode, s.Cur, s.Round, s.Rounds
	g.subj, g.turn, g.intro = s.Subj, s.Turn, s.Intro
	g.stmts, g.guesses, g.seen = s.Stmts, s.Guesses, s.Seen
	if g.seen == nil {
		g.seen = map[string]bool{}
	}
	if s.Pending && g.deck[g.round-1].dare != nil {
		g.pending = &g.deck[g.round-1]
	}
	if s.Active >= 0 && s.Active < maxPairs && slices.Contains(playing, g.pairs[s.Active]) {
		g.pair = g.pairs[s.Active]
	}
	g.lead()
	if g.phase == "question" && g.mode == "rysowanie" {
		g.endDrawing(false)
	}
	g.botsThink() // their timers did not survive the restart
}
