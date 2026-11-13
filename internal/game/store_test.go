package game

import (
	"slices"
	"strings"
	"testing"
)

// "Nie pokazuj więcej": the question is gone from every later game, and the ban goes into the backup.
func TestBan(t *testing.T) {
	g := started(t, "wolisz")
	id := g.cur.id("wolisz")
	if err := g.Act("b", "ban", "", ""); err != nil || !strings.Contains(string(g.view("a")), `"banned":true`) {
		t.Fatalf("ban: %v", err)
	}
	all(g, "quit")
	for range 30 { // whatever is drawn, the banned one never comes back
		all(g, "start")
		if slices.ContainsFunc(g.deck, func(c card) bool { return c.q.id("wolisz") == id }) {
			t.Fatal("a banned question was dealt again")
		}
		all(g, "quit")
	}
	g.Act("a", "config", "mode", "kto")
	g.Act("a", "start", "", "")
	for g.phase != "end" {
		all(g, "skip")
	}
	if err := g.Act("b", "again", "", ""); err != nil || g.phase != "question" || g.round != 1 || g.cfg["mode"] != "kto" {
		t.Fatalf("Jeszcze raz starts the same game again: %v phase=%s", err, g.phase)
	}
	if !New(g.store.dir).store.banned[id] {
		t.Fatal("bans must survive a restart")
	}
	data, _ := g.Backup("a")
	if !strings.Contains(string(data), id) {
		t.Fatal("bans belong in the backup")
	}
}
