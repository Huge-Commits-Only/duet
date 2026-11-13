// Command duet runs the Duet game server.
//
// Configuration (environment): PORT (default 8080), PUBLIC_URL (optional, the address put
// into the QR code), DATA_DIR (default /data; where questions asked, the saved list and
// the running game are kept).
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"duet/internal/game"
	"duet/internal/server"
)

func main() { os.Exit(run(os.Args[1:], os.Getenv)) }

// run is the whole program, arguments and environment in, exit code out, so a test can drive it.
func run(args []string, getenv func(string) string) int {
	port := getenv("PORT")
	if port == "" {
		port = "8080"
	}
	// `duet -health` is the container health check: the image has no shell or curl to do it with.
	if len(args) > 0 && args[0] == "-health" {
		resp, err := http.Get("http://127.0.0.1:" + port + "/healthz")
		if err != nil || resp.StatusCode != http.StatusOK {
			return 1
		}
		return 0
	}
	dataDir := getenv("DATA_DIR") // for running outside the container; the image always uses /data
	if dataDir == "" {
		dataDir = "/data"
	}
	log.Printf("duet: http://0.0.0.0:%s/screen", port)
	// no WriteTimeout: it would cut the long-lived event streams
	srv := &http.Server{Addr: ":" + port, Handler: server.Handler(game.New(dataDir)), ReadHeaderTimeout: 10 * time.Second}
	log.Print(srv.ListenAndServe())
	return 1
}
