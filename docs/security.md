# Security / threat model — обязательный объём review

Активы: payload, recipient tokens, PIN hashes, sessions, API tokens, KEK/DEKs, signing private key, reports/exports, audit chain, backups. Границы: browser↔Traefik↔API, API↔DB/Redis/storage, worker↔outbound SMTP/S3/webhooks, host↔CLI.

Контроли: HTTPS TLS1.2+ / ACME; strict CSP, no-referrer, no-store, no framing; публичная страница без third-party scripts кроме разрешённого CAPTCHA provider; safe GET/HEAD; CSRF; anti-enumeration; rate/connection limits; SSRF DNS rebinding/private IP/redirect/port checks; IPv4 trusted proxy; no request bodies/credentials/raw query logs.

Аутентификация: Argon2id, local weak-password denylist, password history, no periodic forced reset; granular token scopes, IP CIDR allowlist, re-auth privileged sessions, invalidation on security events. Recipient sessions HttpOnly/Secure/SameSite=Strict, opaque file refs hashed in PostgreSQL.

Удаление: enumerate tus chunks, local/S3, migration/re-encryption copies, temp files, ZIP buffers, forensic downloads; never claim physical secure erase. No unmanaged plaintext `/tmp`. Tests must prove deletion workflows and orphan reconciliation.

Audit: tamper-evident chain, verification/checkpoints, admin incidents, immutable security events. User history deletion never deletes administrative audit.

Forensic: admin plaintext/ZIP with re-auth/reason/audit, raw ciphertext only on integrity/crypto failure; no key material export; USER not notified.


## Threat model и review
Конкретные границы/угрозы/контроли и этапы доказательства: [threat model](../wiki/threat-model.md). Результат внутреннего review: [design review](../wiki/design-review.md). Data plane подключён к session/scoped auth и проверен локально. Перечень выше — полный нормативный security scope; Incidents, exportable checkpoints и restore reconciliation реализованы фазой 6; production TLS/ACME, IP/CIDR deployment policy и независимое хранение checkpoints/keys проверяются фазой 7. Текущие границы нельзя смешивать с release acceptance.
