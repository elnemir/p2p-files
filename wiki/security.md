# Security / threat model — обязательный объём review

Активы: payload, recipient tokens, PIN hashes, sessions, API tokens, KEK/DEKs, signing private key, reports/exports, audit chain, backups. Границы: browser↔Traefik↔API, API↔DB/Redis/storage, worker↔outbound SMTP/S3/webhooks, host↔CLI.

Контроли: HTTPS TLS1.2+ / ACME; strict CSP, no-referrer, no-store, no framing; публичная страница без third-party scripts кроме разрешённого CAPTCHA provider; safe GET/HEAD; CSRF; anti-enumeration; rate/connection limits; SSRF DNS rebinding/private IP/redirect/port checks; IPv4 trusted proxy; no request bodies/credentials/raw query logs.

Аутентификация: Argon2id, local weak-password denylist, password history, no periodic forced reset; granular token scopes, IP CIDR allowlist, re-auth privileged sessions, invalidation on security events. Recipient sessions HttpOnly/Secure/SameSite=Strict, opaque file refs hashed in PostgreSQL.

Удаление: enumerate tus chunks, local/S3, migration/re-encryption copies, temp files, ZIP buffers, forensic downloads; never claim physical secure erase. No unmanaged plaintext `/tmp`. Tests must prove deletion workflows and orphan reconciliation.

Audit: tamper-evident chain, verification/checkpoints, admin incidents, immutable security events. User history deletion never deletes administrative audit.

Forensic: admin plaintext/ZIP with re-auth/reason/audit, raw ciphertext only on integrity/crypto failure; no key material export; USER not notified.


## Threat model и review
Конкретные границы/угрозы/контроли и этапы доказательства: [threat model](threat-model.md). Результат внутреннего review: [design review](design-review.md). Data plane подключён к session/scoped auth и проверен локально. Перечень выше — полный нормативный security scope; Incidents, exportable checkpoints и restore reconciliation реализованы фазой 6; локальные TLS/IP/CIDR/security проверки реализованы фазой 7; real ACME/host и независимое хранение checkpoints/keys требуют внешней приёмки. Текущие границы нельзя смешивать с release acceptance.

Реализованная networkPolicy хранится в PG settings: exact trusted proxy /32, single-hop XFF/XFP, client IPv4 CIDR allowlist, обязательные HTTPS/Host guards. Изменение policy/origin меняет generation и требует restart/reconcile/reopen; host reset снимает только client allowlist. [Ingress runbook](production-ingress.md). Public ACME/host/deployment и independent review ещё не подтверждены.

## Scoped WebDAV и доступ владельца
Отдельный digest-only credential действует только для указанного draft, с TTL/отзывом/auth epoch и теми же transport/network/recovery gates. Его пароль не сохраняется в UI storage и не равен паролю аккаунта. PG locks/If защищают также REST/tus/bulk; отзыв/freeze сильнее lock. Owner draft GET не создаёт recipient journal/consume, после publish закрывается. Атомарный encrypted staging сохраняет прежний ready file при ошибке. [Threat model](threat-model.md), [ограничения и реальные клиенты](webdav-upload.md).

## Публичный брендинг
Название/описания/палитры/логотипы намеренно публичны; secrets и сведения об ADMIN в DTO отсутствуют. Запись доступна только текущей ADMIN session со свежим reauth и Origin/CSRF, повторно проверяемыми внутри CAS/audit transaction. Растровые assets ограничены до decode и нормализуются; URL/SVG/HTML/CSS/JS не принимаются. [ADR-013](adr/013-white-label.md) фиксирует threat/architecture/crypto review, [white-label](white-label.md) — поля, ограничения и эксплуатацию. Обновление бренда не меняет security/privacy/one-time/report JSON invariants.
