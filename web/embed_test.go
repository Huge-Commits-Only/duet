package web

import (
	"io/fs"
	"testing"
)

// Every page and asset the server hands out is in the binary (a new file type needs the embed pattern).
func TestFS(t *testing.T) {
	for _, name := range []string{"screen.html", "play.html", "common.js", "style.css", "manifest.json", "icon.svg", "icon-180.png", "icon-512.png"} {
		if b, err := fs.ReadFile(FS, name); err != nil || len(b) == 0 {
			t.Errorf("%s is not embedded: %v", name, err)
		}
	}
}
