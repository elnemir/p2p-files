# Changelog

Опубликованных releases пока нет. Формат Keep a Changelog.

## Unreleased — этапы 1–5

### Added
- Go API/worker/CLI/verifier, Next.js RU/EN owner/recipient UI, themes/mobile layouts, local Compose/Traefik, internal health/readiness/metrics и CI.
- Argon2id/session/CSRF/scoped tokens, account epoch/history, privileged reauthentication, typed settings revisions и HMAC audit.
- tus/HEAD resume, Local/S3 encrypted chunks, server finalization/policy/scanner, draft tree/virtual folders/atomic repath, immutable publish, PIN/links/QR.
- Fenced one-time Range/file/atomic ZIP64 delivery, server checksum manifest/Ed25519/PEM verifier, logical/physical cleanup, freeze/forensic/check/delete.
- FXP1 authenticated header/HKDF, independent binary vectors, durable key usage budgets, reference-preserving class rotation and storage migration.
- SMTP STARTTLS/SMTPS templates, outbox/retries/DLQ, HMAC/SSRF/replay webhook controls, WebSocket/polling/digests/quiet hours/cross-tab notifications.
- Immutable PDF/CSV/JSON reports with three hash layers, public USER verify, trash/history, encrypted ArtifactStorage, Range credentials, bounded export snapshots, private views/bulk jobs.
- 93-operation OpenAPI/AsyncAPI/schema sync, 32-requirement traceability, unit/race/23 PG tests, Chromium E2E/axe/visual and captured response contract checks.

### Fixed
- QR archives now encrypted with recipient mapping and transactional invalidation on link changes.
- Public report verification hides ADMIN/trashed/purged revisions; verifier defaults to signed metadata only.
- Atomic recursive moves, stale upload/maintenance fencing, fair frozen-job scheduling, quiet-hours delivery starvation and admin artifact/job authorization.
- Export cancellation persists even when claim finds no remaining eligible jobs.
- Mobile logo wrapping, PDF Cyrillic/date/state rendering, build loop fail-fast and contract observer cancellation handling.
- Documentation inconsistencies in requirement mapping, restore override, initial scaffold status and implemented contracts.

### Remaining
- Phase 6 backup/restore/reconciliation/checkpoints/operational tooling; phase 7 production TLS/ACME, DR/load/security/SBOM/license/container/secret and full browser/WCAG acceptance. Production bootstrap deliberately rejected.
- Local fixtures do not prove delivery through a real SMTP/S3 account. Remote CI/deploy not executed; residual risks not accepted. See wiki/development.md and wiki/session.md.
