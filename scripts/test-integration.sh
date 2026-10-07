#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
if [[ ! -f .env ]]; then ./scripts/init-env.sh; fi
set -a
source .env
set +a
docker compose up -d postgres redis migrate
pg_ip=$(docker inspect "$(docker compose ps -q postgres)" --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}')
redis_ip=$(docker inspect "$(docker compose ps -q redis)" --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}')
export TEST_DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${pg_ip}:5432/${POSTGRES_DB}?sslmode=disable"
export TEST_REDIS_URL="redis://:${REDIS_PASSWORD}@${redis_ip}:6379/0"
cd apps/api
"${GO_BIN:-go}" test -race "$@" ./...
