# Documentation

These guides describe the current ExpertsAcademy application. Documentation is in English; the publication and studio interfaces are Persian and right-to-left. UI labels in the guides are described in English.

| Guide                                 | What it covers                                                                                |
| ------------------------------------- | --------------------------------------------------------------------------------------------- |
| [Project README](../README.md)        | Docker quick start, first administrator, and project layout                                   |
| [User guide](USER_GUIDE.md)           | Registration, passwords, images, resizing, links, publishing, comments, and author management |
| [Development](DEVELOPMENT.md)         | Native development, dependencies, tests, SQL generation, and contribution workflow            |
| [Architecture](ARCHITECTURE.md)       | Service boundaries, database design, security, caching, media, and email delivery             |
| [API reference](API_REFERENCE.md)     | Authentication, endpoints, request examples, content blocks, errors, and limits               |
| [Operations](OPERATIONS.md)           | Environment variables, production deployment, backups, migrations, and troubleshooting        |
| [Troubleshooting](TROUBLESHOOTING.md) | Common platform problems and the archived static validator's diagnostics                      |
| [Verification](VERIFICATION.md)       | Recorded local checks, their scope, and outstanding production validation                     |
| [Legacy demo](LEGACY_DEMO.md)         | Historical Ghost/GitHub Pages artifacts and publishing caveats                                |

## Common tasks

- **Start the project:** follow the [Docker quick start](../README.md#run-locally), then [create the first administrator](../README.md#create-the-first-administrator).
- **Add images or links to an article:** follow [inline images](USER_GUIDE.md#inline-images-and-resizing) and [article links](USER_GUIDE.md#internal-and-external-links).
- **Manage accounts or comments:** use the separate [author management](USER_GUIDE.md#manage-author-accounts) and [comment management](USER_GUIDE.md#moderate-comments) tabs.
- **Work on application code:** follow [native development](DEVELOPMENT.md#run-the-applications-natively) and [verification commands](DEVELOPMENT.md#verification-commands).
- **Deploy or recover data:** follow [production deployment](OPERATIONS.md#production-deployment) and [backup and restore](OPERATIONS.md#backup-and-restore).
- **Integrate another client:** read the [HTTP conventions](API_REFERENCE.md#http-conventions) before implementing authentication or writes.

## Current scope and limits

Public registration is open and creates authors who can publish their own articles. There is no invitation, email verification, password reset, or editorial approval workflow. Administrator provisioning uses the CLI; the account UI does not change roles or manage administrator accounts.

The studio article list is capped at 200 records per API request and has no pagination. Its archive and ownership filters operate on those returned records. Comment moderation returns the oldest 200 comments in a selected status without pagination. Author management is paginated. Public comment trees are not paginated and allow at most eight levels.

Uploaded media is public even when referenced by a draft. Removing an image reference does not delete its file. The application has no media cleanup job, autosave, automatic slug redirects, or permanent article deletion endpoint.

Next.js caches and local upload volumes require additional coordination for multiple hosts. Server-rendered API requests can share the Next.js server's IP rate limit. Production DNS/TLS, real SMTP delivery, load testing, dependency advisory scans, and a production restore remain separate validation tasks; see the [verification record](VERIFICATION.md).
