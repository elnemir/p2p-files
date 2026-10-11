# ADR-001: завершение выдачи и конкуренция

Статус: принято для реализации 2026-10-10 по поручению самостоятельно разрешить дизайн. Requirements: FR-006–009, OPS-001. Остаточный риск R-02 не принят от имени владельца.

## Контекст и проблема
TCP/HTTP не доказывает сохранение файла клиентом. Нужна однозначная одноразовая бизнес-операция при Range, reconnect, crash и ZIP. Exactly-once delivery байтов через сеть недостижима; гарантируем один успешный commit операции выдачи и исключительное владение серверной операцией.

## Решение
Явный POST создаёт DownloadSession, связанный с recipient session. GET/HEAD landing/metadata не получают lease и не тратят выдачу. Content GET допускается только для уже созданной сессии и opaque hashed file ref. Один активный streaming request на сессию/файл; параллельный запрос получает 409. До первой записи проверяются session, lease, fencing, transfer/link epochs, freeze и expiry.

Одиночный файл: поддерживается один HTTP byte range на запрос (включая suffix/open-ended); multipart ranges отклоняются 416. Успешно записанные полные ranges накапливаются в PG как объединение интервалов `[start,end)`. Неполный ответ не добавляет интервал. После покрытия `[0,size)` и завершающей проверки всех guards одна транзакция переводит session в completed, файл в deleting и создаёт outbox deletion. Для нулевого файла нужен явный content GET и успешное завершение пустого ответа. HEAD никогда не завершает session. Повтор ranges одной незавершённой сессии допустим; после commit content недоступен.

Lease хранится в PostgreSQL, время берётся из DB. Heartbeat, новый fencing при takeover; завершение старого holder отклоняется. Перед каждым bounded write и после него проверять cancellation/lease. При revoke/freeze/expiry увеличивается access epoch, локальные handlers отменяются; межпроцессный watchdog читает PG, потеря PG означает немедленную остановку следующих writes. Уже отправленные в kernel/proxy байты отозвать нельзя. Proxy buffering payload запрещён, write deadlines ограничены. Измеренная задержка остановки и объём in-flight bytes обязательны в фазе 2; без этих tests gate выдачи не закрывается.

Crash после отдачи байтов и до DB commit оставляет операцию незавершённой: возможна повторная передача тех же байтов после восстановления. Это известный предел протокола, не доказательство второй завершённой выдачи. Нельзя скрывать его в отчётах.

## ZIP
Snapshot включает все ещё не удалённые файлы опубликованного manifest. Любой frozen, leased, integrity_failed или storage_missing блокирует всю операцию (409/423), исключать такой файл нельзя. Acquire всех leases и snapshot membership — одна короткая транзакция с фиксированным порядком блокировок. Пустой остаток — 410. ZIP completion: полностью записанный central directory, итоговая проверка guards и один PG commit всех logical deletes. Failure освобождает операцию без logical deletes. Range/resume ZIP также сохраняется в области требований. Для одной session нужны byte-for-byte deterministic ZIP (фиксированные entry order/timestamps/encoding/signing key) и известная полная длина; фаза 3 должна выбрать и проверить layout без хранения plaintext archive. Вариант baseline — ZIP STORE с CRC32, вычисленным при finalization, и проверяемым расчётом ZIP64 overhead. При resume заново формируется тот же byte stream с пропуском уже выданного префикса; immutable snapshot и leases сохраняются в пределах session deadline. Это увеличивает чтение storage, но ограничивает память. После потери session leases возобновлять старый snapshot нельзя: только новая явная операция. Закрыть ZIP gate без проверки deterministic layout/Range нельзя. Дубли байтов до DB commit возможны по той же сетевой причине.

## Пример и тесты
Два download requests конкурируют за один файл: ровно один lease. Revocation побеждает ещё не committed completion. Если completion уже committed, последующий revoke не переписывает историю. Нужны tests overlap/suffix/zero-byte, stale fencing, crash-before-commit, disconnect, Redis loss, frozen member и ZIP failure.
