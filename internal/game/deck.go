// Building the deck of rounds for a game and moving through it.
package game

import (
	"math/rand/v2"
	"slices"
	"time"
)

// card is one round of a game: a question and the mode it is played in.
type card struct {
	mode string
	q    question
	dare *question // "wyzwania": q is the truth, dare the task; the player picks one
}

// pick takes up to want questions from pool, not-yet-asked ones first;
// a used-up pool starts over rather than giving a short game.
func (g *Game) pick(mode string, pool []question, want int) []question {
	want = min(want, len(pool))
	var fresh []question
	for _, q := range pool {
		if !g.store.asked[q.id(mode)] {
			fresh = append(fresh, q)
		}
	}
	if len(fresh) < want {
		ids := make([]string, len(pool))
		for i, q := range pool {
			ids[i] = q.id(mode)
		}
		g.store.forget(ids...)
		fresh = slices.Clone(pool)
	}
	rand.Shuffle(len(fresh), func(i, j int) { fresh[i], fresh[j] = fresh[j], fresh[i] })
	return fresh[:want]
}

// cards draws up to want rounds of one mode in the chosen tone.
func (g *Game) cards(mode string, all map[string][]question, want int) []card {
	var pool, picked []question
	for _, q := range all[mode] {
		if q.Tone == g.cfg["tone"] && q.valid(mode) {
			pool = append(pool, q)
		}
	}
	if mode == "rozmowy" { // a third of the cards per depth level, shallow first
		for level := 1; level <= 3; level++ {
			var lv []question
			for _, q := range pool {
				if q.Level == level {
					lv = append(lv, q)
				}
			}
			picked = append(picked, g.pick(mode, lv, want/3)...)
		}
	} else {
		want = min(want, len(pool))
		if subjectModes[mode] && want > 1 {
			want -= want % 2 // both players are in the spotlight equally often
		}
		if mode == "wyzwania" { // every round offers one truth and one dare to choose from
			truths, dares := byKind(pool, "prawda"), byKind(pool, "wyzwanie")
			if n := min(want, len(truths), len(dares)); n > 0 {
				ts, ds := g.pick(mode, truths, n), g.pick(mode, dares, n)
				out := make([]card, n)
				for i := range out {
					out[i] = card{mode, ts[i], &ds[i]}
				}
				return out
			}
		}
		picked = g.pick(mode, pool, want)
	}
	out := make([]card, len(picked))
	for i, q := range picked {
		out[i] = card{mode: mode, q: q}
	}
	return out
}

func byKind(pool []question, kind string) []question {
	var out []question
	for _, q := range pool {
		if q.Kind == kind {
			out = append(out, q)
		}
	}
	return out
}

func (g *Game) start() error {
	// Every complete couple that is here plays; a half-seated one would be left out, so it blocks the start instead.
	var playing []*pair
	for _, p := range g.live() {
		if len(p.players) < 2 {
			return errAlone
		}
		playing = append(playing, p)
	}
	if len(playing) == 0 {
		return ErrBad
	}
	if len(playing) > 1 && !multiModes[g.cfg["mode"]] {
		return errSolo
	}
	all, length := loadQuestions(g.store.dir), g.cfg["length"]
	var deck []card
	if g.cfg["mode"] == "mix" {
		// Shuffled modes take turns, two rounds each, until the evening is full.
		modes := slices.Clone(mixModes)
		if len(playing) > 1 {
			modes = slices.DeleteFunc(modes, func(m string) bool { return !multiModes[m] })
		}
		rand.Shuffle(len(modes), func(i, j int) { modes[i], modes[j] = modes[j], modes[i] })
		total, share := roundsFor[length], map[string]int{}
		for i := 0; i < total/2; i++ {
			share[modes[i%len(modes)]] += 2
		}
		piles := map[string][]card{}
		for _, m := range modes {
			piles[m] = g.cards(m, all, share[m])
		}
		for more := true; more; {
			more = false
			for _, m := range modes {
				n := min(2, len(piles[m]))
				deck, piles[m] = append(deck, piles[m][:n]...), piles[m][n:]
				more = more || n > 0
			}
		}
	} else {
		deck = g.cards(g.cfg["mode"], all, roundsOf(g.cfg["mode"], length))
	}
	if len(deck) == 0 {
		return errNoQuestions
	}
	g.playing = playing
	for _, p := range playing {
		p.matches, p.score, p.points, p.found = 0, 0, [2]int{}, nil
	}
	g.deck, g.rounds, g.round, g.showList, g.showStats, g.turn = deck, len(deck), 0, false, false, 0
	g.phase = "question" // before lead(): from here on the first couple playing is the leading one
	g.lead()
	g.seen = map[string]bool{}
	g.epoch++
	g.next()
	return nil
}

func (g *Game) next() {
	if g.round == g.rounds {
		g.phase = "end"
		g.store.record(g)
		return
	}
	g.mode, g.cur = g.deck[g.round].mode, g.deck[g.round].q
	g.round++
	for _, p := range g.playing {
		p.answers, p.texts = [2]int{-1, -1}, [2]string{}
	}
	g.stmts, g.subj, g.pending = nil, -1, nil
	if g.deck[g.round-1].dare != nil {
		g.pending = &g.deck[g.round-1]
	}
	g.strokes, g.ink, g.guesses = nil, 0, nil
	g.intro, g.seen[g.mode] = !g.seen[g.mode], true
	if subjectModes[g.mode] {
		g.subj = g.turn % 2
		g.turn++
	}
	if g.mode == "rysowanie" {
		g.deadline = time.Now().Add(drawTime)
		epoch, round := g.epoch, g.round
		time.AfterFunc(drawTime, func() {
			g.mu.Lock()
			defer g.mu.Unlock()
			if g.epoch == epoch && g.round == round && g.phase == "question" { // still this very round
				g.endDrawing(false)
				g.save()
				g.broadcast()
			}
		})
	}
	g.phase = "question"
}
