package game

import (
	"strings"
	"testing"
)

// With guests at the table nobody can show, change, download or overwrite the household's saved data.
func TestGuestsCannotTouchHouseholdData(t *testing.T) {
	g := New(t.TempDir())
	g.store.addYes("Kolacja przy świecach")
	sa := g.Subscribe("a")
	g.join("a", "Natalia", "#ff0000")
	if g.Act("a", "list", "", "") != nil || !strings.Contains(string(g.view("")), "Kolacja") {
		t.Fatal("alone at home the list shows")
	}
	g.Subscribe("c")
	if g.JoinAs("c", "Gość", "#0000ff", "", g.key, 1) != nil {
		t.Fatal("a guest joins")
	}
	if strings.Contains(string(g.view("")), "Kolacja") || strings.Contains(string(g.view("c")), "Kolacja") {
		t.Fatal("the list must disappear once a guest sits down")
	}
	for _, act := range []string{"list", "stats", "yesdone", "yesdel"} {
		if g.Act("c", act, "", "Kolacja przy świecach") != ErrBad {
			t.Errorf("a guest can %s", act)
		}
	}
	if _, err := g.Backup("c"); err != ErrBad {
		t.Error("a guest can download the backup")
	}
	if g.Restore("a", []byte(`{"asked":[],"yes":[],"history":[]}`)) != ErrBad || len(g.store.yes) != 1 {
		t.Error("the backup can be restored with guests around")
	}
	// the household's phones go dark: the guest is alone at the table, and still a guest
	g.Unsubscribe(sa)
	if _, err := g.Backup("c"); err != ErrBad || g.Act("c", "list", "", "") != ErrBad {
		t.Error("a guest alone at the table gets the household's data")
	}
	if strings.Contains(string(g.view("")), "Kolacja") {
		t.Error("the list stays on the projector for a guest")
	}
}
