package game

import (
	"encoding/json"
	"testing"
)

// A backup file comes from the user's disk: whatever it holds, Restore must not panic,
// and anything it accepts must survive another Backup/Restore round trip.
func FuzzRestore(f *testing.F) {
	f.Add([]byte(`{"asked":[],"yes":[],"history":[]}`))
	f.Add([]byte(`{"asked":["q"],"yes":[{"text":"Kolacja"}],"history":[{}],"banned":["x"]}`))
	f.Add([]byte(`{"asked":null,"yes":null,"history":null}`))
	f.Add([]byte(`[]`))
	f.Fuzz(func(t *testing.T, data []byte) {
		g := New(t.TempDir())
		g.Subscribe("a")
		g.join("a", "Natalia", "#ff0000")
		if g.Restore("a", data) != nil {
			return
		}
		b, err := g.Backup("a")
		if err != nil {
			t.Fatalf("backup after an accepted restore: %v", err)
		}
		if err := g.Restore("a", b); err != nil {
			t.Fatalf("own backup rejected: %v\n%s", err, b)
		}
	})
}

// Every tap comes from a phone: any action, key and value in any mode must leave a game
// that still renders for the projector and both phones.
func FuzzAct(f *testing.F) {
	for _, s := range [][3]string{
		{"answer", "", "0"}, {"answer", "", "1"}, {"next", "", ""}, {"skip", "", ""}, {"no", "", ""},
		{"draw", "", "1,2,3,4"}, {"clear", "", ""}, {"undo", "", ""}, {"stake", "", "5"}, {"giveup", "", ""},
		{"quit", "", ""}, {"again", "", ""}, {"config", "tone", "pikantnie"}, {"bot", "", ""}, {"kick", "", "b"},
	} {
		f.Add(uint8(0), s[0], s[1], s[2], true)
	}
	f.Fuzz(func(t *testing.T, mode uint8, act, key, value string, byA bool) {
		g := New(t.TempDir())
		g.Subscribe("a")
		g.Subscribe("b")
		g.join("a", "Natalia", "#ff0000")
		g.join("b", "Kuba", "#00ff00")
		modes := menu["mode"]
		g.Act("a", "config", "mode", modes[int(mode)%len(modes)].ID)
		g.Act("a", "start", "", "")
		id := "a"
		if !byA {
			id = "b"
		}
		g.Act(id, act, key, value)
		g.Act("a", act, key, value) // the same tap twice, as on a double tap
		for _, who := range []string{"", "a", "b"} {
			if !json.Valid(g.view(who)) {
				t.Fatalf("view %q is not JSON after %q %q %q", who, act, key, value)
			}
		}
	})
}
