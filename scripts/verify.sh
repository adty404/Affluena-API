#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$ROOT_DIR"

TEST_DATABASE_URL="${AFFLUENA_TEST_DATABASE_URL:-postgres://affluena:affluena@localhost:5432/affluena?sslmode=disable}"
API_HEALTH_URL="${AFFLUENA_API_HEALTH_URL:-http://localhost:8080/healthz}"

step() {
	printf '\n==> %s\n' "$1"
}

step "Check Go formatting"
unformatted="$(find . -name '*.go' -not -path './.git/*' -print0 | xargs -0 gofmt -l)"
if [ -n "$unformatted" ]; then
	printf '%s\n' "$unformatted"
	printf 'Run gofmt on the files above.\n' >&2
	exit 1
fi

step "Run unit tests"
go test ./... -count=1

step "Run go vet"
go vet ./...

step "Build API"
go build -o /tmp/affluena-api ./cmd/api

step "Validate Postman collection JSON"
python3 -m json.tool postman/Affluena.postman_collection.json >/tmp/affluena_postman_validated.json

step "Rebuild and start Docker Compose"
docker compose up -d --build

step "Run integration tests"
AFFLUENA_TEST_DATABASE_URL="$TEST_DATABASE_URL" go test ./internal/db ./internal/debt ./internal/server -count=1

step "Check API health"
attempt=1
while [ "$attempt" -le 30 ]; do
	if curl -fsS "$API_HEALTH_URL" >/tmp/affluena_health.json; then
		cat /tmp/affluena_health.json
		printf '\n'
		exit 0
	fi
	attempt=$((attempt + 1))
	sleep 1
done

printf 'API health check failed at %s\n' "$API_HEALTH_URL" >&2
exit 1
