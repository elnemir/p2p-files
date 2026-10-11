# Changelog

Опубликованных releases пока нет. Формат Keep a Changelog.

## Unreleased — этапы 1–6 и техническая реализация фазы 7

### Added
- Воспроизводимая настройка GitHub/main/Actions, CODEOWNERS, Issue/PR templates, SECURITY policy, Dependabot для четырёх ecosystems, CodeQL/dependency review, repository validation и общие editor/Git/Node defaults. Применение и CI подтверждаются в session.
- Скрин-тур по локальному интерфейсу, записанный видеотур с русской озвучкой/субтитрами, автономная галерея и воспроизводимый сценарий на отдельном Compose-стенде. Внешняя приёмка не подменяется демонстрацией.
- Единая wiki: current state, developer workflow, карта компонентов, configuration/testing reference, backlog с зависимостями и полная внешняя acceptance matrix. Исторические результаты отделены от текущих инструкций.
- Staticcheck, ESLint/hooks/JSX-a11y, Prettier и generated OpenAPI UI types; conservative breaking diff для OpenAPI/AsyncAPI/JSON Schemas.
- Pinned Trivy/Gitleaks, source/history/worktree/container scans, CycloneDX SBOM и third-party license inventory с отдельным owner review.
- Пять browser/device profiles в pinned Linux container, 25 E2E scenarios, reviewed visual baselines, keyboard/error/320px/theme checks и bounded fuzz campaigns.
- Multi-GB tus/Local→TLS S3/rotation/Redis-loss concurrency, large-export SIGKILL/retry/VACUUM, backup/restore/reconcile/reopen/ZIP hash performance harness.
- Source-bound release evidence, fail-closed readiness, пять cross-built verifier targets/checksums и unpublished release review bundle.
- Фаза 7.1: audited host upgrade prepare, generation/drain, отдельный controlled image replacement/downtime drill с verified backup, startup faults и explicit reopen.
- Фаза 7.1: trusted proxy/CIDR network policy в PG settings, effective IP для login rate limit, runtime fencing и host reset при ошибочной сетевой конфигурации.
- Генератор HTTPS/ACME Compose с immutable images, non-root/private networks и isolated TLS acceptance harness; production startup guard сохранён.
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
- OpenAPI: 107 операций; traceability: 32 требования; unit/race, 36 PG integration tests, 25 browser E2E, TLS ingress и controlled upgrade drills, axe/visual и response contract validation.

### Fixed
- Галерея тура принимает только локальные PNG paths перед src/href; тестовый S3 fixture строит имя файла из канонического UUID. Добавлены негативные browser/wire проверки после сигналов первого CodeQL анализа, без исключения файлов из scan.
- 20 страниц проектной документации перенесены в wiki; ссылки, AGENTS/entrypoints, CLI help, docs checker, matrix generator и release bundle используют канонический путь. Устаревшие статусы фаз/проверок исправлены без принятия внешних blockers.
- Vulnerable npm tooling удалён из web runtime; PostgreSQL gosu заменён su-exec; исправлены zlib/nghttp2/OpenSSL и Go/x/net в инфраструктурных images. Generator требует пять проверенных immutable digests.
- Skip link переводит keyboard focus в main; UI использует точные типы API и привязывает recovery acknowledgement к просмотренному report.
- Startup и read-only preflight используют strict непрерывный migration prefix; unknown/checksum/hole отклоняются до SQL. Pending DDL требует closed gate/exclusive runtime lock и атомарно инвалидирует прежнюю generation/reconciliation; bootstrap не удерживает shared lock при ожидании exclusive.
- Повторяющиеся test IDs OPS-005 устранены; docs gate проверяет уникальность всех traceability test IDs.
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
- Внешнее окружение не подготовлено (решение владельца 2026-10-11): real host/DNS/ACME/SMTP/S3/backups/KEK/checkpoints и target load/DR acceptance заблокированы. Реальные browser/device/assistive-technology, независимый review, лицензии и R-01–06 остаются открытыми. Production bootstrap отклоняется.
- Local fixtures не доказывают совместимость реальных SMTP/S3 accounts. CI реализации фазы 7 подтверждён для 7dce226; статус последующих изменений — в session. Production deployment не выполнялся, residual risks не приняты. См. [историю evidence](wiki/history/development-evidence-2026-10-11.md) и [session](wiki/session.md).
