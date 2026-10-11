# Deployment and operations

Run Compose commands from the repository root. See the [README](../README.md) for local startup and administrator bootstrap, and [development](DEVELOPMENT.md) for native Go/Next.js setup.

## Local services and storage

```bash
bash scripts/init-env.sh
docker compose up --build -d
docker compose ps -a
docker compose logs --tail=100 api web migrate
```

The initializer creates a private `.env` with independent PostgreSQL, Redis, JWT, and revalidation secrets. It preserves existing settings and adds a missing/empty revalidation secret. It does not rotate other credentials or fill an incomplete existing configuration. Generated hex credentials work directly in connection URLs; manually chosen credentials containing reserved URL characters require percent-encoding.

Startup order is PostgreSQL readiness, migration completion, Redis readiness, API, web, then gateway. Migrations are a separate one-shot service rather than API startup DDL. `migrate` exiting with code 0 is normal. No sample content or default account is created.

| Service    | Purpose                                           | Persistent data                           |
| ---------- | ------------------------------------------------- | ----------------------------------------- |
| `postgres` | PostgreSQL 16                                     | `postgres_data`                           |
| `redis`    | Redis 7.4, authentication, AOF, 256 MB/noeviction | `redis_data`                              |
| `migrate`  | Apply numbered SQL migrations                     | Schema version in PostgreSQL              |
| `api`      | HTTP API, media, mail worker                      | `uploads_data` mounted at `/data/uploads` |
| `web`      | Next.js server and runtime cache                  | Cache is disposable; no content database  |
| `gateway`  | Caddy, TLS, reverse proxy                         | `caddy_data`, `caddy_config`              |

Only gateway ports are published: 3000/3443 by default, mapping to container ports 80/443. Local `CADDY_SITE=:80` serves HTTP, so the default local URL is `http://localhost:3000`. PostgreSQL and Redis have no host ports in the standard configuration.

```bash
docker compose down
docker compose up -d
```

`down` preserves volumes. Do not use `down -v` to restart or repair a database authentication error; it deletes persistent data. Uploaded images survive API restarts and image replacement.

## Configuration reference

Use [root .env.example](../.env.example) for Compose, [API .env.example](../apps/api/.env.example) for native Go, and [web .env.example](../apps/web/.env.example) for native Next.js. Go reads exported process variables and does not load an env file itself. Next.js loads `.env.local` for local development.

### Root Compose variables

| Variable                         | Default/example and purpose                                                                                |
| -------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| `POSTGRES_DB`                    | `expertsacademy`; initialized database name                                                                |
| `POSTGRES_USER`                  | `expertsacademy`; initialized database role                                                                |
| `POSTGRES_PASSWORD`              | Required; generated secret; does not change an existing database role automatically                        |
| `REDIS_PASSWORD`                 | Required; generated Redis authentication secret                                                            |
| `JWT_SECRET`                     | Required; independent secret of at least 32 bytes; changing it invalidates existing sessions               |
| `REVALIDATION_SECRET`            | Required; independent API/web webhook secret of at least 32 bytes                                          |
| `APP_ENV`                        | `development`; use `production` for secure cookies and production validation                               |
| `SITE_URL`                       | `http://localhost:3000`; full origin with no trailing slash/path; becomes API Origin and frontend site URL |
| `CADDY_SITE`                     | `:80` locally; production hostname without scheme                                                          |
| `HTTP_PORT`                      | `3000` locally, `80` in production                                                                         |
| `HTTPS_PORT`                     | `3443` locally, `443` in production                                                                        |
| `MAIL_DRIVER`                    | `mock` locally; `smtp` required in production                                                              |
| `SMTP_HOST`                      | SMTP server hostname; required with SMTP                                                                   |
| `SMTP_PORT`                      | `587`; adapter requires STARTTLS, not implicit TLS                                                         |
| `SMTP_USERNAME`, `SMTP_PASSWORD` | Provider authentication credentials                                                                        |
| `SMTP_FROM`                      | Provider-approved sender address; required with SMTP                                                       |

Only `SITE_URL` is forwarded as the public web build argument. Never expose database credentials, JWT secrets, SMTP credentials, or `REVALIDATION_SECRET` with a `NEXT_PUBLIC_` name. Changing `SITE_URL` requires rebuilding web because public values are embedded in client output. The root `.env` is executable shell input for the integration helper; keep values properly quoted when editing it manually.

### Direct application variables

| Variable               | Scope and behavior                                                                                   |
| ---------------------- | ---------------------------------------------------------------------------------------------------- |
| `DATABASE_URL`         | API/admin process; PostgreSQL connection URL; supplied by Compose                                    |
| `REDIS_URL`            | API; Redis URL; native fallback `redis://localhost:6379/0`                                           |
| `HTTP_ADDR`            | API; defaults to `:8080`                                                                             |
| `CORS_ORIGIN`          | API; browser origin without a trailing slash; Compose derives it from `SITE_URL`                     |
| `TRUSTED_PROXY_CIDRS`  | API; comma-separated trusted peer networks; empty trusts no forwarded IPs                            |
| `UPLOAD_DIR`           | API; defaults to `./uploads`, Compose uses `/data/uploads`                                           |
| `WEB_REVALIDATION_URL` | API; Compose uses `http://web:3000/api/revalidate`; empty disables webhook calls                     |
| `API_INTERNAL_URL`     | Web server and rewrite configuration; `http://api:8080` in Compose, `http://localhost:8080` natively |
| `NEXT_PUBLIC_SITE_URL` | Web public origin for canonical URLs, feeds, sitemap, and metadata                                   |
| `REVALIDATION_SECRET`  | API and web runtime; values must match                                                               |

`ADMIN_EMAIL`, `ADMIN_NAME`, and `ADMIN_PASSWORD` are inputs only to the explicit administrator CLI. The password must be 12–72 UTF-8 bytes. The CLI creates an account rather than resetting one. `TEST_DATABASE_URL`, `TEST_REDIS_URL`, and optional `GO_BIN` are test configuration; see [development](DEVELOPMENT.md#verification-commands).

## Production deployment

1. Prepare a Linux container host with Docker/Compose, storage for images and backups, and SMTP connectivity.
2. Point the domain's A record, and AAAA record if used, to the host. Allow inbound ports 80 and 443 so Caddy can obtain and renew certificates.
3. Generate `.env` with `bash scripts/init-env.sh` and configure the production values:

```dotenv
APP_ENV=production
SITE_URL=https://blog.expertsacademy.ir
CADDY_SITE=blog.expertsacademy.ir
HTTP_PORT=80
HTTPS_PORT=443
MAIL_DRIVER=smtp
SMTP_PORT=587
```

4. Configure `SMTP_HOST`, `SMTP_USERNAME`, `SMTP_PASSWORD`, and `SMTP_FROM`. Use a sender permitted by the provider and the organization's domain mail policy. The adapter requires verified STARTTLS, typically on 587; implicit TLS on 465 is unsupported.
5. Build/start using `docker compose up --build -d`, inspect readiness and logs, then bootstrap the administrator according to the README.
6. Publish an article, submit a guest comment and reply, approve them, and verify both real notification deliveries. Check image loading, article metadata, RSS, and sitemap using the actual public origin.
7. Schedule private off-host backups and monitor dependency readiness, disk usage, Redis memory, and exhausted email jobs.

Production configuration rejects non-HTTPS origins and mock mail. Public signup is open and grants authors the ability to publish their own content. Confirm that this policy fits the deployment before making the studio public; there is no built-in signup-disable setting or invitation workflow.

Compose uses subnet `172.30.58.0/24`, which the API trusts for forwarded IPs. If it conflicts with an existing network, change the subnet and API `TRUSTED_PROXY_CIDRS` together in `docker-compose.yml`. A CDN or extra proxy needs its own verified forwarding setup. The API uses the last forwarded IP only when the direct peer is trusted; do not expose the API directly to the internet.

The API runs as UID 10001 with a read-only root filesystem and dropped capabilities. Web runs without root, but `.next` is writable for ISR and image caching. Local uploads and web caches are not automatically shared across hosts. A multi-host deployment needs shared media storage and cache invalidation reaching every web replica.

## Backup and restore

### Create a coordinated database/media backup

PostgreSQL holds articles, users, comments, and the email outbox. Uploaded image bytes live in `uploads_data` and are not included in a SQL dump. For a consistent pair, briefly stop application writers; this also pauses the mail worker and causes publication downtime. Any other client writing directly to the database/storage must be paused too.

```bash
umask 077
mkdir -p backups
backup_stamp=$(date -u +%Y%m%d-%H%M%S)
docker compose stop gateway web api
docker compose exec -T postgres sh -c 'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Fc' > "backups/blog-${backup_stamp}.dump"
docker compose run --rm -T --no-deps --entrypoint sh api -c 'tar -C /data/uploads -cf - .' > "backups/uploads-${backup_stamp}.tar"
docker compose up -d api web gateway
```

Check that both commands succeeded and keep matching timestamps together. Resume services if a backup step fails. Copy backups to private storage away from the host and practice recovery on a separate environment. Back up configuration securely as well; it contains secrets. `caddy_data` contains private certificate keys and needs protected handling if included in disaster recovery.

For a live database-only snapshot, the `pg_dump` command can run without stopping the stack, but it does not provide a coordinated media snapshot.

### Restore into an empty recovery environment

Use a separate recovery host/environment and empty target volumes. Preserve a backup of any existing target data before replacement. Do not restore on top of initialized application tables using the commands below. Start only infrastructure first, so the migrate service has not created the schema:

```bash
docker compose build api web
docker compose up -d postgres redis
docker compose exec -T postgres sh -c 'pg_restore -U "$POSTGRES_USER" -d "$POSTGRES_DB" --no-owner --exit-on-error' < backups/selected.dump
docker compose run --rm -T --no-deps --entrypoint sh api -c 'tar -C /data/uploads -xf -' < backups/matching-uploads.tar
docker compose up -d
```

The image's default user extracts media as UID 10001 into its writable upload volume. Ensure sufficient space and restore only trusted archives. The database dump includes migration state; subsequent `migrate up` applies any newer migrations from the chosen release. Verify login, article content, comments, and actual cover/inline image URLs before routing traffic to the recovery stack.

Redis is not the durable store for articles or view counts. Losing Redis may reset caches/rate-limit state but does not remove PostgreSQL content. Fresh Redis is acceptable for recovery. Guest emails and outbox data in backups are personal information; follow the organization's retention policy. No automatic backup/restore scheduler or media cleanup job is included.

## Email outbox

Inspect exhausted jobs without displaying recipients or bodies:

```bash
docker compose exec -T postgres sh -c 'psql -X -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"' <<'SQL'
SELECT id, attempts, created_at, available_at
FROM email_outbox
WHERE sent_at IS NULL AND attempts >= 12
ORDER BY created_at;
SQL
```

After fixing SMTP, requeue only a reviewed job by its UUID:

```sql
UPDATE email_outbox
SET attempts = 0, available_at = now()
WHERE id = '<reviewed-job-uuid>' AND sent_at IS NULL;
```

Delivery is at least once, so retry after an ambiguous SMTP result can duplicate an email. The worker uses five-minute leases, polls every two seconds, and stops after 12 attempts. A mock driver logs job IDs and marks deliveries processed without sending external email. Purge old sent jobs only under an approved retention policy.

## Migrations and releases

Before deploying changed code, run applicable Go tests/vet, frontend type checks/tests/build, integration checks, and sqlc generation. Keep `schema.sql` aligned with new numbered up/down migrations; never rewrite an already applied production migration. Back up before schema changes. Review backward compatibility before rolling application images back: an old image and newer database may not be compatible.

```bash
bash scripts/init-env.sh
docker compose up --build -d
docker compose ps -a
docker compose logs --tail=100 migrate api web
```

Running `migrate up` at the current version is a no-op. Down migrations can discard data and must not be used as routine repair. CI performs down/up checks only on a disposable database. The local integration helper also applies migrations to the configured development Compose database before isolated-schema tests.

Run `npm audit` from `apps/web` and `govulncheck ./...` from `apps/api` in an environment with advisory-network access. Prior local scans were blocked by network failures; see [verification](VERIFICATION.md). The repository records dependency versions and locks, but a successful build alone does not establish absence of vulnerabilities.

## Health, logs, and cache recovery

Go serves internal `/healthz` and `/readyz`; the gateway routes only `/api/v1/*` to Go. Inspect container health and logs:

```bash
docker compose ps -a
docker compose logs --tail=100 postgres redis migrate api web
```

API request logs include request IDs, route names, methods, statuses, and durations. Configure log retention and collection in the deployment. Readiness requires PostgreSQL and Redis. Redis failure may still permit uncached public API reads, while writes fail with 503 and dependent restarts may wait for health.

After a save or status change, the API calls `WEB_REVALIDATION_URL` with the private secret. If content remains stale, check API logs for `site cache revalidation failed`, connectivity to web, and matching `REVALIDATION_SECRET` values. Missing/short webhook configuration returns 503; an invalid token returns 401. A committed save is not rolled back when cache revalidation fails.

For urgent removal of a cached article, confirm the PostgreSQL status first, then recreate web:

```bash
docker compose up -d --force-recreate web
```

This resets that container's local cache and may briefly interrupt requests. Apply equivalent invalidation to all web replicas and any external CDN caches. Uploaded image URLs use a one-year immutable cache and are independent of article publication status; changing a draft or deleting an article does not revoke a previously served image.

## Troubleshooting

### PostgreSQL password authentication fails

`POSTGRES_PASSWORD` initializes a role only when the data volume is new. Editing `.env` or rebuilding a container does not update an existing database password. The Compose TCP health check detects a mismatch.

First recreate the PostgreSQL service with the intended current environment. It can remain unhealthy until its stored role is updated:

```bash
docker compose up -d postgres
```

For this local Compose PostgreSQL instance, align the existing role with the container's configured password without deleting data:

```bash
docker compose exec -T postgres sh -c 'exec psql -X -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"' <<'SQL'
\getenv configured_user POSTGRES_USER
\getenv configured_password POSTGRES_PASSWORD
SELECT format('ALTER ROLE %I PASSWORD %L', :'configured_user', :'configured_password')
\gexec
SQL

docker compose up -d api web gateway
```

This changes the database role password; update other clients of the same role too. It does not print the password. It assumes the role/database names still match the existing installation and the image's local socket access is available. If names also changed, use the original role/database through your database administration process. No volume deletion or forced migration version is needed.

### Other common failures

| Symptom                                       | Check                                                                                                              |
| --------------------------------------------- | ------------------------------------------------------------------------------------------------------------------ |
| Port 3000/80/443 already in use               | Stop the conflicting service or adjust gateway host ports; update `SITE_URL` and rebuild web if the origin changes |
| Compose subnet overlaps another network       | Change both subnet and trusted proxy CIDRs                                                                         |
| Migration fails                               | Inspect migrate/PostgreSQL logs and credentials; fix the underlying cause rather than forcing a version            |
| Signup or other writes return 403             | Exact `Origin` must match configured origin, including scheme and port                                             |
| Writes return 503                             | Redis connectivity, credentials, memory pressure, and readiness                                                    |
| Requests return 429                           | Wait for the one-minute bucket; SSR requests from web can share a read quota                                       |
| Upload returns 413                            | File/output size, 12-million-pixel limit, or 8,000-pixel axis limit                                                |
| Upload returns 400                            | Actual PNG/JPEG data, exactly one `image` field, and no extra multipart fields                                     |
| Media becomes 404 after a move                | Missing upload volume or mismatched database/media backup pair                                                     |
| Save fails with 409                           | Duplicate slug or an article archived/deleted since its editor was opened                                          |
| Email is absent                               | Mock driver, SMTP STARTTLS/authentication, provider policy, spam folder, or exhausted outbox jobs                  |
| Changed site origin still appears in metadata | Rebuild the web image; public site URL is a build-time value too                                                   |
