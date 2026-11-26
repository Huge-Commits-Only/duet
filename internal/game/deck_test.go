package game

import (
	"testing"
)

func TestMix(t *testing.T) {
	g := started(t, "mix")
	if g.rounds != 8 {
		t.Fatalf("a short mix is 8 rounds, got %d", g.rounds)
	}
	modes, spotlight := map[string]int{}, [2]int{}
	for i, c := range g.deck {
		modes[c.mode]++
		if i%2 == 1 && c.mode != g.deck[i-1].mode {
			t.Fatalf("rounds come in pairs of the same mode: %d", i)
		}
	}
	if len(modes) != 4 {
		t.Fatalf("8 rounds are 4 modes, 2 rounds each: %v", modes)
	}
	// play it through with whatever each mode needs
	for g.phase != "end" {
		if s := g.subject(); s >= 0 {
			spotlight[s]++
		}
		ids := [2]string{"a", "b"}
		switch g.mode {
		case "wyzwania":
			g.Act(ids[g.subject()], "answer", "", "1")
			g.Act(ids[1-g.subject()], "answer", "", "0")
			continue
		case "blef":
			g.Act(ids[g.subject()], "answer", "0", "raz\ndwa\ntrzy")
			g.Act(ids[1-g.subject()], "answer", "", "0")
		case "telepatia":
			g.Act("a", "answer", "", "x")
			g.Act("b", "answer", "", "x")
		case "rysowanie":
			g.Act(ids[1-g.subject()], "answer", "", g.cur.Q)
		default:
			g.Act("a", "answer", "", "0")
			g.Act("b", "answer", "", "0")
		}
		if g.phase == "reveal" {
			all(g, "next")
		}
	}
	if spotlight[0] != spotlight[1] {
		t.Fatalf("spotlight must be shared equally: %v", spotlight)
	}
}

func TestMixedTone(t *testing.T) {
	g := New(t.TempDir())
	all := loadQuestions(t.TempDir())
	tones := func() map[string]bool {
		seen := map[string]bool{}
		for _, c := range g.cards("wolisz", all, 1000) {
			seen[c.q.Tone] = true
		}
		return seen
	}
	if got := tones(); len(got) != 1 || !got["lekko"] {
		t.Fatalf("a single tone draws only from it: %v", got)
	}
	g.cfg["tone"] = "mieszany"
	if got := tones(); len(got) != 3 {
		t.Fatalf("the mixed tone draws from all three: %v", got)
	}
}
