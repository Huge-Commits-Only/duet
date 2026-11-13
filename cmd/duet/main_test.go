package main

import (
	"net"
	"strconv"
	"testing"
	"time"
)

// The server starts on PORT with its data in DATA_DIR, and `duet -health` tells whether it answers.
func TestRun(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0") // a free port
	if err != nil {
		t.Fatal(err)
	}
	port := strconv.Itoa(l.Addr().(*net.TCPAddr).Port)
	l.Close()
	env := map[string]string{"PORT": port, "DATA_DIR": t.TempDir()}
	getenv := func(k string) string { return env[k] }

	if run([]string{"-health"}, getenv) != 1 {
		t.Fatal("health must fail while nothing listens")
	}
	go run(nil, getenv) // serves until the test ends
	for i := 0; run([]string{"-health"}, getenv) != 0; i++ {
		if i > 100 {
			t.Fatal("the server never became healthy")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if run(nil, getenv) != 1 {
		t.Fatal("a second server on a taken port must fail")
	}
}
