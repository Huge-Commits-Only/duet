package game

import (
	"encoding/json"
	"errors"
	"slices"
)

// backup is everything Duet remembers between evenings, as one file to keep somewhere safe.
type backup struct {
	Asked   []string  `json:"asked"`   // questions already asked
	Yes     []yesItem `json:"yes"`     // the saved list of mutual yeses
	History []record  `json:"history"` // finished games
}

var errNotBackup = errors.New("to nie jest plik kopii zapasowej Duet")

// Backup returns the saved data as JSON. Only a seated player may take it, and only with no guests.
func (g *Game) Backup(id string) ([]byte, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if p, _ := g.find(id); p != g.pairs[0] || !g.home() { // household data, not for guests
		return nil, ErrBad
	}
	b := backup{Asked: []string{}, Yes: g.store.yes, History: g.store.history}
	for q := range g.store.asked {
		b.Asked = append(b.Asked, q)
	}
	slices.Sort(b.Asked)
	return json.MarshalIndent(b, "", "  ")
}

// Restore replaces the saved data with a backup. Only from the menu, by a seated player, with no guests.
func (g *Game) Restore(id string, data []byte) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if p, _ := g.find(id); p != g.pairs[0] || !g.home() || g.phase != "lobby" {
		return ErrBad
	}
	var raw map[string]json.RawMessage
	var b backup
	// all three lists must be there: an unrelated JSON file must not wipe everything
	if json.Unmarshal(data, &raw) != nil || raw["asked"] == nil || raw["yes"] == nil || raw["history"] == nil || json.Unmarshal(data, &b) != nil {
		return errNotBackup
	}
	s := g.store
	s.asked = map[string]bool{}
	for _, q := range b.Asked {
		s.asked[q] = true
	}
	s.yes, s.history = b.Yes, b.History
	s.saveAsked()
	s.write("yes.json", s.yes)
	s.write("history.json", s.history)
	g.broadcast()
	return nil
}
