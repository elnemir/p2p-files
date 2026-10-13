# Эксплуатация, backup и восстановление

## Контекст и задача
FileX хранит authoritative metadata в PostgreSQL, а зашифрованные payload и artifacts — в отдельных Local/S3 хранилищах. Фаза 6 добавляет управляемое восстановление этих метаданных, расследование расхождений и явное открытие сервиса. Копия БД не возвращает физически удалённые файлы и может воскресить ранее отозванные ссылки/API-токены. Поэтому успешный import сам по себе не разрешает выдачу.

[ADR-006](adr/006-operations-recovery.md) объясняет формат и locks; [CLI](cli.md) содержит полный интерфейс; [session](session.md) фиксирует фактически выполненные проверки. Production остаётся фазой 7.

## Компоненты и данные
- `internal/operations`: encrypted archive, logical snapshot/import, KEK recovery, integrity checks, incidents, checkpoints, schedules, reconciliation.
- `internal/maintenance`: runtime gate. HTTP handlers и обычные worker tasks удерживают shared PostgreSQL advisory lock; restore/rewrap/reconciliation после закрытия gate получают exclusive lock. Gate использует отдельный pool до 24 connections на процесс, основной pool — 10.
- `ops_control`: `mode`, `generation`, `instance_id`, последний restore/reconciliation. `ops_jobs`, `ops_events`, `ops_schedules`, `ops_provider_history`, `ops_secrets`, `ops_checkpoints`, `ops_reconciliations` не откатываются вместе с приложением.
- `security_incidents` и encrypted `security_incident_reports`: состояние расследования и неизменяемые ревизии JSON. PDF строится из конкретного JSON; `X-Content-SHA256` обозначает hash JSON, не PDF. Эти данные входят в backup и не имеют TTL/user purge.
- ADMIN UI показывает задания, pre-backup, причины ошибок без secrets, инциденты, контрольные точки и результат сверки. Публичный status сообщает только operational/unavailable. Recovery подробности не отправляются владельцам в notifications.

## Проверка конкуренции idle worker / reopen, 2026-10-13
Во время квалификации Node 26 mobile-WebKit получил `409 conflict` на единственный reopen: generation, свежий passed report/id/hash и acknowledgement совпадали, до/после запроса jobs completed, incidents пусты. Gate остался закрыт; последующий USER login закономерно получил 503. Это не timeout отображения. Точный trace сохранён в `.cache/node26-review/browser-failure-1`, исходный прогон остаётся FAIL.

Детерминированно воспроизведённый механизм: автоматический `Operations.Run` берёт общий runner advisory lock даже без queued/running jobs и выполняет пустой UPDATE. Reopen использует тот же try-lock, поэтому idle tick может дать ложный conflict. Regression PG-тест: SHARE lock на ops_jobs допускает SELECT, но задерживает пустой UPDATE; параллельный idle Run не должен мешать валидному Reopen.

Исправленный минимальный поток: проверить generation → для автоматического Run прочитать наличие queued/running jobs → при пустой очереди вернуть без runner lock/UPDATE; при наличии работы выполнить прежний locked executor. Running jobs должны по-прежнему проходить recovery interrupted; explicit Execute/QueueAndExecute и все Reopen guards остаются прежними. Задание, поставленное после пустой проверки, получает следующий worker tick. Contracts/state machines/crypto не меняются; тест сначала завершился RED с `operation state conflict`, после изменения — GREEN. Второй PG-тест подтвердил queued job, interrupted running job, generation fence и запрет автоматического reopen. Исходный browser trace не содержит PostgreSQL lock timeline; тест отдельно доказывает этот механизм конфликта. Полная повторная квалификация фиксируется в session.

## Backup: вход → обработка → результат
1. ADMIN с web session и re-auth не старше 5 минут выбирает включённый, не draining BackupProvider и вводит причину (5…2000 Unicode-символов, минимум 5 после удаления пробелов по краям). Host CLI использует ту же реализацию без дополнительного ADMIN password.
2. Worker получает один session advisory lock исполнителя операций для этой схемы PostgreSQL. Зафиксированное расписание не создаёт дубликат незавершённого job этого вида.
3. `REPEATABLE READ READ ONLY` snapshot включает таблицы FileX, encrypted settings/key metadata и identity sequence high-water marks. External payload/artifacts, Redis, PostgreSQL roles/extensions, bootstrap env/KEK не копируются. Это FileX logical archive, не `pg_dump`.
4. Поток NDJSON разбивается на 4 MiB chunks. Новый UUID/salt/HKDF key создаётся на каждый backup; chunks и manifest защищены AES-GCM. Descriptor содержит wrapped backup DEK, ciphertext manifest, salt и IDs. Он записывается последним, после chunks.
5. Сервис читает объекты обратно и проверяет AEAD, длины, SHA-256 и совместимость схемы. Только затем job становится completed. Failed/incomplete archive нельзя использовать как source. Статус completed относится к проверенным данным на момент создания, а не гарантирует дальнейшую сохранность внешнего хранилища.

Local использует create-exclusive + file/directory fsync. S3 — TLS, signed requests и conditional Put без overwrite. Для backup выделяется отдельный root/bucket; автоматической retention/deletion backup в этой фазе нет. Секреты S3 в PostgreSQL и сохранённых provider descriptors зашифрованы; штатные API не возвращают их.

## Restore через ADMIN UI
1. Проверить выбранные UUID, дату и hash копии. Выбрать доступный BackupProvider для **новой предварительной копии** текущей БД. Он может отличаться от source provider.
2. «Подготовить восстановление» показывает последствия; отдельное действие запускает восстановление. Gate закрывается при постановке restore в очередь. USER/recipient/data-plane requests прекращаются; новый пользовательский login не допускается. ADMIN recovery login остаётся доступен вне exclusive import.
3. Worker отменяет обычные active handlers/tasks и ждёт освобождения их locks. Source проверяется полностью. До изменения прикладных строк создаётся и проверяется pre-restore backup; его ID связан с restore job. Ошибка source/provider/pre-backup оставляет текущие прикладные строки на месте и gate закрытым. Override отсутствует.
4. Atomic SQL import заменяет только application tables, проверяя текущую migration fingerprint и набор таблиц. FK/check constraints включены, user lifecycle triggers временно отключаются внутри transaction. При ошибке все прикладные изменения и sequence restart откатываются. Durable KEK wrap reservations не возвращаются.
5. Восстановленные web/recipient sessions и artifact credentials отзываются. Download/upload leases/fences меняются, незавершённые upload/export отменяются, bulk/migration/rotation приостанавливаются. Незавершённые deliveries/outbox не переотправляются автоматически. Существующие links/API tokens восстанавливаются как данные и требуют проверки.
6. Generation меняется. Перезапустить **все** API и worker процессы с текущим KEK; старый runtime возвращает `restart_required`. Worker выходит при обнаружении чужой generation; конкретный supervisor может его перезапустить. Сам gate от restart не открывается.
7. ADMIN входит повторно, запускает «Сверить восстановленные данные». Проверяются audit chains/checkpoints, wrapped keys/configuration/signing key consistency, полный SHA/AEAD файлов/artifacts, inventory всех текущих и сохранённых прежних payload/artifact providers.
8. Missing/tampered файлы fail closed; повреждённые или отсутствующие artifacts теряют действующий TTL. Объекты без записей в БД — warnings, они не публикуются и не удаляются. Недоступный provider/inventory, crypto/audit ошибка или незакрытый critical incident блокируют reopening.
9. Проверить capabilities и findings. При необходимости отозвать **все** восстановленные links/API tokens отдельным действием и повторить сверку. Reopen требует текущие report ID/hash/generation, возраст до 15 минут, `passed=true`, отсутствие pending jobs/critical incidents, причину и явное подтверждение проверки capabilities. Изменения административных данных инвалидируют предыдущую сверку.

Открытие gate не принимает production residual risks и не делает отсутствующие файлы доступными. Их восстановление из независимой копии payload требует отдельного ADMIN check по исходному SHA; orphan publication не является способом восстановления.

## KEK: rewrap и чистая БД
`kek rewrap` принимает новый независимо сгенерированный KEK на stdin, закрывает gate, создаёт проверенный pre-backup и атомарно rewrap DEKs, bootstrap audit/sentinel и operations key. Plaintext key material и audit HMAC identity сохраняются. После успеха оператор обновляет bootstrap secret всех процессов, перезапускает их, выполняет reconciliation/reopen. При неудавшейся попытке rewrap использовать другой свежий новый KEK; не переиспользовать ключ с неизвестным числом seal attempts.

Исторический backup descriptor остаётся под историческим KEK. Для него `backup verify/import/restore` принимает `recoveryKek` на stdin. Потеря исторического KEK делает соответствующие backups нечитаемыми. KEK escrow хранится отдельно от backup и сервера.

Восстановление в пустую БД: тот же binary/schema → **новый** MASTER_KEK → provision BackupProvider → `backup import` по известному backup UUID с историческим recoveryKek → `backup restore` → restart → reconcile → review/reopen. Тот же старый KEK на новой instance ID отклоняется: новая БД не знает историю его использования. При restore все прежние non-signing DEK становятся read-only/exhausted, четыре новые active class keys обслуживают записи; historical signing keys сохраняются.

## Audit, инциденты и alerts
Проверка audit/crypto по умолчанию раз в час, файлов — раз в сутки. Backup schedule выключен до выбора provider. Интервал 0 отключает конкретную optional проверку; другие допустимые значения — 60…2592000 секунд. Mandatory lifecycle cleanup остаётся отдельным worker task и возобновляется после открытия gate. Restore-test выполняется вручную, без напоминаний.

Критическая ошибка сначала durable закрывает gate, даже если повреждённый configuration key не позволяет сохранить encrypted incident report. В этом случае требуется восстановить читаемые ключи/данные прежде, чем можно закончить расследование. Ordinary missing/integrity file findings — предупреждения с quarantine, а не разрешение выдавать повреждённые bytes.

Обычный checkpoint экспортирует HMAC anchor обоих журналов. Recovery checkpoint требует закрытого gate, причины и текущего evidence hash **обоих** журналов. Он сохраняет повреждённые строки, binds hash принятого исторического prefix и начинает проверяемый suffix. Последующее изменение prefix снова обнаруживается. Разрешить audit/crypto incident можно лишь после успешной соответствующей проверки. Каждое изменение incident создаёт новую encrypted report revision; старые revision доступны по номеру.

Сохранить JSON checkpoint вне этого сервера. HMAC внутри той же БД не доказывает отсутствие полного rollback/truncation при компрометации host+DB+keys. Ed25519 по-прежнему подписывает только ZIP SHA256SUMS. Независимые checkpoints и инфраструктурный audit — release/операторская граница, не автоматически принятый риск.

Internal `/metrics` содержит aggregate request counter, `filex_data_plane_open`, unresolved incident counts, failed/pending operation counts и timestamp последнего successful backup. Ни user IDs, ни пути, ни причины, ни provider topology в labels не выводятся. `/ready` не ready во время maintenance/stale generation; `/health` описывает живой процесс. Traefik не публикует эти routes.

## ADMIN exports
Отдельная панель показывает последние 100 exports с необязательным owner UUID filter. Чтение чужого export требует fresh ADMIN re-auth и причины, создаёт отдельную одноразовую credential, привязанную к текущей web session и флагу `admin_access`. Каждый Range/stream guard повторно проверяет ADMIN role/reauth, TTL и отзыв session. Demotion или истёкшая re-auth прекращает чтение. Причина фиксируется в `administrative_actions` и HMAC audit, USER не уведомляется.

TTL можно изменить только у completed, ready, ещё не истёкшего export: 60…604800 секунд от текущего времени (инженерный bound). Expired/deleted artifact не воскресает. Обычный owner-scoped credential route не получил исключения для чужих exports.

## Ограничения и продолжение
- Archive ≤1 GiB, record ≤64 MiB, inventory ≤1 000 000 объектов, findings ≤10 000; превышение прерывает операцию с закрытым gate. Worker execution ≤5 минут, CLI ≤15 минут; это bounds, не RPO/RTO/SLA.
- Schema fingerprint должен совпадать; сначала восстановить matching binary/schema, затем отдельно выполнять [controlled upgrade](controlled-upgrade.md). Прямые DDL/operator writes вне FileX — доверенная граница и не должны конкурировать с backup/restore.
- Local inventory использует bounded memory и повторный directory scan на страницу. Большие каталоги требуют отдельного load/DR измерения в фазе 7. Для purposes следует использовать разные root/bucket; общий backend может дать повторные inventory warnings.
- Списки UI ограничены последними 100 элементами. Старые backup/report ID доступны через CLI/конкретный endpoint. Metadata и security history растут бессрочно; capacity планируется оператором.
- PostgreSQL backup без payload — осознанный контракт. Отдельная durability/storage snapshot стратегия, реальные S3 credentials/endpoints, disaster recovery на production topology и release security acceptance проверяются фазой 7.

## WebDAV после обновления и восстановления
Migration 0017 включена в schema fingerprint backup. Backup схемы 16 проверен до обновления. После migration команда verify тоже требует совпадения schema fingerprint и отклоняет его на схеме 17; для recovery нужен matching-schema стенд, прямого импорта нет. Restore/reopen сохраняют прежние правила проверки ссылок/API tokens; DAV lock не отменяет recovery gate, а восстановленные locks оцениваются по credential/auth epoch/expiry. Заброшенный stage не публикуется, cleanup учитывает freeze. Owner должен заново подключить папку после generation/restart. [Upgrade](controlled-upgrade.md), [WebDAV](webdav-upload.md).

## Брендинг в backup/restore

Migration 0018 добавляет branding revisions и report presentation snapshots в schema discovery. Restore возвращает активный бренд вместе с прикладными данными; incident control plane по-прежнему сохраняется отдельно. Существующий matching-schema guard действует и для 0018: копия схемы 17 не импортируется напрямую в 18. PG recovery test проверяет смену бренда до/после backup и возврат сохранённой ревизии. [White labeling](white-label.md), текущий статус — [session](session.md).
