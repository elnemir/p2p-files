# Путь разработки и критерии закрытия

## Контекст
Исходная нумерация 0–7 сохранена. Текущее поручение — реализовать и проверить этапы 1–5 без дополнительных согласований. Все 32 MUST остаются в MVP; эксплуатация и release acceptance выполняются последующими фазами. Закрытие development-фаз не разрешает production запуск.

**Статус на 2026-10-11: этапы 1–5 закрыты по локальным development gates.** Полная release приёмка остаётся в 6–7.

## Фазы и результаты

| Фаза | Результат | Проверяемая граница |
|---|---|---|
| 0. Контрактная база | Requirements, ADR, threat model, state machines, traceability | Устранены противоречия, полный набор requirements сохранён; contracts синхронизированы с runtime. |
| 1. Платформа | Go/Next.js/Compose, migrations, auth/roles/CSRF, settings revisions, internal metrics, CLI provisioning, CI | Unit/race, DB bootstrap/upgrade/checksum, negative auth tests, TS/build, local proxy isolation. |
| 2. Передача | tus/resume, Local/S3, SHA/scanner, draft tree, immutable publish, links/PIN, acquire/Range/fencing, cleanup | Browser upload→publish→recipient ZIP, interrupted PATCH/HEAD resume, concurrency/revoke/fault/retention tests. |
| 3. ZIP/криптография | Deterministic ZIP64/Ed25519, verifier, FXP1 AEAD, class key rotation, migration, freeze/forensic | Independent binary vectors, tamper/nonce bounds, atomic ZIP/Range, stale worker/rotation/hold tests. |
| 4. Коммуникации | SMTP TLS/templates/outbox, HMAC webhooks, retry/DLQ, WS+poll, preferences/digest/quiet hours | TLS fixture delivery and certificate rejection, SSRF/replay transaction, delivery faults, session revocation/privacy. |
| 5. Отчёты | Immutable PDF/CSV/JSON snapshots, three hashes, public verify/trash, export jobs, credentials, saved views/bulk, encrypted QR | Snapshot/retry/privacy/Range/authorization/invalidation tests, browser PDF/JSON/CSV, visual inspection. |
| 6. Эксплуатация | BackupProvider, encrypted backup, mandatory pre-restore backup, reconciliation, reopen gate, incidents/checkpoints, expanded CLI | Restore drills, orphan/audit/crypto reconciliation, key recovery, operational dashboards. **Не закрыта.** |
| 7. Приёмка | Production TLS/ACME, performance/DR/security, browser matrix, accessibility acceptance, SBOM/license/container/secret scans | Все MUST с release evidence, residual risks приняты владельцем, нет critical/high. **Не закрыта.** |

Итог финального прогона этапов 1–5: [session](session.md). Доказательства и ограничения: [development](development.md). Статус `verified` в [матрице](../docs/test-matrix.md) относится к указанному локальному тесту; требования, захватывающие фазы 6–7, остаются `partial`.

## Порядок продолжения
После 1–5: (6.1) backup encrypted DB без payload и KEK recovery → (6.2) обязательный backup перед restore, invalidate sessions и заблокировать data plane → (6.3) reconciliation и явное ADMIN reopen → (6.4) incidents/checkpoints/CLI. Затем (7.1) production deploy configuration, (7.2) load/DR/security/browser/a11y evidence, (7.3) release review. Deployment требует отдельной команды пользователя.

## Принцип закрытия
Фаза закрывается при реализации вертикального сценария, зелёных относящихся к нему проверках и синхронных wiki/contracts/changelog. Наличие файлов, успешная компиляция или пропущенные integration tests не являются доказательством. Ни ограничения env, ни открытые release risks нельзя скрывать отметкой «готово».
