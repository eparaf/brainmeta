#!/usr/bin/env bash
# Disci Brain — tek komutla baslat (WSL / Linux / macOS terminali).
#   ./start.sh
# Bos bir port otomatik secilir; UI o porta baglanir. Ctrl+C ikisini de durdurur.
# Gereksinim: `go` ve `npm` PATH'te olmali.
set -euo pipefail
cd "$(dirname "$0")"

# --- bos bir backend portu bul (8090'dan baslayarak) ---
is_free() { ! (exec 3<>"/dev/tcp/127.0.0.1/$1") 2>/dev/null; }
PORT=8090
while ! is_free "$PORT"; do PORT=$((PORT+1)); done
echo "→ Backend portu: $PORT  (UI: 5173)"

# --- backend'i arka planda baslat ---
BRAIN_ADDR=":$PORT" BRAIN_SNAPSHOT="brain-data/snapshot.json" go run ./cmd/brain serve &
BACK=$!

cleanup() { echo; echo "durduruluyor…"; kill "$BACK" 2>/dev/null || true; }
trap cleanup EXIT INT TERM

# --- backend hazir olana kadar bekle ---
for i in $(seq 1 60); do
  if (exec 3<>"/dev/tcp/127.0.0.1/$PORT") 2>/dev/null; then echo "→ backend hazir"; break; fi
  sleep 0.5
done

# --- UI ---
cd ui
[ -d node_modules ] || npm install
echo "→ UI baslatiliyor:  http://localhost:5173"
BACKEND_URL="http://localhost:$PORT" npm run dev
