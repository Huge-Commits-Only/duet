package game

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"slices"
)

// store is what survives between evenings, as JSON files in dir:
// asked.json (ids of questions already asked) and yes.json (ideas both said yes to).
type store struct {
	dir     string
	asked   map[string]bool
	yes     []yesItem
	history []record        // finished games, oldest first (history.json)
	banned  map[string]bool // ids of questions the couple never wants to see again (banned.json)
}

// yesItem is one idea from "Wspólne TAK" on the couple's saved list.
type yesItem struct {
	Text string `json:"text"`
	Done bool   `json:"done"`
}

func openStore(dir string) *store {
	s := &store{dir: dir, asked: map[string]bool{}, banned: map[string]bool{}}
	// So own packs can be dropped in with `docker cp` (the scratch image has no shell to mkdir).
	if err := os.MkdirAll(filepath.Join(dir, "questions"), 0o755); err != nil {
		log.Printf("katalog danych %s jest niedostępny: %v", dir, err)
	}
	var ids []string
	s.read("asked.json", &ids)
	for _, id := range ids {
		s.asked[id] = true
	}
	s.read("history.json", &s.history)
	ids = nil
	s.read("banned.json", &ids)
	for _, id := range ids {
		s.banned[id] = true
	}
	// yes.json used to be a plain list of texts
	var raw []json.RawMessage
	s.read("yes.json", &raw)
	for _, r := range raw {
		var it yesItem
		if json.Unmarshal(r, &it.Text) == nil || json.Unmarshal(r, &it) == nil {
			s.yes = append(s.yes, it)
		}
	}
	return s
}

func (s *store) mark(id string) { s.asked[id] = true; s.saveAsked() }
func (s *store) forget(ids ...string) {
	for _, id := range ids {
		delete(s.asked, id)
	}
	s.saveAsked()
}

func (s *store) yesIndex(text string) int {
	return slices.IndexFunc(s.yes, func(it yesItem) bool { return it.Text == text })
}

func (s *store) addYes(text string) {
	if s.yesIndex(text) < 0 {
		s.yes = append(s.yes, yesItem{Text: text})
		s.write("yes.json", s.yes)
	}
}

// toggleYes ticks an idea off as done (or back); removeYes drops it. Both report whether it was on the list.
func (s *store) toggleYes(text string) bool {
	i := s.yesIndex(text)
	if i >= 0 {
		s.yes[i].Done = !s.yes[i].Done
		s.write("yes.json", s.yes)
	}
	return i >= 0
}

func (s *store) removeYes(text string) bool {
	i := s.yesIndex(text)
	if i >= 0 {
		s.yes = slices.Delete(s.yes, i, i+1)
		s.write("yes.json", s.yes)
	}
	return i >= 0
}

func (s *store) saveAsked() { s.write("asked.json", sorted(s.asked)) }

// ban drops a question from every future game.
func (s *store) ban(id string) {
	s.banned[id] = true
	s.write("banned.json", sorted(s.banned))
}

func sorted(set map[string]bool) []string {
	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

func (s *store) read(name string, v any) {
	if b, err := os.ReadFile(filepath.Join(s.dir, name)); err == nil {
		if err := json.Unmarshal(b, v); err != nil {
			log.Printf("uszkodzony plik %s, zaczynam od pustego: %v", name, err)
		}
	}
}

func (s *store) write(name string, v any) {
	b, _ := json.Marshal(v)
	tmp := filepath.Join(s.dir, name+".tmp")
	err := os.WriteFile(tmp, b, 0o644)
	if err == nil {
		err = os.Rename(tmp, filepath.Join(s.dir, name)) // atomic: never a half-written file
	}
	if err != nil {
		log.Printf("nie zapisano %s (gra działa dalej): %v", name, err)
	}
}
