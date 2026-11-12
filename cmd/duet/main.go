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

	"duet/internal/game"
	"duet/internal/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	// `duet -health` is the container health check: the image has no shell or curl to do it with.
	if len(os.Args) > 1 && os.Args[1] == "-health" {
		resp, err := http.Get("http://127.0.0.1:" + port + "/healthz")
		if err != nil || resp.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		return
	}
	dataDir := os.Getenv("DATA_DIR") // for running outside the container; the image always uses /data
	if dataDir == "" {
		dataDir = "/data"
	}
	log.Printf("duet: http://0.0.0.0:%s/screen", port)
	log.Fatal(http.ListenAndServe(":"+port, server.Handler(game.New(dataDir))))
}
