# Первый запуск: веб-мастера запущенных контейнеров

## Контекст и проблема
Мастера заменяют ручное создание первого ADMIN FileX и заполнение OIDC env портала. Они работают внутри уже запущенных API/web контейнеров и не устанавливают Docker, не выполняют shell commands, не получают Docker socket. Решение, угрозы, состояния и review: [ADR-017](adr/017-first-run-wizards.md).

## Что подготовить на сервере
FileX: PostgreSQL, Redis, KEK, local/S3 storage, public build trust и миграции согласно [development](development.md)/[licensing operations](licensing-operations.md). Портал: отдельная БД с owner migration/разделением ролей, signer с приватным Ed25519 seed, HTTPS origin/ingress. Веб-мастер никогда не принимает vendor private key. FileX `APP_ENV=production` всё ещё блокируется до release acceptance.

Мастер включается только временным host capability. Подготовить **новый** закрытый каталог вне Git и backup БД:

```sh
python3 scripts/setup-access.py product /absolute/private/filex-bootstrap
# Для отдельного портала:
python3 scripts/setup-access.py issuer /absolute/private/issuer-bootstrap
```

Скрипт создаёт каталог 0700, bootstrap.env и token.txt 0600 через O_EXCL. Срок токена — один час. stdout содержит только срок, секреты не печатаются. `token.txt` открыть локальным редактором и ввести в браузер; не пересылать ссылкой/query и не публиковать в логах. bootstrap.env содержит SHA-256 токена и RFC3339 expiry; issuer дополнительно получает независимый случайный `ISSUER_CONFIG_KEY` (base64 32 bytes). **Сохранить этот ключ отдельно от backup БД и не регенерировать при перезапуске/истечении токена.**

Переменные product: `FILEX_SETUP_TOKEN_HASH`, `FILEX_SETUP_EXPIRES_AT`. Переменные issuer: `ISSUER_SETUP_TOKEN_HASH`, `ISSUER_SETUP_EXPIRES_AT`, `ISSUER_CONFIG_KEY`, плюс прежний `ISSUER_ORIGIN`. Пример подключения файла поверх уже существующего env:

```sh
docker compose --env-file .env --env-file /absolute/private/filex-bootstrap/bootstrap.env -f deploy/compose.yaml up -d api
# Только после issuer migrate, с подготовленным signer и прочим env:
docker compose --env-file /absolute/private/issuer.env --env-file /absolute/private/issuer-bootstrap/bootstrap.env -f deploy/issuer-compose.yaml up -d api web signer
```

Это операторские примеры, не выполненный deployment. `up -d api` пересоздаёт контейнер для передачи env; одной команды restart недостаточно. Существующие volumes/KEK не удалять. Для старой схемы FileX migration 0021 применяется штатным [controlled upgrade](controlled-upgrade.md); мастер не обходит maintenance gate. Existing issuer env mode поддерживается; при выборе database setup удалить OIDC env/ISSUER_OPERATORS, смешение режимов отвергается. Использованный issuer с issuances/sessions/flows не принимает первичную config.

## FileX: доступ → администратор → активация
1. Открыть `/setup` на настроенном publicOrigin (development: `http://localhost:8090`). На главной пустой установки появляется ссылка. Exact Origin обязателен; `127.0.0.1` вместо настроенного localhost — другой origin.
2. Ввести токен. Указать email, пароль 12–1024 байта и подтверждение. Проверить email на шаге подтверждения; секрет не показывается.
3. Завершение атомарно создаёт ADMIN/password history/audit и закрывает `ops_setup`. Войти обычным паролем.
4. В Администрирование → Лицензия скачать installation request, передать издателю, загрузить подписанный JWS, проверить и применить. Все прежние offline guards действуют; мастер не выдаёт бесплатную лицензию. Прочие настройки — существующие ADMIN panels.

При existing users мастер закрывается migration; CLI user create также закрывает latch. Невозможно повторно получить ADMIN через POST или открытие страницы. Штатный restore сохраняет `ops_*`; полный rollback VM/root вне этой гарантии.

## Портал: доступ → OIDC/MFA → вход
1. Открыть `/issuer/setup` на HTTPS origin, ввести token.
2. В IdP зарегистрировать callback `<ISSUER_ORIGIN>/issuer/auth/callback`. Внести HTTPS issuer, client ID/secret, реальное MFA ACR и стабильный `sub` первого оператора. Email не заменяет sub. Первый оператор получает `security_admin`.
3. Перед сохранением проверяется OIDC discovery с timeout 10 секунд. Client secret/config шифруются существующим AES-GCM в immutable `issuer_setup`. Этот отдельный key отсутствует у signer и не связан с seed лицензий.
4. Перейти ко входу и пройти настоящий OIDC MFA: подпись/audience/issuer/nonce/ACR/auth_time/allowlist проверяются существующей auth. Discovery не доказывает, что client secret, subject и MFA верны — это подтверждает первый вход.

После успеха удалить token/hash/expiry из окружения и приватный token.txt согласно политике. Сохранить `ISSUER_CONFIG_KEY` в окружении **всех API replicas**: без него database setup не работает. HTTP не предоставляет сброс, замену OIDC или просмотр secret. Неверный subject/ACR/client secret после сохранения требует owner recovery: остановить API replicas, сохранить encrypted row и ключ, подготовить исправленный env mode через существующий runbook, не удалять issuance/audit. Публичного password fallback нет.

## Сбои, ограничения и API
GET `/api/v1/setup` или `/issuer/api/setup` возвращает только `state`: pending/disabled/complete. POST с exact Origin и `X-Setup-Token` принимает documented JSON ([customer](../contracts/openapi.json), [issuer](../contracts/issuer-openapi.json)); max16KiB, no-store, нет CORS. Ошибки: 400 input/discovery, 403 capability/origin/expiry, 409 already complete, 503 unavailable.

При потере ответа перечитать status: complete означает commit уже состоялся. Если pending — повторить; password/config никогда не возвращаются. Сбой до commit не закрывает мастер; concurrency допускает одного победителя. При истечении срока host может выпустить новый token/hash/expiry, **сохраняя исходный ISSUER_CONFIG_KEY**; завершённый мастер это не откроет.

Формы русскоязычные, responsive, без localStorage/URL secrets. Domain/TLS/storage/key provisioning остаются host-задачами; wizard не меняет эти защищённые границы. Реальные IdP/MFA и escrow требуют отдельной приёмки. Tests и актуальные результаты — [session](session.md).

## Внутренний review и локальная квалификация

Проверены границы HTTP → capability → PostgreSQL: точный Origin, срок и digest токена, strict bounded JSON, отсутствие readback secrets, atomic latch. FileX хранит завершение в ops_setup и не меняет license/maintenance guards. Issuer использует отдельный AES-GCM key/AAD, unique singleton, только SELECT/INSERT для web role, отсутствие доступа signer к config. Установка с existing issuance/session/flow не принимает первичный bootstrap. Env mode и database mode нельзя случайно смешать.

Browser проверяет оба настоящих API, PostgreSQL, confirmation/reload, первый ADMIN и synthetic OIDC MFA login, 390px reflow и axe; скриншоты просмотрены. Исправлена отсутствующая цель skip-link; длинные значения переносятся в review. Host helper проверен на 0700/0600, независимость секрета от hash и отказ перезаписи.

В `make e2e` включены обычный issuer workflow и оба setup workflow. Для отдельного запуска после `npm run build`: `FILEX_SETUP_BROWSER=1 python3 scripts/test-issuer-browser.py` с Go из go.mod, Docker и Playwright Chromium. Артефакты — `.cache/licensing-issuer-browser/setup-*-mobile.png`. Актуальные hashes, результаты и границы — [session](session.md); внешний crypto/security review и настоящая MFA policy не заявляются выполненными.
