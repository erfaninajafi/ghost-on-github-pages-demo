# ExpertsAcademy Blog

A Persian, right-to-left publication for `https://blog.expertsacademy.ir`, with a Go API, PostgreSQL, Redis, and a Next.js frontend. The studio supports author accounts, article publishing, plain-text and HTML content with multiple resizable images and links, and separate administrator tabs for comments and authors.

The application interface remains Persian. Project documentation is in English.

## Run locally

Run these commands from the repository directory containing `docker-compose.yml`.

Requirements: Docker Engine with Compose v2, Bash, Python 3, internet access for the first build, and at least 6 GB of free disk space for the initial images and build cache. Native Go and Node.js installations are not required for this Docker workflow.

```bash
bash scripts/init-env.sh
docker compose up --build -d
docker compose ps -a
```

- Publication: <http://localhost:3000>
- Author and administrator studio: <http://localhost:3000/studio>

The initialization script creates a private `.env` with four independent random secrets. It preserves an existing configuration and adds `REVALIDATION_SECRET` if missing or empty. Migrations run before the API starts. PostgreSQL, Redis, and uploaded images use persistent volumes; database and Redis ports are not published to the host.

A fresh database has no default accounts or sample articles. An exited `migrate` container with exit code `0` is expected.

## Create the first administrator

After the services start, use a unique email address and a password of 12–72 UTF-8 bytes:

```bash
export ADMIN_EMAIL='you@example.com'
export ADMIN_NAME='Academy Admin'
read -rs -p 'Admin password (12–72 bytes): ' ADMIN_PASSWORD
echo
export ADMIN_PASSWORD
docker compose run --rm -e ADMIN_EMAIL -e ADMIN_NAME -e ADMIN_PASSWORD --entrypoint admin api
unset ADMIN_PASSWORD
```

Sign in at `/studio`. Administrators can create categories and tags, publish articles, manage author accounts, and moderate comments. Public registration creates an author account, requires password confirmation in the browser, and signs the new author in immediately. Both registration password fields and the login password are hidden until their eye button is pressed. Authors can publish their own articles without an approval step.

## Write and manage articles

The editor accepts plain text or HTML, optional cover images, and ordered text, image, and link blocks. Use the inline upload button to insert one or more PNG/JPEG images and set each image's width from 80 to 2,000 pixels. Images retain their aspect ratio and fit the available page width. Links can point to a published article found through search or an external HTTP/HTTPS address. Preview the result and save explicitly; there is no autosave.

Administrators have separate **Articles**, **Comment management**, and **Author management** tabs. Switching tabs preserves the current editor state in the page; reloading or leaving the page loses unsaved changes. See the [user guide](docs/USER_GUIDE.md) for the complete workflow and current limits.

## Stop and restart

```bash
docker compose down
docker compose up -d
```

`down` preserves volumes. `down -v` deletes persistent data and is not an ordinary restart command. Run `docker compose up --build -d` after changing application code; rebuild the web image after changing `SITE_URL`.

## Repository structure

| Path                              | Purpose                                                                                    |
| --------------------------------- | ------------------------------------------------------------------------------------------ |
| `apps/api`                        | Go domain, use cases, HTTP delivery, PostgreSQL repository, media storage, and mail worker |
| `apps/api/db`                     | SQL schema, migrations, and sqlc query sources                                             |
| `apps/web`                        | Persian publication, SEO, search, comments, and studio                                     |
| `docker-compose.yml`, `Caddyfile` | Services, readiness checks, persistent storage, and HTTPS gateway                          |
| `docs`                            | English documentation and verification history                                             |

## Documentation

- [Documentation index](docs/README.md)
- [Author and administrator guide](docs/USER_GUIDE.md)
- [Development and testing](docs/DEVELOPMENT.md)
- [Architecture](docs/ARCHITECTURE.md)
- [API reference](docs/API_REFERENCE.md)
- [Deployment, configuration, backups, and troubleshooting](docs/OPERATIONS.md)
- [Verification history and unverified areas](docs/VERIFICATION.md)

## Checks

With Go and Node.js installed:

```bash
cd apps/api
go test -race ./...
go vet ./...
cd ../web
npm ci
npm run typecheck
npm test
npm run build
```

Database and Redis integration tests are skipped unless their `TEST_DATABASE_URL` and `TEST_REDIS_URL` variables are available. On Linux with Docker Engine, run `bash scripts/test-integration.sh` from the repository root to provide real dependencies. PostgreSQL tests use temporary schemas, while Redis tests need an isolated test database. See [development and testing](docs/DEVELOPMENT.md) for details.

## Production

Set `APP_ENV=production`, `SITE_URL=https://blog.expertsacademy.ir`, `CADDY_SITE=blog.expertsacademy.ir`, host ports `80` and `443`, and real SMTP credentials in `.env`. Configure DNS and network access before starting the stack. Caddy obtains and renews TLS certificates. Production rejects the mock mail driver and non-HTTPS origins. Follow [operations](docs/OPERATIONS.md) for the full deployment and backup procedure.

Deployment requires a host that can run Docker containers and provide persistent storage for PostgreSQL and uploaded images.

## License

See [LICENSE](LICENSE).
