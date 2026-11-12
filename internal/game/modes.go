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

// menu is what the phones can pick before a game; keys match game.cfg.
var menu = map[string][]opt{
	"mode": {{"wolisz", "Co wolisz"}, {"kto", "Kto prędzej…"}, {"znasz", "Jak dobrze mnie znasz"}, {"skala", "Skala"},
		{"telepatia", "Telepatia"}, {"tak", "Wspólne TAK"}, {"rozmowy", "Głębokie rozmowy"},
		{"wyzwania", "Wyzwania"}, {"blef", "Blef"}, {"rysowanie", "Rysowanie"}, {"mix", "Wieczór mix"}},
	"tone":   {{"lekko", "Lekko"}, {"romantycznie", "Romantycznie"}, {"pikantnie", "Pikantnie 18+"}},
	"length": {{"krotka", "ok. 10 min"}, {"srednia", "ok. 25 min"}, {"dluga", "ok. 45 min"}},
}

// Even numbers: in subject modes both players are asked about equally often.
// rules is the one-line explanation of each game, shown on the projector in the menu
// and again whenever a mode comes up for the first time in a game.
var rules = map[string]string{
	"wolisz":    "Oboje wybieracie w tajemnicy jedną z dwóch opcji. Liczy się, czy wybierzecie to samo.",
	"kto":       "Oboje wskazujecie, kogo z Was dotyczy pytanie. Liczy się, czy się zgadzacie.",
	"znasz":     "Jedno odpowiada o sobie, drugie zgaduje jego odpowiedź. Trafienie to punkt.",
	"skala":     "Jedno ustawia suwak od 0 do 100, drugie zgaduje. Im bliżej, tym więcej punktów.",
	"telepatia": "Wpiszcie pierwsze skojarzenie ze słowem z ekranu. Wygrywacie razem, gdy wpiszecie to samo.",
	"tak":       "Odpowiadacie tak albo nie na pomysły. Ekran pokaże tylko to, na co zgodziliście się oboje.",
	"rozmowy":   "Karty do rozmowy, od lekkich po najgłębsze. Bez punktów — po prostu rozmawiajcie.",
	"wyzwania":  "Ekran losuje pytanie albo zadanie dla jednego z Was. Partner ocenia, czy zaliczone.",
	"blef":      "Jedno pisze o sobie dwie prawdy i jedno kłamstwo. Drugie szuka kłamstwa.",
	"rysowanie": "Jedno rysuje hasło palcem na telefonie, drugie zgaduje. Macie 60 sekund.",
	"mix":       "Gra sama przeplata tryby po dwie rundy i liczy wspólny wynik.",
}

var roundsFor = map[string]int{"krotka": 8, "srednia": 20, "dluga": 36}

// talkRounds: "rozmowy" cards take minutes each; a multiple of 3, one third per depth level.
var talkRounds = map[string]int{"krotka": 3, "srednia": 9, "dluga": 15}

// Slower modes get fewer rounds for the same evening length.
var (
	dareRounds = map[string]int{"krotka": 6, "srednia": 14, "dluga": 24}
	blefRounds = map[string]int{"krotka": 4, "srednia": 8, "dluga": 12}
)

func roundsOf(mode, length string) int {
	switch mode {
	case "rozmowy":
		return talkRounds[length]
	case "wyzwania":
		return dareRounds[length]
	case "blef", "rysowanie":
		return blefRounds[length]
	}
	return roundsFor[length]
}

// subjectModes put one player in the spotlight (alternating) and give individual points.
var subjectModes = map[string]bool{"znasz": true, "skala": true, "wyzwania": true, "blef": true, "rysowanie": true}

// drawTime is how long a "rysowanie" round lasts; the only timed mode. A var so tests can shorten it.
var drawTime = 60 * time.Second

// maxInk caps one drawing (numbers, i.e. two per point) so a stuck phone cannot grow it without bound.
const maxInk = 12000

// mixModes are what "Wieczór mix" interleaves, two rounds at a time. No "rozmowy": it has no score.
// multiModes can be played by several couples at once: everyone answers at the same time.
// The others need the projector for one couple (a drawing, a dare, a talk).
var multiModes = map[string]bool{"wolisz": true, "kto": true, "znasz": true, "skala": true, "telepatia": true, "tak": true, "mix": true}

var mixModes = []string{"wolisz", "kto", "znasz", "skala", "telepatia", "tak", "wyzwania", "blef", "rysowanie"}

// viewQ is the current question in the shape the pages render: a list of options, or a 0-100 scale.
type viewQ struct {
	Kind    string   `json:"kind"` // choice | scale | text | yesno | talk | pick | dare | write | draw
	Text    string   `json:"text"`
	Options []string `json:"options"` // choices; the two end labels of a scale; the depth label of a talk card
}

// p and me are the viewer's couple and seat there (-1 for the projector):
// in "rysowanie" only the one drawing may see the word before the reveal, and "kto" offers the couple's own names.
func (g *Game) viewQ(p *pair, me int) *viewQ {
	q := g.cur.gendered(g.forms(p, me))
	switch g.mode {
	case "rysowanie":
		if me != g.subj && g.phase != "reveal" {
			return &viewQ{"draw", "", nil}
		}
		return &viewQ{"draw", q.Q, nil}
	case "wolisz":
		return &viewQ{"choice", "Co wolisz?", []string{q.A, q.B}}
	case "kto":
		if me < 0 && g.multi() { // every couple has its own two names, so the projector lists none
			return &viewQ{"choice", q.Q, nil}
		}
		return &viewQ{"choice", q.Q, []string{p.players[0].Name, p.players[1].Name}}
	case "znasz":
		return &viewQ{"choice", q.Q, q.Options}
	case "telepatia":
		return &viewQ{"text", q.Q, nil}
	case "tak":
		return &viewQ{"yesno", q.Q, []string{"Tak", "Nie"}}
	case "wyzwania":
		if g.pending != nil {
			return &viewQ{"pick", "Prawda czy wyzwanie?", []string{"Prawda", "Wyzwanie"}}
		}
		return &viewQ{"dare", q.Q, []string{map[string]string{"prawda": "Prawda", "wyzwanie": "Wyzwanie"}[q.Kind]}}
	case "blef":
		if g.stmts == nil {
			return &viewQ{"write", q.Q, nil}
		}
		return &viewQ{"choice", "Które zdanie to kłamstwo?", g.stmts}
	case "rozmowy":
		return &viewQ{"talk", q.Q, []string{[]string{"Poziom 1 · Rozgrzewka", "Poziom 2 · Głębiej", "Poziom 3 · Od serca"}[q.Level-1]}}
	}
	lo, hi := q.A, q.B
	if lo == "" || hi == "" {
		lo, hi = "wcale", "bardzo"
	}
	return &viewQ{"scale", q.Q, []string{lo, hi}}
}

// subject is the seat in the spotlight this round, or -1 when both answer for themselves.
func (g *Game) subject() int { return g.subj }

func (g *Game) maxAnswer() int {
	switch g.mode {
	case "blef":
		return len(g.stmts) - 1 // -1 until the statements exist: nothing to pick yet
	case "skala":
		return 100
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
		if g.mode == "skala" {
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
	case "skala":
		d := max(a-b, b-a)
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
		case "skala":
			return strconv.Itoa(a)
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
