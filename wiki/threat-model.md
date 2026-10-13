# Модель угроз и реализованные границы

## Контекст
Активы: payload, capability/PIN/session/API secrets, key material, reports/exports, audit, backups. Browser/recipient/другой USER недоверен. ADMIN управляет инфраструктурой и имеет audited forensic доступ. Host/DB superuser compromise требует внешних checkpoints и независимого KEK backup.

| Граница / угроза | Контроль | Проверка / оставшийся gate |
|---|---|---|
| Cross-owner read/mutation | Auth + scopes + owner predicates + neutral404 | PG identity/ownership/tree tests |
| CSRF/prefetch consume | SameSite Strict, exact Origin, CSRF, explicit acquire POST | Identity, recipient HEAD/Range, browser |
| Double consume/stale worker | PG leases/epochs/fences, atomic coverage commit | Concurrent acquire/revoke/stale upload/ZIP tests |
| Path traversal/ZIP slip | NFC safe path, root filesystem API, UUID object keys, collisions | Domain negative/fuzz and ZIP tests |
| Forged ready/checksum | Server full finalize and MIME/scanner boundary | Upload/policy/tamper integration |
| Ciphertext/metadata swap | FXP1 authenticated header, per-attempt subkeys, SHA-256, bounded parser | Independent vectors/every-byte tamper/rotation tests |
| Secret leaks | Generic errors, no body/raw query logs, CSP/no-store/no-referrer, encrypted secrets | HTTP/browser contract privacy tests |
| Webhook SSRF/replay | HTTPS443/public IPv4, DNS per connect, pin, no redirects/proxy, signed eventId/timestamp, transactional receiver dedup | Webhook guards and receipt rollback tests |
| SMTP downgrade | Mandatory verified STARTTLS/SMTPS before AUTH | Wire TLS/certificate/plaintext rejection tests |
| ADMIN privilege loss | Current role/reauth/account epoch during stream and job commit | ADMIN artifact/forensic/export demotion tests |
| Stale QR/public report | Link change invalidation, private trash tombstones, source-state guards | QR invalidation/report privacy tests |
| Resource exhaustion | 4 MiB chunks, 8 streams, 4 Argon2 ops, quotas/timeouts/row bounds | Negative tests и bounded 2050 MiB local load; target capacity acceptance отдельно |
| Retained copies | Compensation rows before Put, idempotent cleanup, hold/write lease guards | Cleanup/migration/expiry tests; restore orphan inventory/quarantine tests |
| Restore resurrection | Verified pre-backup, generation/session fencing, reconciliation/capability review, explicit reopen | Recovery/fresh DB/drain/rollback PG drills |
| Incompatible binary / concurrent DDL | Strict migration prefix, closed gate, generation drain, exclusive migration lock, explicit reopen | Controlled upgrade tests по ADR-008; текущий результат в session |
| Backup tamper / KEK budget rollback | Chunk AEAD, verified encrypted descriptor, exact schema, durable wrap reservation, fresh master on empty DB | Local/S3 wire tamper/truncate/key and fresh recovery tests |
| Audit rewrite by host attacker | HMAC chain detects modification without key; external checkpoints required | Audit+ops tamper/recovery checkpoint/immutable report tests; independent host escrow/security acceptance phase 7 |

## Примеры и ограничения
Утёкший recipient URL — bearer capability, PIN является дополнительной защитой; невозможно запретить получателю копировать уже полученные bytes. Переключение hold/epoch прерывает stream на следующем bounded write, уже переданные bytes не отзываются. SMTP/webhook имеют at-least-once delivery, поэтому повтор не маскируется как exactly-once.

По умолчанию rate limits используют socket peer и email. При настроенной networkPolicy effective IP берётся только от точного доверенного IPv4 /32 Traefik peer; duplicate/chain/malformed headers отвергаются. RequireProxy/HTTPS/Host и IPv4 client allowlist применяются до login. Policy/origin change меняет generation и закрывает gate, включая privileged streams; host reset сохраняет transport trust. HTTPS/ACME профиль генерируется и проверяется локально, public issuance/host acceptance остаются открытыми; [ADR-007](adr/007-ingress-and-network-policy.md). S3/SMTP network access разрешается доверенной ADMIN конфигурацией, private S3 endpoint не считается untrusted webhook. В development нет публичного обхода auth; production fail closed. Residual risk acceptance и независимый security audit не выполнены.

## WebDAV: дополнительная граница доверия
[ADR-011](adr/011-native-webdav.md) сохраняет scoped Basic credential, HTTPS/loopback transport, owner/draft/hold/recovery gates. Угрозы: утечка пароля, cross-draft/path/Destination injection, подмена If/Lock-Token, межпротокольный обход lock, потеря старого файла, скрытый stage в publish, crash cleanup и metadata amplification.

Контроли: digest-only token с TTL/отзывом/auth epoch; PG parent-first locks; positive If token только того же credential; отсутствие Destination network fetch; NFC/SimpleFold и Unicode-safe repath; atomic ready swap после scanner; скрытые stages/lease cleanup с freeze fence. GET выдаёт owner bytes только до publish, проверяет guard на bounded output; ранее полученные bytes отозвать невозможно. Криптография FXP1 и recipient semantics сохранены.

Ресурсные бюджеты: 8 общих stream slots, XML input 64 KiB/depth 16, If 8 KiB, 10 000 namespace nodes, ответ 8 MiB до 207, properties 32 KiB/256 на ресурс и 4 MiB aggregate, 256 active locks с owner XML 4 KiB. PROPPATCH проверяет aggregate в той же parent-locked transaction; oversized update откатывается. Количество узлов и метаданные ограничены независимо, чтобы маленький PROPFIND не создавал неограниченный ответ/объект в памяти.

Негативные/concurrency/fault cases перечислены в [матрице](test-matrix.md). Внутренний review и локальные native clients не заменяют независимый security audit, Windows стенд или external acceptance.
