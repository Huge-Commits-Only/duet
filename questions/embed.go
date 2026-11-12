// Package questions holds the built-in question packs, one JSON file per game mode.
package questions

import "embed"

// FS is the built-in packs. Extra packs can be added at runtime in <data dir>/questions.
//
//go:embed *.json
var FS embed.FS
