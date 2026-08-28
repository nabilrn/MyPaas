# Beta backup and restore drill

**Status:** Historical qualification procedure; retained for recovery provenance.  
**Current source of truth:** `scripts/backup-restore.py`, current production configuration, and [`../engineering/beta-readiness-gates.md`](../engineering/beta-readiness-gates.md).

This document records the intent of the beta backup/restore qualification that has already been completed. It is **not** an instruction to repeat the full drill after unrelated changes.

## What the qualification established

The retained beta runtime verification records fresh-host recovery of relevant MyPaaS state, including:

- control-plane PostgreSQL state;
- production configuration required to decrypt/use persisted project environment values;
- static releases;
- managed persistent application state within the supported backup boundary;
- Compose/runtime recovery state;
- project/deployment history;
- Caddy route recovery;
- DB Studio state/connectivity within its supported database boundary.

A successful restore does not imply every Docker/Podman volume layout is portable. Current migration/backup preflight owns that decision.

## Current operational rule

Use the current script contract rather than copying historical commands blindly:

```bash
python3 scripts/backup-restore.py --help
```

Before destructive recovery:

1. verify the backup/bundle with the current tool;
2. use a checkout compatible with the backup metadata and current restore contract;
3. preserve the production encryption/configuration material required by the backup;
4. use a disposable or intentionally selected recovery target;
5. run the current production deployment/verification path after restore;
6. verify application routes, persistent sentinels, project history, encrypted-env usability, and supported database access without exposing secrets.

The exact current command flags are defined by `scripts/backup-restore.py`; this historical document must not override them.

## Evidence safety

Backup bundles and recovery evidence can contain or depend on sensitive configuration.

Never publish:

- production `.env` contents;
- registry credentials;
- OAuth/JWT secrets;
- decrypted project environment variables;
- cookies/tokens;
- database credentials;
- backup material containing those secrets.

A qualification report may retain tested Git SHA, environment shape, timestamps, checksums, project identifiers, and pass/fail status when those values are not secrets.

## Regression rule

Repeat a fresh-host backup/restore qualification only when a change materially touches:

- backup bundle composition;
- encryption/config restoration;
- persistent-storage capture/restore;
- database restore behavior;
- migration portability rules;
- route/runtime reconstruction after restore.

Do not repeat this drill merely because an unrelated deployment, UI, template, compatibility, or documentation change landed.

## Historical failure rule

During qualification, missing mandatory state, checksum mismatch, unreadable encrypted data, lost persistent sentinel data, failed database restore, or failed recovered route was treated as a real failure rather than a caveat. That principle remains valid even though the original beta workstream is complete.
