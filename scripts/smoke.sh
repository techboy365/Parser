#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "==> Go build"
go build -o /tmp/parser ./cmd/parser

echo "==> Go tests"
go test ./...

echo "==> Sidecar typecheck"
cd captcha-solver
npm run typecheck

echo "==> Sidecar health (background)"
node --import tsx src/index.ts &
SID=$!
sleep 1
curl -sf http://127.0.0.1:8787/health | head -c 200
echo ""
kill $SID

echo ""
echo "All smoke checks passed."
echo ""
echo "Full integration test (requires Kameleo + hotspot + API keys):"
echo "  1. cp captcha-solver/.env.example captcha-solver/.env  # add keys"
echo "  2. cd captcha-solver && npm start"
echo "  3. go run ./cmd/parser --dork 'site:example.com'"
