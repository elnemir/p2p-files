# WebDAV: черновик как сетевая папка

## Контекст и статус
WebDAV позволяет владельцу работать с файлами **одного существующего черновика** через системный файловый менеджер. До публикации доступны загрузка, чтение, замена, переименование и удаление. Публикация выполняется в FileX и закрывает DAV-доступ. Получатель продолжает пользоваться одноразовой ссылкой: его выдача и журнал не меняются.

[ADR-011](adr/011-native-webdav.md) заменяет ограничения исторического [upload-only профиля](adr/010-webdav-upload.md). Фактические результаты и source hash — в [сессии](session.md). Production deployment не выполнялся.

| Клиент | Проверка на локальном стенде | Граница |
|---|---|---|
| macOS 27.0.1, штатный `mount_webdav` | PASS: подключение, mkdir, загрузка 5 700 000 bytes, чтение серверного файла, замена, rename, delete, rmdir, отключение | Реальный системный backend Finder; автоматизирована файловая система, не клики Finder |
| Linux, Ubuntu 24.04 / GVfs 1.54.4 / GLib 2.80.0 / `gio` | PASS: тот же цикл, 5 600 000 bytes, сверка содержимого | Настоящий GVfs в изолированном Docker/D-Bus, без рабочего стола GNOME; не проверены KDE/davfs2 |
| Windows WebClient / Проводник | **Ожидает Windows-стенда** | Реализация протокола и сценарий подготовлены; фактическая совместимость не объявлена подтверждённой |
| rclone 1.75.2, macOS arm64 | PASS: mkdir/copyto/list/moveto/deletefile/rmdir, серверный SHA-256 | Отдельный клиентский тест; WinSCP/Cyberduck/Office не квалифицированы |

## Подключение пользователя
1. Откройте черновик и секцию «Загрузка через WebDAV». Выберите файловый менеджер.
2. Подтвердите пароль **аккаунта** и задайте срок подключения. Скопируйте адрес, имя `filex` и отдельный пароль **WebDAV**. Пароль аккаунта в файловый менеджер не передаётся.
3. Подключите адрес по инструкции ниже. Копируйте файлы и каталоги; для возобновляемой загрузки используйте браузерный tus.
4. Закройте открытые файлы и отключите папку. В FileX нажмите «Обновить файлы после WebDAV», проверьте содержимое и готовность, затем опубликуйте передачу.
5. Если доступ больше не нужен, отзовите его в панели или списке API-токенов. Пароль показывается только при выдаче; после потери создайте новое подключение.

| ОС | Действия |
|---|---|
| macOS | Finder → Переход → Подключение к серверу (`⌘K`) → адрес `https://host/api/v1/webdav/<uuid>/` → зарегистрированный пользователь `filex` и пароль подключения. Для отключения извлечь сетевой диск. |
| Windows | Проводник → Этот компьютер → Добавить сетевое расположение → HTTPS-адрес → `filex` и пароль подключения. Нужны установленная/работающая служба WebClient и доверенный сертификат. Конкретную доступность WebClient и ограничения размера проверяет администратор стенда. |
| Linux / GNOME Files | Другие места → Подключиться к серверу → `davs://host/api/v1/webdav/<uuid>/` (замена `https://`); пользователь `filex`. Нужен GVfs DAV backend, в Ubuntu пакет `gvfs-backends`. Локальному HTTP соответствует `dav://`. |

Не отключайте TLS-проверку и не разрешайте передачу Basic-паролей по внешнему HTTP ради подключения Windows. Его системный предел размера файла может быть ниже лимита FileX; большие файлы можно загрузить через браузер. macOS может создавать `._*`/`.DS_Store`: это обычные видимые файлы, их следует проверить и при необходимости удалить перед публикацией. Автоматическая фильтрация имён не добавлялась.

В dev HTTP разрешён только для настроенного loopback PublicOrigin с совпадающим Host. Compose публикует порт на loopback; его нельзя выставлять наружу. Реальный домен требует HTTPS и reviewed ingress policy; внешние ACME/DNS и Windows TLS acceptance ещё не выполнены.

## Поток данных и атомарность
Файловый менеджер → существующий Traefik/network/recovery gate → Basic credential → PG-проверка owner/scope/TTL/auth epoch → состояние draft/hold → DAV method. Credential содержит scope только своего draft, в БД хранится digest. Срок 60 секунд–30 дней; отзыв/отключение пользователя/смена auth epoch прекращают доступ и действие его locks.

PUT → скрытая staged-запись с lease 5 минут → поток encrypted FXP1 chunks по 4 MiB → серверные checksum/MIME/scanner → короткая parent-first транзакция → повторные условия/права/locks/namespace/quota → атомарная смена видимого файла. Прежний файл остаётся читаемым до успешного commit. При ошибке отменяется только кандидат; старые bytes сохраняются. Старый ciphertext после успешной замены попадает в общий deletion ledger.

Размер задаётся Content-Length или macOS X-Expected-Entity-Length. При неизвестном размере первый скрытый ciphertext использует фиксированную верхнюю длину; после EOF выполняется bounded decrypt/re-encrypt с фактической длиной и новым file ID/salt. FXP1 не меняется, plaintext spool отсутствует. Неизвестная длина требует второго прохода и дополнительного места; это не resume. Content-Range в PUT запрещён.

Stage скрыт из REST/DAV списка и не входит в publish. Worker отменяет заброшенный stage после lease expiry; freeze сохраняет его до снятия hold. Оборванный процесс не оставляет пользователю видимую половину замены. При storage/DB ошибке cleanup остаётся в durable ledger. Физическое secure erase не обещается.

GET/HEAD/один Range доступны только владельцу готового файла черновика, без consume/recipient journal. Каждый bounded output проверяет актуальные credential, state, freeze и recovery generation; write deadline 5 секунд. Уже переданные клиенту bytes отозвать невозможно. PUT проверяет права перед записью/финализацией и использует read deadline 30 секунд на блок.

## Блокировки и свойства
LOCK создаёт exclusive write lock в PostgreSQL, Depth 0 или infinity, максимум на час; пустой LOCK с If обновляет срок, UNLOCK снимает его. Блокировка привязана к credential. Чужой credential не может воспользоваться даже известным token. Depth 0 папки защищает добавление/удаление непосредственных детей, но не запись существующего дочернего файла. Рекурсивная блокировка защищает поддерево.

`Timeout: Second-N` разбирается как unsigned 32-bit seconds. Допустимое положительное значение выше 3600 ограничивается до 3600 **до** преобразования в платформенный `int`; ноль, переполнение 32 бит и malformed значения отклоняются либо пропускаются при поиске следующей допустимой альтернативы в списке. Например, `Second-4294967295` даёт 3600, а одиночный `Second-4294967296` — HTTP 400. Явная проверка верхней границы сохраняет прежний контракт `min(N,3600)` и позволяет CodeQL доказать безопасность narrowing conversion на 32-bit target. Граничные случаи проверяет `TestWebDAVTimeoutBounds`; scanner suppression не используется.

If поддерживает tagged/untagged lists, Not, lock tokens и ETag; false → 412, отсутствие нужной блокировки → 423. Те же guards действуют в REST/tus/tree/bulk/publish (REST сообщает 409). Публикация ждёт снятия активных locks и завершения stages. Отзыв, freeze и административное containment имеют приоритет над DAV lock. Locks, корень которых перенесён/удалён, удаляются; ancestor lock остаётся на своей коллекции.

LOCK отсутствующего пути сначала создаёт обычный проверенный пустой файл. Политика типов/scanner может запретить такой файл; это ограничивает приложения, начинающие запись с LOCK-null. MOVE атомарно заменяет файл при разрешённом overwrite, но не коллекцию; Overwrite:F сохраняет существующую цель. COPY не реализован. Корень нельзя удалить или перенести, Destination ограничен тем же origin/draft и не вызывает сетевых запросов.

PROPFIND Depth 0/1 возвращает live properties, timestamps, lockdiscovery и сохранённые namespaced dead properties. PROPPATCH атомарен: запрещённое live property получает 403, зависимые изменения 424 без записи. Разрешены file/folder creation/modified timestamps и Microsoft Win32 time properties; времена корня защищены. Миграция даёт старым файлам время миграции как baseline, историческое время не восстанавливается. Свойства сохраняются при MOVE/замене. Системный файловый менеджер — поддерживаемый способ копирования; совместное редактирование Office не квалифицировано.

## Ограничения и ошибки
Новые настройки/env/порты не добавлены. Миграция `0017_native_webdav.sql` обязательна; rollback на старый бинарник после неё запрещён проверкой migration ledger. Обновление — по [runbook](controlled-upgrade.md).

| Ограничение | Значение |
|---|---|
| Файл / encrypted chunk | 1 TiB / 4 MiB; дополнительно действуют настройки количества файлов и путей |
| Namespace | 10 000 узлов с корнем и неявными родителями; проверяется весь draft |
| XML request / depth / response | 64 KiB / 16 / 8 MiB; ответ собирается до 207, превышение даёт 507 |
| Свойства | 256 и 32 KiB JSON на ресурс; 4 MiB PostgreSQL JSON сверх пустых [] на видимый draft |
| Locks / owner XML / If | 256 активных на draft / 4 KiB / 8 KiB |
| Параллельные запросы | Все DAV methods используют общие 8 stream slots |

DTD/entities запрещены. Безопасные пути используют NFC/Unicode SimpleFold, не допускают traversal, encoded slash/backslash, query. MOVE сохраняет сегменты исходного суффикса и корректен при разных регистрах/Unicode. Root URL с завершающим `/` и без него эквивалентны для GVfs discovery.

| Ответ | Действие пользователя |
|---|---|
| 201 / 204 | Создано / заменено или выполнено. Если ответ потерян, проверить ETag/содержимое до повтора. |
| 401 / 403 / 404 | Проверить отдельный credential, TTL/отзыв, draft/hold, транспорт и владение. Чужие передачи не раскрываются. |
| 400 / 405 / 415 | Исправить XML/условия/метод/encoding; MKCOL требует пустое тело, COPY и partial PUT не поддерживаются. |
| 409 / 412 / 423 | Обновить список; создать родителя, разрешить конфликт версии или закрыть файл/снять свою блокировку. |
| 413 / 429 / 507 | Уменьшить запрос/метаданные, дождаться stream slot, разделить слишком большой draft. Меньший Depth помогает только при лимите ответа, а не общего namespace. |
| 500 / 503 | Storage/scanner/DB ошибка или maintenance; прежний файл остаётся. Проверить сервер, затем повторить. |

## Воспроизведение проверки
Только disposable ресурсы, не production. Серверные HTTP tests и реальные клиенты проверяются отдельно:

```sh
FILEX_RCLONE_PATH=/absolute/path/to/rclone python3 scripts/test-integration.py TestIntegrationWebDAV
FILEX_NATIVE_WEBDAV_CLIENT=macos python3 scripts/test-integration.py TestNativeWebDAVClients
docker build -f scripts/webdav/Dockerfile -t filex-webdav-gvfs-test scripts/webdav
FILEX_NATIVE_WEBDAV_CLIENT=linux python3 scripts/test-integration.py TestNativeWebDAVClients
```

Без rclone/native env соответствующий opt-in test отмечает skip; это не PASS нативного клиента. Mac использует временную точку монтирования и снимает её в finally. Linux использует реальный GVfs в отдельном D-Bus и локальный Docker Desktop bridge; тест фиксирует GLib/GVfs версии. Server-created fixture доказывает GET независимо от кеша записавшего клиента, данные больше одного encrypted chunk проверяют границы.

Для Windows подготовлен [windows.ps1](../scripts/webdav/windows.ps1). На выделенном Windows 10/11 стенде с рабочим WebClient и trusted HTTPS запустить из PowerShell 7 (корректное чтение UTF-8 исходника с кириллицей):

```powershell
./scripts/webdav/windows.ps1 -Url 'https://test.example/api/v1/webdav/<draft-uuid>/'
```

Если доступен только Windows PowerShell 5.1, сначала сохранить локальную копию скрипта как UTF-8 с BOM: без BOM он может прочитать кириллицу как ANSI. Это требование к проверочному скрипту, а не к Проводнику. [Объяснение Microsoft](https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.core/about/about_character_encoding).

Пароль вводится через Get-Credential; скрипт не меняет реестр, службу и TLS security. Он проверяет подключение/каталог/файл >5 MB/хеш/чтение/замену/rename/delete и удаляет собственные данные/drive. Дополнительно вручную прочитать файл, загруженный через FileX до подключения, проверить отзыв, publish/freeze/lock конфликт и согласованный большой файл. Записать Windows build/WebClient, лимиты станции, TLS chain, серверный source hash и результаты; **до этого Windows — ожидает стенда**.

Где менять: [HTTP adapters](../internal/httpapi/webdav.go), [PUT](../internal/httpapi/webdav_put.go), [read](../internal/httpapi/webdav_read.go), [locks](../internal/transfer/dav.go), [staging](../internal/files/staging.go), [migration](../internal/database/migrations/0017_native_webdav.sql), [UI](../apps/web/components/WebDAVPanel.tsx). Контракт [webdav.json](../contracts/webdav.json) и OpenAPI x-webdav генерируются; AsyncAPI не получил новых событий. Traceability — [matrix](test-matrix.md), кадры и демонстрация — [тур](screen-tour.md).

Источники протокола/клиентов: [RFC 4918](https://www.rfc-editor.org/rfc/rfc4918.html), [Apple webdavfs](https://github.com/apple-oss-distributions/webdavfs), [GNOME GVfs](https://github.com/GNOME/gvfs), [Microsoft WebDAV Redirector](https://learn.microsoft.com/en-us/iis/publish/using-webdav/using-the-webdav-redirector).
