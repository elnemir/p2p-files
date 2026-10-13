# ADR-010: WebDAV для загрузки в черновик

Дата: 2026-10-12. **Исторический профиль: ограничения GET/LOCK/overwrite/unknown length заменены [ADR-011](011-native-webdav.md).** Выбор протокола подтверждён владельцем; реализован первый срез upload inbox. Реализация и фактическая совместимость фиксируются в [руководстве](../webdav-upload.md) и [session](../session.md).

## Контекст и проблема
FileX принимает файлы через tus из браузера. Для загрузки из файлового менеджера нужен WebDAV. Прямое предоставление Local/S3 каталога обошло бы ownership, FXP1, policy/scanner, immutable publish и удаление. В FXP1 полный размер файла аутентифицирован в каждом chunk: неизвестный Content-Length нельзя поддержать простой записью с изменяемым size.

## Решение и поток данных
Владелец открывает draft → подтверждает пароль в веб-интерфейсе → получает отдельный API token со scope `webdav:<transfer UUID>` и сроком до 30 дней → подключает URL `/api/v1/webdav/<UUID>/`, username `filex`, password token. Credentials доступны один раз, digest/expiry/auth epoch хранятся в существующей api_tokens; отзыв, password reset и recovery invalidation общие. Generic API token и cookie сами по себе не дают WebDAV-доступа.

WebDAV → тот же Traefik/network/maintenance gate → Basic token auth → owner + scope + draft + freeze → метод → существующие files/storage и draft tree. PUT с известным Content-Length 0–1 TiB → reserve → bounded 4 MiB chunks → FXP1 encrypted Local/S3 → SHA-256/CRC32/MIME/scanner → ready → 201. Ни crypto primitives, ни payload header, ни storage provider selection не меняются. Auth проверяется на границах записи/финализации в PG transaction; запрещённое состояние не даёт завершить upload. При неуспехе сервер отменяет созданный upload и регистрирует cleanup. При падении процесса незавершённый файл виден как uploading; удалить его перед повтором можно в UI/WebDAV.

## Границы протокола
- Корень подключения — один существующий draft. OPTIONS, PROPFIND depth 0/1, HEAD, MKCOL, PUT, MOVE и DELETE обслуживают дерево; неизвестные свойства получают 404 propstat, изменение свойств — 403. Только фиксированные live properties; Входной XML ограничен 64 KiB/16 уровней, полный ответ — 8 MiB; ответ строится по одному ресурсу и переполнение даёт 507 до отправки 207. Внешние entities не разрешены, metadata tree ограничен 10 000 nodes.
- GET/COPY payload запрещены политикой upload inbox; recipient flow не расширяется. LOCK/UNLOCK не реализуются, class 2 не объявляется. Подключение не является диском общего назначения или редактором Office.
- PUT создаёт новое имя. Существующий файл не перезаписывается; требуется явное удаление или другое имя. MOVE только внутри того же draft, без overwrite. Destination не вызывает исходящих HTTP requests.
- Нет chunked PUT/Content-Range/resume для WebDAV; 411/400 до записи. tus остаётся протоколом resume. Обрыв PUT не выдаётся за success.
- Невалидные пути, `..`, encoded slash/backslash, query, absolute cross-origin Destination и case-fold collisions отвергаются. Каталоги из существующих вложенных tus paths видны как виртуальные ancestors. Рекурсивные операции выбирают descendants по общему Unicode SimpleFold PathKey; `ſeed`/`SEED` относятся к одной папке, суффиксы сохраняются по сегментам.
- Basic допускается через HTTPS с проверенной proxy policy; HTTP — только при настроенном loopback development PublicOrigin и совпадающем Host. Dev Compose публикует ingress исключительно на loopback; нельзя открывать этот dev-профиль в LAN/Internet. Внутренний адрес контейнера Traefik не принимается за client IP, непроверенный XFF не используется. Cookies и чужой Origin не используются для авторизации. Доступ закрывается после publish и при freeze/maintenance/revocation.

## Architecture consistency / crypto review до реализации
PG остаётся authoritative; операции дерева сериализуются parent-first. Shared runtime gate охватывает запрос; Все DAV methods входят в общий лимит 8 тяжёлых потоков. Таймаут чтения обновляется на bounded blocks вместо бесконечного whole-file upload. Нет нового listener, внешних хранилищ или временного plaintext файла. Сохраняются scopes, encrypted chunk compensation и server finalize. AsyncAPI и JSON event payloads не расширяются: upload не порождает новых событий; публикация/уведомления выполняются прежним потоком.

Остаточные ограничения: upload-only профиль не гарантирует работу Finder/Windows WebClient/Office; клиент, требующий locks или неизвестную длину, получает явную ошибку. Совместимость объявляется только после запуска конкретного клиента. Это внутренний design review, не независимая release acceptance.

## Проверка
Позитивный XML browse/UTF-8/MKCOL/PUT/move/delete и отдельный WebDAV-клиент; checksums/FXP1/no plaintext; wrong owner/scope/password, expired/revoked token, CSRF/Origin, traversal/XML faults, collision/race, truncated body/storage fault, freeze/publish/maintenance, auth revoked midstream. REST/tus/recipient regression, route contracts, ingress и UI/a11y обязательны.

Источники: [RFC 4918](https://www.rfc-editor.org/rfc/rfc4918.html), [rclone WebDAV](https://rclone.org/webdav/).
