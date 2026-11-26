package game

import (
	"errors"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Duet remembers the couple: a question comes back months later to compare the answers,
// letters wait a year in the time capsule, the anniversary evening is built from their own history,
// and the first game of a month sums up the one before.

var clock = time.Now // tests move the time

const (
	againAfter  = 60 * 24 * time.Hour  // a question comes back two months after it was last asked, at the earliest
	letterAfter = 365 * 24 * time.Hour // a letter opens after a year
	maxLetter   = 200                  // characters
	minHistory  = 4                    // rounds an anniversary evening needs
)

var errHistory = errors.New("za mało wspólnej historii: zagrajcie najpierw kilka wieczorów we dwoje")

// align returns what each of the given seats said in r, matched by name (the two may have swapped seats).
func (r roundRec) align(names [2]string) [2]string {
	var out [2]string
	for i, n := range names {
		if j := slices.Index(r.Names[:], n); j >= 0 {
			out[i] = r.Said[j]
		}
	}
	return out
}

// sameCouple reports whether r was played by these two.
func (r roundRec) sameCouple(names [2]string) bool {
	return r.Names == names || r.Names == [2]string{names[1], names[0]}
}

// latest returns, per question, its most recent round that can be asked again.
func (s *store) latest() map[string]roundRec {
	out := map[string]roundRec{}
	for _, r := range s.rounds {
		if r.ID != "" && r.Mode != "" {
			out[r.ID] = r
		}
	}
	return out
}

// lookup finds the question a round was about in today's packs.
func lookup(all map[string][]question, r roundRec) (question, bool) {
	i := slices.IndexFunc(all[r.Mode], func(q question) bool { return q.id(r.Mode) == r.ID })
	if i < 0 {
		return question{}, false
	}
	return all[r.Mode][i], true
}

// household reports whether the game being set up is the couple at home, with no bots: their history applies.
func (g *Game) household() bool { return len(g.playing) == 1 && g.home() && !g.hasBots() }

func (g *Game) names() [2]string {
	p := g.playing[0].players
	return [2]string{p[0].Name, p[1].Name}
}

// usable reports whether a past round can be played again tonight: same couple, the question still
// exists and is not banned, and its tone is not hotter than the evening's ("mieszany" is last, so it takes any).
func (g *Game) usable(all map[string][]question, r roundRec) (question, bool) {
	q, ok := lookup(all, r)
	hotter := slices.IndexFunc(menu["tone"], func(o opt) bool { return o.ID == q.Tone }) >
		slices.IndexFunc(menu["tone"], func(o opt) bool { return o.ID == g.cfg["tone"] })
	return q, ok && r.sameCouple(g.names()) && !g.store.banned[r.ID] && !hotter
}

// withRepeat swaps one card of the deck for a question the couple answered at least two months ago,
// in a mode the deck already has, so the screen can show what they said then.
func (g *Game) withRepeat(deck []card, all map[string][]question) []card {
	if !g.household() {
		return deck
	}
	var due []roundRec
	for _, r := range g.store.latest() {
		in := slices.ContainsFunc(deck, func(c card) bool { return c.mode == r.Mode })
		dup := slices.ContainsFunc(deck, func(c card) bool { return c.q.id(c.mode) == r.ID })
		if _, ok := g.usable(all, r); ok && in && !dup && clock().Sub(r.When) >= againAfter {
			due = append(due, r)
		}
	}
	if len(due) == 0 {
		return deck
	}
	slices.SortFunc(due, func(a, b roundRec) int { return a.When.Compare(b.When) }) // map order is random; the pick should not be
	r := due[rand.IntN(len(due))]
	q, _ := g.usable(all, r)
	var at []int
	for i, c := range deck {
		if c.mode == r.Mode {
			at = append(at, i)
		}
	}
	deck[at[rand.IntN(len(at))]] = card{mode: r.Mode, q: q, before: &r}
	return deck
}

// anniversary builds an evening from the couple's history: first the questions they differed on
// most, then the oldest ones, played in the order they were first asked.
func (g *Game) anniversary(all map[string][]question, want int) ([]card, error) {
	if !g.household() {
		return nil, errHistory
	}
	var past []roundRec
	for _, r := range g.store.latest() {
		if _, ok := g.usable(all, r); ok {
			past = append(past, r)
		}
	}
	slices.SortFunc(past, func(a, b roundRec) int { return a.When.Compare(b.When) }) // map order is random
	slices.SortStableFunc(past, func(a, b roundRec) int { return b.Gap - a.Gap })
	apart := slices.IndexFunc(past, roundRec.close)
	if apart < 0 {
		apart = len(past)
	}
	differ, rest := past[:apart], past[apart:]
	slices.SortFunc(rest, func(a, b roundRec) int { return a.When.Compare(b.When) })
	chosen := append(slices.Clone(differ[:min(len(differ), want/2)]), rest...)
	chosen = append(chosen, differ[min(len(differ), want/2):]...)
	chosen = chosen[:min(want, len(chosen))]
	if len(chosen) < minHistory {
		return nil, errHistory
	}
	slices.SortFunc(chosen, func(a, b roundRec) int { return a.When.Compare(b.When) })
	deck := make([]card, len(chosen))
	for i, r := range chosen {
		q, _ := g.usable(all, r)
		deck[i] = card{mode: r.Mode, q: q, before: &chosen[i]}
	}
	return deck, nil
}

// viewBefore is what the couple said the last time a question came up, in the order of the seats now.
type viewBefore struct {
	When string    `json:"when"` // "w marcu", "w marcu 2025"
	Said [2]string `json:"said"`
}

var monthsIn = []string{"w styczniu", "w lutym", "w marcu", "w kwietniu", "w maju", "w czerwcu", "w lipcu", "w sierpniu", "we wrześniu", "w październiku", "w listopadzie", "w grudniu"}
var monthsName = []string{"styczeń", "luty", "marzec", "kwiecień", "maj", "czerwiec", "lipiec", "sierpień", "wrzesień", "październik", "listopad", "grudzień"}

// whenText says when something was, the way people say it: "w marcu", or with the year when it was another one.
func whenText(t time.Time) string {
	s := monthsIn[t.Month()-1]
	if t.Year() != clock().Year() {
		s += " " + strconv.Itoa(t.Year())
	}
	return s
}

// beforeView is shown with a question asked again. Caller holds mu.
func (g *Game) beforeView(p *pair) *viewBefore {
	if g.before == nil || len(p.players) < 2 {
		return nil
	}
	return &viewBefore{whenText(g.before.When), g.before.align([2]string{p.players[0].Name, p.players[1].Name})}
}

// letter is one sentence to the two of them a year from now (letters.json); nobody sees it before then.
type letter struct {
	When   time.Time `json:"when"`
	Name   string    `json:"name"`
	Text   string    `json:"text"`
	Opened bool      `json:"opened"`
}

type viewLetter struct {
	When string `json:"when"`
	Name string `json:"name"`
	Text string `json:"text"`
}

func viewLetters(ls []letter) []viewLetter {
	out := []viewLetter{}
	for _, l := range ls {
		out = append(out, viewLetter{whenText(l.When), l.Name, l.Text})
	}
	return out
}

// writeLetter keeps one letter per person and evening: writing again replaces it. Caller holds mu.
func (g *Game) writeLetter(name, text string) error {
	text = strings.Join(strings.Fields(text), " ")
	if n := utf8.RuneCountInString(text); n < 1 || n > maxLetter {
		return ErrBad
	}
	s, now := g.store, clock()
	i := slices.IndexFunc(s.letters, func(l letter) bool { return l.Name == name && sameDay(l.When, now) })
	if i >= 0 {
		s.letters[i].Text = text
	} else {
		s.letters = append(s.letters, letter{When: now, Name: name, Text: text})
	}
	s.write("letters.json", s.letters)
	return nil
}

func sameDay(a, b time.Time) bool { return a.Format(time.DateOnly) == b.Format(time.DateOnly) }

// myLetter is the letter this person wrote tonight, so the phone can show it was kept.
func (s *store) myLetter(name string) string {
	now := clock()
	for _, l := range s.letters {
		if l.Name == name && sameDay(l.When, now) {
			return l.Text
		}
	}
	return ""
}

// due lists the letters waiting to be opened.
func (s *store) due() []int {
	var out []int
	for i, l := range s.letters {
		if !l.Opened && clock().Sub(l.When) >= letterAfter {
			out = append(out, i)
		}
	}
	return out
}

// openLetters marks the due letters as read and returns them, for the projector.
func (s *store) openLetters() []letter {
	var out []letter
	for _, i := range s.due() {
		s.letters[i].Opened = true
		out = append(out, s.letters[i])
	}
	s.write("letters.json", s.letters)
	return out
}

func (s *store) opened() []letter {
	return slices.DeleteFunc(slices.Clone(s.letters), func(l letter) bool { return !l.Opened })
}

// monthView sums up the previous month, shown in the menu until someone moves on.
type monthView struct {
	Key      string     `json:"key"`  // "2026-10"
	Name     string     `json:"name"` // "październik"
	Evenings int        `json:"evenings"`
	Games    int        `json:"games"`
	Compat   int        `json:"compat"`         // average, percent
	Best     *statLine  `json:"best,omitempty"` // the topic agreed on most that month
	Change   *statLine  `json:"change,omitempty"`
	Badges   []statLine `json:"badges"` // earned that month
}

// month returns the summary of the previous month, or nil when nothing was played then.
func (s *store) month() *monthView {
	now := clock()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	from, before := start.AddDate(0, -1, 0), start.AddDate(0, -2, 0)
	m := &monthView{Key: from.Format("2006-01"), Name: monthsName[from.Month()-1], Badges: []statLine{}}
	days, sum, counted := map[string]bool{}, 0, 0
	for _, r := range s.history {
		if r.When.Before(from) || !r.When.Before(start) {
			continue
		}
		m.Games++
		days[r.When.Format(time.DateOnly)] = true
		if c, ok := r.compat(); ok {
			sum, counted = sum+c, counted+1
		}
	}
	if m.Games == 0 {
		return nil
	}
	m.Evenings = len(days)
	if counted > 0 {
		m.Compat = sum / counted
	}
	// agreement per topic in the month and in the one before
	per := func(a, b time.Time) map[string]int {
		n, c := map[string]int{}, map[string]int{}
		for _, r := range s.rounds {
			if !r.When.Before(a) && r.When.Before(b) {
				n[r.Topic]++
				if r.close() {
					c[r.Topic]++
				}
			}
		}
		out := map[string]int{}
		for t, k := range n {
			if k >= minRounds {
				out[t] = c[t] * 100 / k
			}
		}
		return out
	}
	now1, prev := per(from, start), per(before, from)
	for _, t := range topics {
		v, ok := now1[t.ID]
		if !ok {
			continue
		}
		if m.Best == nil || v > m.Best.Value {
			m.Best = &statLine{Name: t.Name, Value: v}
		}
		if p, ok := prev[t.ID]; ok && v != p && (m.Change == nil || abs(v-p) > abs(m.Change.Value)) {
			m.Change = &statLine{Name: t.Name, Value: v - p}
		}
	}
	had := s.badgesAt(from)
	for i, b := range s.badgesAt(start) {
		if b.Value == 1 && had[i].Value == 0 {
			m.Badges = append(m.Badges, b)
		}
	}
	return m
}

func abs(n int) int { return max(n, -n) }

// migrate moves what is kept per question (asked, banned, rounds) to the new id of a question whose
// text was corrected: the pack lists its old ids in "prev".
func (s *store) migrate(all map[string][]question) {
	renamed := map[string]string{}
	for mode, qs := range all {
		for _, q := range qs {
			for _, old := range q.Prev {
				renamed[old] = q.id(mode)
			}
		}
	}
	moved := 0
	for old, id := range renamed {
		for _, set := range []map[string]bool{s.asked, s.banned} {
			if set[old] {
				delete(set, old)
				set[id] = true
				moved++
			}
		}
	}
	for i := range s.rounds {
		if id, ok := renamed[s.rounds[i].ID]; ok {
			s.rounds[i].ID = id
			moved++
		}
	}
	if moved > 0 {
		s.saveAsked()
		s.write("banned.json", sorted(s.banned))
		s.write("rounds.json", s.rounds)
	}
}
