# File delivery: продолжение работы инженером

## Контекст и проблема
Критичная часть FileX — дать только одну подтверждённую сервером выдачу, сохранить Range resume и не удалить forensic evidence. Компоненты: internal/files, storage, bundle, httpapi/recipients, worker; правила и порядок locks — [core-design](core-design.md).

## Реализованный поток
1. Owner создаёт transfer и tus upload. При выборе provider используются enabled/non-draining payload candidates по priority, затем fallback. Upload сохраняет выбранный provider.
2. PATCH ≤4 MiB получает file lease/epoch, шифрует FXP1 chunk, регистрирует compensation до Put и атомарно фиксирует offset/chunk. Прерванный ответ восстанавливается HEAD. Поддерживаются Tus-Resumable 1.0.0 и method override PATCH/DELETE, duplicate/invalid metadata отвергаются.
3. Finalize читает и проверяет фактические bytes; SHA-256, CRC32, MIME policy, optional scanner. Повреждение не маскируется клиентским checksum.
4. Publish фиксирует immutable manifest и capability links. Recipient POST acquire затем GET/Range выдаёт file/ZIP, guards проверяются на каждом ограниченном write.
5. Coverage completion фиксирует logical delete и outbox. Worker удаляет Local/S3 chunks/неиспользуемые копии, учитывая holds и active write leases.

## Storage, rotation и forensic
Local использует rooted file operations, O_EXCL, fsync и generated keys; S3 SDK — TLS и conditional Put, bounded Get и idempotent Delete. Keys/salts независимы от имени пользовательского файла. [FXP1](encryption-format.md) связывает metadata, identity и chunk index.

Migration: dry-run → paused+drain source → run → copy/authenticate/read-back → parent/file/job fence check → pointer swap → cleanup old copy. Pause/cancel предотвращает следующий commit, уже завершённые файлы сохраняются. Rotation меняет active class key сразу, background re-encryption сохраняет hashes; frozen файлы ожидают освобождения hold. Fair checked_at не позволяет одному frozen job остановить другие.

ADMIN re-auth/reason/audit обязательны для hold/check/delete/forensic. Plaintext forensic не расходует recipient rights. Forensic ZIP требует transfer hold; raw ciphertext ограничен object конкретного integrity_failed файла. Снятие полномочий/hold во время stream прерывает выдачу. USER не получает административное уведомление.

## Проверки и эксплуатационные границы
PG tests покрывают acquire races, stale upload/migration, hold во время stream, TTL/revoke, missing/corrupt files, atomic ZIP, Range union и cleanup. Browser через Traefik подтвердил PATCH interruption→resume и ZIP download. S3 wire test использует локальный TLS fixture; реальный provider требует отдельного подключения. Память bounded по chunk, ZIP/maintenance metadata O(N); Physical orphan reconciliation реализована и проверена фазой 6; bounded 2050 MiB local load выполнен фазой 7. Target load/DR и реальные providers остаются внешней приёмкой. Уже доставленные байты нельзя вернуть, а successful write не доказывает сохранение на диске получателя.

Cleanup держит parent retention lock во время bounded physical Delete (20 s context). Только эта transaction увеличивает idle-in-transaction timeout до 30 s: обычные 10 s pool меньше 15 s S3 request, поэтому без локального override PostgreSQL мог бы снять lock до окончания удаления. Общий DB timeout остальных операций не меняется. Уже начавшийся внешний Delete нельзя отменить задним числом при катастрофическом DB/network отказе; local drain/restore fault tests выполнены в фазе 6, а такие инфраструктурные failure/DR границы проверяются отдельно в фазе 7.
