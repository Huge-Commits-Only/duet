package game

import (
	"slices"
	"strconv"
	"time"
	"unicode"
)

// The couple's profile: where the two of them agree and where they differ, topic by topic,
// across the evenings. Built from every answered round of the household (rounds.json).

// topics of the questions both answer about the same thing; the order of the menu.
var topics = []opt{
	{"codziennosc", "🏠 Codzienność"}, {"podroze", "✈️ Podróże"}, {"pieniadze", "💰 Pieniądze i praca"},
	{"przyszlosc", "🔮 Przyszłość"}, {"rozrywka", "🎉 Rozrywka i znajomi"}, {"uczucia", "💞 Uczucia"},
	{"charakter", "🧠 Charakter"}, {"seks", "🔥 Seks"},
}

const (
	closeGap  = 15 // a gap up to this counts as agreeing, as in "skala"
	minRounds = 3  // fewer rounds of a topic say nothing yet
	trendDays = 30 // the trend compares the latest month with everything before it
)

// roundRec is one answered round: its topic, how far apart the two answers were (0-100) and the question.
type roundRec struct {
	When  time.Time `json:"when"`
	Topic string    `json:"topic,omitempty"`
	Gap   int       `json:"gap"`
	Text  string    `json:"text"`
}

func (r roundRec) close() bool { return r.Gap <= closeGap }

// gap says how far apart the couple's answers were, 0 (the same) to 100; ok is false in modes
// where the two do not answer the same question (drawing, dares, bluffs) or answer freely (telepatia).
func (g *Game) gap(p *pair) (gap int, ok bool) {
	a, b := p.answers[0], p.answers[1]
	switch g.mode {
	case "skala", "szacowanie":
		d := max(a-b, b-a)
		if g.mode == "szacowanie" {
			d = d * 100 / g.cur.Max
		}
		return min(d, 100), true
	case "wolisz", "kto", "nigdy", "znasz", "tak":
		if a == b {
			return 0, true
		}
		return 100, true
	}
	return 0, false
}

// talkText is the question as the projector would put it, for "Do pogadania".
func (g *Game) talkText(p *pair) string {
	q := g.cur.gendered(g.forms(p, -1))
	if g.mode == "wolisz" { // "Góry czy morze?": the second choice continues the sentence, unless it starts with a name
		r := []rune(q.B)
		if len(r) > 1 && unicode.IsLower(r[1]) {
			r[0] = unicode.ToLower(r[0])
		}
		return q.A + " czy " + string(r) + "?"
	}
	return q.Q
}

// note keeps a scored round of a game played by one couple: for the end of this game
// and, when it is the household without bots, for the profile. Caller holds mu.
func (g *Game) note(p *pair) {
	gap, ok := g.gap(p)
	if !ok || len(g.playing) != 1 || g.hasBots() {
		return
	}
	r := roundRec{When: time.Now(), Topic: g.cur.Topic, Gap: gap, Text: g.talkText(p)}
	g.talk = append(g.talk, r)
	if g.home() {
		g.store.rounds = append(g.store.rounds, r)
		g.store.write("rounds.json", g.store.rounds)
	}
}

// widest returns the texts of up to n rounds where the answers differed the most, the widest first.
func widest(rounds []roundRec, n int) []string {
	apart := slices.DeleteFunc(slices.Clone(rounds), roundRec.close)
	slices.SortStableFunc(apart, func(a, b roundRec) int { return b.Gap - a.Gap })
	out := []string{}
	for _, r := range apart[:min(n, len(apart))] {
		out = append(out, r.Text)
	}
	return out
}

// profile is what the projector shows about the couple.
type profile struct {
	Rounds int        `json:"rounds"` // rounds counted so far
	Topics []statLine `json:"topics"` // agreement per topic in percent, best first; Note: the trend, e.g. "▲ 12"
	Talk   []string   `json:"talk"`   // the widest differences of the latest evening
	Badges []statLine `json:"badges"` // Value 1 = earned; Note: what it takes
}

func (s *store) profile(now time.Time) *profile {
	pr := &profile{Rounds: len(s.rounds), Topics: []statLine{}}
	since := now.AddDate(0, 0, -trendDays)
	type agg struct{ n, close, newN, newClose int }
	per := map[string]*agg{}
	for _, r := range s.rounds {
		a := per[r.Topic]
		if a == nil {
			a = &agg{}
			per[r.Topic] = a
		}
		c := 0
		if r.close() {
			c = 1
		}
		a.n, a.close = a.n+1, a.close+c
		if r.When.After(since) {
			a.newN, a.newClose = a.newN+1, a.newClose+c
		}
	}
	for _, t := range topics {
		a := per[t.ID]
		if a == nil || a.n < minRounds {
			continue
		}
		l := statLine{Name: t.Name, Value: a.close * 100 / a.n}
		if oldN := a.n - a.newN; a.newN >= minRounds && oldN >= minRounds {
			switch d := a.newClose*100/a.newN - (a.close-a.newClose)*100/oldN; {
			case d >= 5:
				l.Note = "▲ " + strconv.Itoa(d)
			case d <= -5:
				l.Note = "▼ " + strconv.Itoa(-d)
			}
		}
		pr.Topics = append(pr.Topics, l)
	}
	slices.SortStableFunc(pr.Topics, func(a, b statLine) int { return b.Value - a.Value })
	if n := len(s.rounds); n > 0 {
		day := s.rounds[n-1].When.Format(time.DateOnly)
		first := n
		for first > 0 && s.rounds[first-1].When.Format(time.DateOnly) == day {
			first--
		}
		pr.Talk = widest(s.rounds[first:], 3)
	}
	pr.Badges = s.badges()
	return pr
}

// badges are small goals that give a reason to come back; all are listed, the earned ones lit.
func (s *store) badges() []statLine {
	best, run, day := 0, 0, ""
	seen := map[string]bool{}
	for _, r := range s.rounds {
		if d := r.When.Format(time.DateOnly); d != day { // a run lasts one evening
			day, run = d, 0
		}
		if r.close() {
			run++
			best = max(best, run)
		} else {
			run = 0
		}
		seen[r.Topic] = true
	}
	allTopics := !slices.ContainsFunc(topics, func(t opt) bool { return !seen[t.ID] })
	perfect, days, played := false, map[string]bool{}, map[string]bool{}
	for _, r := range s.history {
		days[r.When.Format(time.DateOnly)] = true
		played[r.Mode] = true
		c, ok := r.compat()
		perfect = perfect || ok && c == 100 && r.Rounds >= 5
	}
	allModes := !slices.ContainsFunc(menu["mode"], func(o opt) bool { return !played[o.ID] })
	b := func(name, note string, earned bool) statLine {
		l := statLine{Name: name, Note: note}
		if earned {
			l.Value = 1
		}
		return l
	}
	return []statLine{
		b("🔥 Seria 5", "5 zgodnych odpowiedzi z rzędu", best >= 5),
		b("⚡ Seria 10", "10 zgodnych odpowiedzi z rzędu", best >= 10),
		b("💯 Komplet", "gra bez jednej różnicy", perfect),
		b("🌙 10 wieczorów", "10 wieczorów z Duetem", len(days) >= 10),
		b("🧭 Wszystkie tematy", "pytanie z każdego tematu", allTopics),
		b("🎮 Wszystkie gry", "każdy tryb choć raz", allModes),
	}
}
