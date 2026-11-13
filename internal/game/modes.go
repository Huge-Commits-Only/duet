// The game modes: what the menu offers, how many rounds each takes, how a question is shown and scored.
package game

import (
	"strconv"
	"strings"
	"time"
)

type opt struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// modeDef is everything that describes a game mode in one place. A new mode is one entry
// here plus how its question looks (viewQ), how a round is scored (score) and, when it
// does not follow the usual "both answer, then reveal" round, its own actions in do.
type modeDef struct {
	id, name string
	rule     string                // one line of rules, shown in the menu and when the mode first comes up in a game
	rounds   [3]int                // for a short, medium and long evening; slower modes get fewer
	subject  bool                  // one player is in the spotlight (alternating) and individual points are given
	multi    string                // with several couples: "all" answer at once, couples take "turns", "" = one couple only
	mix      bool                  // part of "Wieczór mix"
	valid    func(q question) bool // does a question from a pack have what this mode needs
}

var modes = []modeDef{
	{
		id: "wolisz", name: "Co wolisz",
		rule:   "Oboje wybieracie w tajemnicy jedną z dwóch opcji. Liczy się, czy wybierzecie to samo.",
		rounds: [3]int{8, 20, 36}, subject: false, multi: "all", mix: true,
		valid: func(q question) bool { return q.A != "" && q.B != "" },
	},
	{
		id: "kto", name: "Kto prędzej…",
		rule:   "Oboje wskazujecie, kogo z Was dotyczy pytanie. Liczy się, czy się zgadzacie.",
		rounds: [3]int{8, 20, 36}, subject: false, multi: "all", mix: true,
		valid: func(q question) bool { return q.Q != "" },
	},
	{
		id: "nigdy", name: "Nigdy przenigdy",
		rule:   "Oboje w tajemnicy odpowiadacie, czy Wam się zdarzyło. Ekran odsłania obie odpowiedzi.",
		rounds: [3]int{8, 20, 36}, subject: false, multi: "all", mix: true,
		valid: func(q question) bool { return q.Q != "" },
	},
	{
		id: "znasz", name: "Jak dobrze mnie znasz",
		rule:   "Jedno odpowiada o sobie, drugie zgaduje jego odpowiedź. Trafienie to punkt.",
		rounds: [3]int{8, 20, 36}, subject: true, multi: "all", mix: true,
		valid: func(q question) bool { return q.Q != "" && len(q.Options) >= 2 && len(q.Options) <= 4 },
	},
	{
		id: "skala", name: "Skala",
		rule:   "Jedno ustawia suwak od 0 do 100, drugie zgaduje. Im bliżej, tym więcej punktów.",
		rounds: [3]int{8, 20, 36}, subject: true, multi: "all", mix: true,
		valid: func(q question) bool { return q.Q != "" },
	},
	{
		id: "szacowanie", name: "Szacowanie",
		rule:   "Jedno podaje liczbę o sobie, drugie zgaduje. Im bliżej, tym więcej punktów.",
		rounds: [3]int{8, 20, 36}, subject: true, multi: "all", mix: true,
		valid: func(q question) bool { return q.Q != "" && q.Max >= 1 && q.Max <= 1000 },
	},
	{
		id: "telepatia", name: "Telepatia",
		rule:   "Wpiszcie pierwsze skojarzenie ze słowem z ekranu. Wygrywacie razem, gdy wpiszecie to samo.",
		rounds: [3]int{8, 20, 36}, subject: false, multi: "all", mix: true,
		valid: func(q question) bool { return q.Q != "" },
	},
	{
		id: "tak", name: "Wspólne TAK",
		rule:   "Odpowiadacie tak albo nie na pomysły. Ekran pokaże tylko to, na co zgodziliście się oboje.",
		rounds: [3]int{8, 20, 36}, subject: false, multi: "all", mix: true,
		valid: func(q question) bool { return q.Q != "" },
	},
	{
		id: "rozmowy", name: "Głębokie rozmowy",
		rule:   "Karty do rozmowy, od lekkich po najgłębsze. Bez punktów — po prostu rozmawiajcie.",
		rounds: [3]int{3, 9, 15}, subject: false, multi: "", mix: false,
		valid: func(q question) bool { return q.Q != "" && q.Level >= 1 && q.Level <= 3 },
	},
	{
		id: "wyzwania", name: "Wyzwania",
		rule:   "Ekran losuje pytanie albo zadanie dla jednego z Was. Partner ocenia, czy zaliczone.",
		rounds: [3]int{6, 14, 24}, subject: true, multi: "turns", mix: true,
		valid: func(q question) bool { return q.Q != "" && (q.Kind == "prawda" || q.Kind == "wyzwanie") },
	},
	{
		id: "blef", name: "Blef",
		rule:   "Jedno pisze o sobie dwie prawdy i jedno kłamstwo. Drugie szuka kłamstwa.",
		rounds: [3]int{4, 8, 12}, subject: true, multi: "turns", mix: true,
		valid: func(q question) bool { return q.Q != "" },
	},
	{
		id: "rysowanie", name: "Rysowanie",
		rule:   "Jedno rysuje hasło palcem na telefonie, drugie zgaduje. Macie 60 sekund.",
		rounds: [3]int{4, 8, 12}, subject: true, multi: "turns", mix: true,
		valid: func(q question) bool { return q.Q != "" },
	},
	{
		id: "mix", name: "Wieczór mix",
		rule:   "Gra sama przeplata tryby po dwie rundy i liczy wspólny wynik.",
		rounds: [3]int{8, 20, 36}, subject: false, multi: "all", mix: false,
		valid: func(q question) bool { return false },
	},
}

// The lookups the rest of the code uses, all derived from modes.
var (
	modeByID     = map[string]modeDef{}
	rules        = map[string]string{}
	subjectModes = map[string]bool{} // one player in the spotlight; individual points
	multiModes   = map[string]bool{} // playable by several couples
	turnModes    = map[string]bool{} // …one couple at a time, the others watch
	mixModes     []string            // what "Wieczór mix" interleaves, two rounds at a time

	// menu is what the phones can pick before a game; keys match Game.cfg.
	menu = map[string][]opt{
		"tone":   {{"lekko", "Lekko"}, {"romantycznie", "Romantycznie"}, {"pikantnie", "Pikantnie 18+"}},
		"length": {{"krotka", "ok. 10 min"}, {"srednia", "ok. 25 min"}, {"dluga", "ok. 45 min"}},
	}
)

func init() {
	for _, m := range modes {
		modeByID[m.id], rules[m.id] = m, m.rule
		menu["mode"] = append(menu["mode"], opt{m.id, m.name})
		subjectModes[m.id], multiModes[m.id], turnModes[m.id] = m.subject, m.multi != "", m.multi == "turns"
		if m.mix {
			mixModes = append(mixModes, m.id)
		}
	}
}

// roundsFor is the length of an evening in rounds for the quick modes (and for the mix).
var roundsFor = map[string]int{"krotka": 8, "srednia": 20, "dluga": 36}

func roundsOf(mode, length string) int {
	return modeByID[mode].rounds[map[string]int{"krotka": 0, "srednia": 1, "dluga": 2}[length]]
}

// drawTime is how long a "rysowanie" round lasts; the only timed mode. A var so tests can shorten it.
var drawTime = 60 * time.Second

// maxInk caps one drawing (numbers, i.e. two per point) so a stuck phone cannot grow it without bound.
const maxInk = 12000

// drawColors is how many pen colors there are (PALETTE in common.js).
const drawColors = 6

// viewQ is the current question in the shape the pages render: a list of options, or a 0-100 scale.
type viewQ struct {
	Kind    string   `json:"kind"` // choice | scale | text | yesno | talk | pick | dare | write | draw
	Text    string   `json:"text"`
	Options []string `json:"options"`       // choices; the two end labels of a scale; the depth label of a talk card
	Max     int      `json:"max,omitempty"` // "szacowanie": the top of the scale (0 = the usual 0-100)
}

// p and me are the viewer's couple and seat there (-1 for the projector):
// in "rysowanie" only the one drawing may see the word before the reveal, and "kto" offers the couple's own names.
func (g *Game) viewQ(p *pair, me int) *viewQ {
	q := g.cur.gendered(g.forms(p, me))
	switch g.mode {
	case "rysowanie":
		if drawing := p == g.pair && me == g.subj; !drawing && g.phase != "reveal" {
			return &viewQ{"draw", "", nil, 0}
		}
		return &viewQ{"draw", q.Q, nil, 0}
	case "wolisz":
		return &viewQ{"choice", "Co wolisz?", []string{q.A, q.B}, 0}
	case "kto":
		if me < 0 && g.multi() { // every couple has its own two names, so the projector lists none
			return &viewQ{"choice", q.Q, nil, 0}
		}
		return &viewQ{"choice", q.Q, []string{p.players[0].Name, p.players[1].Name}, 0}
	case "znasz":
		return &viewQ{"choice", q.Q, q.Options, 0}
	case "nigdy":
		return &viewQ{"choice", q.Q, nigdyOptions, 0}
	case "szacowanie":
		return &viewQ{"scale", q.Q, []string{q.A, q.B}, q.Max}
	case "telepatia":
		return &viewQ{"text", q.Q, nil, 0}
	case "tak":
		return &viewQ{"yesno", q.Q, []string{"Tak", "Nie"}, 0}
	case "wyzwania":
		if g.pending != nil {
			return &viewQ{"pick", "Prawda czy wyzwanie?", []string{"Prawda", "Wyzwanie"}, 0}
		}
		return &viewQ{"dare", q.Q, []string{map[string]string{"prawda": "Prawda", "wyzwanie": "Wyzwanie"}[q.Kind]}, 0}
	case "blef":
		if g.stmts == nil {
			return &viewQ{"write", q.Q, nil, 0}
		}
		return &viewQ{"choice", "Które zdanie to kłamstwo?", g.stmts, 0}
	case "rozmowy":
		return &viewQ{"talk", q.Q, []string{[]string{"Poziom 1 · Rozgrzewka", "Poziom 2 · Głębiej", "Poziom 3 · Od serca"}[q.Level-1]}, 0}
	}
	lo, hi := q.A, q.B
	if lo == "" || hi == "" {
		lo, hi = "wcale", "bardzo"
	}
	return &viewQ{"scale", q.Q, []string{lo, hi}, 0}
}

// nigdyOptions are the two answers in "Nigdy przenigdy"; the cards are short phrases, so no gendered verbs.
var nigdyOptions = []string{"Zdarzyło mi się", "Nigdy przenigdy"}

// subject is the seat in the spotlight this round, or -1 when both answer for themselves.
func (g *Game) subject() int { return g.subj }

func (g *Game) maxAnswer() int {
	switch g.mode {
	case "blef":
		return len(g.stmts) - 1 // -1 until the statements exist: nothing to pick yet
	case "skala":
		return 100
	case "szacowanie":
		return g.cur.Max
	case "znasz":
		return len(g.cur.Options) - 1
	}
	return 1
}

// score settles a round for one couple once both its answers are in.
func (g *Game) score(p *pair) {
	a, b := p.answers[0], p.answers[1]
	p.gain, p.hit = 0, false
	defer func() {
		if p.hit {
			p.matches++
		}
		// what couples are ranked by: closeness in "skala", otherwise one per match
		if g.mode == "skala" || g.mode == "szacowanie" {
			p.score += p.gain
		} else if p.hit {
			p.score++
		}
	}()
	switch g.mode {
	case "blef": // spotting the lie scores for the guesser, a successful bluff for the bluffer
		if p.hit = a == b; !p.hit {
			p.points[g.subj]++
			return
		}
		p.gain = 1
	case "telepatia":
		p.hit = foldWord(p.texts[0]) == foldWord(p.texts[1])
		return
	case "tak":
		if p.hit = a == 0 && b == 0; p.hit { // option 0 is "Tak"
			p.found = append(p.found, g.cur.Q)
			if len(g.playing) == 1 { // the saved list belongs to the couple at home, not to an evening with guests
				g.store.addYes(g.cur.Q)
			}
		}
		return
	case "skala", "szacowanie":
		d := max(a-b, b-a)
		if g.mode == "szacowanie" { // closeness relative to the range: 2 off out of 10 is far, out of 1000 spot on
			d = d * 100 / g.cur.Max
		}
		switch {
		case d <= 5:
			p.gain = 3
		case d <= 15:
			p.gain = 2
		case d <= 30:
			p.gain = 1
		}
		p.hit = d <= 15
	case "znasz":
		if p.hit = a == b; p.hit {
			p.gain = 1
		}
	default:
		p.hit = a == b
	}
	if s := g.subject(); s >= 0 {
		p.points[1-s] += p.gain
	}
}

// detail says what a couple answered, for the ranking shown when several couples play.
func (g *Game) detail(p *pair) string {
	said := func(seat int) string {
		a := p.answers[seat]
		switch g.mode {
		case "telepatia":
			return p.texts[seat]
		case "skala", "szacowanie":
			return strconv.Itoa(a)
		case "nigdy":
			return nigdyOptions[a]
		case "kto":
			return p.players[a].Name
		case "wolisz":
			return []string{g.cur.A, g.cur.B}[a]
		case "znasz":
			return g.cur.Options[a]
		}
		return ""
	}
	return p.players[0].Name + ": " + said(0) + " · " + p.players[1].Name + ": " + said(1)
}

var diacritics = strings.NewReplacer("ą", "a", "ć", "c", "ę", "e", "ł", "l", "ń", "n", "ó", "o", "ś", "s", "ź", "z", "ż", "z")

// foldWord makes "Plaża " and "plaza" the same answer.
func foldWord(s string) string {
	return diacritics.Replace(strings.ToLower(strings.TrimSpace(s)))
}
