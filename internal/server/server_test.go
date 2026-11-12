package server

import (
	"net/http/httptest"
	"testing"

	"duet/internal/game"
)

func TestBaseURL(t *testing.T) {
	r := httptest.NewRequest("GET", "http://duet.example.duckdns.org/qr.png", nil)
	if got := baseURL(r); got != "http://duet.example.duckdns.org" {
		t.Error(got)
	}
	r.Header.Set("X-Forwarded-Proto", "https")
	if got := baseURL(r); got != "https://duet.example.duckdns.org" {
		t.Error(got)
	}
	t.Setenv("PUBLIC_URL", "http://10.0.0.5:8080/")
	if got := baseURL(r); got != "http://10.0.0.5:8080" {
		t.Error(got)
	}
}

func TestHealth(t *testing.T) {
	w := httptest.NewRecorder()
	Handler(game.New(t.TempDir())).ServeHTTP(w, httptest.NewRequest("GET", "/healthz", nil))
	if w.Code != 200 {
		t.Fatalf("healthz: %d", w.Code)
	}
}
