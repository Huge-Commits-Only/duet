package game

import (
	"slices"
	"testing"
)

func TestBuiltinPacks(t *testing.T) {
	packs := loadQuestions(t.TempDir())
	// the same question twice in one mode and tone would come up twice in an evening
	for mode, qs := range packs {
		seen := map[string]bool{}
		for _, q := range qs {
			if key := q.Tone + "\x00" + q.id(mode); seen[key] {
				t.Errorf("%s/%s: duplicate question %q", mode, q.Tone, q.Q+q.A)
			} else {
				seen[key] = true
			}
		}
	}
	for _, mode := range menu["mode"] {
		if mode.ID == "mix" {
			continue // draws from the other modes' packs
		}
		for _, tone := range menu["tone"] {
			n := 0
			for _, q := range packs[mode.ID] {
				if q.Tone == tone.ID && q.valid(mode.ID) {
					n++
				}
			}
			if n < roundsFor["krotka"] {
				t.Errorf("%s/%s: %d valid questions, need at least %d", mode.ID, tone.ID, n, roundsFor["krotka"])
			}
			for level := 1; mode.ID == "rozmowy" && level <= 3; level++ {
				if !slices.ContainsFunc(packs[mode.ID], func(q question) bool { return q.Tone == tone.ID && q.Level == level }) {
					t.Errorf("rozmowy/%s: no card on level %d", tone.ID, level)
				}
			}
		}
	}
}
