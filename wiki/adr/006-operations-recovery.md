# ADR-006: эксплуатационный контур и восстановление FileX

Статус: реализовано в фазе 6, 2026-10-11. OPS-003–004, SEC-002–003. Реализация и evidence записываются в [session](../sessions/2026-10-11-phase7.md).

## Контекст и проблема
Restore возвращает более раннее состояние PostgreSQL, но не откатывает Local/S3. Возвращённые capability links/API tokens требуют проверки; физически удалённые файлы могут отсутствовать. Блокировка обслуживания и журнал самой операции не должны откатываться вместе с прикладными данными. Ротация KEK не должна менять ключ HMAC существующей audit chain.

## Решение и поток данных
1. `ops_` control tables сохраняют maintenance state/generation, задания и журнал восстановления. Их evidence входит в backup, но существующий control plane не заменяется при restore.
2. FileX logical backup v1 сохраняет данные всех таблиц своей схемы и fingerprint применённых migrations, без payload, Redis, PostgreSQL roles и внешних extensions. Используется согласованный `REPEATABLE READ READ ONLY` snapshot. Версия схемы при restore должна совпадать; это не универсальный PostgreSQL dump и не средство cross-version migration.
3. Поток JSON records ограничен 1 GiB, отдельная запись — 64 MiB. Он разбивается на 4 MiB encrypted objects в отдельном BackupProvider Local/S3. Каждый backup имеет новый UUID/salt/HKDF subkey class `backups`; descriptor содержит wrapped DEK и encrypted manifest, поэтому чтение не зависит от наличия старых key_versions в текущей БД. В descriptor нет plaintext KEK, credentials или строк БД. Публикация descriptor последняя, проверка читает ciphertext обратно и проверяет AEAD, длины и SHA-256.
4. Restore закрывает durable gate, отменяет/дожидается runtime readers/writers под PostgreSQL advisory locks, проверяет source и создаёт проверенный pre-restore backup. При любой ошибке исходные данные не заменяются, gate остаётся закрытым до проверки/явного открытия.
5. Замена прикладных строк выполняется одной transaction: проверка fingerprint/набора таблиц, truncate, загрузка в FK order, возврат пользовательских triggers, восстановление sequence high-water marks, invalidation web/recipient/artifact sessions и jobs/leases. Только эта transaction временно отключает пользовательские triggers; FK/check constraints продолжают проверяться. Raw SQL из backup не выполняется.
6. После restore generation меняется. Старые API/worker процессы не могут возобновить работу с закэшированным ключом аудита: требуется restart. Сервис запускается в закрытом состоянии. ADMIN повторно входит, запускает audit/crypto/storage reconciliation, проверяет quarantine/orphans/capabilities и явно открывает gate с идентификатором и hash свежего отчёта.
7. Audit key сохраняется wrapped отдельно от производного bootstrap KEK. KEK rewrap выполняется host CLI под закрытым gate и сохраняет plaintext DEKs/audit key; старые backup descriptors остаются под историческим KEK. Для восстановления такой копии CLI принимает recovery KEK через stdin и rewrap восстановленных ключей под текущий KEK. Секреты не передаются в argv/logs.
8. Restore не откатывает бюджет nonce/wrap использования: существующий KEK counter резервируется до import и переносится в восстановленную БД. Все восстановленные root DEKs, кроме signing seed, становятся read-only/exhausted; для новых записей создаются свежие active class keys. Backup содержит instance ID: перенос на свежую БД требует нового bootstrap KEK и отдельно исторического recovery KEK, поскольку история использования старого KEK вне исходной БД неизвестна.

Control envelopes используют случайный salt на каждую запись и отдельный HKDF subkey из сохранённого operations key. AAD следуют существующему формату `filex/<class>/v1/<identifier>`: `ops_provider`, `ops_inventory`, `backups`; дополнительные slash в identifier не допускаются. Backup chunks имеют suffix `_chunk_<index>`, manifest — `_manifest`.

## Audit и incident recovery
Критическая ошибка закрывает gate отдельной durable записью до создания encrypted incident report. Повреждение configuration key не должно мешать containment. Checkpoint хранит HMAC-signed anchor для application audit и сохраняемого ops journal. Recovery требует evidence hash обоих журналов под их locks; hash исторического prefix проверяется при каждой дальнейшей верификации, suffix проверяется по HMAC. Испорченная история сохраняется. Ревизии encrypted incident JSON не перезаписываются, PDF строится из выбранной ревизии.

Crypto health читает ключи и references в согласованном read-only snapshot, чтобы concurrent class rotation не создавала ложный missing-key incident. Signing seed/public key соответствие также проверяется. ADMIN export credential отмечается admin_access и проверяет role/reauth на каждом stream guard. [Полный runbook](../operations-recovery.md).

## Критерии проверки
Faults: unavailable backup provider, truncated/swapped ciphertext, wrong KEK/schema, failed pre-backup, rollback failed import, process interruption. Concurrency: active stream/worker drain, concurrent restore, stale reopen, stale runtime generation. Reconciliation: missing/tampered payload/artifact, orphan inventory, audit corruption, restored credentials, no automatic orphan publication/deletion. Полный цикл проверяется на disposable PostgreSQL и Local/S3 fixtures; реальные хранилища не изменяются.

## Ограничения и обоснование
Backup не копирует payload/ArtifactStorage: их durability обеспечивается отдельно, reconciliation сообщает расхождения. Большие recovery operations имеют явные resource/time bounds; это не заявленный RPO/RTO. Состояние после commit restore требует restart всех API/worker экземпляров. Миграции/upgrade не запускаются одновременно с restore. Доступ superuser/host остаётся доверенной границей. Инциденты не считаются принятым residual risk; production acceptance остаётся фазой 7.

Нормативная база snapshot/locks: [PostgreSQL isolation](https://www.postgresql.org/docs/18/transaction-iso.html), [advisory locks](https://www.postgresql.org/docs/18/explicit-locking.html), [JSON record conversion](https://www.postgresql.org/docs/18/functions-json.html). Формат является решением FileX, а не рекомендацией этих источников.
