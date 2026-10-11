# Historical Ghost content seed

This folder contains `demo-posts.json`, a Ghost export used to reproduce the earlier static demo. It is not a PostgreSQL seed for ExpertsAcademy, and Docker startup does not import it. A fresh platform database contains no sample articles or default accounts.

## Reproduce the historical demo

Use a dedicated Ghost/archive environment. The export metadata identifies Ghost 6.50.0. Set up its administrator account before importing. In Ghost Admin, use Settings → Advanced → Import/Export and the Universal import option to upload `demo-posts.json`, following the [official Ghost import guide](https://ghost.org/help/imports/).

For a compatible Ghost CLI installation, start the server before running its authenticated import command. `/absolute/path/to/repository` must point to this checkout:

```bash
cd ~/.ghost/current
ghost start
ghost import /absolute/path/to/repository/content-seed/demo-posts.json
```

The CLI import prompts for credentials; see Ghost's [restore instructions](https://ghost.org/docs/reinstall/). The export contains demo posts about v3, getting started, and the Classic v2 archive. `scripts/sanitize-seed.js` is the repository's helper for stripping sensitive export data before committing it. Configure credentials through Ghost's own administrator workflow. Historical `seed-content.js`/`seed-content.ps1` scripts use fixed local demo credentials and modify Ghost content/settings; they are not platform seeders.

For Windows archive generation, run from the repository root:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/generate-static.ps1
```

That script generates and postprocesses `static/` itself; it does not deploy ExpertsAcademy. Review its hardcoded Ghost paths and historical domain first. The Bash `scripts/deploy-demo.sh` is an archive publication script that commits and force-pushes its Ghost publish checkout; read the [legacy guide](../docs/LEGACY_DEMO.md) before using it.

## Current application

Use the [project README](../README.md) to start ExpertsAcademy and the [user guide](../docs/USER_GUIDE.md) to create articles through its studio. Moving Ghost content into the current database requires an explicit import implementation; one is not included.
