package game

import (
	"slices"
	"strconv"
	"time"
)

// record is one finished game in history.json.
type record struct {
	When   time.Time    `json:"when"`
	Mode   string       `json:"mode"`
	Tone   string       `json:"tone"`
	Rounds int          `json:"rounds"`
	Pairs  []recordPair `json:"pairs"`
}

type recordPair struct {
	Names   []string `json:"names"`
	Matches int      `json:"matches"`
	Points  [2]int   `json:"points"`
	Score   int      `json:"score"`
}

// compat is how well the couples agreed in this game, in percent; ok is false where that means nothing.
func (r record) compat() (percent int, ok bool) {
	if r.Mode == "rozmowy" || r.Rounds == 0 || len(r.Pairs) == 0 {
		return 0, false
	}
	sum := 0
	for _, p := range r.Pairs {
		sum += p.Matches
	}
	return sum * 100 / (r.Rounds * len(r.Pairs)), true
}

// winner names who won, or "" for a draw and for games without a winner.
func (r record) winner() string {
	if len(r.Pairs) == 1 {
		p := r.Pairs[0]
		if p.Points[0] == p.Points[1] {
			return ""
		}
		if p.Points[0] > p.Points[1] {
			return p.Names[0]
		}
		return p.Names[1]
	}
	best, name := -1, ""
	for _, p := range r.Pairs {
		switch {
		case p.Score > best:
			best, name = p.Score, p.Names[0]+" i "+p.Names[1]
		case p.Score == best:
			name = "" // shared first place
		}
	}
	return name
}

// record appends the game that just ended. Called with g's lock held.
func (s *store) record(g *Game) {
	if g.hasBots() {
		return // a try-out, not an evening
	}
	r := record{When: time.Now(), Mode: g.cfg["mode"], Tone: g.cfg["tone"], Rounds: g.rounds}
	for _, p := range g.playing {
		r.Pairs = append(r.Pairs, recordPair{[]string{p.players[0].Name, p.players[1].Name}, p.matches, p.points, p.score})
	}
	s.history = append(s.history, r)
	s.write("history.json", s.history)
}

// stats is what the projector shows about the evenings so far.
type stats struct {
	Games    int        `json:"games"`
	Evenings int        `json:"evenings"` // distinct days with a finished game
	Compat   int        `json:"compat"`   // average, percent
	Series   []int      `json:"series"`   // compatibility of the latest games, oldest first
	Modes    []statLine `json:"modes"`    // average compatibility per game, most played first
	Wins     []statLine `json:"wins"`     // most wins first
	Recent   []statLine `json:"recent"`   // latest games, newest first
}

type statLine struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
	Note  string `json:"note,omitempty"`
}

func (s *store) stats() *stats {
	st := &stats{Games: len(s.history), Series: []int{}}
	days, total, counted := map[string]bool{}, 0, 0
	type agg struct{ games, sum, counted int }
	perMode, wins := map[string]*agg{}, map[string]int{}
	modeName := func(id string) string {
		for _, o := range menu["mode"] {
			if o.ID == id {
				return o.Name
			}
		}
		return id
	}
	for _, r := range s.history {
		days[r.When.Format(time.DateOnly)] = true
		a := perMode[r.Mode]
		if a == nil {
			a = &agg{}
			perMode[r.Mode] = a
		}
		a.games++
		if c, ok := r.compat(); ok {
			total, counted = total+c, counted+1
			a.sum, a.counted = a.sum+c, a.counted+1
			st.Series = append(st.Series, c)
		}
		if w := r.winner(); w != "" {
			wins[w]++
		}
	}
	st.Evenings = len(days)
	if counted > 0 {
		st.Compat = total / counted
	}
	st.Series = st.Series[max(0, len(st.Series)-20):]
	for id, a := range perMode {
		l := statLine{Name: modeName(id), Note: strconv.Itoa(a.games) + " ×"}
		if a.counted > 0 {
			l.Value = a.sum / a.counted
		}
		st.Modes = append(st.Modes, l)
	}
	slices.SortFunc(st.Modes, func(a, b statLine) int { return len(b.Note) - len(a.Note) + cmpStr(b.Note, a.Note) })
	for name, n := range wins {
		st.Wins = append(st.Wins, statLine{Name: name, Value: n})
	}
	slices.SortFunc(st.Wins, func(a, b statLine) int { return 2*(b.Value-a.Value) + cmpStr(a.Name, b.Name) })
	for i := len(s.history) - 1; i >= 0 && len(st.Recent) < 6; i-- {
		r := s.history[i]
		l := statLine{Name: r.When.Format("02.01") + " · " + modeName(r.Mode), Note: r.winner()}
		if c, ok := r.compat(); ok {
			l.Value = c
		}
		st.Recent = append(st.Recent, l)
	}
	return st
}

func cmpStr(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}
