# API reference

The public gateway exposes the Go API under `/api/v1`. Examples use `http://localhost:3000`; the intended production origin is `https://blog.expertsacademy.ir`. Paths in endpoint tables are relative to `/api/v1` unless stated otherwise.

## HTTP conventions

Requests and responses use UTF-8 JSON, UUID identifiers, and RFC3339 timestamps. Ordinary JSON bodies are limited to 1 MiB. Unknown request fields, malformed JSON, and multiple JSON documents are rejected. Uploads use a separate multipart contract. List endpoints return arrays or the explicitly documented page envelope.

Every write must send `Origin` equal to `CORS_ORIGIN`, even when using a Bearer token. In Compose, that value comes from `SITE_URL`. JSON writes require `Content-Type: application/json`; multipart writes let the client generate the boundary. Foreign Origins are rejected on reads too when supplied. Requests through the frontend are same-origin.

The API normally sets `Cache-Control: no-store`; successful media responses instead use immutable public caching. Human-facing error messages are generally Persian. Clients should branch on status and `error.code` rather than message text.

## Authentication and roles

| Method and path     | Body                                 | Success                     | Access       |
| ------------------- | ------------------------------------ | --------------------------- | ------------ |
| `POST /auth/signup` | `email`, `password`, optional `name` | 200 User and session cookie | Public       |
| `POST /auth/login`  | `email`, `password`                  | 200 User and session cookie | Public       |
| `POST /auth/logout` | No body required                     | 204 and cleared cookie      | Public       |
| `GET /auth/me`      | None                                 | 200 User                    | Author/admin |

Signup always creates an author. `name` is optional; the server defaults it to the Persian word for author. If supplied, a name must contain 2–100 Unicode characters. Email must be valid and at most 254 bytes. Passwords must contain 12–72 UTF-8 bytes and are hashed with bcrypt cost 12.

The web registration form asks for confirmation and validates the match locally. There is no `confirm_password`, `password_again`, or role field in the signup API; sending unknown fields causes 400. Eye controls affect browser display only. There is no invitation, email verification, password-reset, or password-update endpoint.

A User response contains `id`, `email`, `name`, `role`, and nullable `disabled_at`; password hashes are excluded. Signup/login set `ea_session`, an HttpOnly, SameSite=Lax cookie with an eight-hour lifetime and Secure in production. They do not return a separate token property. A JWT can also be supplied as `Authorization: Bearer <token>`; it takes precedence over the cookie. The server reloads the user for each authenticated request.

Logout clears the browser cookie rather than revoking copied JWTs. Disabling an author blocks subsequent authenticated requests. Invalid browser cookies are cleared on public GETs, which continue as guest; an invalid explicit Bearer token still returns 401.

### Local cookie-client example

In Bash, prompt for an existing account's password and keep the cookie jar private:

```bash
umask 077
cookie_jar=$(mktemp /tmp/expertsacademy-session.XXXXXX)
export LOGIN_EMAIL='you@example.com'
read -rs -p 'Password: ' LOGIN_PASSWORD
echo
export LOGIN_PASSWORD
python3 - <<'PY' | curl --fail-with-body -sS \
  -c "$cookie_jar" \
  -H 'Origin: http://localhost:3000' \
  -H 'Content-Type: application/json' \
  --data-binary @- http://localhost:3000/api/v1/auth/login
import json, os
print(json.dumps({'email': os.environ['LOGIN_EMAIL'], 'password': os.environ['LOGIN_PASSWORD']}))
PY
unset LOGIN_PASSWORD
curl --fail-with-body -sS -b "$cookie_jar" http://localhost:3000/api/v1/auth/me
```

Do not share the cookie file. After your API session:

```bash
curl --fail-with-body -sS -b "$cookie_jar" \
  -H 'Origin: http://localhost:3000' \
  -X POST http://localhost:3000/api/v1/auth/logout
rm -f "$cookie_jar"
```

## Public articles and taxonomy

| Method and path                   | Result and query parameters                                                                                           |
| --------------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| `GET /posts`                      | `{items,total,page,limit}`; defaults `page=1`, `limit=12`; page 1–10,000, limit 1–50; optional category/tag **slugs** |
| `GET /posts/popular`              | Up to six public article summaries by lifetime views                                                                  |
| `GET /posts/search-suggest?q=...` | Up to eight summaries; queries outside 2–100 normalized characters return an empty array                              |
| `GET /posts/{slug}`               | Published article and tags; counts one view unless `count_view=false`                                                 |
| `GET /taxonomy`                   | `{categories:[],tags:[]}`; taxon fields are `id`, `name`, `slug`                                                      |

Public articles must be `published` with a publication time at or before now. Draft, archived, and deleted articles are inaccessible. Category and tag filters use slugs, while save requests use UUIDs. Public lists and suggestions omit usable body content by returning `content_html: ""` and `content_json: {}`. Sort order is publication time descending with a stable ID tie-breaker; popularity uses accumulated views.

A full Post contains `id`, `author_id`, optional `author_name`, nullable `category_id`, `slug`, `title`, `content_html`, `content_json`, nullable `cover_image_url`, `excerpt`, `reading_time_minutes`, `views_count`, `status`, nullable `published_at`, `created_at`, `updated_at`, and `tags`.

Use a once-encoded Unicode slug as the path segment. Requests for previews, SEO, and automated reads should use `count_view=false` to avoid incrementing views. Comment paths below use the **post UUID**, not its slug.

## Comments

| Method and path             | Contract                                                       | Success                           |
| --------------------------- | -------------------------------------------------------------- | --------------------------------- |
| `GET /posts/{id}/comments`  | Fully approved root/reply trees for a published article        | 200 array                         |
| `POST /posts/{id}/comments` | `author_name`, `author_email`, `content`, nullable `parent_id` | 201 Comment with `status=pending` |

Guest names contain 2–100 characters and content 2–5,000; the email must be valid. For a signed-in user, account name/email replace client-supplied identity fields. Every comment starts pending, including author/admin comments.

`parent_id` must identify an approved comment on the same article. Maximum tree depth is eight levels including the root. A public reply is visible only if all ancestors are approved. The public tree has no pagination.

Comment responses contain `id`, `post_id`, optional `user_id`, nullable `parent_id`, `author_name`, `content`, `status`, timestamps, and `replies`. `author_email` is never serialized, including moderation responses. New comments queue an author notification; approved replies queue a notification to the parent commenter.

## Article workspace

| Method and path                    | Contract                                                                                        | Access and success      |
| ---------------------------------- | ----------------------------------------------------------------------------------------------- | ----------------------- |
| `GET /studio/posts`                | Latest 200 non-deleted articles; admins include archived, authors see their own active articles | Author/admin; 200 array |
| `GET /studio/posts?status=deleted` | Latest 200 deleted articles; other non-empty status filters are invalid                         | Admin; 200 array        |
| `POST /studio/posts`               | Complete create/update payload described below                                                  | Owner/admin; 200 Post   |
| `PATCH /studio/posts/{id}/status`  | `{ "status": "archived" }` or `{ "status": "draft" }`                                           | Admin; 200 Post         |
| `DELETE /studio/posts/{id}`        | No body; soft deletion                                                                          | Admin; 204              |

The studio list is not paginated. Archive and ownership UI filters operate only on the returned 200 records. A deleted article cannot be archived directly; restore it to draft first. Restore never republishes an article. Neither archive nor delete removes stored content, comments, tags, or uploaded files. No permanent deletion endpoint exists.

An empty/omitted `id` creates an article; an existing UUID updates it. The author is assigned by the server and preserved on administrator edits. `POST /studio/posts` is a complete save, not a partial patch: send all values you want retained, including tags and cover. Archived/deleted records reject ordinary saves, preventing stale forms from overriding moderation.

### Save fields

| Field             | Type and constraints                                              |
| ----------------- | ----------------------------------------------------------------- |
| `id`              | Optional UUID string; empty for creation                          |
| `title`           | Required, 1–240 Unicode characters                                |
| `slug`            | Required, unique, 1–200 Unicode letters/digits/hyphens            |
| `excerpt`         | String, at most 500 characters                                    |
| `status`          | `draft` or `published`                                            |
| `category_id`     | Existing category UUID or null                                    |
| `tag_ids`         | Array of at most 20 distinct existing tag UUIDs                   |
| `cover_image_url` | Existing local upload URL or null                                 |
| `content_format`  | `plain_text` or `html` for current clients                        |
| `content_blocks`  | Optional ordered block array; preferred current editor format     |
| `content_text`    | Legacy plain-text source; at most 500,000 bytes                   |
| `content_html`    | Legacy HTML source; at most 500,000 bytes                         |
| `content_json`    | Legacy JSON metadata; regenerated when an explicit format is used |

Publication requires at least one word after sanitization; images alone are insufficient. Reading time is calculated at 200 words/minute, rounded up with a minimum of one minute. Republish preserves the first publication date. Category/tag changes and article save are transactional.

### Mixed content blocks

When `content_blocks` is present and non-null, it must contain 1–200 blocks and an explicit supported `content_format`. The combined UTF-8 size of all `text`, `src`, `alt`, and `href` values is limited to 500,000 bytes. This content limit is separate from the 1 MiB JSON request limit.

| Block | Required values                                 | Rules                                                                                            |
| ----- | ----------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| Text  | `type: "text"`, `text`                          | Other block properties must be empty or omitted; text may be empty                               |
| Image | `type: "image"`, `src`, `width`; optional `alt` | Existing local upload; integer width 80–2,000; alt at most 300 characters; src at most 500 bytes |
| Link  | `type: "link"`, `text`, `href`                  | Nonblank text 1–500 characters after trimming; href at most 2,048 bytes                          |

Do not include incompatible properties, such as `href` on an image. Managed image and cover URLs must start with `/api/v1/media/` and identify an existing server upload. Authors can attach their permitted uploads; administrators can attach another account's uploads. Remote image URLs can be used in sanitized HTML source but are not accepted as managed image blocks or covers.

Managed links permit `http://`, `https://`, or a root-relative path starting with one `/`. They reject user credentials, protocol-relative `//` URLs, backslashes, tabs, newlines, and leading/trailing whitespace. Internal links are ordinary paths such as `/blog/architecture-basics`; they are not foreign keys and do not automatically follow slug changes.

Example save payload with two independently sized images and both link types:

```json
{
  "title": "Architecture walkthrough",
  "slug": "architecture-walkthrough",
  "excerpt": "A guide with diagrams and related reading.",
  "status": "draft",
  "category_id": null,
  "tag_ids": [],
  "cover_image_url": null,
  "content_format": "plain_text",
  "content_blocks": [
    { "type": "text", "text": "Start with the system boundaries.\n\n" },
    {
      "type": "image",
      "src": "/api/v1/media/11111111-1111-4111-8111-111111111111/22222222-2222-4222-8222-222222222222.png",
      "alt": "System boundary diagram",
      "width": 640
    },
    { "type": "text", "text": "\nRelated reading: " },
    {
      "type": "link",
      "text": "Architecture basics",
      "href": "/blog/architecture-basics"
    },
    { "type": "text", "text": " and " },
    {
      "type": "link",
      "text": "Go documentation",
      "href": "https://go.dev/doc/"
    },
    {
      "type": "image",
      "src": "/api/v1/media/11111111-1111-4111-8111-111111111111/33333333-3333-4333-8333-333333333333.jpg",
      "alt": "Detail view",
      "width": 320
    }
  ]
}
```

Replace the illustrative media URLs with URLs returned by actual uploads, and use an existing published article for the internal destination. For HTML mode, set `content_format` to `html` and put HTML source in text blocks. The server escapes plain text, renders managed images/links, and sanitizes the combined HTML. It stores a version 2 editor document:

```json
{
  "version": 2,
  "format": "plain_text",
  "content": "",
  "blocks": [{ "type": "text", "text": "Example", "alt": "" }]
}
```

The server may include empty `text`/`alt` fields on blocks; clients should read the block type. Version 2 HTML source is editing data and must not be rendered directly. Public consumers should use sanitized `content_html`.

Older clients can omit/null `content_blocks` and send an explicit format with `content_text` or `content_html`; the server creates `{version:1,format,content}`. Omitting `content_format` retains the earlier sanitized HTML plus client JSON contract. Existing documents continue to open in the editor.

## Media

| Method and path             | Contract                                                                  | Access                                 |
| --------------------------- | ------------------------------------------------------------------------- | -------------------------------------- |
| `POST /studio/media`        | Multipart form with exactly one file field named `image`, no extra fields | Author/admin; 201 `{url,width,height}` |
| `GET /media/{owner}/{file}` | Server-generated account/file UUID path with `.png` or `.jpg`             | Public image bytes                     |

PNG/JPEG content is decoded rather than accepted based on its filename or claimed MIME type. Limits: 5 MiB file input and re-encoded output, 12 million pixels, and 8,000 pixels per axis. The multipart body permits an additional 64 KiB overhead. The server re-encodes the image and generates its storage name.

Using the logged-in cookie jar from the example above:

```bash
curl --fail-with-body -sS -b "$cookie_jar" \
  -H 'Origin: http://localhost:3000' \
  -F 'image=@/absolute/path/to/diagram.png' \
  http://localhost:3000/api/v1/studio/media
```

Do not manually set the multipart Content-Type. A successful result includes a root-relative `url` and the original decoded dimensions. Set an image block's `width` independently for display. Media is public immediately and uses `Cache-Control: public, max-age=31536000, immutable`. No upload-delete endpoint exists, and removing an article reference does not delete the file.

## Administrator endpoints

| Method and path                    | Body or parameters                                         | Success                                   |
| ---------------------------------- | ---------------------------------------------------------- | ----------------------------------------- |
| `GET /studio/authors`              | Defaults page 1/limit 20; page 1–10,000, limit 1–50        | 200 `{items,total,page,limit}`            |
| `POST /studio/authors`             | `{name,email,password}`; same account validation as signup | 201 User; preserves administrator session |
| `PATCH /studio/authors/{id}`       | Complete `{name,email,disabled}` values                    | 200 User; author accounts only            |
| `GET /studio/comments`             | `status=pending` default, or `approved`/`spam`             | 200 oldest 200 comments in that status    |
| `PATCH /studio/comments/{id}`      | `{status:"approved"}`, `pending`, or `spam`                | 200 Comment                               |
| `POST /studio/taxonomy/categories` | `{name,slug}`                                              | 201 Taxon                                 |
| `POST /studio/taxonomy/tags`       | `{name,slug}`                                              | 201 Taxon                                 |

These endpoints require an enabled administrator. Updating an author cannot change a role, administrator account, or password. Disabling does not hide published articles. Taxonomy names contain 1–100 characters and slugs follow article slug syntax; duplicate names/slugs conflict. There are no taxonomy rename/delete endpoints.

Comment moderation is unpaginated. Processing pending items permits later pending records to enter the returned set; refreshing approved/spam lists alone does not expose records beyond their oldest 200. Reply approval requires an approved parent. Reapproving an already approved reply is notification-idempotent in that status; changing away and back can queue a new email.

## Rate limits and errors

Each IP has independent Redis buckets: 180 GETs/minute, 20 other writes/minute, and eight auth POSTs/minute. Media requests also pass through the read bucket. A 429 response includes `Retry-After: 60`. Only explicitly trusted proxy peers can supply the forwarded client address. Redis failure permits read fallback but causes write requests to return 503.

Errors have this envelope; the sample preserves the code structure while human-readable messages are generally Persian:

```json
{ "error": { "code": "invalid_input", "message": "<localized error message>" } }
```

| Status | Codes and meaning                                                                            |
| ------ | -------------------------------------------------------------------------------------------- |
| 400    | `invalid_input`: validation, invalid JSON, or invalid relationships                          |
| 401    | `unauthorized`: missing or invalid protected session                                         |
| 403    | `forbidden`, `origin_denied`, `csrf_rejected`: role, ownership, or Origin failure            |
| 404    | `not_found`: missing or non-public record/media                                              |
| 409    | `conflict`: uniqueness, invalid database relation, or archived/deleted save                  |
| 413    | `image_too_large`: upload size/dimension limits; oversized ordinary JSON instead returns 400 |
| 429    | `rate_limited`: per-IP bucket exhausted                                                      |
| 503    | `rate_limit_unavailable`: write limiter unavailable                                          |
| 500    | `internal_error`: unexpected server failure; database details are not exposed                |

## Internal health and cache webhook

Go exposes `/healthz` for process liveness and `/readyz` for PostgreSQL/Redis readiness on its internal address, outside `/api/v1`. Readiness returns 503 with `not_ready` when dependencies fail. The standard gateway does not expose these paths as API health routes.

Next.js exposes `POST /api/revalidate`, separate from Go's API prefix. It requires `Authorization: Bearer <REVALIDATION_SECRET>` and a JSON body such as `{ "slugs": ["old-slug", "new-slug"] }`, at most two valid canonical slugs and 4,096 body bytes. It revalidates `posts`, each article tag/path, the home page, sitemap, and RSS. This is an internal integration credential rather than a user JWT. Success is `{ "revalidated": true }`; webhook errors use their own `{error: string}` shape and return 400, 401, 413, or 503.
