# Development and testing

For the fastest complete setup, use the [Docker quick start](../README.md#run-locally). The commands below assume a Bash shell and start from the repository root unless stated otherwise.

## Toolchain

| Tool            | Repository configuration                                                          |
| --------------- | --------------------------------------------------------------------------------- |
| Go              | `go.mod` declares 1.24.0; Docker and CI use 1.27                                  |
| Node.js         | Docker and CI use 22                                                              |
| npm             | Install frontend dependencies with `npm ci` and the committed `package-lock.json` |
| sqlc            | 1.30.0, required when changing SQL queries or generated models                    |
| PostgreSQL      | 16, with `citext`, `uuid-ossp`, and `pg_trgm` extensions                          |
| Redis           | 7.4                                                                               |
| golang-migrate  | Compose uses `migrate/migrate:v4.18.3`                                            |
| Docker Compose  | v2                                                                                |
| Bash and Python | Environment initialization and the local integration helper                       |

Next.js, React, TypeScript, and other JavaScript versions are pinned in [package.json](../apps/web/package.json). Go dependencies are recorded in [go.mod](../apps/api/go.mod) and `go.sum`.

## Run the applications natively

### 1. Start infrastructure and apply migrations

Use a development checkout and a development `.env`. The standard Compose configuration intentionally has no host database or Redis ports. The following temporary override exposes them only on loopback, using ports 55432 and 56379:

```bash
bash scripts/init-env.sh
dev_compose=$(mktemp /tmp/expertsacademy-dev.XXXXXX)
cat > "$dev_compose" <<'YAML'
services:
  postgres:
    ports:
      - "127.0.0.1:55432:5432"
  redis:
    ports:
      - "127.0.0.1:56379:6379"
YAML
docker compose -f docker-compose.yml -f "$dev_compose" up -d postgres redis migrate
docker compose -f docker-compose.yml -f "$dev_compose" ps -a
```

The migration container should finish with exit code `0`. Keep the override path for subsequent infrastructure commands. If the complete Docker stack is already running, stop `gateway`, `web`, and `api` before starting the native servers so the gateway releases port 3000:

```bash
docker compose stop gateway web api
```

Do not remove persistent volumes when switching development modes. For Docker Desktop, the loopback port mappings avoid relying on inaccessible private container IPs.

### 2. Start the Go API

In the same terminal, export the generated root environment and construct connection URLs with escaped credentials:

```bash
set -a
source .env
set +a
export DATABASE_URL="$(python3 - <<'PY'
import os
from urllib.parse import quote
user = quote(os.environ['POSTGRES_USER'], safe='')
password = quote(os.environ['POSTGRES_PASSWORD'], safe='')
database = quote(os.environ['POSTGRES_DB'], safe='')
print(f'postgres://{user}:{password}@localhost:55432/{database}?sslmode=disable')
PY
)"
export REDIS_URL="$(python3 - <<'PY'
import os
from urllib.parse import quote
password = quote(os.environ['REDIS_PASSWORD'], safe='')
print(f'redis://:{password}@localhost:56379/0')
PY
)"
export APP_ENV=development
export MAIL_DRIVER=mock
export HTTP_ADDR=:8080
export SITE_URL=http://localhost:3000
export CORS_ORIGIN=http://localhost:3000
export TRUSTED_PROXY_CIDRS=''
export UPLOAD_DIR="$PWD/apps/api/uploads"
export WEB_REVALIDATION_URL=http://localhost:3000/api/revalidate
cd apps/api
go run ./cmd/api
```

The Go process reads exported variables; it does not load `.env` automatically. The alternative [API environment template](../apps/api/.env.example) can be copied to `apps/api/.env`, configured, and sourced with `set -a` before running the API. The template credentials are examples and must match the actual database and Redis settings. Keep local env files private and untracked.

Native uploads in `apps/api/uploads` and Docker's `uploads_data` are different stores. If this database already contains Docker-uploaded images, copy its media snapshot into the native upload directory before using it; switching modes does not move files automatically. Copy native media back before returning that same database to Docker. Keep a matching database/media backup and use the [media backup procedure](OPERATIONS.md#backup-and-restore) for transfers. A fresh development database avoids reusing production media or accounts.

### 3. Start Next.js

In a second terminal, from the repository root:

```bash
cd apps/web
cp .env.example .env.local
chmod 600 .env.local
npm ci
npm run dev
```

Copy the template only for initial setup; edit an existing `.env.local` instead of overwriting it. Set `REVALIDATION_SECRET` there to the same private value exported to the API. The template already points `API_INTERNAL_URL` to `http://localhost:8080` and `NEXT_PUBLIC_SITE_URL` to `http://localhost:3000`. Do not prefix the revalidation secret with `NEXT_PUBLIC_`.

Open <http://localhost:3000/studio>. Next.js rewrites browser `/api/v1/*` requests to the native API. The API's origin must match the browser origin exactly, including its port. Restart Next.js after changing its local environment.

### 4. Bootstrap an administrator

In another terminal with the same API environment exported, run the native equivalent of the Docker bootstrap:

```bash
export ADMIN_EMAIL='you@example.com'
export ADMIN_NAME='Academy Admin'
read -rs -p 'Admin password (12–72 bytes): ' ADMIN_PASSWORD
echo
export ADMIN_PASSWORD
cd apps/api
go run ./cmd/admin
unset ADMIN_PASSWORD
```

Run this from the repository root before `cd`, or omit `cd apps/api` when already in that directory. It creates a new administrator; it does not update or reset an existing account. Keep the database URL pointed at the intended development database.

## Verification commands

### Go

```bash
cd apps/api
go vet ./...
go test -race ./...
```

Unit tests cover content rendering, roles, authentication, sanitization, media validation, and HTTP behavior. Database and Redis integration tests report a skip when their dependency variables are absent.

On Linux with Docker Engine, from the repository root:

```bash
bash scripts/test-integration.sh
```

The helper sources the root `.env`, starts PostgreSQL/Redis/migrations, discovers private container IPs, uses Redis database 15, and runs Go tests with the race detector. PostgreSQL tests create and clean up isolated schemas. The helper still applies migrations to the configured Compose database; use a development instance. Redis tests write application cache/ranking/revision keys as well as temporary limiter keys, so database 15 should be reserved for these tests. Do not run another integration suite against that same Redis database concurrently.

For Docker Desktop or separate infrastructure, set `TEST_DATABASE_URL` to a disposable PostgreSQL database with the required extensions available, and `TEST_REDIS_URL` to an isolated Redis database accessible from the host. Then run `go test -race ./...` from `apps/api`. The database user must be able to create schemas and extensions. `GO_BIN=/path/to/go bash scripts/test-integration.sh` selects a specific Go executable for the helper.

### Frontend

```bash
cd apps/web
npm ci
npm run typecheck
npm test
npm run format:check
npm run build
```

Tests exercise editor document compatibility, images and links, password confirmation and UTF-8 limits, safe HTML, article paths, XML, and SEO helpers. `npm run format` writes formatting changes across the frontend; review the resulting diff. A build validates compilation but does not prove live API, SMTP, or domain behavior.

### SQL generation and migrations

Edit `db/queries/blog.sql` or add a numbered migration under `db/migrations`, and keep `db/schema.sql` aligned with the final schema. From `apps/api`:

```bash
sqlc generate
git diff -- internal/repository/db
```

Generated files in `internal/repository/db` must not be edited by hand. Commit generated changes with their SQL sources. Do not rewrite migrations already applied to shared environments. Migration down/up checks belong on a disposable database because down migrations can drop data.

## CI and manual acceptance

[platform.yml](../.github/workflows/platform.yml) runs Go vet/tests against PostgreSQL and Redis, verifies sqlc output, applies migrations, and performs a migration down/up round trip on its disposable database. Its web job installs dependencies, type-checks, tests, and builds. The separate [validate.yml](../.github/workflows/validate.yml) checks the archived static demo.

For editor or studio changes, supplement automated checks with a browser pass: register with matching and mismatching passwords, exercise eye buttons, save and reopen both content formats with two different image widths and both link types, and check role-specific tabs. Confirm that an unsaved draft survives tab switching and that a refresh requires a saved article. Verify public output at narrow and wide viewport sizes.

Record actual commands, outcomes, fixture scope, and unverified areas in [VERIFICATION.md](VERIFICATION.md). Do not describe mock mail or an in-memory browser fixture as production SMTP delivery or a real database browser test.

## Contribution boundaries

Keep authorization and business rules in use cases, database access behind domain contracts, and transport concerns in HTTP delivery. Preserve compatibility for existing version 1 editor documents and older HTML API clients. Never commit `.env`, passwords, uploads, database dumps, or temporary test accounts. Update the API, user, or operations guide when behavior or configuration changes.
