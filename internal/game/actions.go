// Everything a phone can do during a game.
package game

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// act is every in-game action a phone can take.
func (g *Game) Act(id, act, key, value string) error { return g.ActAt("", id, act, key, value) }

// step names the screen the phones are on; it changes with every new game, round, phase,
// truth-or-dare pick and bluff written.
func (g *Game) step() string {
	return fmt.Sprint(g.epoch, ":", g.round, ":", g.phase, ":", g.pending != nil, ":", g.stmts != nil)
}

// ActAt is Act from a phone that saw the screen step; a tap meant for an earlier screen (double tap,
// both tapping at once) is dropped without an error. An empty step skips the check.
func (g *Game) ActAt(step, id, act, key, value string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if step != "" && step != g.step() {
		return nil
	}
	if err := g.do(id, act, key, value); err != nil {
		return err
	}
	g.botsThink()
	if act != "draw" && act != "clear" && act != "undo" { // strokes are not saved, and there are many of them
		g.save()
	}
	g.broadcast()
	return nil
}

// do applies one action; people and bots go through the same rules. Caller holds mu.
func (g *Game) do(id, act, key, value string) error {
	p, me := g.find(id)
	if p == nil || g.phase != "lobby" && !slices.Contains(g.playing, p) { // seated, and not just watching a game in progress
		return ErrBad
	}
	// When couples take turns, only the one playing answers, draws or gives up; anyone may skip, go on or quit.
	if g.phase == "question" && turnModes[g.mode] && p != g.pair && act != "skip" && act != "quit" {
		return ErrBad
	}
	// the saved list, statistics and backup belong to the household (couple 1), and only with no guests around
	if (p != g.pairs[0] || !g.home()) && slices.Contains([]string{"list", "stats", "yesdone", "yesdel"}, act) {
		return ErrBad
	}
	switch {
	case act == "config" && g.phase == "lobby":
		ok := false
		for _, o := range menu[key] {
			ok = ok || o.ID == value
		}
		if !ok {
			return ErrBad
		}
		g.cfg[key] = value
	case act == "start" && g.phase == "lobby":
		if err := g.start(); err != nil {
			return err
		}
	case act == "stake" && g.phase == "lobby":
		value = strings.TrimSpace(value)
		if utf8.RuneCountInString(value) > 80 {
			return ErrBad
		}
		p.stakes[me] = value
	case act == "bot" && g.phase == "lobby":
		if err := g.toggleBot(p); err != nil {
			return err
		}
		g.lead()
	case act == "botpair" && g.phase == "lobby":
		if err := g.toggleBotPair(); err != nil {
			return err
		}
	case act == "kick" && g.phase == "lobby":
		// someone took a seat by mistake, or left their phone behind: key is the couple's number, value the seat
		n, err := strconv.Atoi(key)
		seat, err2 := strconv.Atoi(value)
		if err != nil || err2 != nil || n < 1 || n > maxPairs || seat < 0 || seat >= len(g.pairs[n-1].players) {
			return ErrBad
		}
		kp := g.pairs[n-1]
		kp.players = slices.Delete(kp.players, seat, seat+1)
		kp.stakes = [2]string{}
		g.lead()
	case act == "list" && g.phase == "lobby":
		g.showList, g.showStats = !g.showList, false
	case act == "stats" && g.phase == "lobby":
		g.showStats, g.showList = !g.showStats, false
	case act == "yesdone" && g.phase == "lobby":
		if !g.store.toggleYes(value) {
			return ErrBad
		}
	case act == "yesdel" && g.phase == "lobby":
		if !g.store.removeYes(value) {
			return ErrBad
		}
	case act == "draw" && g.phase == "question" && g.mode == "rysowanie" && me == g.subj:
		// value: "x,y,x,y…" on a 0-1000 square; key "new" starts a stroke, otherwise the last one continues
		fields := strings.Split(value, ",")
		if len(fields)%2 != 0 || g.ink+len(fields) > maxInk {
			return ErrBad
		}
		pts := make([]int, len(fields))
		for i, f := range fields {
			n, err := strconv.Atoi(f)
			if err != nil || n < 0 || n > 1000 {
				return ErrBad
			}
			pts[i] = n
		}
		if key == "new" || len(g.strokes) == 0 {
			g.strokes = append(g.strokes, nil)
		}
		last := len(g.strokes) - 1
		g.strokes[last] = append(g.strokes[last], pts...)
		g.ink += len(pts)
	case act == "clear" && g.phase == "question" && g.mode == "rysowanie" && me == g.subj:
		g.strokes, g.ink = nil, 0
	case act == "undo" && g.phase == "question" && g.mode == "rysowanie" && me == g.subj:
		if n := len(g.strokes); n > 0 {
			g.ink -= len(g.strokes[n-1])
			g.strokes = g.strokes[:n-1]
		}
	case act == "giveup" && g.phase == "question" && g.mode == "rysowanie" && me != g.subj:
		g.endDrawing(false) // shows the word, unlike "skip"
	case act == "answer" && g.phase == "question" && g.mode == "rysowanie":
		// only the partner guesses, as often as they like until the time is up
		value = strings.TrimSpace(value)
		if n := utf8.RuneCountInString(value); me == g.subj || n < 1 || n > 30 {
			return ErrBad
		}
		right := func(w string) bool { return foldWord(w) == foldWord(value) }
		if !right(g.cur.Q) && !slices.ContainsFunc(g.cur.Alt, right) {
			g.guesses = append(g.guesses, value)
			g.guesses = g.guesses[max(0, len(g.guesses)-3):]
			break
		}
		g.points[me] += 2 // guessing is worth more than being understood
		g.points[g.subj]++
		g.endDrawing(true)
	case act == "answer" && g.phase == "question" && g.mode == "wyzwania" && g.pending != nil:
		// the one in the spotlight picks: 0 = truth, 1 = dare; only the picked card is used up
		if me != g.subj || value != "0" && value != "1" {
			return ErrBad
		}
		if value == "1" {
			g.cur = *g.pending.dare
		}
		g.pending = nil
	case act == "answer" && g.phase == "question" && g.mode == "wyzwania":
		// only the partner judges: 0 = done (a point for the one in the spotlight), 1 = gave up
		if me == g.subj || value != "0" && value != "1" {
			return ErrBad
		}
		if value == "0" {
			g.points[g.subj]++
			g.matches++
			g.pair.score++
		}
		g.store.mark(g.cur.id(g.mode))
		g.next()
	case act == "answer" && g.phase == "question" && g.mode == "blef" && g.stmts == nil:
		// the bluffer sends three statements (one per line) and which one is the lie (key)
		parts := strings.Split(value, "\n")
		lie, err := strconv.Atoi(key)
		if me != g.subj || len(parts) != 3 || err != nil || lie < 0 || lie > 2 {
			return ErrBad
		}
		for i, p := range parts {
			parts[i] = strings.TrimSpace(p)
			if n := utf8.RuneCountInString(parts[i]); n < 1 || n > 80 {
				return ErrBad
			}
		}
		g.stmts, g.answers[me] = parts, lie
	case act == "answer" && g.phase == "question" && g.mode == "telepatia":
		value = strings.TrimSpace(value)
		if n := utf8.RuneCountInString(value); n < 1 || n > 30 {
			return ErrBad
		}
		p.texts[me], p.answers[me] = value, 0
		g.settle()
	case act == "answer" && g.phase == "question" && g.mode != "rozmowy":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 || n > g.maxAnswer() {
			return ErrBad
		}
		p.answers[me] = n
		g.settle()
	case act == "next" && (g.phase == "reveal" || g.phase == "question" && g.mode == "rozmowy"):
		g.store.mark(g.cur.id(g.mode)) // talk cards have no answers; they count as asked once passed
		g.next()
	case act == "skip" && g.phase == "question":
		// a way out when a phone died or someone would rather not answer: no points, on to the next round
		if g.pending == nil { // nothing was shown yet while truth or dare is being picked
			g.store.mark(g.cur.id(g.mode))
		}
		g.next()
	case act == "quit" && g.phase != "lobby":
		g.phase = "lobby"
		g.lead()
	default:
		return ErrBad
	}
	return nil
}

// endDrawing closes a "rysowanie" round: guessed in time, or the clock ran out.
func (g *Game) endDrawing(guessed bool) {
	g.hit, g.gain = guessed, 0
	if guessed {
		g.gain = 2
		g.matches++
		g.pair.score++
	}
	g.store.mark(g.cur.id(g.mode))
	g.phase = "reveal"
}

// settle ends the round once every couple has both answers in.
func (g *Game) settle() {
	pairs := g.playing
	if turnModes[g.mode] {
		pairs = []*pair{g.pair} // the others only watch this round
	}
	for _, p := range pairs {
		if p.answers[0] < 0 || p.answers[1] < 0 {
			return
		}
	}
	for _, p := range pairs {
		g.score(p)
	}
	g.store.mark(g.cur.id(g.mode))
	if g.mode == "tak" {
		g.next() // no reveal: showing a card that did not match would expose who said yes
		return
	}
	g.phase = "reveal"
}
