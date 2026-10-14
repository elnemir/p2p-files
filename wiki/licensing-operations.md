# Лицензии: сборка, панель издателя и эксплуатация

## Контекст и реализованный срез

Владелец утвердил профиль 2026-10-14. [Модель](licensing.md), [ADR-016](adr/016-offline-licensing.md). Клиентский FileX проверяет подписанный файл локально; панель издателя — отдельные Go API, Next.js UI и PostgreSQL. Секретный ключ доступен только signer, который читает durable outbox из БД. В клиентские API/worker/CLI images signer и vendor API не включены.

Реализованы subscription/perpetual, ограничения пользователей и размера передачи, три feature flags, expiry continuity, будущая активация продления, revision/CAS, обнаружение обычного отката часов, отдельная подпись release metadata и проверка обновления до миграции. Это не anti-root/anti-clone механизм. Production deployment не выполнялся; внешняя квалификация ключевого хранилища, MFA провайдера и независимый security review остаются отдельными задачами.

## Поток клиента

1. Сборка содержит vendor public trust и отдельно проверенные release metadata. Runtime env не может заменить trust roots.
2. Migration 0020 создаёт `ops_license` со случайным installation ID. Штатный restore сохраняет все `ops_*`: identity, принятую revision, license epoch и max observed time.
3. Без лицензии разрешено создать только первого ADMIN через доверенный host CLI; обычные пользователи/загрузки/publish запрещены. ADMIN может войти локальным паролем, подтвердить его и открыть раздел «Лицензия».
4. ADMIN скачивает installation request и передаёт издателю. Издатель выпускает `.jws`; ADMIN загружает файл, проверяет preview и применяет его с expectedEpoch. Текущий сеанс/роль/reauth проверяются повторно в транзакции.
5. Renew сохраняет entitlementId, повышает revision и меняет licenseId. Future notBefore хранится pending и не вытесняет текущую лицензию раньше времени. Ошибочная подпись, чужая установка, старая revision или CAS conflict оставляют текущую лицензию.

CLI для автоматизации в доверенной host/container границе:

```sh
filex-cli license status
filex-cli license apply < /absolute/private-license-input.json
```

Input содержит `{"envelope":"<полный JWS>","expectedEpoch":0}`. Настоящий envelope не передавать через argv и не логировать. ADMIN API: `GET /api/v1/admin/license`, `POST /api/v1/admin/license/preview`, `PUT /api/v1/admin/license`; [контракт](../contracts/openapi.json). Совместимые v1 error codes: `invalid_input` → 400 (`message=invalid_license`), `forbidden` → 403 (`message=license_restricted`), устаревший epoch/revision → `conflict`/409. Исходный enum Error не расширяется; UI распознаёт два фиксированных diagnostic messages. GET status возвращает claims только ADMIN с reauth, не public status.

## Ограничения и поток данных

Включённые USER+ADMIN, включая JIT, занимают места. Получатели не занимают места. Первичный bootstrap ADMIN возможен только при полном отсутствии users; это не периодический бесплатный слот. Квота и создание пользователя атомарны под `ops_license FOR UPDATE`; отключённые учётные записи не считаются. Прямое администрирование БД не является недоверенным API.

Upload резервирует объявленный plaintext размер. Unknown-length DAV резервирует по фактическим bounded chunks; provisional и final encrypted candidates одного пути учитываются как максимум, а не сумма. Переименование синхронизирует `license_path_key` через SQL trigger, чтобы повторное создание старого пути не обходило квоту. Отмена/cleanup освобождают reservation вместе с существующим lifecycle. Publish повторно проверяет полный размер. Limitation: expired/deleting state учитывается по существующим draft state rules; физическая ёмкость temporary ciphertext не является коммерческой квотой.

API, host CLI, JIT и worker-facing domain methods используют общие guards. `webdav` закрывает новые credentials и DAV mutations, кроме удаления/освобождения locks; `federated_auth` закрывает завершение новых SSO/LDAP sessions; `white_label` закрывает применение branding. Существующий бренд автоматически не сбрасывается. Локальный ADMIN fallback сохраняется. Новый write API token и новый export job требуют активной лицензии; ранее принятые export jobs завершаются. Уже опубликованные recipient downloads/Range/ZIP, one-time consume, revoke/TTL/freeze, audit, backup/recovery и mandatory cleanup сохраняют исходную семантику.

Снижение maxEnabledUsers ниже текущего usage не отключает существующих пользователей. Оно запрещает новые аккаунты. При превышении transfer limit разрешено удаление, увеличение/publish запрещены. UI объясняет состояние; сервер остаётся authority. Нет runtime bypass по APP_ENV.

## Ключи и поставка

[Signer](../cmd/filex-license-signer/main.go) имеет owner-only команды подготовки ключей и release signature. Ниже операторские примеры, не выполненный production bootstrap:

```sh
filex-license-signer keygen vendor-2026-01 /secure/vendor-license.seed > /secure/vendor-trust.json
filex-license-signer keygen release-2026-01 /secure/vendor-release.seed > /secure/release-trust.json
ISSUER_SIGNING_KEY_FILE=/secure/vendor-release.seed ISSUER_KEY_ID=release-2026-01 \
  filex-license-signer sign-release < /secure/release-input.json > /secure/release.jws
```

Keygen использует CSPRNG, пишет raw Ed25519 seed в новый файл mode 0600 через O_EXCL и выводит только public trust JSON. Не использовать test/qualification keys в поставке. Seed должен быть доступен uid signer (в Compose 65532), остальные пользователи не должны читать его; signer отвергает group/world-readable файл. Хранилище, резервирование и escrow выбирает оператор; private key не хранится в issuer PostgreSQL и не экспортируется UI.

Release input: `version`, `sourceSha256` (64 lowercase hex, digest проверенного исходного snapshot), `publishedAt` (UTC). Release metadata подписывается отдельным ключом/типом `filex-release+jws`; license verifier не принимает этот тип. Четыре **build arguments**, а не runtime settings:

| Аргумент | Значение |
|---|---|
| FILEX_LICENSE_TRUST | Compact JSON `kid → base64 public key` лицензий |
| FILEX_RELEASE_TRUST | Compact JSON `kid → base64 public key` release metadata |
| FILEX_RELEASE_METADATA | Полная подпись `.jws` release input |
| FILEX_RELEASE_SOURCE | Тот же sourceSha256 квалифицированного исходного snapshot |

[Dockerfile.api](../deploy/Dockerfile.api) встраивает значения через Go linker в customer API/worker/CLI. Пустой trust не разрешает ни одну лицензию. Подмена даты runtime env невозможна; patched binary/root остаётся вне гарантии. Build pipeline обязан вычислить sourceSha256 из фактического snapshot и передать совпадающий digest; само равенство двух входных строк не доказывает provenance. Проверенный candidate не должен включать qualification-only keys. Штатный bootstrap проверяет право обновления **до** `database.Migrate`, если есть pending migrations. При точном совпадении схемы запуск разрешён для восстановления/continuity даже с повреждённой лицензией; runtime guards закрывают новые платные действия, а ADMIN/CLI позволяют исправить права. Новая сборка без права обновления на той же схеме работает только в этом ограниченном режиме. Если лицензия не разрешает дату релиза, сначала продлить её старым разрешённым CLI, затем повторить upgrade. Автоматического downgrade БД нет.

Key rotation: сначала доставить сборку, доверяющую старому и новому public key; затем начать новые выдачи новым kid. Не переиспользовать kid с другим ключом. При compromise нужен новый доверенный релиз и перевыпуск лицензий; изолированный клиент не узнает об отзыве сам.

## Развёртывание закрытой панели

[Compose](../deploy/issuer-compose.yaml) не связан с customer Compose/volumes. Он использует отдельную БД `filex_issuer`, PostgreSQL owner только для migration, роли `filex_issuer_app` и `filex_issuer_signer` с отдельными паролями. API не может изменить issued claims/artifact; signer не создаёт issuance. UPDATE/DELETE issued record и audit history блокируются. БД owner/root может изменить это правило; резервный журнал и внешнее escrow остаются обязанностью оператора.

Переменные панели:

- `ISSUER_OWNER_PASSWORD`, `ISSUER_APP_PASSWORD`, `ISSUER_SIGNER_PASSWORD`: независимые URL-safe случайные пароли минимум 24 символа; хранить в закрытом env/secret store.
- `ISSUER_ORIGIN`: точный HTTPS origin без path/query. Панель доступна по `/issuer`.
- `ISSUER_OIDC_URL`, `ISSUER_OIDC_CLIENT_ID`, `ISSUER_OIDC_CLIENT_SECRET`: выделенный OIDC client; callback `<origin>/issuer/auth/callback`.
- `ISSUER_MFA_ACR`: **реальное assurance value провайдера**, соответствующее enforced MFA. Не указывать значение password-only policy.
- `ISSUER_OPERATORS`: JSON mapping стабильного OIDC `sub` в `viewer`, `issuer` или `security_admin`. Email и произвольные IdP role claims не дают полномочия.
- `ISSUER_KEY_ID`, `ISSUER_SIGNING_KEY_FILE`: текущий kid и абсолютный путь приватного seed, монтируемого только signer.

При отдельном разрешении на размещение оператор запускает postgres, затем one-shot `migrate` с profile `setup`, затем api/signer/web. Migration выполняет `ProvisionRoles` от owner; runtime не получает owner password. Пример команд после подготовки закрытого `/secure/issuer.env` и разрешения на размещение:

```sh
docker compose --env-file /secure/issuer.env -f deploy/issuer-compose.yaml up -d --build --wait postgres
docker compose --env-file /secure/issuer.env -f deploy/issuer-compose.yaml --profile setup run --rm --build migrate
docker compose --env-file /secure/issuer.env -f deploy/issuer-compose.yaml up -d --build api signer web
```

API и web публикуются только на loopback 8082/3002. Signer не имеет входного HTTP endpoint и внешней сети; только internal database network. [Traefik пример](../deploy/issuer-routes.example.yaml) предназначен для host-proxy: заменить exact domain и настроить TLS. Если Traefik сам в контейнере, loopback URLs из примера не работают — использовать выделенную сеть/доступные private addresses. Не открывать эти loopback upstream напрямую наружу.

MFA flow: PKCE S256 + state + browser binding + nonce → проверка issuer/audience/signature/nonce/ACR/auth_time → allowlisted subject → PostgreSQL cookie session на 15 минут. Выпуск требует Origin/CSRF и auth_time не старше пяти минут; reauth снова проходит IdP. Logout требует session/Origin/CSRF, но не свежего MFA. Панель не реализует парольный запасной вход. Проверять реальный provider MFA policy до использования реального signing key.

## Выпуск и повтор после сбоя

Форма принимает customer/order identifier, installation ID, модель, начало действия (UTC; пусто означает сейчас), сроки, лимиты и features. Будущая дата позволяет заранее выдать pending renewal. Сначала preview, затем явное подтверждение. Права, license ID и idempotency key фиксируются на эту попытку; retry после transport error повторяет тот же запрос. Новый payload с тем же key возвращает conflict. Issued claims и artifact неизменяемы. Продление из истории копирует entitlement и предлагает следующую revision; конкурирующие issuers получают conflict вместо двойной выдачи.

API transaction → immutable issuance + audit + outbox → signer `FOR UPDATE SKIP LOCKED` → стандартная подпись → локальная Verify → artifact/hash/audit/queue completion в одной transaction. Сбой до commit не публикует файл; после restart повторяется та же выдача. Signer при ошибке завершает процесс с нейтральным сообщением; оператор проверяет health/БД/ключ и перезапускает, approved job остаётся в очереди. Не редактировать ошибочный issued record вручную.

История UI показывает последние 200 выдач; полный реестр в PostgreSQL. Отдельные customer CRM CRUD, платежи, удалённый отзыв, migration grants, recovery clock grants и веб-редактор key rotation не реализованы. Перенос на новую installation identity оформляется отдельной цепочкой после проверки договора; отключить старую offline копию нельзя. Эти функции не маскируются кнопками-заглушками. Бэкап БД плюс независимо защищённый backup seed нужны для disaster recovery; реальная процедура escrow ещё требует внешней квалификации.

## Локальная разработка и проверка

`init-dev.py` создаёт одноразовые fixture keys, public build arguments в `.env` и private material только в `.cache/dev-license.json` mode 0600. После `make up` выполнить `python3 scripts/activate-dev-license.py`. Это ограниченная по времени локальная лицензия (сутки); существующие env/keys не перезаписываются. Повторная активация текущим простым fixture script предназначена для новой тестовой БД; для renewal действующей установки используется панель с той же entitlement chain. Existing dev volumes требуют штатного закрытия maintenance gate перед migration, не удаления данных.

[Тесты подписи](../internal/licensing/license_test.go), [quota/continuity tests](../internal/httpapi/licensing_integration_test.go), [issuer race/fault tests](../internal/issuer/store_integration_test.go), [OIDC/MFA wire tests](../internal/issuer/auth_integration_test.go). Скрипты qualification генерируют ephemeral public trust и подписанную лицензию вместо отключения guard. `scripts/test-issuer-browser.py` упражняет настоящий API/UI/signing с **синтетическим** TLS IdP; это не внешняя MFA приёмка. Сначала собрать `npm run build --workspace @filex/issuer` с `NEXT_TELEMETRY_DISABLED=1`. Общий browser gate запускает issuer-сценарий после customer profiles, используя тот же закреплённый Linux browser server; самостоятельный скрипт использует локально установленный Playwright Chromium.

Актуальные команды, source boundary и результаты — [session](session.md). Исторические PASS не распространяются на новый diff. Квалификация customer images и двух vendor images разделяется; [supply chain](../scripts/supply_chain.py) включает обе новые сборки. Independent security review и deployment остаются отдельными gates.

## Внутренний review реализации — 2026-10-14

Проверены границы доверия: клиент содержит только public trust; vendor seed монтируется только signer; OIDC allowlist/MFA/CSRF выполняются сервером; PostgreSQL роли разделяют approval и signing. Cryptography использует стандартный Ed25519/JWS, точные signed bytes, strict base64url/JSON, точный регистр имён полей (включая вложенные limits/header/release) и фиксированный алгоритм. Release и license signatures имеют разные типы/ключи; customer ZIP keys не переиспользуются.

Проверены quota linearization и lock order, fail-closed для новых операций, continuity/TTL/one-time при expiry и restore exclusion ops_*. Найденные при review проблемы исправлены: квота после переименования DAV; recovery startup с повреждённой лицензией; maintenance lock для CLI; missing claims/limits; сохранение Error enum v1. Негативные/concurrency/fault тесты подтверждают эти границы. Проверяющий Node crypto в customer browser fixture подписывает совместимые лицензии независимо от Go signer.

Это внутренний инженерный review, не внешний криптоаудит. Полная доверенная supply chain, customer root/VM rollback, реальный MFA assurance/escrow, договорной перенос и принятые остаточные риски остаются внешними условиями. Производственный guard FileX не снят.

## Веб-настройка первого запуска

[Мастера запущенных контейнеров](first-run-setup.md): одноразовый ADMIN FileX и encrypted OIDC bootstrap портала; [ADR-017](adr/017-first-run-wizards.md). Docker/DB/ключи готовит host; production guard сохранён.
