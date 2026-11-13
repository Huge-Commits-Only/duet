package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRestartKeepsTheGame(t *testing.T) {
	g := started(t, "skala")
	dir := g.store.dir
	g.Act("a", "answer", "", "50")
	g.Act("b", "answer", "", "52")
	all(g, "next")
	g.Act("b", "answer", "", "30") // b already answered round 2 when the server goes down

	r := New(dir)
	if r.phase != "question" || r.round != 2 || r.cur.Q != g.cur.Q || r.points != [2]int{0, 3} || r.answers != [2]int{-1, 30} ||
		r.subj != 1 || r.key != g.key || r.stakes != g.stakes || r.seat("a") != 0 || r.seat("b") != 1 {
		t.Fatalf("restored: phase=%s round=%d points=%v answers=%v subj=%d", r.phase, r.round, r.points, r.answers, r.subj)
	}
	r.Subscribe("a")
	if err := r.Act("a", "answer", "", "31"); err != nil || r.phase != "reveal" || r.points != [2]int{3, 3} {
		t.Fatalf("the restored game must simply continue: %v phase=%s points=%v", err, r.phase, r.points)
	}

	// the join code: needed for a new phone, not for one that already has a seat
	if r.JoinAs("c", "Obcy", "#123456", "", "zły kod", 0) != ErrKey || r.JoinAs("a", "Natalia", "#ff00aa", "f", "", 0) != nil || r.players[0].Form != "f" {
		t.Fatal("join code must guard free seats only")
	}
	if r.JoinAs("a", "Natalia", "#ff00aa", "x", r.key, 0) != ErrBad {
		t.Fatal("unknown form must be rejected")
	}
	if strings.Contains(string(r.view("a")), r.key) || !strings.Contains(string(r.view("")), r.key) {
		t.Fatal("only the projector shows the join code")
	}

	// a drawing round interrupted by a restart counts as out of time; a broken file gives a fresh lobby
	d := started(t, "rysowanie")
	if r := New(d.store.dir); r.phase != "reveal" || r.hit {
		t.Fatalf("interrupted drawing: phase=%s", r.phase)
	}
	os.WriteFile(filepath.Join(dir, "game.json"), []byte(`{"Phase":"question","Round":7,"Deck":[]`), 0o644)
	if r := New(dir); r.phase != "lobby" || r.key == "" {
		t.Fatalf("broken state file: phase=%s", r.phase)
	}
}

// Statistics shown on the projector are still there after a restart.
func TestRestartKeepsTheStats(t *testing.T) {
	g := New(t.TempDir())
	g.Subscribe("a")
	g.join("a", "Natalia", "#ff0000")
	g.Act("a", "stats", "", "")
	if !New(g.store.dir).showStats {
		t.Fatal("the statistics must survive a restart")
	}
}
