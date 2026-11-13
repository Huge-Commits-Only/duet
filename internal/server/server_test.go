package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

// The whole HTTP surface, the way a projector and a phone use it.
func TestAPI(t *testing.T) {
	g := game.New(t.TempDir())
	srv := httptest.NewServer(Handler(g))
	defer srv.Close()
	noRedirect := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	do := func(method, path, body string) (int, string, http.Header) {
		t.Helper()
		req, _ := http.NewRequest(method, srv.URL+path, strings.NewReader(body))
		resp, err := noRedirect.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b := make([]byte, 1<<16)
		n, _ := resp.Body.Read(b)
		return resp.StatusCode, string(b[:n]), resp.Header
	}
	expect := func(what string, got, want int) {
		t.Helper()
		if got != want {
			t.Errorf("%s: status %d, want %d", what, got, want)
		}
	}

	// pages and assets
	code, _, h := do("GET", "/", "")
	expect("root redirects to the projector", code, http.StatusFound)
	if h.Get("Location") != "/screen" {
		t.Errorf("redirect to %q", h.Get("Location"))
	}
	for _, path := range []string{"/screen", "/play", "/static/style.css", "/static/common.js", "/static/icon-180.png", "/static/manifest.json", "/healthz"} {
		code, _, _ := do("GET", path, "")
		expect(path, code, http.StatusOK)
	}
	code, _, h = do("GET", "/qr.png", "")
	expect("qr", code, http.StatusOK)
	if h.Get("Content-Type") != "image/png" {
		t.Errorf("qr content type %q", h.Get("Content-Type"))
	}

	// joining needs the code from the QR link
	join := func(id, name, color, key string) int {
		body, _ := json.Marshal(map[string]any{"id": id, "name": name, "color": color, "key": key})
		code, _, _ := do("POST", "/api/join", string(body))
		return code
	}
	expect("join without the code", join("a", "Ola", "#ff00aa", ""), http.StatusForbidden)
	expect("join", join("a", "Ola", "#ff00aa", g.Key()), http.StatusNoContent)
	expect("join with bad data", join("b", "", "#ff00aa", g.Key()), http.StatusBadRequest)
	code, _, _ = do("POST", "/api/join", "not json")
	expect("join with a broken body", code, http.StatusBadRequest)

	// actions
	code, _, _ = do("POST", "/api/act", `{"id":"a","act":"config","key":"tone","value":"romantycznie"}`)
	expect("act", code, http.StatusNoContent)
	code, msg, _ := do("POST", "/api/act", `{"id":"a","act":"start"}`)
	expect("start alone", code, http.StatusBadRequest)
	if msg == "" {
		t.Error("a refused action must say why")
	}
	code, _, _ = do("POST", "/api/act", "not json")
	expect("act with a broken body", code, http.StatusBadRequest)

	// the live stream starts with the current state
	resp, err := http.Get(srv.URL + "/events?id=a")
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 4096)
	n, _ := resp.Body.Read(buf)
	resp.Body.Close()
	if resp.Header.Get("Content-Type") != "text/event-stream" || resp.Header.Get("X-Accel-Buffering") != "no" ||
		!strings.HasPrefix(string(buf[:n]), "data: {") || !strings.Contains(string(buf[:n]), `"romantycznie"`) {
		t.Errorf("event stream: %q", buf[:n])
	}

	// backup and restore
	code, _, _ = do("GET", "/api/backup?id=stranger", "")
	expect("backup for a stranger", code, http.StatusForbidden)
	code, backup, h := do("GET", "/api/backup?id=a", "")
	expect("backup", code, http.StatusOK)
	if !strings.Contains(h.Get("Content-Disposition"), "duet-kopia-") || !strings.Contains(backup, `"history"`) {
		t.Errorf("backup: %q %q", h.Get("Content-Disposition"), backup)
	}
	code, _, _ = do("POST", "/api/restore?id=a", backup)
	expect("restore", code, http.StatusNoContent)
	code, _, _ = do("POST", "/api/restore?id=a", `{"unrelated":true}`)
	expect("restore of a foreign file", code, http.StatusBadRequest)
}
