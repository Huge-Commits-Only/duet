// Package server is the HTTP side of Duet: the pages, the live event stream
// and the endpoints the phones call. The game itself lives in internal/game.
package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"duet/internal/game"
	"duet/web"
	"strings"
	"time"

	qrcode "github.com/skip2/go-qrcode"
)

// app adds the HTTP handlers to a game.
type app struct{ *game.Game }

func (g app) handleEvents(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no") // nginx (Nginx Proxy Manager) must not buffer the stream
	ch := g.Subscribe(r.URL.Query().Get("id"))
	defer g.Unsubscribe(ch)
	ping := time.NewTicker(25 * time.Second) // keeps the proxy from timing out an idle stream
	defer ping.Stop()
	for {
		var err error
		select {
		case b := <-ch:
			_, err = fmt.Fprintf(w, "data: %s\n\n", b)
		case <-ping.C:
			_, err = io.WriteString(w, ": ping\n\n")
		case <-r.Context().Done():
			return
		}
		if err != nil {
			return // client went away
		}
		fl.Flush()
	}
}

func (g app) handleJoin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID, Name, Color, Form, Key string
		Pair                       int
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&in) != nil {
		http.Error(w, game.ErrBad.Error(), http.StatusBadRequest)
		return
	}
	switch err := g.JoinAs(in.ID, in.Name, in.Color, in.Form, in.Key, in.Pair); err {
	case nil:
		w.WriteHeader(http.StatusNoContent)
	case game.ErrKey:
		http.Error(w, err.Error(), http.StatusForbidden)
	case game.ErrFull, game.ErrColor:
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func (g app) handleAct(w http.ResponseWriter, r *http.Request) {
	var in struct{ ID, Act, Key, Value string }
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in) != nil {
		http.Error(w, game.ErrBad.Error(), http.StatusBadRequest)
		return
	}
	if err := g.Act(in.ID, in.Act, in.Key, in.Value); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// baseURL is the address phones should open: PUBLIC_URL, or whatever the screen was opened with.
func baseURL(r *http.Request) string {
	if u := os.Getenv("PUBLIC_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func (g app) handleQR(w http.ResponseWriter, r *http.Request) {
	png, err := qrcode.Encode(baseURL(r)+"/play?k="+g.Key(), qrcode.Medium, 512) // key never changes after start
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(png) // nothing to do if the client went away
}

// Handler serves one game: the pages, their assets and the API.
func Handler(gm *game.Game) http.Handler {
	g := app{gm}
	page := func(name string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) { http.ServeFileFS(w, r, web.FS, name) }
	}
	mux := http.NewServeMux()
	mux.Handle("GET /{$}", http.RedirectHandler("/screen", http.StatusFound))
	mux.HandleFunc("GET /screen", page("screen.html"))
	mux.HandleFunc("GET /play", page("play.html"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(web.FS)))
	mux.HandleFunc("GET /qr.png", g.handleQR)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "ok") })
	mux.HandleFunc("GET /events", g.handleEvents)
	mux.HandleFunc("POST /api/join", g.handleJoin)
	mux.HandleFunc("POST /api/act", g.handleAct)
	return mux
}
