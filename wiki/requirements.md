# Актуальные требования

## Контекст, область и приёмка

Все 32 MUST ниже входят в согласованный MVP: 16 FR, 7 SEC, 5 OPS, 4 NFR. Перенос документации 2026-10-11 не меняет их формулировок. Реализация/тесты — [generated matrix](test-matrix.md), семантика — [decisions](decisions.md) и ADR, текущие границы — [current state](current-state.md). `verified` в матрице связан с локальным evidence; выпуск требует полной [приёмки](acceptance-criteria.md). Исключения и принятие рисков не выполняются автоматически.

Нет согласованного отдельного урезанного релиза, организационной tenancy/billing или production SLA. Исходное интервью до вопроса 978 отсутствует, поэтому эти записи являются доступным нормативным baseline. Новое требование проходит решение владельца и traceability, а не добавляется по предположению.

Дополнение владельца 2026-10-12: WebDAV для штатных файловых менеджеров Windows/macOS/Linux. [ADR-011](adr/011-native-webdav.md) расширяет прежний upload-only профиль: владелец читает, заменяет и переименовывает файлы своего черновика, применяются durable locks/conditions. После publish DAV закрыт; recipient one-time, FR-002–004, SEC-001/003/005/006 и NFR-001 сохранены. tus остаётся каналом resume. Фактическая проверка Windows ожидает стенда по решению владельца; Office collaboration не входит в подтверждённую совместимость. Привязка новых проверок к существующим инвариантам — в matrix; исходные 32 MUST ниже не исключены и не объявлены заново принятыми.

## Функции
- **FR-001**: RU/EN минималистичный B2B/SaaS, light/dark/system, responsive, WCAG 2.2 AA, доступная DnD зона.
- **FR-002**: draft, загрузка файлов/каталогов через tus, автоматический resume, cancel, finalizing с серверным SHA-256, file-type allow/deny/disabled, optional scanner abstraction.
- **FR-003**: draft tree/table, virtual folders, multi-select, rename/move/delete, поиск/сортировка, bulk operations. После publish immutable file/path/name set; минимум один ready файл.
- **FR-004**: atomic publish с pre-review и immutable published manifest + manifestHash, TTL от publishedAt, absolute max TTL; post-publish редактируются только разрешённые metadata/links.
- **FR-005**: общая опциональная и индивидуальные recipient links, независимые PIN/password, QR PNG/SVG, QR bundle, rotation/revoke, recoverable encrypted access token и verification hash.
- **FR-006**: recipient GET/HEAD не инициируют download, явное действие; Range/resume, session-bound opaque file refs, PostgreSQL leases/fencing/heartbeat.
- **FR-007**: individual successful download → logical delete → asynchronous physical delete; revoke/expiry немедленно прерывают поток. Download all — atomic streaming ZIP из всех оставшихся файлов; failure не удаляет ничего.
- **FR-008**: ZIP сохраняет относительные пути, безопасные имена, содержит SHA256SUMS.txt, Ed25519 Base64 sig и PEM public key. Сам payload byte-for-byte immutable.
- **FR-009**: USER download journal без recipient correlation/IP/UA; Download all одна агрегированная запись (time/count/total bytes/status); ADMIN extended audit/report.
- **FR-010**: SMTP STARTTLS/SMTPS, RU/EN versioned templates, resend, retry/DLQ, webhook HMAC/SSRF/replay protection, expiry reminder и финальные письма.
- **FR-011**: WebSocket + polling, in-app/browser/email notifications, digest hourly/daily/weekly, quiet hours только для пользовательских писем, cross-tab synchronization.
- **FR-012**: user/admin reports PDF/CSV/JSON с immutable revisions/canonical SHA-256; публичная verification только user reports; trash/restore/purge history; exports CSV/JSON.
- **FR-013**: async exports, streaming compression ZIP/GZIP по размеру, progress/cancel/retry, per-user/global concurrency, admin priority anti-starvation, encrypted ArtifactStorage, one-time hashed artifact credentials + Range.
- **FR-014**: большие exports используют согласованный snapshot значений на момент начала job; гибрид PostgreSQL + encrypted ArtifactStorage, короткие DB snapshot transactions, crash recovery и cleanup.
- **FR-015**: личные saved views, cursor pagination, filters, URL без raw search text; bulk user/admin actions с dry-run, server-side selection snapshot, progress, cancel.
- **FR-016**: `/help` RU/EN, глобальный banner, `/status` только текущий агрегированный статус, no PWA.

## Безопасность
- **SEC-001**: USER/ADMIN, email login, Argon2id, password history, no forced periodic rotation, login history, anti-enumeration, CSRF, session management, granular API scopes.
- **SEC-002**: tamper-evident audit, integrity checks, incidents, checkpoints, security reports, no user-controlled audit purge.
- **SEC-003**: KEK bootstrap, class DEKs, key rotation, encrypted configuration secrets, encrypted ArtifactStorage/BackupProvider, chunked AEAD payload encryption when enabled.
- **SEC-004**: Ed25519 signs only ZIP checksum manifest, signing key encrypted in PostgreSQL, historical public keys forever, offline Go verifier CLI.
- **SEC-005**: strict TLS/Traefik/ACME, CSP/no-referrer/no-store, secrets redaction, no body/raw query logging, SSRF controls, trust only Traefik forwarding, IPv4 required.
- **SEC-006**: freeze transfer/file indefinitely, forensic plaintext/ZIP for ADMIN and raw ciphertext only for integrity failures, with re-auth/reason/audit; USER not notified.
- **SEC-007**: file integrity_failed/storage_missing fail closed; file-level freeze blocks Download all; admin repair/check/force delete semantics.

## Эксплуатация и контракты
- **OPS-001**: PostgreSQL source of truth, Redis ephemeral, transactional outbox, idempotent workers, scheduler distributed leases.
- **OPS-002**: multiple payload Local/S3 providers with priority/fallback, sticky tus, drain/disable, migration dry-run/pause/resume/cancel; one active ArtifactStorage; separate BackupProvider Local/S3.
- **OPS-003**: encrypted PostgreSQL backups without payload, restore CLI/Admin UI, pre-restore backup, post-restore audit/crypto/storage reconciliation, explicit admin gate before data plane resumes.
- **OPS-004**: scheduled storage checks, audit verification, mandatory cleanup; health/readiness/metrics internal only; admin dashboards and alerts.
- **OPS-005**: Docker Compose one VPS, non-root, isolated networks, no Docker socket, production safety checks, auto migrations at startup under lock, controlled upgrade downtime.
- **NFR-001**: REST/JSON `/api/v1`, tus, Range, WebSocket, OpenAPI 3.1 + curl, AsyncAPI, versioned JSON Schemas, backward-compatible major versions.
- **NFR-002**: JSON camelCase, enum snake_case, RFC3339 UTC API, IANA timezone UI, PDF locale preferences, CSV UTF-8 no BOM/comma/RFC4180/formula injection protection.
- **NFR-003**: tests unit/integration/E2E/race/fuzz/fault/visual/a11y, dependency/container/secret/license scans, SBOM, release checklist, requirement→test matrix.
- **NFR-004**: proprietary code; no unapproved copyleft, no critical/high security findings at release, no production deployment without explicit authorization.

## Дополнение владельца: white labeling
2026-10-13 разрешён настраиваемый общий брендинг установки: название, логотип/favicon, две палитры, стиль иконок, ADMIN preview/apply/reset и согласованное отображение в UI/письмах/новых PDF. [ADR-013](adr/013-white-label.md). Расширяет FR-001/010/012/016, сохраняет исходные 32 MUST и strict contracts.
