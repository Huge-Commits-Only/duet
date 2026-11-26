package game

import (
	"encoding/json"
	"slices"
	"strconv"
	"testing"
	"time"
)

func TestProfile(t *testing.T) {
	now := time.Date(2026, 11, 26, 21, 0, 0, 0, time.UTC)
	old, recent := now.AddDate(0, 0, -40), now.AddDate(0, 0, -1)
	r := func(when time.Time, topic string, gap int, text string) roundRec {
		return roundRec{when, topic, gap, text}
	}
	s := &store{rounds: []roundRec{
		// uczucia: far apart a month ago, close lately
		r(old, "uczucia", 100, "a"), r(old, "uczucia", 100, "b"), r(old, "uczucia", 40, "c"),
		r(recent, "uczucia", 0, "d"), r(recent, "uczucia", 10, "e"), r(recent, "uczucia", 15, "f"),
		// seks: only two rounds, too few to say anything
		r(recent, "seks", 0, "g"), r(recent, "seks", 100, "h"),
		// the latest evening: three differences, the widest first in "Do pogadania"
		r(now, "podroze", 0, "i"), r(now, "podroze", 20, "j"), r(now, "podroze", 100, "k"), r(now, "podroze", 60, "l"), r(now, "podroze", 30, "m"),
	}}
	p := s.profile(now)
	want := []statLine{{Name: "💞 Uczucia", Value: 50, Note: "▲ 100"}, {Name: "✈️ Podróże", Value: 20}}
	if !slices.Equal(p.Topics, want) {
		t.Errorf("topics = %+v, want %+v", p.Topics, want)
	}
	if !slices.Equal(p.Talk, []string{"k", "l", "m"}) {
		t.Errorf("talk = %v, want the widest gaps of the latest evening", p.Talk)
	}
	earned := func(p *profile, name string) bool {
		i := slices.IndexFunc(p.Badges, func(b statLine) bool { return b.Name == name })
		return i >= 0 && p.Badges[i].Value == 1
	}
	// d, e, f, g: four close answers in a row; i is close too, but on another evening
	if earned(p, "🔥 Seria 5") {
		t.Error("a run must not carry over to the next evening")
	}
	s.rounds = append(s.rounds, r(now, "seks", 0, "n"), r(now, "seks", 5, "o"), r(now, "seks", 15, "p"), r(now, "seks", 0, "q"), r(now, "seks", 0, "s"))
	if p := s.profile(now); !earned(p, "🔥 Seria 5") || earned(p, "⚡ Seria 10") {
		t.Errorf("badges = %+v, want Seria 5 only", p.Badges)
	}
}

// A game of one couple notes every round: the profile keeps them, the end shows where the two differed most.
func TestGameKeepsRounds(t *testing.T) {
	g := started(t, "skala")
	gaps := []int{5, 40, 15, 70}
	for i := 0; g.phase == "question"; i++ {
		g.Act("a", "answer", "", "20")
		g.Act("b", "answer", "", strconv.Itoa(20+gaps[i%len(gaps)]))
		all(g, "next")
	}
	if g.phase != "end" || len(g.store.rounds) != g.rounds || len(g.talk) != g.rounds {
		t.Fatalf("phase %s, %d rounds kept of %d", g.phase, len(g.store.rounds), g.rounds)
	}
	for _, r := range g.store.rounds {
		if !slices.ContainsFunc(topics, func(o opt) bool { return o.ID == r.Topic }) || r.Text == "" {
			t.Fatalf("round without a topic or text: %+v", r)
		}
	}
	var v struct{ Talk []string }
	json.Unmarshal(g.view(""), &v)
	if len(v.Talk) != 3 || v.Talk[0] != g.talk[3].Text {
		t.Errorf("talk = %v, want the 70-apart round first", v.Talk)
	}

	// the profile is shown on the projector from the menu, like the statistics
	all(g, "quit")
	if err := g.Act("a", "profile", "", ""); err != nil {
		t.Fatal(err)
	}
	var lobby struct {
		HasProfile bool
		Profile    *profile
	}
	json.Unmarshal(g.view(""), &lobby)
	if !lobby.HasProfile || lobby.Profile == nil || lobby.Profile.Rounds != g.rounds || len(lobby.Profile.Badges) != 6 {
		t.Fatalf("profile view: %+v", lobby)
	}

	// and it travels with the backup
	data, err := g.Backup("a")
	if err != nil {
		t.Fatal(err)
	}
	kept := len(g.store.rounds)
	g.store.rounds = nil
	if err := g.Restore("a", data); err != nil || len(g.store.rounds) != kept {
		t.Fatalf("restore: %v, %d rounds of %d", err, len(g.store.rounds), kept)
	}
}

// Every question where both answer about the same thing has a known topic.
func TestPacksHaveTopics(t *testing.T) {
	for mode, qs := range loadQuestions(t.TempDir()) {
		if _, ok := (&Game{mode: mode, cur: question{Max: 1}}).gap(&pair{}); !ok {
			continue
		}
		for _, q := range qs {
			if !slices.ContainsFunc(topics, func(o opt) bool { return o.ID == q.Topic }) {
				t.Errorf("%s: %q has topic %q", mode, q.Q+q.A, q.Topic)
			}
		}
	}
}
