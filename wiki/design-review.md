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

## Review фазы 6
- Restore не выполняет SQL из archive: exact migration/table fingerprint, quoted known tables, FK/check constraints и atomic rollback проверены fault injection. `ops_*` gate/ledger не возвращается в старое состояние.
- Shared runtime locks охватывают streams/worker side effects, exclusive restore ждёт их освобождения. Persisted generation запрещает старым процессам продолжить работу после import.
- KEK usage reservations переживают failed import, root DEKs после restore исчерпываются, fresh-instance recovery требует новый master. Rewrap сохраняет plaintext audit key; исторические backups проверяются историческим KEK.
- Incident containment коммитится до попытки encrypted report. Recovery checkpoints bind historical evidence обоих журналов; последующая порча старого prefix обнаруживается, старые строки не переписываются.
- Crypto-health snapshot согласован при class rotation. Restore не выдаёт missing/tampered bytes и не публикует orphan inventory. Reopen привязан к свежему matching report и reviewed capability resurrection.
- ADMIN чужого export использует отдельный audited credential: reauth/role проверяются непрерывно; истёкший artifact не возвращается в ready изменением TTL.

## Поток доказательств
Requirements → ADR/contracts → SQL guards/storage/transport → negative/concurrency/fault tests → browser through Traefik → visual/axe/contracts. Датированные локальные evidence: [история проверок](history/development-evidence-2026-10-11.md), [matrix](test-matrix.md). Binary primitives остаются стандартными AES-GCM/HKDF-SHA256/Ed25519, [format](encryption-format.md) содержит offsets и limits.

## Границы и следующий этап
SMTP/S3 проверены локальными TLS wire fixtures, реальная инфраструктура не подключалась. HTTP completed доказывает server-observed coverage, не fsync устройства получателя. HMAC audit в одной БД не противостоит host+DB+key compromise без внешних checkpoints. IPv4/DNS pin относится к outbound webhooks; SMTP/S3 endpoints — privileged ADMIN configuration. Локальные TLS/forwarding, five-profile browser, load/DR и SBOM/security проверки реализованы и прошли для 7dce226; реальные ACME/host/devices/AT/providers и independent acceptance остаются открытыми. R-01–06 не приняты. Production env запрещён.

## Review первого среза фазы 7
Network settings используют прежние draft/validate/CAS/audit вместо отдельного config store. При изменении policy/origin generation fencing отменяет также ADMIN streams, а reset не открывает gate. Exact proxy /32 и single-valued XFF/XFP исключают доверие arbitrary chains; allowlist и login bucket используют один effective IP. Generation и policy применяются одной transaction. HTTPS generator не развёртывает сервисы; standalone profile не публикует DB/Redis/API ports и не монтирует Docker socket. Production startup guard сохраняется до полной release приёмки; [ADR-007](adr/007-ingress-and-network-policy.md).

## Review controlled upgrade
[ADR-008](adr/008-controlled-upgrade.md) сохраняет existing gate, runtime/runner locks и audit. Startup и preflight должны одинаково отвергать неизвестную/повреждённую/неполную историю migrations. Pending DDL требует closed gate и exclusive runtime lock; bootstrap не удерживает shared lock при ожидании exclusive. Drain меняет generation и не открывает сервис при timeout. Backup остаётся явным operator step до миграций; ciphertext/key formats и обязательный pre-restore backup не меняются. Downgrade после DDL не предполагается безопасным. Реализация и evidence текущего шага проверяются отдельно в session.

## Review технического завершения фазы 7
[ADR-009](adr/009-release-evidence.md) описывает source-bound gates и отдельные решения владельца. Staticcheck/ESLint/strict types не меняют HTTP schemas; типы UI генерируются из существующего OpenAPI. Reconciliation acknowledgement связан с ID просмотренного report. Skip link получает фокусируемый main. Контрактный comparator отклоняет изменения существующих schema/operation относительно baseline f68fc08.

Container review проверяет фактический image ID, а не название upstream tag. Web runtime не содержит npm/yarn; PostgreSQL entrypoints используют su-exec вместо старого gosu; Redis OS libraries и Traefik Go/x/net обновлены. В scanner allowlist разрешён лишь точный публичный crypto test vector. Сканирование не отключает unfixed findings. Собственные TLS S3 и fault/load fixtures не попадают в runtime images; они не являются production implementations.

Release readiness закрыт при отсутствующем/устаревшем/failed evidence, неполной browser matrix, отсутствии multi-GB/large-export профиля, неполной traceability или отсутствующих human decisions. Скрипт не подтверждает подлинность решения аудитора, не заменяет независимый review и не выполняет publish/deploy. Production startup guard сохранён. Решение владельца от 2026-10-11: внешнее окружение не подготовлено, соответствующие блокеры остаются открытыми.
