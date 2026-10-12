# ADR-011: штатные WebDAV-клиенты и редактируемый черновик

Дата: 2026-10-12. Статус: реализовано по прямому запросу владельца; локальная квалификация в [session](../session.md), Windows ожидает стенда. Заменяет ограничения первого профиля [ADR-010](010-webdav-upload.md), сохраняя его credential/transport/ownership boundary.

## Контекст и проблема
Finder/mount_webdav требует LOCK для записи. WebClient и GVfs используют условия, timestamps, чтение и замену при сохранении. Upload-only с запретом overwrite/GET не обеспечивает этот сценарий. Нативный клиент подключается к одному существующему черновику; это не опубликованная recipient-ссылка.

## Решение и поток данных
Credential → общий ingress/auth/recovery → draft/hold guard → DAV method. GET/HEAD/Range читают готовый файл владельца без consume и recipient journal. Каждый bounded output проверяет credential, состояние и freeze. PUT принимает Content-Length, macOS X-Expected-Entity-Length либо chunked поток. Partial Content-Range не поддерживается; resume остаётся tus.

PUT резервирует скрытый staged file с lease. Прежний файл остаётся видимым и читаемым. Поток → FXP1 chunks → checksum/MIME/scanner → parent-first transaction проверяет условия, namespace/quota, credential и locks → прежний файл отменяется с deletion jobs, новый получает канонический path_key. При ошибке удаляется только staging; исходный файл сохраняется. Публикация блокируется при незавершённом staging.

Неизвестная длина: первый скрытый ciphertext использует неизменный верхний размер 1 TiB. После EOF фактическая длина известна; bounded decrypt/re-encrypt в новый FXP1 file с этой длиной, затем обычная финализация. Первый объект никогда не публикуется. Никаких plaintext temporary files, новых primitives или изменений FXP1. Цена — второй проход только без известного размера.

Exclusive write locks хранятся в PostgreSQL с UUID token, корнем, Depth 0/infinity, credential, сроком до часа. LOCK/refresh/UNLOCK сериализуются по transfer. If поддерживает tagged/untagged lists, Not, tokens и ETags; неверное условие 412, отсутствие нужного lock token 423. Locks защищают REST/tus/tree/publish тоже, чтобы другой протокол не обходил блокировку. Отозванный credential не удерживает активную блокировку. Freeze и административные containment-операции не требуют DAV token.

Метаданные: сохранённые creation/modified timestamps, bounded dead properties, live supportedlock/lockdiscovery. PROPPATCH атомарен: запрещённое live property → 403/424 без частичной записи. MOVE сохраняет свойства, может атомарно заменить файл; COPY остаётся явно неподдерживаемым. Root delete и выход за draft запрещены.

## Модель данных и состояния
Миграция 0017: files staged_until/created_at/modified_at/dav_properties, timestamps/properties у draft_folders и root properties у transfers; таблица dav_locks с FK/index по transfer/credential/expiry. Старым файлам недоступное историческое время не придумывается: используется явно документированное время миграции как baseline metadata.

PUT: reserved hidden → uploading → finalizing → ready hidden → atomic visible swap. Failure → cancelled + deletion jobs. Lease expiry → worker cancels abandoned stages под parent/freeze fence. DAV lock: absent → active → refreshed/released/expired/revoked. Publish требует отсутствия active locks и stages.

## Architecture / crypto review до реализации
Стек, keys, immutable published payload, one-time recipient и приватность не меняются. PG rows authoritative; Redis/in-memory lock не используется. Parent-first transaction исключает гонку LOCK/mutation/publish. Storage I/O вне metadata transaction; existing compensation ledger до PUT, guard после I/O. Read checks на bounded writes с deadline ограничивают окно отзыва; уже переданные bytes не возвращаются. Staging cleanup не удаляет frozen bytes. Криптография остаётся FXP1/HKDF/AEAD; второй проход использует новые ID/salt/key binding. AsyncAPI не меняется: новые domain events не нужны.

## Пример и проверка
`LOCK /api/v1/webdav/<id>/notes.txt` → `Lock-Token: <urn:uuid:…>`; последующий PUT содержит `If: (<urn:uuid:…>)`; закрытие редактора выполняет UNLOCK. PROPFIND возвращает времена и lock discovery. UI предлагает отдельные инструкции Finder, Windows WebClient и GNOME Files.

Негативные проверки: wrong token, expired/revoked lock, условия, same-name writers, scanner/storage failure при overwrite, unknown stream, freeze/revoke между chunks, скрытый staging и publish, cleanup после crash, Range без consume. Реальные macOS/Linux tests отдельно от HTTP tests. Windows подготовлен, фактический запуск pending по ответу владельца; Office совместное редактирование не обещается.

Источники: [RFC 4918](https://www.rfc-editor.org/rfc/rfc4918.html), [Apple webdavfs](https://github.com/apple-oss-distributions/webdavfs), [GNOME GVfs](https://github.com/GNOME/gvfs), [Microsoft WebDAV Redirector](https://learn.microsoft.com/en-us/iis/publish/using-webdav/using-the-webdav-redirector).

## Итоговые уточнения реализации
Свойства ограничены 32 KiB/256 на ресурс и 4 MiB PostgreSQL JSON на видимый draft; активных locks ≤256, owner XML ≤4 KiB. Root timestamps защищены, dead properties разрешены. Depth 0 папки защищает membership, не существующие child bytes; тот же guard добавлен REST folders/cancel и bulk. Архитектурное решение не меняется. Новый upgrade drill проверяет additive 0017 и отказ старого binary; точные результаты — в session.
