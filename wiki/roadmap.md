# Путь разработки и критерии закрытия

## Контекст
Исходная нумерация 0–7 сохранена. После этапов 1–5 по поручению владельца реализована и проверена фаза 6. Все 32 MUST остаются в MVP; инфраструктурная приёмка и release acceptance выполняются фазой 7. Закрытие development-фаз не разрешает production запуск.

**Статус на 2026-10-11: этапы 1–6 закрыты по локальным development gates.** Полная release приёмка остаётся в фазе 7.

## Фазы и результаты

| Фаза | Результат | Проверяемая граница |
|---|---|---|
| 0. Контрактная база | Requirements, ADR, threat model, state machines, traceability | Устранены противоречия, полный набор requirements сохранён; contracts синхронизированы с runtime. |
| 1. Платформа | Go/Next.js/Compose, migrations, auth/roles/CSRF, settings revisions, internal metrics, CLI provisioning, CI | Unit/race, DB bootstrap/upgrade/checksum, negative auth tests, TS/build, local proxy isolation. |
| 2. Передача | tus/resume, Local/S3, SHA/scanner, draft tree, immutable publish, links/PIN, acquire/Range/fencing, cleanup | Browser upload→publish→recipient ZIP, interrupted PATCH/HEAD resume, concurrency/revoke/fault/retention tests. |
| 3. ZIP/криптография | Deterministic ZIP64/Ed25519, verifier, FXP1 AEAD, class key rotation, migration, freeze/forensic | Independent binary vectors, tamper/nonce bounds, atomic ZIP/Range, stale worker/rotation/hold tests. |
| 4. Коммуникации | SMTP TLS/templates/outbox, HMAC webhooks, retry/DLQ, WS+poll, preferences/digest/quiet hours | TLS fixture delivery and certificate rejection, SSRF/replay transaction, delivery faults, session revocation/privacy. |
| 5. Отчёты | Immutable PDF/CSV/JSON snapshots, three hashes, public verify/trash, export jobs, credentials, saved views/bulk, encrypted QR | Snapshot/retry/privacy/Range/authorization/invalidation tests, browser PDF/JSON/CSV, visual inspection. |
| 6. Эксплуатация | BackupProvider, encrypted backup, mandatory pre-restore backup, reconciliation, reopen gate, KEK recovery/rewrap, incidents/checkpoints, schedules/metrics, expanded CLI и ADMIN exports | **Закрыта локально:** 8 новых PG/race drills, archive tamper/Local/S3 unit tests, Chromium backup→restore→restart→reauth→reconcile→reopen, PDF/mobile visual, contracts/build/security. |
| 7. Приёмка | Production TLS/ACME, performance/DR/security, browser matrix, accessibility acceptance, SBOM/license/container/secret scans | Все MUST с release evidence, residual risks приняты владельцем, нет critical/high. **В работе: начат 7.1, HTTPS/proxy/CIDR и локальная приёмка. Не закрыта.** |

Итог текущего прогона: [session](session.md). Доказательства и ограничения: [development](development.md), устройство восстановления: [runbook](operations-recovery.md). В [матрице](../docs/test-matrix.md) 27 `verified` по локальным тестам и 5 `partial` с оставшимся scope фазы 7. Ни одно требование не исключено.

## Порядок продолжения
Следующая работа: (7.1) production TLS/ACME/trusted-proxy/CIDR configuration и controlled upgrade, (7.2) load/DR/security/SBOM/container/license/secret/browser/a11y evidence, (7.3) release review и явное рассмотрение residual risks владельцем. Начат 7.1: [HTTPS/proxy/CIDR и host reset](production-ingress.md); результат локальных проверок — в session. Реальный deployment требует отдельной команды пользователя.

## Принцип закрытия
Фаза закрывается при реализации вертикального сценария, зелёных относящихся к нему проверках и синхронных wiki/contracts/changelog. Наличие файлов, успешная компиляция или пропущенные integration tests не являются доказательством. Ни ограничения env, ни открытые release risks нельзя скрывать отметкой «готово».
