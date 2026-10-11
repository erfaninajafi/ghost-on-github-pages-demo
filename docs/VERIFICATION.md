# Verification record

This document records checks performed during implementation on 2026-10-07 and 2026-10-11. It describes local evidence, not approval of the public production domain or organizational SMTP delivery. It is a historical record rather than a guarantee that every check was rerun during subsequent documentation edits.

For reproducible commands, see [development and testing](DEVELOPMENT.md). Deployment and recovery procedures are in [operations](OPERATIONS.md).

## Initial platform — 2026-10-07

The following checks completed successfully in the local implementation environment:

- Built API and web images from the repository Dockerfiles and ran PostgreSQL, Redis, migrations, Go, Next.js, and Caddy through Compose.
- Applied the initial migration to PostgreSQL 16 and exercised down/up in an isolated schema, recreating all seven application tables.
- Ran `go test -race ./...` with real PostgreSQL and Redis. Coverage included signup/password hashing, publication permissions, HTML sanitization, Persian filtering/search, article/tag transaction rollback, comment trees, cross-article foreign keys, concurrent moderation, same-status notification idempotency, concurrent view increments, and atomic rate limiting.
- Ran `go vet ./...`, regenerated with sqlc 1.30.0, and checked schema/migration consistency.
- Passed frontend type checks, tests, formatting checks, and a Next.js 15.5.27 build. Article output used SSG/ISR.
- Exercised gateway HTTP behavior: signup, HttpOnly cookies, protected studio access, foreign-Origin rejection, publication, script sanitization, views, guest comments, replies, moderation, suggestions, and popular articles.
- Inspected actual RSS/sitemap XML and article canonical, OpenGraph, and JSON-LD output. Checked invalid pagination (400), missing articles (404), and unauthenticated protected access (401).
- Checked the browser's Persian RTL layout, local font, light/dark theme, debounced search, nested replies, branch-local reply form, sign-in, article lists, and approved-comment controls.
- Checked a mobile viewport for horizontal overflow and verified article text/reply forms fit.
- Verified invalid-cookie regression handling: public reads continue as guest and clear the cookie; writes, protected routes, and invalid Bearer credentials still fail with 401.

Temporary QA accounts, three articles, their comments, and notification jobs were removed after this pass, and web cache was refreshed. No default-password account or artificial training content was retained. The local `.env` used generated secrets and was ignored by Git. Existing `.cursorrules` was preserved.

## Administrator/author separation — 2026-10-11

- Applied migration 2 while preserving local data. Both migrations completed down/up checks in a temporary schema.
- Passed Go race tests against real PostgreSQL/Redis and Go vet. HTTP tests rejected author access to account management, archive/delete/trash, taxonomy creation, and moderation. Disabling an account rejected both its existing JWT and subsequent login.
- Database tests covered author creation/update, protection of administrator accounts, preserved article ownership during administrator edits, archiving/deleting administrator and author articles, removal from suggestions, and private restoration without content loss. A focused check also passed after adding author-name joins.
- Regeneration with sqlc 1.30.0 produced identical output. Frontend type checks/tests and Docker/Next.js builds passed.
- Browser checks created an author from the administrator account without replacing its session. The author created a draft and had no management controls. The administrator archived, deleted, and restored that article as a draft while preserving the original author's name.
- Removed the two QA accounts and their single temporary article while preserving existing user data. PostgreSQL, Redis, API, and web were healthy after the update.

## Optional covers and Persian article paths — 2026-10-11

- Reproduced the Persian-path 404 on an existing article: Go returned it, but the Next.js route encoded its already encoded slug again. The same article loaded in the browser after the fix.
- Applied migration 3 without removing data and checked all three migrations in an isolated schema. Generated sqlc 1.30.0 output from current schema/query sources.
- Passed Go race tests with real PostgreSQL/Redis, Go vet, frontend type checks/tests, and Docker/Next.js builds. Coverage included actual PNG/JPEG validation, byte/dimension limits, stripping appended data, file ownership, path traversal rejection, persistent storage, multipart contract, and Persian slugs.
- A temporary author uploaded a PNG through the studio and published a Persian-slug article. A page previously cached as 404 showed the title, text, and cover immediately after publication. Next.js optimized image loading and the OpenGraph cover URL were checked.
- Removed the cover reference and saved; the article stayed available without a cover.
- Confirmed the separate upload volume. Removed only QA account/article/media/ranking data and invalidated their cache, preserving the two pre-existing articles and accounts.

## Inline images, links, passwords, and studio tabs — 2026-10-11

- New Go tests covered rendering and save/read round trips with multiple independently sized images and internal/external links in both formats. They also covered HTML-looking plain text, unsafe links, foreign-account media, invalid widths, and excessive block count/size.
- Passed `go test -race ./...`, `go vet ./...`, and `scripts/test-integration.sh` with real PostgreSQL and Redis. PostgreSQL integration tests used isolated schemas rather than deleting the main content.
- Frontend tests/type checks covered editor block retention, link/image insertion and removal, safe preview output, and matching/UTF-8 byte limits for passwords. Local and Docker builds passed.
- Browser checks used a temporary API fixture with in-memory accounts/articles, not the primary database. Passwords began masked, eye controls changed visibility, and mismatched confirmation was rejected. Added two images at widths 320 and 480, selected a published article through internal search, and inserted an external URL. Saving plain-text and HTML versions, reloading, and reopening preserved widths and links.
- Confirmed separate author/comment tabs and retained unsaved article state while switching tabs. The inspected viewport showed no horizontal overflow, and image elements displayed at the selected 320/480 widths.
- This browser fixture created no account or article in the primary PostgreSQL database. Server persistence behavior was covered separately by Go tests. Temporary browser servers were stopped. The rebuilt local stack returned 200 for the home page/studio, with healthy long-running services and migrations exiting successfully.

## Documentation revision — 2026-10-11

Documentation was rewritten in English and checked against current routes, validation, editor behavior, configuration, migrations, and CI definitions. The revision adds a documentation index, native development guide, and API reference, and clarifies storage backups, role boundaries, cache behavior, and pagination limits. Documentation checks validate local links/anchors, JSON examples, and Bash example syntax; they do not execute deployment, restore, or account-creation examples.

The documentation pass checked 13 Markdown files, 80 local links/anchors, three JSON examples, and 29 Bash examples. Local targets and anchors existed, JSON parsed, Bash syntax checks passed, and Prettier plus `git diff --check` passed. No Persian prose remains in these Markdown guides.

## Repository cleanup — 2026-10-11

Removed 88 tracked archive files, including generated sites, seed data, the redirect page, obsolete publishing helpers, and archive-specific documentation/CI. Updated the remaining documentation and ignore rules. The current application code, Compose deployment, environment/integration scripts, and platform workflow were unchanged. Existing license attribution was retained.

Checked the remaining nine Markdown guides, 58 local links/anchors, three JSON examples, and 27 Bash examples. Documentation formatting, remaining script syntax, removed-path checks, and `git diff --check` passed.

## Not verified by these local passes

- Production DNS, certificate issuance/renewal for `blog.expertsacademy.ir`, and real organizational SMTP delivery. Local mail used the mock driver.
- A complete dependency advisory scan: `npm audit` could not reach its registry successfully, and `govulncheck` received HTTP 403 from `vuln.go.dev`. No claim of vulnerability-free dependencies is made; rerun scans in a network-enabled environment.
- Production load, an organizational backup restore, multi-host operation, CDN invalidation, or delivery guarantees beyond the implemented at-least-once outbox behavior.
- Execution of the newly documented native-development and recovery recipes as a complete end-to-end session; their configuration/command syntax has been reviewed, but those environments were not started by the documentation revision.
- GitHub Actions success after a push. [platform.yml](../.github/workflows/platform.yml) defines the CI checks; historical local success is not a recorded remote CI run.

Known implementation limits include unpaginated 200-record article/moderation lists, unpaginated public comment trees, shared server-side API rate buckets, local media/cache coordination across hosts, organization-only structured-data authors, and no autosave, password recovery, automatic redirects, media cleanup, or permanent article deletion. See the [documentation index](README.md#current-scope-and-limits) for current scope.
