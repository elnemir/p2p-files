# Architecture consistency и crypto design review

Дата: 2026-10-11. Внутренний review кода/документации; independent audit и принятие release risks не заявляются.

## Контекст и результат
Исходный комплект расходился с частично реализованным кодом. Review проверяет реальные потоки, не приравнивает green build к выполнению MUST. PostgreSQL source of truth, immutable publish, transactional outbox, USER privacy и one-time completion сохранены.

## Исправленные расхождения
- QR bundle теперь encrypted temporary artifact с email mapping, transactional link revision и invalidation.
- Public report verify подтверждает только active USER, включая exact tuple POST; trash/purge дают neutral404. Hash слои document/canonical/artifact разделены.
- Offline verifier по умолчанию проверяет signed metadata; payload hashing — --verify-files.
- Payload FXP1 связывает весь header; новый chunk UUID/subkey для каждого retry предотвращает reuse deterministic nonce. Root envelope attempts имеют durable budget и fail closed. Fixed vectors независимы от Go implementation.
- Writer guards проверяют role/session/hold/expiry перед и после bounded writes. Artifact credential issuance исключает concurrent key rotation; ADMIN export worker сверяет authorization epoch.
- Maintenance fair scheduling обходит frozen jobs; config rotation не завершает job при пропущенных locked references. Atomic recursive draft moves исключают half-applied state.
- Export snapshot хранит values. ADR-003 уточняет PG staging → verified encrypted final artifact, crash/retry сохраняет committed snapshot. Empty claim commit сохраняет отмену jobs при смене полномочий.

## Поток доказательств
Requirements → ADR/contracts → SQL guards/storage/transport → negative/concurrency/fault tests → browser through Traefik → visual/axe/contracts. Текущие локальные evidence: [development](development.md), [matrix](../docs/test-matrix.md). Binary primitives остаются стандартными AES-GCM/HKDF-SHA256/Ed25519, [format](../docs/encryption-format.md) содержит offsets и limits.

## Границы и следующий этап
SMTP/S3 проверены локальными TLS wire fixtures, реальная инфраструктура не подключалась. HTTP completed доказывает server-observed coverage, не fsync устройства получателя. HMAC audit в одной БД не противостоит host+DB+key compromise без внешних checkpoints. IPv4/DNS pin относится к outbound webhooks; SMTP/S3 endpoints — privileged ADMIN configuration. TLS/ACME/trusted forwarding deployment, full a11y/browser, load/DR и SBOM/security acceptance ещё входят в 6–7. R-01–06 не приняты. Production env запрещён.
