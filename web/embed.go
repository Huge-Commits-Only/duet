// Package web holds the projector and phone pages, embedded into the server binary.
package web

import "embed"

// FS is the pages and their assets, served as they are (there is no build step).
//
//go:embed *.html *.css *.js *.json *.svg *.png
var FS embed.FS
