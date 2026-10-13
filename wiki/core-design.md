# Ядро передачи: реализация и инварианты

## Контекст
API, UI и worker используют один PostgreSQL repository и storage service. Один transfer принадлежит user ID; организации/tenant billing не вводились. Redis не хранит права на выдачу. Требования: FR-002–009, OPS-001.

## Поток
Owner session/scope → draft → tus reserve либо WebDAV encrypted staging (sticky provider) → encrypted chunks + committed offset → finalize с server SHA-256/CRC32/MIME/scanner → ready. Клиент не может назначить checksum или ready через JSON. Draft tree/table поддерживает виртуальные папки, recursive move/delete и атомарный batch repath с collision validation.

Publish берёт parent lock, проверяет owner/expected revision, минимум один ready file и отсутствие незавершённых файлов, live WebDAV stages и активных DAV locks; фиксирует manifest/hash/TTL/recipient links/outbox одной транзакцией. HMAC fingerprint включает полный publish request, включая PIN/recipients: повтор с теми же параметрами идемпотентен, иной запрос — 409. После publish file/path/name set immutable; разрешены title/message и операции links. Source: [repository](../internal/transfer/store.go), [transport](../internal/httpapi/transfers.go).

Recipient token находится во fragment URL. Явный POST session проверяет token hash/PIN, создаёт HttpOnly session. GET list возвращает session-bound opaque refs без расходования доступа. POST acquire берёт один файл или весь remaining set; busy/frozen/error member блокирует весь ZIP. Range requests используют ту же reservation и fence. Полное объединённое покрытие успешно записанных сервером ranges → atomic logical deletion всех reservation files → aggregate journal/outbox → asynchronous cleanup. HEAD не расходует права.

## Пути и ZIP
NFC UTF-8 relative paths, запрет dot/backslash/control/Windows-reserved/trailing dot-space и checksum names; case-insensitive collision key. Manifest сортируется по UTF-8 path bytes. ZIP64 STORE имеет deterministic layout/timestamps и SHA256SUMS.txt/sig/pub; при Range выдаёт только пересекаемые сегменты. Это сохраняет resume без накопления plaintext ZIP на диске.

## Конкурентность и ошибки
Порядок mutation locks: transfer → files. Upload, content writer, maintenance и artifact jobs используют fences/leases. Перед и после каждого 32 KiB write проверяются session, account, epochs, revoke/expiry/hold; writer deadline 2 s. Старый writer не может commit после смены fence. Уже отправленные bytes отозвать физически невозможно. [ADR-001](adr/001-download-completion.md) определяет точный смысл completed.

Порча/missing блокирует consume. Обычный TTL удерживает integrity_failed для диагностики; revoke помечает retained payload deleting, физический hold сохраняется до ADMIN решения. Frozen TTL и hard deadline — [ADR-002](adr/002-retention-precedence.md). Upload/delete/migration регистрируют компенсационные object references до внешней записи, поэтому штатный crash не создаёт неизвестную копию.

## Пример и границы
Две параллельные acquire одного файла: один lease, другой 409; полный ZIP выдаётся одной reservation и одной journal записью payload bytes/count. Test cases: `TestIntegrationConcurrentAcquireAndRevoke`, `TestIntegrationOneTimeRangesZIPAndCleanup`, `TestIntegrationFileLeaseFencesStaleUpload`. Все cleanup операции идемпотентны; secure erase S3/disk не обещается. Полное orphan reconciliation после restore реализовано фазой 6: inventory findings остаются private, orphan bytes не публикуются и не удаляются, [runbook](operations-recovery.md).

## Межпротокольные guards черновика
WebDAV чтение относится к владельцу черновика и не создаёт downloads/download_journal. Замена видимого файла происходит только после проверки кандидата. Exclusive DAV locks в PG проверяются также в tus reserve/append/finalize/cancel, REST tree/folders, bulk cancel и publish. Поэтому открытый файл нельзя незаметно удалить другим протоколом. При конфликте DAV сообщает 423, REST — 409; пользователь закрывает файл/отключает папку или отзывает credential. Revocation/hold и administrative containment не требуют согласия файлового менеджера. [Подключение и ограничения](webdav-upload.md).
