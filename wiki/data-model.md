# Модель данных

## Контекст и изменения
PostgreSQL authoritative для identity, доступа, jobs, immutable snapshots и audit. Миграции 0001–0017 встроены в binaries и применяются под advisory lock с SHA-256 checksum. Изменение уже применённого SQL блокирует startup. Внешние bytes располагаются в Local/S3 отдельно от DB.

| Группа | Таблицы / ответственность |
|---|---|
| Bootstrap/config | schema_migrations, bootstrap_secrets, app_settings; KEK sentinel и revisions/active/validated. |
| Identity | users/auth_epoch, password_history, web_sessions, api_tokens, login_history, auth_limits. |
| Audit | audit_events, audit_checkpoints, security_incidents/security_incident_reports: serialized HMAC chain, actor/action/subject; user не может purge. |
| Передачи | transfers: immutable manifest/request hash/timestamps + lifecycle/access_epoch/hold/links_revision; files: paths/checksums/upload offset/epoch/provider/key/salt/hold; draft_folders. |
| Storage/crypto | storage_providers с purpose payload/artifacts/backups; key_versions с active/version/seal_count, crypto_usage KEK budget, signing_keys historical public keys. |
| Payload | file_chunks: stable chunk UUID/index/range/object key/plain SHA; deletion_jobs: compensation/leases/retries. |
| Recipient | recipient_links с encrypted token/hash/PIN/epoch; recipient_sessions; recipient_file_refs hashed. |
| Выдача | downloads writer/fence/coverage, download_files selection, download_journal owner-safe aggregate. |
| Maintenance | storage_migrations и rotation_jobs, pause/resume/cancel/progress/checked_at. |
| Коммуникации | outbox, notifications, notification_preferences, communication_configs (encrypted), deliveries, webhook_receipts dedup. |
| Отчёты | reports immutable revision/canonical hash/encrypted snapshot/render fence/state; report_artifacts; append-only report_history. |
| Artifacts | artifacts state/key/salt/provider/TTL/SHA/maintenance lease, artifact_chunks, artifact_credentials hash/session/coverage/fence/admin_access; qr_bundles provenance. |
| Recovery | ops_control mode/generation/instance_id, ops_jobs encrypted provider descriptors/pre-backup/progress, ops_events/ops_checkpoints, ops_secrets ledger key, ops_schedules, ops_reconciliations immutable report/hash, ops_provider_history. |
| Export/организация | export_jobs authorization epoch/snapshot timestamp/fence, export_rows frozen values, saved_views, bulk_jobs/bulk_items frozen selections. |

## Поток и guards
Migrate → checksum → transaction → version insert. Parent locks сериализуют publish, move, hold, cleanup и completion. Settings apply и audit append выполняются в одной transaction. Recipient/download/credential fences живут в DB и не зависят от API process memory. FK/indexes поддерживают owner/provider/key/queue lookups. SQL `SKIP LOCKED` применяется к очередям, не к требованию атомарного «все файлы».

QR link mutation increment links_revision и инвалидирует связанные artifacts триггером. Report immutable identity/hash не меняется при trash; purge удаляет content, сохраняя private tombstone/history. Export job фиксирует owner auth_epoch, worker не продолжает ADMIN snapshot после отзыва роли. Payload crypto header сверяет DB metadata, а payload bytes остаются вне DB.

## Пример и ограничения
Если S3 Put завершился, а DB commit не состоялся, заранее записанный deletion_job позже удалит orphan object. Если worker crash произошёл после export materialization, другая fenced попытка использует те же export_rows. [ADR-003](adr/003-export-snapshot.md). Backup/restore используют отдельный BackupProvider. `ops_*` сохраняется при замене прикладных данных; данные этих таблиц входят в archive только как evidence. Restore запрещает откат KEK budget, инвалидирует sessions и меняет generation; детали и ограничения — [runbook](operations-recovery.md).

## Network policy фазы 7.1
`app_settings.value.networkPolicy` расширяет settings v1 без новой SQL migration: trustedProxyCidrs, allowedClientCidrs, requireProxy, requireHttps. Apply с expectedRevision атомарно меняет active revision; network/origin change дополнительно закрывает `ops_control`, меняет generation и инвалидирует reconciliation. Host network reset создаёт новую revision и `network.allowlist_reset` audit event, сохраняет HTTPS/proxy/origin. Старые settings без policy остаются совместимыми. Поток/границы — [ingress](production-ingress.md).

## Native WebDAV: migration 0017
Контекст: штатным клиентам нужны durable locks и метаданные, а overwrite не должен уничтожать прежний файл при неудачной загрузке. `files.staged_until` и частичный expiry index выделяют скрытые кандидаты; timestamps/properties добавлены files/draft_folders, root properties — transfers. Старым ресурсам назначается время миграции, не предполагаемое время создания.

`dav_locks` связывает transfer, credential, canonical path_key, recursive scope, безопасно сериализованный owner XML и expires_at. Active view дополнительно сверяет expiry/revocation/auth epoch/disabled пользователя. Locks не являются in-memory mutex. Parent transfer lock сериализует выдачу/снятие lock, swap, rename/delete и publish. Depth 0 папки защищает membership, subtree lock — descendants. Files properties ограничены БД 64 KiB JSONB representation и приложением 32 KiB входного JSON; aggregate draft budget 4 MiB. Owner XML ≤4 KiB.

Пример: PUT существующего `notes.txt` пишет `#staged:<uuid>`; после finalize одна transaction отменяет прежний file ID с deletion jobs и делает новый ID видимым (`staged_until=NULL`, canonical path_key). Старые properties/creation time переносятся. Worker забирает expired stages parent-first с учётом hold. [Поток/ограничения](webdav-upload.md), [ADR-011](adr/011-native-webdav.md).

## Брендинг: migration 0018
`branding_revisions` хранит публичный typed JSON, identity revision, время и ADMIN creator FK. Revision 0 виртуальна и соответствует embedded defaults. Apply сериализуется отдельным PG advisory lock; expectedRevision сверяется после захвата, новая строка и audit коммитятся вместе. Update ревизии запрещён trigger; sequence допускает пропуски после rollback. Данные не содержат secrets и входят в обычный backup schema discovery.

`reports.branding` и `security_incident_reports.branding` — nullable JSONB snapshots оформления. Новая запись заполняется в transaction создания отчёта; trigger запрещает менять snapshot. NULL обозначает legacy renderer, а не текущий бренд. Это отдельная presentation metadata, canonical report JSON/hash и encrypted envelope не расширяются. Incident tables сохраняют прежнюю recovery/control-plane семантику. [Data flow, поля, bounds и API](white-label.md), [ADR-013](adr/013-white-label.md).
