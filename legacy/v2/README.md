# Classic v2 archive — May 2018

This folder preserves the original Ghost on GitHub Pages demo associated with [Classic v2.0.1](https://github.com/paladini/ghost-on-github-pages/releases/tag/v2.0.1). Its pages were generated from Ghost 1.22.5 using Buster, a Python 2 static generator. The retained files are historical static output, not the current application's server code.

## Known defects preserved in the snapshot

| Issue                        | Example                                                        |
| ---------------------------- | -------------------------------------------------------------- |
| Localhost metadata and links | `http://localhost:2373/` in canonical, OpenGraph, and tag URLs |
| Incorrect sitemap location   | `robots.txt` references a localhost sitemap                    |
| External assets              | Some images depend on `casper.ghost.org`                       |

These defects are intentionally retained for historical comparison. See the upstream [localhost-link issue](https://github.com/paladini/ghost-on-github-pages/issues/22) for context.

## Find the relevant documentation

- [Frozen static files](static/)
- [Historical Ghost demo guide](../../docs/LEGACY_DEMO.md)
- [Current ExpertsAcademy application](../../README.md)

The historical archive URL recorded by the demo was `https://paladini.github.io/ghost-on-github-pages-demo/legacy/v2/static/`; current remote availability is not verified here. Modern Compose deployment serves ExpertsAcademy rather than this snapshot.
