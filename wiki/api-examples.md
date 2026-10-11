# API: локальные curl примеры

## Контекст
Точный контракт: [OpenAPI](../contracts/openapi.json). Примеры используют только собственный dev stack. Файлы login/session/cookies содержат секреты, их нельзя коммитить или логировать; каталог с ними ограничить владельцем. UUID и revision ниже заменяются значениями ответов.

```sh
umask 077
filex_api=http://localhost:8090/api/v1
curl --fail --silent --show-error "$filex_api/auth/login" \
  -H 'Origin: http://localhost:8090' -H 'Content-Type: application/json' \
  --data-binary @private-login.json --cookie-jar private-cookies \
  --output private-session.json
filex_csrf=$(jq -r .csrfToken private-session.json)
curl --fail --silent --show-error "$filex_api/transfers" \
  -X POST --cookie private-cookies -H 'Origin: http://localhost:8090' \
  -H "X-CSRF-Token: $filex_csrf"
```

private-login.json имеет `{ "email": "your-local-user@example.test", "password": "your actual local password" }`. Не использовать показанный текст как готовый пароль. POST draft возвращает id/revision. API token с transfers:write может заменить cookie через Authorization: Bearer; ADMIN scopes не выдаются API tokens.

## tus и publish

```sh
curl --include --fail "$filex_api/transfers/<transfer-uuid>/uploads" \
  -X POST --cookie private-cookies -H 'Origin: http://localhost:8090' \
  -H "X-CSRF-Token: $filex_csrf" -H 'Tus-Resumable: 1.0.0' \
  -H 'Upload-Length: 5' -H 'Upload-Metadata: filename YS50eHQ='
```

Location указывает на /api/v1/uploads/UUID; filename metadata здесь a.txt. PATCH отправляет bytes ≤4 MiB с Content-Type application/offset+octet-stream, Upload-Offset и Tus-Resumable. При неизвестном результате PATCH выполнить HEAD и продолжить с returned Upload-Offset. POST на `<Location>/finalize` запускает server validation. Затем GET transfer возвращает текущую revision, которую нужно включить в publish:

```json
{"expectedRevision":3,"ttlSeconds":3600,"commonLink":true,"password":"123456","recipients":[{"email":"recipient@example.test","password":"654321"}]}
```

Payload publish отправляется в POST /transfers/{id}/publish с теми же cookie/Origin/CSRF. Число revision — пример, не фиксированная константа. Password PIN только демонстрационный. Изменение уже выполненного publish request возвращает conflict.

## Отчёт и encrypted artifact
POST /transfers/{id}/reports `{kind:"user",locale:"ru",timezone:"Europe/Moscow"}` создаёт revision; GET /reports/{id} после renderState=ready возвращает artifact IDs и SHA. Для каждого скачивания: POST /artifacts/{id}/credentials → token → POST /artifacts/{id}/claim `{token}` с cookie jar → GET returned content URL с original session и path-scoped artifact cookie. Range и HEAD поддерживаются, полный coverage расходует credential. QR bundle следует тому же протоколу после POST /transfers/{id}/qr-bundle.

Public POST /api/v1/verify-report получает JSON reportId/revision/canonicalHash. Только точное совпадение active USER revision даёт verified=true; чужой hash, trash/purge/ADMIN/unknown дают один и тот же 404. JSON report сравнивается по точным canonical bytes, а не после произвольного JSON reformat.

## Ошибки
400 invalid_input, 401 unauthenticated, 403 forbidden, 404 not_found, 409 conflict, 429 rate_limited. Для tus/Range есть дополнительные protocol statuses, описанные OpenAPI. Generic internal_error не содержит DB credentials или request body. Browser GET/HEAD получателя не приобретает download lease; acquire выполняется явным POST.

## Эксплуатация: ADMIN cookie + fresh re-auth

Создание backup использует отдельный BackupProvider UUID. Credential cookies/CSRF хранить приватно; значения ниже — env placeholders существующей локальной сессии.

```sh
curl --fail -b "$COOKIE_JAR" -H "Origin: $ORIGIN" -H "X-CSRF-Token: $CSRF" \
  -H 'Content-Type: application/json' -d @backup-request.json \
  "$ORIGIN/api/v1/admin/operations/jobs"
```

`backup-request.json`: `{ "kind": "backup", "providerId": "<backup-provider-uuid>", "reason": "Перед плановым обслуживанием" }`. Ответ 202 `{id}` означает queue, не completed. GET `/admin/operations` показывает state/result. Restore: kind=restore + sourceId=completed backup UUID + providerId=pre-backup destination. Gate закрывается при queue; restart и reconciliation/reopen обязательны. [Runbook](operations-recovery.md), полный [OpenAPI](../contracts/openapi.json).

## Роли, scopes и идемпотентность

| Клиент | Разрешённая граница |
|---|---|
| USER web session | Собственные transfers/reports/artifacts/views; mutation требует exact Origin и CSRF |
| API token | Account owner boundary плюс выданные scopes; не ADMIN re-auth и не способ получить чужие данные |
| Recipient | Отдельная capability/PIN session и opaque refs; GET/HEAD не приобретают lease |
| ADMIN web session | Privileged routes с current role и re-auth ≤5 min; отдельный reason/audit для forensic/recovery/чужих exports |
| Host CLI | Доверенная host/container граница; сохраняет guards, см. CLI |

Доступные token scopes: account:read, transfers:read, transfers:write, reports:read, reports:write, exports:write, notifications:read, notifications:write, webhooks:write. TTL token 60…2592000 s; credential секрет возвращается при создании, в БД hash. Scope account:write не выдаётся API tokens, ADMIN scopes также не выдаются. Точный route/security contract — OpenAPI и internal/httpapi/app.go.

Универсального Idempotency-Key API не обещает. Publish повторяет тот же request fingerprint идемпотентно; другой request после publish даёт conflict. Settings/tree mutations используют expected revision/CAS. Worker jobs имеют собственные lease/fence/idempotency правила. SMTP/webhooks at-least-once: потребитель дедуплицирует stable eventId. После неизвестного tus PATCH результата читается HEAD offset, а не вслепую повторяются новые bytes.

503 может означать закрытый maintenance gate/недоступность зависимости; generic ошибка не раскрывает topology. Восстановление после network change/restore не обходится повторным login до restart/reconciliation. API examples подтверждаются локальными contract/E2E tests; реальный SMTP/S3 acceptance описан [отдельно](environment-acceptance.md).
