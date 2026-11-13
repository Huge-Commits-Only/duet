package game

import (
	"crypto/sha256"
	"duet/questions"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

// question fields are used per mode:
//   - wolisz: a, b (the two choices)
//   - kto:    q ("Kto prędzej…?"; the choices are the two players)
//   - znasz:  q + 2-4 options (asked about one player, in second person)
//   - skala:  q, optionally a and b as labels for 0 and 100
//   - szacowanie: q (a number about one player, in second person) + max (top of the range, 1-1000),
//     optionally a and b as labels for 0 and max
//   - nigdy:  q (a short phrase: something one has done or never did)
//   - telepatia: q (the word to associate with)
//   - tak:    q (an idea both can say yes or no to)
//   - rozmowy: q + level 1-3 (depth)
//   - wyzwania: q + kind "prawda" (a question to answer honestly) or "wyzwanie" (a task)
//   - blef:   q (a theme for two truths and a lie)
//   - rysowanie: q (the word to draw; one word, it is compared with what the partner types),
//     optionally alt: other words that count as a correct guess
type question struct {
	Tone    string   `json:"tone"`
	Q       string   `json:"q,omitempty"`
	A       string   `json:"a,omitempty"`
	B       string   `json:"b,omitempty"`
	Options []string `json:"options,omitempty"`
	Level   int      `json:"level,omitempty"`
	Kind    string   `json:"kind,omitempty"`
	Alt     []string `json:"alt,omitempty"`
	Max     int      `json:"max,omitempty"`
}

// valid reports whether the question has what its mode needs, so a broken hand-written pack cannot crash a game.
func (q question) valid(mode string) bool {
	m, ok := modeByID[mode]
	return ok && m.valid(q)
}

// id is derived from the text, so hand-written packs need no ids.
func (q question) id(mode string) string {
	sum := sha256.Sum256([]byte(mode + "\x00" + q.Q + "\x00" + q.A + "\x00" + q.B))
	return hex.EncodeToString(sum[:8])
}

// loadQuestions reads the built-in packs plus any in <dir>/questions/*.json.
// File format: {"mode":"wolisz","items":[{"tone":"lekko","a":"...","b":"..."}]}; fields per mode: see question.
func loadQuestions(dir string) map[string][]question {
	out := map[string][]question{}
	add := func(fsys fs.FS) {
		names, _ := fs.Glob(fsys, "*.json")
		for _, n := range names {
			var p struct {
				Mode  string
				Items []question
			}
			b, err := fs.ReadFile(fsys, n)
			if err == nil {
				err = json.Unmarshal(b, &p)
			}
			if err != nil {
				log.Printf("pomijam paczkę pytań %s: %v", n, err)
				continue
			}
			out[p.Mode] = append(out[p.Mode], p.Items...)
		}
	}
	add(questions.FS)
	add(os.DirFS(filepath.Join(dir, "questions")))
	return out
}
