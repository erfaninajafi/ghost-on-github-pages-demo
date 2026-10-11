# Troubleshooting

## ExpertsAcademy platform

Start with the [operations troubleshooting guide](OPERATIONS.md#troubleshooting) for database credentials, migrations, Origin errors, rate limits, upload failures, and mail. The [health and cache section](OPERATIONS.md#health-logs-and-cache-recovery) covers readiness and stale published content. For editor controls and user-facing problems, see the [user guide](USER_GUIDE.md#common-problems).

From the repository root:

```bash
docker compose ps -a
docker compose logs --tail=100 postgres redis migrate api web
```

An exited migration service with code 0 is normal. Preserve data volumes while troubleshooting; deleting them is not a password or migration repair procedure.

## Historical static demo

The error message in `scripts/validate-static.sh` points to this document. That script checks generated static files, not the Go/Next.js application. Run it against the v3 output directory:

```bash
bash scripts/validate-static.sh static
```

If it reports localhost addresses, inspect the affected generated URLs and the Ghost export's production-domain configuration. The script matches all occurrences, including tutorial text that legitimately mentions localhost; distinguish those from incorrect `href`, `src`, or metadata attributes. The static CI workflow specifically checks URL attributes and sitemap/robots targets.

Broken image extensions such as `.jpegg`, `.jpegpg`, or `.jpegjpg` require correcting the source asset reference and regenerating/postprocessing the export. Review `scripts/post-process-static.js` and the dedicated archive workflow described in the [legacy guide](LEGACY_DEMO.md). The Classic v2 snapshot intentionally retains localhost defects; do not rewrite it as part of platform troubleshooting.

The historical deployment script commits and force-pushes a Ghost publish checkout. It is not a cache repair or deployment command for ExpertsAcademy.
