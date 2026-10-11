# Реализация этапов 1–5

## Контекст и задача
Пользователь разрешил автономно привести документацию в порядок, выбрать путь разработки и закрыть этапы 1–5. Сохранены стек, все 32 MUST, one-time semantics, USER/ADMIN privacy и запрет самостоятельного production deploy. Точная проверяемая карта — [матрица](../docs/test-matrix.md), команды/evidence — [development](development.md).

## 1. Платформа и identity
Go API/worker/CLI/verifier, Next.js, Compose/Traefik, PG/Redis, CI; migrations 0001–0013. Argon2id 64 MiB/t3/p2, 12–1024 byte passwords, последние 5 паролей, no forced rotation. Web sessions/API token hashes, exact Origin+CSRF, SameSite Strict, auth_epoch revocation, granular scopes, session/token/login history UI. ADMIN re-auth 5 min. Typed settings revisions validate/apply/rollback и HMAC-chain audit. Internal health/ready/metrics недоступны через public proxy.

## 2. Передачи
tus с HEAD resume, файловые и каталоговые uploads, tree/table/search/sort/multi-select, virtual folders, atomic rename/move, cancellation. Local/S3 provider selection, server finalization и policy/scanner. Pre-review publish, independent recipient PIN/password, rotate/revoke links. PostgreSQL acquire/leases/fencing, single Range, strict stream guards, atomic ZIP consume и hold-aware physical cleanup. [Подробности](file-delivery.md).

## 3. ZIP, crypto и retention
Deterministic ZIP64 STORE с checksum/signature/PEM. Offline verifier разделяет signed manifest check и явный --verify-files. FXP1 AAD фиксирует IDs/salt/key/length/index; HKDF chunk identity предотвращает retry nonce reuse. Root-key reservations 2^20, object bounds и independent vectors. Rotation classes, artifact credential coordination, migrations dry-run/pause/resume/cancel, indefinite freeze и audited forensic/check/delete. [Crypto review](adr/005-cryptography.md).

## 4. Коммуникации
Outbox → owner notifications + recipient invitations + signed webhook deliveries. SMTP STARTTLS/SMTPS, certificate verification, RU/EN templates v1, stable Message-ID. At-least-once retry до 8 попыток/backoff/DLQ и manual resend. Webhook HTTPS443/public IPv4/DNS pin/no proxy/no redirects; HMAC timestamp+body, signed stable eventId. Reference receiver AcceptWebhook выполняет dedup+callback в одной PG transaction, не является публичным ingress API. Потребитель должен использовать аналогичный transactional dedup. WS owner-safe cursor/events и polling fallback, cross-tab BroadcastChannel, browser permission, digest/quiet hours; invitations не задерживаются quiet hours.

## 5. Отчёты и artifacts
Repeatable-read immutable snapshots, encrypted at rest, PDF/CSV/JSON из одного revision. USER без recipient correlation; ADMIN expanded report защищён current role/reauth. documentContentHash исключает своё поле; canonicalHash включает final JSON; artifact SHA считает bytes конкретного формата. Public verification принимает только active USER; trash/purge/ADMIN/mismatch → neutral404, private history retained.

Exports materialize bounded values в PG и формируют encrypted final artifacts, с fencing, cancel/retry, auth epoch, 2 global/1 owner и aging priority. Credentials hashed/one-use claim/5 min, session-bound Range, проверка роли/TTL/report state на каждом write. QR ZIP с recipients.csv — encrypted artifact, изменение links инвалидирует его транзакционно. Saved views не раскрывают raw query в URL; bulk фиксирует server-side selection, dry-run/paused, run/cancel/progress и ADMIN reason.

## Изменённые решения и примеры
QR direct download заменён явным POST → artifact credentials; это исправление исходного privacy contract. Public tombstones скрыты по исходным decisions. Export ADR уточнён: immutable PG staging → verified encrypted final artifact без лишней промежуточной копии snapshot. Snapshot timestamp не меняется при retry уже committed values. GET/HEAD сами по себе не приобретают recipient lease; acquire — только POST.

## Ограничения и следующий шаг
Реальные внешние mail/S3 accounts не настраивались. Local Chromium/axe/visual evidence не закрывает всю browser/WCAG release приёмку. Phase 6 backup/restore/checkpoints/operations и phase 7 load/DR/security/SBOM остаются обязательными. Production env отвергается. Session фиксирует финальные результаты, риски автоматически не принимаются.
