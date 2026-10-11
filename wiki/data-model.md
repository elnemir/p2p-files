# Модель данных

## Контекст и изменения
PostgreSQL authoritative для identity, доступа, jobs, immutable snapshots и audit. Миграции 0001–0013 встроены в binaries и применяются под advisory lock с SHA-256 checksum. Изменение уже применённого SQL блокирует startup. Внешние bytes располагаются в Local/S3 отдельно от DB.

| Группа | Таблицы / ответственность |
|---|---|
| Bootstrap/config | schema_migrations, bootstrap_secrets, app_settings; KEK sentinel и revisions/active/validated. |
| Identity | users/auth_epoch, password_history, web_sessions, api_tokens, login_history, auth_limits. |
| Audit | audit_events: serialized HMAC chain, actor/action/subject; user не может purge. |
| Передачи | transfers: immutable manifest/request hash/timestamps + lifecycle/access_epoch/hold/links_revision; files: paths/checksums/upload offset/epoch/provider/key/salt/hold; draft_folders. |
| Storage/crypto | storage_providers с purpose payload/artifacts/backups; key_versions с active/version/seal_count, crypto_usage KEK budget, signing_keys historical public keys. |
| Payload | file_chunks: stable chunk UUID/index/range/object key/plain SHA; deletion_jobs: compensation/leases/retries. |
| Recipient | recipient_links с encrypted token/hash/PIN/epoch; recipient_sessions; recipient_file_refs hashed. |
| Выдача | downloads writer/fence/coverage, download_files selection, download_journal owner-safe aggregate. |
| Maintenance | storage_migrations и rotation_jobs, pause/resume/cancel/progress/checked_at. |
| Коммуникации | outbox, notifications, notification_preferences, communication_configs (encrypted), deliveries, webhook_receipts dedup. |
| Отчёты | reports immutable revision/canonical hash/encrypted snapshot/render fence/state; report_artifacts; append-only report_history. |
| Artifacts | artifacts state/key/salt/provider/TTL/SHA/maintenance lease, artifact_chunks, artifact_credentials hash/session/coverage/fence; qr_bundles provenance. |
| Export/организация | export_jobs authorization epoch/snapshot timestamp/fence, export_rows frozen values, saved_views, bulk_jobs/bulk_items frozen selections. |

## Поток и guards
Migrate → checksum → transaction → version insert. Parent locks сериализуют publish, move, hold, cleanup и completion. Settings apply и audit append выполняются в одной transaction. Recipient/download/credential fences живут в DB и не зависят от API process memory. FK/indexes поддерживают owner/provider/key/queue lookups. SQL `SKIP LOCKED` применяется к очередям, не к требованию атомарного «все файлы».

QR link mutation increment links_revision и инвалидирует связанные artifacts триггером. Report immutable identity/hash не меняется при trash; purge удаляет content, сохраняя private tombstone/history. Export job фиксирует owner auth_epoch, worker не продолжает ADMIN snapshot после отзыва роли. Payload crypto header сверяет DB metadata, а payload bytes остаются вне DB.

## Пример и ограничения
Если S3 Put завершился, а DB commit не состоялся, заранее записанный deletion_job позже удалит orphan object. Если worker crash произошёл после export materialization, другая fenced попытка использует те же export_rows. [ADR-003](adr/003-export-snapshot.md). Backup/restore metadata, incidents/checkpoints и reconciliation tables добавляются в фазе 6; существующий purpose=backups не означает готовый BackupProvider.
