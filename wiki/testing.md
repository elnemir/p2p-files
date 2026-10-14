# Проверки, CI и воспроизводимое evidence

## Контекст

Проверка подтверждает конкретный commit/source hash и профиль. `make check` не запускает PostgreSQL integration, browser, scanners или полный load. Общий зелёный build не закрывает внешнюю приёмку. Источники команд — [Makefile](../Makefile), [acceptance.py](../scripts/acceptance.py), [CI](../.github/workflows/ci.yml).

## Базовые и целевые команды

Для WebDAV добавлены bounded XML/path fuzz, PG/race/fault/transport tests и отдельный UI сценарий во всех пяти профилях. Срез UX добавляет три USER/recipient/ADMIN сценария к native WebDAV, branding и SSO: набор содержит 12 случаев на профиль (60 суммарно). TLS ingress fixture также проверяет issuance/MKCOL/PUT/PROPFIND/DELETE/revoke через настоящий proxy после reviewed reconcile/reopen. Независимый rclone test запускается дополнительно при `FILEX_RCLONE_PATH` с явно установленным проверенным бинарником; иначе он явно skipped. Это dev/test tool, новая runtime dependency не добавляется. Команды/подтверждённые версии и ограничения — [WebDAV](webdav-upload.md); текущий source-bound результат — [session](session.md). Исторические счётчики в более ранних evidence не означают квалификацию нового среза.

| Команда | Что проверяет | Требования / ограничение |
|---|---|---|
| make check | gofmt/vet/staticcheck, TS strict/ESLint/Prettier, Go race, contract/generator/Python negative tests, docs/traceability, Go/Next builds | Go/Node/Python; без тестового DB URL PG tests пропускаются |
| make integration-docker | Все TestIntegration* с race на disposable PostgreSQL; 62 top-level tests (один требует FILEX_RCLONE_PATH) | Docker; каждый test получает отдельную schema, cleanup своего container |
| make test-integration | Те же тесты через FILEX_TEST_DATABASE_URL | Только специально выделенная тестовая БД; URL обязателен |
| make security | npm audit и govulncheck | Network; source reachable findings отдельно от module-only advisories |
| make e2e | 5 Linux browser/emulation profiles × 12 scenarios = 60; contracts, axe/keyboard/responsive/visual | Docker, npm ci, свободный 8090; не реальные устройства |
| make ingress | Настоящий Traefik/test CA, TLS, Host/forwarded/CIDR/secure cookie/reset/private endpoints | Isolated project/random ports, public ACME не вызывается |
| make upgrade-drill | Разные images baseline 28ab4f3/candidate, backup/drain/0017+0018+0019/fault/reopen/one-time | Git history; Compose additive 16→19 с точным ledger/hash и отказом downgrade; PG suite также проверяет prefix/faults |
| make fuzz | Пять кампаний PathKey/FXP1/Range/WebDAV XML + path/If | Default 15 s на кампанию плюс compile; Go seeds/unit сами по себе не fuzz campaign |
| make performance | 32 MiB/2000 rows, load/fault/DR/ZIP hash/resources | Только disposable локальный project; не release-size profile |
| make supply-chain | Source/history/worktree secrets, licenses, пять image scans и SBOM | Docker/network; HIGH/CRITICAL fail incl. unfixed |
| python3 scripts/contract_compatibility.py | Консервативный breaking diff к f68fc08 | Existing schemas/operations/literals protected; baseline не сдвигать ради PASS |
| python3 scripts/release_bundle.py | 5 cross-built verifier targets, checksums/build metadata/release notes | Не запуск на всех OS, не публикация |
| make docs-check | Ссылки, wiki layout, полный набор requirement IDs/test paths и generated matrix | Быстрый самостоятельный documentation gate |
| make repository-check | Repository policy, YAML/forms, SHA pins, required job names и негативные тесты GitHub settings CLI | Также входит в make check; без обращения к GitHub |
| python3 scripts/github_repository.py audit | Реальные GitHub settings, защита main, security и CodeQL ruleset против baseline | Read-only; gh с admin access, сеть; drift возвращает nonzero |

## 13 записываемых gates

IDs: `check`, `integration`, `dependencies`, `source`, `licenses`, `images`, `fuzz`, `browser`, `ingress`, `upgrade`, `performance`, `compatibility`, `verifier`.

```sh
python3 scripts/acceptance.py run check
python3 scripts/acceptance.py run integration
python3 scripts/acceptance.py run browser
FILEX_PERF_MIB=2050 FILEX_PERF_ROWS=20000 python3 scripts/acceptance.py run performance
make release-check
```

Это примеры запуска отдельных gates, не полный release batch. Для полного пакета выполнить каждый ID. Wrapper сохраняет `.cache/evidence/gates/<gate>.json`: sourceSha256 до/после, sourceUnchanged, Git commit, время, exitCode, passed и профиль. Изменение исходников во время запуска запрещает зачёт. Обычный `make e2e` или `make performance` сохраняет собственные отчёты, но не заменяет gate record wrapper.

Fingerprint включает code/config/contracts/scripts/CI/visual baselines и правила AGENTS/CONTRIBUTING. Markdown внутри wiki и корневые README/CHANGELOG/MEMORY исключены, чтобы публикация измерений не меняла кандидат. Ссылки/матрица проверяются отдельно. Изменение скрипта пути после переноса документации **меняет fingerprint**, хотя бизнес-логика прежняя. Не редактировать старые gate records для нового hash.

## Browser и load profiles

`FILEX_E2E_PROJECTS=chromium make e2e` годится для адресной диагностики, не для полного release gate. Baselines находятся в `scripts/e2e/visual-baselines/linux/<project>/`; 10 изображений для RU light/EN dark. Обновление `FILEX_UPDATE_VISUALS=1 make e2e` допустимо после объяснённого UI изменения и просмотра всех затронутых изображений. Затем запуск без update flag обязателен; acceptance wrapper отвергает baseline update.

Performance принимает 8…8192 MiB и 100…40000 rows; release readiness требует ≥2048 MiB и ≥20000 rows. Нет параметра внешнего URL. [Performance runbook](performance-testing.md) описывает synthetic TLS S3, реальный SIGKILL и искусственное истечение job lease. Эти fixtures не подтверждают SigV4 реального провайдера или production crash RTO.

## CI и хранение результатов

Каждый push в main, PR или workflow_dispatch запускает functional и supply-chain jobs на ubuntu-latest, timeout 45 min каждый, а также repository-policy (10 min). Для functional/supply-chain checkout содержит полную историю, Node 26 из .nvmrc, Go из go.mod, npm ci; deployment отсутствует. Эти два jobs сохраняют evidence при успехе/ошибке, retention 14 дней. Security workflow выполняет CodeQL для Go/JS/TS/Python/Actions на PR/main/manual и раз в неделю, dependency review — на PR; aggregate security-checks обязателен. Проверять conclusion всех required checks и **exact headSha**, а не предыдущего commit. CI performance compact; полный профиль нужен отдельно. [Настройки GitHub и правила merge](repository-governance.md).

`.cache/evidence/` ignored. Browser traces содержат synthetic sessions и не публикуются без review. Исходные logs/scanner JSON, screenshot baseline review, hashes и окружение нужны для расследования; перенос чисел в wiki не заменяет raw evidence. Для release хранение долговременнее CI — задача DEV-04 в [плане](implementation-plan.md).

## Как выбирать регрессию

- Каждый завершённый code slice: базовый check и относящиеся негативные/concurrency/fault tests; shared DB/guards/migrations требуют PG suite.
- UI: пять browser profiles, keyboard/error/locale/theme/mobile и визуальный review изменённых baselines.
- Storage/crypto/recovery/network: профильный integration/fuzz/drill и соответствующие security scans; изменение runtime images проверяется по фактическому image ID.
- Перенос документации с правкой путей generators/CLI: docs/traceability/contracts, check и фактическая сборка bundle. Новый PG/load/DR результат не заявлять без запуска; полная release квалификация всё равно требует 13 свежих gates.

Обязательные проверки [AGENTS](../AGENTS.md) и CI не отключать. Дополнительную регрессию расширять по новым изменениям/ошибкам, а не повторять бесконечно после PASS. Для документационного переноса выбранный набор и его результат записаны в [session](session.md).

## Почему release-check ожидаемо blocked

Нужны 13 matching gates моложе 24 часов, полная browser matrix, большой performance profile, clean Git, все 32 verified MUST и все 14 реальных внешних решений с reviewer/evidence. Сейчас окружение не подготовлено и эти решения pending. Dirty/stale/missing evidence — дополнительные технические причины. `readiness.json` и nonzero exit являются ожидаемым отказом выпуска; их нельзя скрывать фиктивным accepted. Проверка полноты не является подписью reviewer и не переключает APP_ENV.

## Готовность публичной формы и независимый recovery fixture

Повторный main run [38167045971](https://github.com/elnemir/p2p-files/actions/runs/38167045971) на исходниках объединения обнаружил гонку `/verify#report=<uuid>` в WebKit. SSR отдавал активную кнопку и пустой required input, а client effect подставлял ID позже. В trace снимок перед кликом содержит пустое поле; API-запрос `/verify/reports/...` отсутствует, следующий снимок уже имеет ID. Это потерянный ранний клик, а не медленный ответ backend. Flow остановился до создания CSV; recovery-сценарий зависел от этого CSV и упал вторично.

Регрессия в [flow.spec.ts](../scripts/e2e/flow.spec.ts) задерживает загрузку JavaScript, открывает страницу до hydration и проверяет disabled input/button. После разрешения scripts она ждёт ID из fragment и проверяет, что один клик доходит до настоящего public API (404 для заведомо отсутствующего UUID). Позитивный сценарий с реальным report JSON и hash остаётся отдельной частью того же browser pack. FileX связывает интерактивность с существующим loaded state; пустой ID также не разрешает submit. Никаких повторных кликов, retries или увеличенных timeout для обхода ошибки.

Recovery test самостоятельно создаёт CSV обычного USER до backup, закрывает отдельный USER browser context после drain observer и затем в независимом ADMIN context проходит restore/reconcile/reopen/download/TTL. Это сохраняет проверку административного доступа к чужому export и убирает зависимость от успеха другого теста. На этом историческом кандидате браузерный набор содержал 25 сценариев на пяти профилях; case с задержанным JavaScript встроен в существующий public-pages test.

Изменение готовности формы не меняет API, privacy/reauth/crypto или server lifecycle. Fragment остаётся клиентским входом, затем явное действие запрашивает metadata отчёта; файл для сравнения читается локально браузером. Без JavaScript форма не должна принимать действия, которые обработчик ещё не может выполнить. Новые результаты записываются по новому source hash; PASS исходного PR #16 не зачтён исправлению автоматически.

До исправления новый delayed-hydration test воспроизвёл enabled input в отдельном WebKit на прежнем web image (контрольный RED). Первый Compose-контроль не дошёл до браузера из-за unexpected EOF у Go proxy; он не считается результатом теста. Recovery fixture ждёт completed именно нового export ID и после restore выбирает в admin list именно эту запись; наличие CSV от другого сценария не требуется.

В [PR run 38167905201](https://github.com/elnemir/p2p-files/actions/runs/38167905201) проявилась другая гонка: contract observer оставлял requestfinished listener после finish, а новые response promises могли отклониться уже при закрытии контекста. В том исправлении finish сначала синхронно снимал listeners, затем дожидался всех ранее принятых ответов. Текущая реализация после обнаружения navigation race описана ниже. Rejection handler присоединяется сразу; ошибка capture или JSON остаётся failure, а не игнорируется. Static resources и artifact content отфильтровываются до обращения к response. Тела/секреты ошибок не записываются. Четыре Node fault/lifecycle tests входят в make check через test:contracts; все четыре были RED на прежнем helper и PASS после исправления. На этом историческом кандидате browser matrix содержала 25 cases, retries=0 и прежние timeout.

Traceability: FR-012 связан с delayed-hydration public-page case; OPS-003 сохраняет recovery case; NFR-003 включает lifecycle/fault tests наблюдателя. Env/config/migrations/contracts API не менялись. На новом source обязательные gates выполняются заново; результат фиксируется в session.

Первый прогон новой квалификации выявил в подготовке recovery fixture потерю response body при logout/navigation. USER и ADMIN теперь используют отдельные browser contexts: USER создаёт export → observer завершает захват/валидацию → context закрывается → ADMIN работает в собственной сессии. Payload ответа считывается до компиляции schema, уменьшая окно потери browser resource. Capture failures не подавляются; test проверяет все принятые observer ответы и точный export ID после восстановления. Неуспешный прогон source 11a18ce7 сохранён отдельно; его первые пять PASS не переносятся автоматически на новый hash.

Итог локального исправления: все 13 gates PASS на source `8390f72c6ac08927e609dc9eccfb78d93ebc7fbbdae8363c47e449cf50feb210`; 25/25 browser, 8/8 Node contract/lifecycle tests, без retries/baseline updates. Точные raw records — `.cache/consolidation/hydration-gates/`; GitHub CI проверяется отдельно на commit этого source.

Код доставлен через PR #17 в main `548359d9d722641a5a66acd53afd8702d98876c6`: [development CI 38172130775](https://github.com/elnemir/p2p-files/actions/runs/38172130775) подтвердил все 13 matching gates для merge ref `83cd5aa67fd44f8b6ec60a5b41f57362641c99e8`, [security CI 38172130794](https://github.com/elnemir/p2p-files/actions/runs/38172130794) — четыре CodeQL категории без findings/errors/warnings. Исторические main/PR failures остаются evidence расследования, успешный новый run относится только к исправленному source 8390f72c.

## Подтверждённый draft перед backup и publish

Контекст: `createDraft` в `scripts/e2e/transfers.ts` — общий UI fixture для upload/publish и recovery; он не изменяет продуктовый API. В [PR #18 run 38174057287](https://github.com/elnemir/p2p-files/actions/runs/38174057287) mobile WebKit после `fill` отправил PATCH с пустым названием. Снимок перед Save уже содержит пустое поле; восстановленная запись имеет исходный ID и то же пустое название. Следовательно, этот failure не доказывает потерю данных при restore. Точный механизм единичного сбоя ввода не установлен: 35 локальных повторов с наблюдением focus/input/change сохранили текст. Изменять product state на основании неподтверждённой гипотезы не потребовалось.

Поток fixture теперь явный: UI создаёт draft → POST 201 даёт ID → фокус и последовательный клавиатурный ввод → DOM содержит точный текст → Save → PATCH того же ID содержит точный title и возвращает 204 → отдельный GET возвращает ID/title/draft → заголовок страницы подтверждает обновление. Например, для «Сохранённая передача до восстановления» резервная копия создаётся только после этих проверок. После restore проверяются исходный ID и title, а поздний ID должен отсутствовать; совпадение чужого названия больше не может подменить результат.

Клавиатурный ввод заменяет одноразовую вставку `fill` только в этом fixture. Произвольные sleeps, повторные заполнения, retries и увеличение timeout не добавлены. Любая потеря ввода останавливает сценарий до backup/publish с точной стадией ошибки. Контракты API, env, миграции и runtime UI не изменены; ограничения внешней приёмки прежние. Проверки fixture входят в существующие 25 случаев на пяти browser profiles. Исторический PASS source `8390f72c` не переносится на изменённые тесты; новая квалификация записывается в [session](session.md).

Локальная квалификация fixture завершена: source `2a8380c6a3d34503a7117e842c46a0c2bb049108ff2bb8369e449146f852b8ab`, все 13 gates PASS; browser reports каждого из пяти профилей имеют expected=5, unexpected/flaky/skipped=0. Smoke performance 32 MiB / 2000 rows, не release-size. CI проверяется отдельно на точном head/merge ref PR #18.

## Проверки совместимости зависимостей

`repository-policy` CI и `make check` запускают `npm run check:dependencies` через repository checker и `npm run test:toolchain`. Это full installed peer tree + согласованность engine/config/types/образов + 57 behavioral/negative tests. Подробные причины, PR #9 с `npm ci` exit 0 / `npm ls` exit 1, границы API/CLI TypeScript и миграция 31 a11y правила — в [dependency remediation](dependency-remediation.md). Clean install на Node 26 обязателен; green lint на системном Node другого major не считается эквивалентной квалификацией. [ADR-012](adr/012-node26-runtime.md) определяет Current/LTS policy на дату review и проверку EOL. Host test driver и web application квалифицируются на Node 26; Node внутри закреплённого Playwright browser-server image — отдельная инструментальная зависимость. Existing upgrade drill заменяет Go API/worker baseline/schema 16→17 при текущем web image; он не измеряет отдельное переключение работающего Next.js с Node 24 на 26. Browser/ingress gates проверяют новое web runtime непосредственно.

## Добавление самостоятельных схем и WebDAV контракта

Compatibility comparator по-прежнему запрещает удаление/изменение существующих baseline operations, schemas и security schemes. Новый credential JSON Schema является добавлением: отсутствующий в старом baseline новый файл проверяется как новый документ с dialect 2020-12 и уникальным HTTPS `$id`; отсутствие существующего baseline не пропускается. Новые security-scheme definitions и корневой `x-webdav` разрешены как additions; существующий `x-webdav` после включения в reviewed baseline сравнивается рекурсивно. Шесть unit cases проверяют в том числе duplicate `$id`, неверный dialect, удаление/изменение auth и изменение существующего protocol profile. `make contracts` теперь компилирует все standalone schemas через strict Ajv помимо OpenAPI/AsyncAPI validation. Не требуется автоматическая смена baseline или waiver ради новой схемы.

## Фактические native WebDAV clients
`FILEX_NATIVE_WEBDAV_CLIENT=macos|linux python3 scripts/test-integration.py TestNativeWebDAVClients` запускается отдельно: настоящий mount_webdav либо GVfs/D-Bus, server-created GET, multi-chunk bytes/hash, replace/move/delete и unmount. Linux image собирается из scripts/webdav/Dockerfile, версии backend фиксируются в output. Серверные tests не выдаются за проверку реального клиента. Windows PowerShell scenario подготовлен, запуск **ожидает стенда**. Команды, TLS prerequisites и матрица границ — [WebDAV](webdav-upload.md).


## Захват контрактов при навигации: 2026-10-12
**Контекст/проблема:** requestfinished observer мог получить Chromium `Network.getResponseBody: No resource with given identifier found` при одновременных notification poll и переходе на `/verify`. Успешная загрузка HTTP-ответа не гарантирует сохранность его тела после уничтожения документа. Диагностика trace зафиксирована в этой записи; ошибка не была исключена из assertions.

**Решение/data flow:** `contractObserver` регистрирует async route для API: запрос браузера → один настоящий upstream `route.fetch` без retries/redirect-following → JSON schema validation → `route.fulfill` с исходными bytes/status/headers → dispose. APIResponse владеет телом независимо от старого документа. Payload, метод, заголовки и cookies не подменяются, mutations не повторяются. Capture/JSON/schema ошибки остаются failure. `finish` закрывает регистрацию и дожидается уже начатых jobs; контекст закрывается после проверки. Fault injection PATCH передаёт обычные requests через fallback, а преднамеренный connection reset сохраняется.

**Проверки/ограничения:** шесть Node cases проверяют однократность, lifecycle, отказ транспорта, invalid/undocumented JSON, binary bypass и passthrough non-JSON. Новый browser case использует отдельный локальный HTTP fixture: удерживает ответ, полностью меняет документ, затем отдаёт JSON и требует count=1/requests=1. Это проверка harness, а не подмена FileX API. Общая матрица теперь 35 cases. Бинарные tus `/uploads/{id}`, загрузка контента `/content`, HTML, WebSocket и native WebDAV идут своим транспортом; JSON API в этих E2E проходит прозрачный тестовый proxy и не служит измерением browser-to-server latency. Протокольная безопасность/ошибки HTTP дополнительно проверяются integration/ingress. Основа API — [Playwright Route](https://playwright.dev/docs/api/class-route#route-fetch).

Binary tus специально не проходит `route.fetch`: WebKit не обязан предоставлять исходные File/Blob bytes этому Node API. Это проявилось нулевым прогрессом повторных PATCH при 204 в trace. Транспорт остаётся браузерным, и сценарий connection reset → HEAD → resume → publish/ZIP по-прежнему обязателен. JSON ошибки tus покрывает HTTP integration; этот proxy не заявляет наблюдение binary endpoints.

TLS ingress fixture актуализирован под ADR-011: старое ожидание GET=403 upload-only заменено проверкой owner draft GET=200 с точными bytes, HEAD без body, Range=206 с точным срезом, PUT overwrite=204 и последующим чтением новых bytes. Отзыв credential и остальные TLS/network negative assertions сохраняются.

## Невоспроизведённая задержка WebKit в предварительной квалификации
На source `768864c97a397db92a18a908948b577e4b3bc32fb7a3388e6110843aceafab18` один предварительный operations E2E завершился по штатному 5 s ожиданию owner UI. Trace: POST auth/login=200 (~160 ms), следующий upstream GET auth/me не завершился до закрытия контекста, остальные 6 cases WebKit PASS. Это не ошибка password/schema и не доказанный дефект WebDAV; первопричина задержки не установлена. Private trace сохранён в `.cache/webdav-native-prerun/webkit-auth-me-pending.zip` (синтетические credentials, не публиковать). Проверка повторяется на чистом disposable окружении без изменения кода, таймаутов, retries или assertions. Итог отдельного полного прогона — в session; PASS повтора не объявляется доказательством устранения первопричины этой задержки.

## Idle worker / reopen при Node 26 qualification
Первичный mobile-WebKit trace получил reopen=409 при свежем passed report и завершённых jobs; следующий login=503 — результат оставшегося закрытого gate. PG regression использует SHARE lock на ops_jobs: разрешает чтение, задерживает пустой UPDATE и воспроизводит удержание runner lock idle worker. До изменения тест RED, после предварительного чтения queued/running — GREEN; второй тест сохраняет queued/crash/generation/explicit-reopen guards. [Подробный поток и ограничения вывода](operations-recovery.md). Browser assertions, retries=0 и timeout не ослаблялись; итоговый source требует полной матрицы заново.

## Брендинг

`TestIntegrationBranding*` проверяют права/CSRF/reauth, CAS гонку, audit fault rollback, immutable report snapshot и SMTP. Recovery и audit checkpoint cases дополнительно проверяют branding restore и snapshot инцидента. `branding.spec.ts` покрывает preview/apply/reset/conflict, assets, SSR без JavaScript, публичные страницы и 320px/axe. Подробности — [white labeling](white-label.md); результаты текущего diff — [session](session.md).

Conditional `GET /api/v1/branding` с `If-None-Match` проходит native HTTP в browser harness: WebKit не поддерживает `route.fulfill(304)`. Новый branding сценарий явно сверяет ETag, 304/пустое тело и 200/полный DTO при stale validator. Первичное чтение по-прежнему валидируется contract observer. Unit regression запрещает распространение исключения на другие routes или mutations; запросы не повторяются и статус не подменяется.

## Federated identity regression

[SSO](federated-auth.md) проверяется на подписанных локальных OIDC/SAML fixtures и настоящем TLS/StartTLS LDAP BER обмене. PG/race тесты проверяют JIT USER, отсутствие email auto-link, 16 одновременных завершений одной flow, expiry, actor revoke, provider CAS/revoke, audit rollback, восстановление и ротацию configuration DEK. Browser `identity.spec.ts` проходит реальный SP→IdP form POST→ACS→browser-bound completion, регистрацию, повторную авторизацию, replay rejection и ADMIN disable. Подменяется только внешний тестовый IdP, API FileX работает настоящий. Проверки не подтверждают конфигурацию конкретной организации Keycloak, Entra ID, Google Workspace или AD.

## UX USER/ADMIN

`product-ux.spec.ts` использует настоящий Compose API и три сценария на каждом из пяти профилей:
- Название/message, отмена удаления без DELETE, пресеты/клавиатура ползунка/PIN, сброс review при чужом изменении файла, реальный ZIP → terminal. Сетевая ошибка списка и отмена клиентом автоматического старта до content GET сохраняют доступ/URL продолжения; удержанный старый ответ не возвращает закрытые controls. Проверяются реальный revoke и естественное истечение короткого TTL; новый token в адресе не подменяется прежней cookie-сессией.
- Два POST-поиска с управляемым порядком ответов, empty/reset, timezone/quiet hours через настоящий runtime image, сохранение выбора, неверный пароль reauth без logout и завершение owner session после server revoke прямо в настройках.
- В том же сценарии настоящий editor response удерживается после server read; сессия отзывается до доставки, выполняется вход другим аккаунтом без navigation/reload, затем старый ответ отпускается. Private state старого действия не должен заполнить кабинет нового пользователя.
- Полный цикл Settings draft/validate/explicit apply, CAS против параллельного apply, сохранность network policy и ввода, invalid expert JSON, TTL установки 90 секунд → ограниченная форма → publish с точным TTL. Исходные настройки восстанавливаются в cleanup.

Axe WCAG 2/2.1/2.2 AA и reflow проверяются на draft/recipient/preferences/ADMIN, в том числе 320 px. Автоматические проверки дополняются просмотром снимков: отсутствие document overflow само по себе не доказывает, что длинная подпись помещается внутри кнопки. Baseline update служит только подготовке reviewed изображений; gate запускается отдельно без update.

Branding и keyboard negative-login E2E сначала дожидаются реального HTTP результата и явно сверяют 200/409/401, затем проверяют состояние UI. Contract capture может добавлять задержку перед доставкой ответа; ожидание server mutation не смешивается с пятисекундным ожиданием отображения. Общие timeout/retries и все DOM/contract assertions сохранены. Эти функциональные tests не заменяют отдельный performance gate.


## PR #26: подтверждённое сохранение quiet hours

**Контекст/проблема:** [CI 38462541585](https://github.com/elnemir/p2p-files/actions/runs/38462541585) на head `ac084c2` / merge ref `01ad482` прошёл 59/60 browser cases. Mobile WebKit обнаружил гонку самого сценария: после click тест сразу отправлял GET, который завершился до PUT=204 и прочитал прежние значения. Trace: PUT с `quietStart/quietEnd=null` начат 20:53:04.440Z, длился 49.527 ms; GET начат 20:53:04.463Z, длился 3.912 ms. Это не доказательство потери данных приложением: PUT передал правильные значения и успешно завершился.

**Решение/data flow:** для включения и отключения quiet hours тест регистрирует ожидание PUT → нажимает Save → проверяет HTTP 204 → проверяет сообщение UI → читает сохранённые значения. Нет sleeps, retries, нового timeout или ослабления assertions; runtime/API/contracts не меняются. Новый source SHA-256 `dbdaffeade8e42cf09bc23f58015b74f90ddc54258bf75a980d5353cde21b396` получает повторную квалификацию; старые PASS не переносятся автоматически. Диагностика сохранена в `.cache/product-ux/delivery/pr-functional-failed/` и `pr-failed.log`, прежние gate records — в `.cache/product-ux/delivery/local-gates/`. Синтетические session traces остаются приватным evidence.
