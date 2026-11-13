package questions

import (
	"encoding/json"
	"io/fs"
	"strings"
	"testing"
)

// Every embedded pack is valid JSON named after its mode, with questions in it.
func TestFS(t *testing.T) {
	names, _ := fs.Glob(FS, "*.json")
	if len(names) < 12 {
		t.Fatalf("only %d packs embedded", len(names))
	}
	for _, n := range names {
		var p struct {
			Mode  string
			Items []json.RawMessage
		}
		b, _ := fs.ReadFile(FS, n)
		if err := json.Unmarshal(b, &p); err != nil || p.Mode != strings.TrimSuffix(n, ".json") || len(p.Items) == 0 {
			t.Errorf("%s: mode %q, %d items, %v", n, p.Mode, len(p.Items), err)
		}
	}
}
