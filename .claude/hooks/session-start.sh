#!/bin/bash
# Affluena-API SessionStart hook.
# Warms the Go module cache and toolchain so build/vet/test work in
# Claude Code on the web.
set -euo pipefail

# Only run in the remote (Claude Code on the web) environment.
if [ "${CLAUDE_CODE_REMOTE:-}" != "true" ]; then
  exit 0
fi

cd "$CLAUDE_PROJECT_DIR"

echo "[session-start] Affluena-API: downloading Go modules..."
# Triggers the go.mod toolchain download (go1.26.x) and populates the module
# cache; both persist in the cached container state for later sessions.
go mod download

echo "[session-start] Affluena-API: pre-building packages..."
# Pre-compile so the first `go build`/`go test` in the session is fast.
# Non-fatal: a build issue should not block the session from starting.
go build ./... || echo "[session-start] go build reported issues (continuing)."

echo "[session-start] Affluena-API: Go environment ready."
echo "[session-start] Note: integration tests (./internal/db, ./internal/server) need Postgres/Docker; unit tests and 'go vet ./...' run without it."
