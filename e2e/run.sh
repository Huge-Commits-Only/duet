#!/bin/sh
# Plays every game mode in a real browser: one projector page and two phones.
# Expects the server binary at bin/duet (go build -o bin/duet ./cmd/duet) and `npm install` done in e2e/.
set -eu
cd "$(dirname "$0")/.."
tmp=$(mktemp -d)
trap 'kill $pid 2>/dev/null || true; rm -rf "$tmp"' EXIT
DATA_DIR="$tmp" PORT=18080 ./bin/duet >"$tmp/duet.log" 2>&1 &
pid=$!
for _ in $(seq 50); do curl -fs -o /dev/null http://127.0.0.1:18080/screen && break; sleep 0.2; done
node e2e/play.mjs http://127.0.0.1:18080 || { cat "$tmp/duet.log"; exit 1; }
