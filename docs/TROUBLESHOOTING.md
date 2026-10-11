# Troubleshooting

## ExpertsAcademy platform

Start with the [operations troubleshooting guide](OPERATIONS.md#troubleshooting) for database credentials, migrations, Origin errors, rate limits, upload failures, and mail. The [health and cache section](OPERATIONS.md#health-logs-and-cache-recovery) covers readiness and stale published content. For editor controls and user-facing problems, see the [user guide](USER_GUIDE.md#common-problems).

From the repository root:

```bash
docker compose ps -a
docker compose logs --tail=100 postgres redis migrate api web
```

An exited migration service with code 0 is normal. Preserve data volumes while troubleshooting; deleting them is not a password or migration repair procedure.
