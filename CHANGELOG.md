# Changelog

Опубликованных releases пока нет. Формат Keep a Changelog.

## Unreleased — этапы 1–6

### Added
- Go API/worker/CLI/verifier, Next.js RU/EN owner/recipient UI, themes/mobile layouts, local Compose/Traefik, internal health/readiness/metrics и CI.
- Argon2id/session/CSRF/scoped tokens, account epoch/history, privileged reauthentication, typed settings revisions и HMAC audit.
- tus/HEAD resume, Local/S3 encrypted chunks, server finalization/policy/scanner, draft tree/virtual folders/atomic repath, immutable publish, PIN/links/QR.
- Fenced one-time Range/file/atomic ZIP64 delivery, server checksum manifest/Ed25519/PEM verifier, logical/physical cleanup, freeze/forensic/check/delete.
- FXP1 authenticated header/HKDF, independent binary vectors, durable key usage budgets, reference-preserving class rotation and storage migration.
- SMTP STARTTLS/SMTPS templates, outbox/retries/DLQ, HMAC/SSRF/replay webhook controls, WebSocket/polling/digests/quiet hours/cross-tab notifications.
- Immutable PDF/CSV/JSON reports with three hash layers, public USER verify, trash/history, encrypted ArtifactStorage, Range credentials, bounded export snapshots, private views/bulk jobs.
- Зашифрованный BackupProvider Local/S3, проверка архивов, atomic restore с обязательной предварительной копией, durable maintenance gate, инвалидация сессий и runtime generation.
- Audit/crypto/storage reconciliation, quarantine missing/tampered данных, orphan inventory и явный ADMIN reopen после проверки восстановленных links/API tokens.
- KEK rewrap/recovery, восстановление в новую БД с новым master key, бессрочные encrypted incident reports PDF/JSON, exportable checkpoints обоих журналов.
- Operational UI/CLI, distributed schedules, private alerts/aggregate metrics, read-only upgrade preflight, отдельный ADMIN доступ к чужим exports и управление TTL с reason/audit.
- OpenAPI: 107 операций; traceability: 32 требования; unit/race, 31 PG integration test, 3 Chromium E2E, axe/visual и response contract validation.

### Fixed
- Restore ждёт завершения старых writers, не откатывает KEK usage reservations и отклоняет параллельный import; устаревшая reconciliation не разрешает reopen.
- Критическая crypto ошибка закрывает доступ до попытки записи encrypted incident report; повреждённая историческая audit prefix остаётся проверяемой после recovery checkpoint.
- Кириллица в operational reason проверяется по символам; mobile restore/reconciliation notices помещаются на 390px.
- QR archives now encrypted with recipient mapping and transactional invalidation on link changes.
- Public report verification hides ADMIN/trashed/purged revisions; verifier defaults to signed metadata only.
- Atomic recursive moves, stale upload/maintenance fencing, fair frozen-job scheduling, quiet-hours delivery starvation and admin artifact/job authorization.
- Export cancellation persists even when claim finds no remaining eligible jobs.
- Mobile logo wrapping, PDF Cyrillic/date/state rendering, build loop fail-fast and contract observer cancellation handling.
- Documentation inconsistencies in requirement mapping, restore override, initial scaffold status and implemented contracts.

### Remaining
- Фаза 7: production TLS/ACME/trusted proxy/CIDR, DR/load/security/SBOM/license/container/secret и полная browser/WCAG приёмка. Production bootstrap по-прежнему отклоняется.
- Local fixtures не доказывают совместимость реальных SMTP/S3 accounts. CI этапов 1–5 подтверждён; изменения фазы 6 ещё не опубликованы. Production deployment не выполнялся, residual risks не приняты. См. [development](wiki/development.md) и [session](wiki/session.md).
