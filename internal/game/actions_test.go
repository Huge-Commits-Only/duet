package game

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

// A second tap carrying the step of a screen that is already gone does nothing.
func TestLateTapIsDropped(t *testing.T) {
	twice := func(g *Game, id, act, value string) {
		t.Helper()
		step := g.step()
		for range 2 {
			if err := g.ActAt(step, id, act, "", value); err != nil {
				t.Fatal(err)
			}
		}
	}
	g := started(t, "tak")
	g.Act("a", "answer", "", "1")
	twice(g, "b", "answer", "0")
	if g.round != 2 || g.answers[1] != -1 {
		t.Fatalf("tak: the second yes landed on the next card: round=%d answers=%v", g.round, g.answers)
	}
	g = started(t, "wyzwania")
	g.Act("a", "answer", "", "0")
	twice(g, "b", "answer", "0")
	if g.round != 2 || g.pending == nil {
		t.Fatalf("wyzwania: the judge's second tap picked truth for them: round=%d", g.round)
	}
	g = started(t, "rozmowy")
	twice(g, "a", "next", "")
	if g.round != 2 {
		t.Fatalf("rozmowy: a card was skipped: round=%d", g.round)
	}
	g = started(t, "wolisz")
	g.Act("a", "answer", "", "0")
	g.Act("b", "answer", "", "0")
	step := g.step()
	g.ActAt(step, "a", "next", "", "")
	if err := g.ActAt(step, "b", "next", "", ""); err != nil || g.round != 2 || g.phase != "question" {
		t.Fatalf("both tapping next: err=%v round=%d phase=%s", err, g.round, g.phase)
	}
	twice(g, "a", "skip", "") // proposed twice: still waiting for b
	if g.round != 2 || g.vote == nil {
		t.Fatalf("a skip needs everyone's yes: round=%d", g.round)
	}
	twice(g, "b", "skip", "")
	if g.round != 3 {
		t.Fatalf("skip twice skipped two rounds: round=%d", g.round)
	}
	g = started(t, "blef")
	step = g.step()
	for range 2 { // the second "Gotowe" must not become a guess at the statements
		if err := g.ActAt(step, "a", "answer", "1", "Raz\nDwa\nTrzy"); err != nil {
			t.Fatalf("blef: a double tap on Gotowe shows an error: %v", err)
		}
	}
}

// After a reveal everyone taps "Dalej"; skipping and quitting are votes. Dropped phones hold no one up.
func TestEveryoneMovesOn(t *testing.T) {
	g := New(t.TempDir())
	g.Subscribe("a")
	sb := g.Subscribe("b")
	g.join("a", "Natalia", "#ff0000")
	g.join("b", "Kuba", "#00ff00")
	g.Act("a", "config", "mode", "wolisz")
	g.Act("a", "start", "", "")
	g.Act("a", "answer", "", "0")
	g.Act("b", "answer", "", "0")
	g.Act("a", "next", "", "")
	if g.phase != "reveal" || !strings.Contains(string(g.view("")), `"ready":true`) {
		t.Fatal("one tap of Dalej waits for the other, and the projector shows who is ready")
	}
	g.Act("b", "next", "", "")
	if g.phase != "question" || g.round != 2 {
		t.Fatalf("both tapped Dalej: phase=%s round=%d", g.phase, g.round)
	}

	g.Act("a", "skip", "", "")
	var v struct{ Vote viewVote }
	json.Unmarshal(g.view("b"), &v)
	if g.round != 2 || v.Vote.Kind != "skip" || v.Vote.By != "Natalia" || v.Vote.Agreed || !slices.Equal(v.Vote.Waiting, []string{"Kuba"}) {
		t.Fatalf("a skip is a proposal the partner sees: round=%d vote=%+v", g.round, v.Vote)
	}
	g.Act("b", "no", "", "")
	if g.vote != nil || g.round != 2 {
		t.Fatal("no cancels the proposal")
	}
	g.Act("a", "quit", "", "")
	g.Act("b", "quit", "", "")
	if g.phase != "lobby" {
		t.Fatal("both agreed to end the game")
	}

	g.Act("a", "start", "", "")
	g.Act("a", "skip", "", "")
	g.Unsubscribe(sb) // the partner's phone dies before saying yes
	if g.round != 2 {
		t.Fatalf("a dropped phone does not block a vote: round=%d", g.round)
	}
}
