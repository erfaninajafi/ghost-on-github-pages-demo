# Historical Ghost on GitHub Pages demo

This repository originally demonstrated [Ghost on GitHub Pages](https://github.com/paladini/ghost-on-github-pages). Its generated sites and content export remain as historical artifacts. The active application is now ExpertsAcademy; use the [project README](../README.md) and [operations guide](OPERATIONS.md) to run or deploy it.

## Archived artifacts

| Path                                      | Historical purpose                                                      |
| ----------------------------------------- | ----------------------------------------------------------------------- |
| [static](../static/)                      | Ghost 6.50 static export generated with gssg for the v3 demo            |
| [legacy/v2/static](../legacy/v2/static/)  | Frozen 2018 Ghost 1.22.5/buster output with known localhost URL defects |
| [legacy/v2](../legacy/v2/README.md)       | Classic demo background                                                 |
| [content-seed](../content-seed/README.md) | Ghost JSON export and reproduction notes                                |
| [index.html](../index.html)               | Historical root redirect to the static demo                             |

The historical publication target recorded in scripts and metadata is `https://paladini.github.io/ghost-on-github-pages-demo/`. Its current availability is not part of this project's local verification. The archived static directories contain generated pages; they are not a Ghost server installation.

## Relationship to the current platform

Compose serves `apps/api` and `apps/web`, not the archived `static` directory or root `index.html`. Ghost exports are not automatically imported into PostgreSQL. The current studio, registration, moderation, image storage, and API need server processes and cannot run on GitHub Pages.

The former upstream v3 setup and migration references remain available as historical context: [getting started](https://github.com/paladini/ghost-on-github-pages/blob/master/docs/v3/GETTING-STARTED.md) and [migration guide](https://github.com/paladini/ghost-on-github-pages/blob/master/docs/MIGRATION.md). Consult the upstream project's applicable version before rebuilding an archive.

## Historical maintainer scripts

- `scripts/generate-static.ps1` generates gssg output from a local Ghost instance at `~/.ghost/current`, replaces this checkout's `static` directory, and runs the postprocessor. Its domain is hardcoded to the historical demo target.
- `scripts/post-process-static.js` adjusts the generated demo assets and paths.
- `scripts/validate-static.sh` and `.github/workflows/validate.yml` check the static archive separately from platform CI.
- `scripts/deploy-demo.sh` invokes the upstream deployment in `~/.ghost`, merges preserved archive assets into `~/.ghost/publish`, commits, and **force-pushes** its `master` and `gh-pages` targets. It preserves selected legacy/script/seed/CI files and the root README, but does not package the current `apps`, `docs`, or Compose platform for deployment.

Use these scripts only in a dedicated archive-maintenance checkout after reviewing their paths and remote targets. They are not the ExpertsAcademy production release process. No historical deploy script was executed as part of the documentation revision.

## Attribution and license

The original demo was created by [Fernando Paladini](https://github.com/paladini). The preserved site was generated with [Ghost](https://ghost.org/) and, for Classic v2, [Buster](https://github.com/axitkhurana/buster). See the repository [LICENSE](../LICENSE).
