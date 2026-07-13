# Restore Drill Report

- Backup started: 2026-07-13T07:22:15.9629386Z
- Restore completed: 2026-07-13T07:22:18.0432289Z
- Format: PostgreSQL custom logical dump
- Dump size: 22,519 bytes
- Restore target: fresh local `stockrush_restore_drill` database
- Migration version after restore: 5
- Invariant check: PASS, zero violations
- Restored API readiness: PASS
- Data: synthetic only

The dump remained under Docker `/tmp`, was not committed, and the restored API ran in a separate container against the fresh database.
