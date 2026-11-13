package game

import (
	"strings"
	"testing"
)

func TestHistoryAndStats(t *testing.T) {
	g := started(t, "skala")
	dir := g.store.dir
	all(g, "quit")
	if len(g.store.history) != 0 {
		t.Fatal("an abandoned game is not recorded")
	}
	for range 2 { // two finished games: b guesses seat 0's number exactly in round 1, the rest is skipped
		all(g, "quit") // back to the menu after the previous game
		g.Act("a", "start", "", "")
		g.Act("a", "answer", "", "40")
		g.Act("b", "answer", "", "40")
		all(g, "next")
		for g.phase == "question" {
			all(g, "skip")
		}
	}
	st := openStore(dir).stats() // from disk: what the next evening will see
	if st.Games != 2 || st.Evenings != 1 || len(st.Series) != 2 || st.Series[0] != 12 || st.Compat != 12 ||
		len(st.Wins) != 1 || st.Wins[0] != (statLine{Name: "Jan", Value: 2}) || st.Modes[0].Name != "Skala" || len(st.Recent) != 2 {
		t.Fatalf("stats: %+v", st)
	}
	all(g, "quit")
	if strings.Contains(string(g.view("")), `"stats"`) || g.Act("b", "stats", "", "") != nil || !strings.Contains(string(g.view("")), `"stats"`) {
		t.Fatal("stats are sent only while a phone asked to show them")
	}
	g.Act("a", "list", "", "")
	if g.showStats {
		t.Fatal("the list and the stats replace each other on the projector")
	}
}
