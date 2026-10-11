# Проверки, CI и воспроизводимое evidence

## Контекст

Проверка подтверждает конкретный commit/source hash и профиль. `make check` не запускает PostgreSQL integration, browser, scanners или полный load. Общий зелёный build не закрывает внешнюю приёмку. Источники команд — [Makefile](../Makefile), [acceptance.py](../scripts/acceptance.py), [CI](../.github/workflows/ci.yml).

## Базовые и целевые команды

| Команда | Что проверяет | Требования / ограничение |
|---|---|---|
| make check | gofmt/vet/staticcheck, TS strict/ESLint/Prettier, Go race, contract/generator/Python negative tests, docs/traceability, Go/Next builds | Go/Node/Python; без тестового DB URL PG tests пропускаются |
| make integration-docker | Все TestIntegration* с race на disposable PostgreSQL; сейчас 36 top-level tests | Docker; каждый test получает отдельную schema, cleanup своего container |
| make test-integration | Те же тесты через FILEX_TEST_DATABASE_URL | Только специально выделенная тестовая БД; URL обязателен |
| make security | npm audit и govulncheck | Network; source reachable findings отдельно от module-only advisories |
| make e2e | 5 Linux browser/emulation profiles × 5 scenarios = 25; contracts, axe/keyboard/responsive/visual | Docker, npm ci, свободный 8090; не реальные устройства |
| make ingress | Настоящий Traefik/test CA, TLS, Host/forwarded/CIDR/secure cookie/reset/private endpoints | Isolated project/random ports, public ACME не вызывается |
| make upgrade-drill | Разные images baseline 50de642/candidate, backup/drain/fault/reopen/one-time | Git history; Compose same-schema 16; реальный SQL prefix upgrade проверяется PG suite |
| make fuzz | Три кампании PathKey/FXP1/Range | Default 15 s на кампанию плюс compile; Go seeds/unit сами по себе не fuzz campaign |
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

Каждый push в main, PR или workflow_dispatch запускает functional и supply-chain jobs на ubuntu-latest, timeout 45 min каждый, а также repository-policy (10 min). Для functional/supply-chain checkout содержит полную историю, Node 24 из .nvmrc, Go из go.mod, npm ci; deployment отсутствует. Эти два jobs сохраняют evidence при успехе/ошибке, retention 14 дней. Security workflow выполняет CodeQL для Go/JS/TS/Python/Actions на PR/main/manual и раз в неделю, dependency review — на PR; aggregate security-checks обязателен. Проверять conclusion всех required checks и **exact headSha**, а не предыдущего commit. CI performance compact; полный профиль нужен отдельно. [Настройки GitHub и правила merge](repository-governance.md).

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

Recovery test самостоятельно создаёт CSV обычного USER до backup, закрывает отдельный USER browser context после drain observer и затем в независимом ADMIN context проходит restore/reconcile/reopen/download/TTL. Это сохраняет проверку административного доступа к чужому export и убирает зависимость от успеха другого теста. Браузерный набор по-прежнему содержит 25 сценариев на пяти профилях; case с задержанным JavaScript встроен в существующий public-pages test.

Изменение готовности формы не меняет API, privacy/reauth/crypto или server lifecycle. Fragment остаётся клиентским входом, затем явное действие запрашивает metadata отчёта; файл для сравнения читается локально браузером. Без JavaScript форма не должна принимать действия, которые обработчик ещё не может выполнить. Новые результаты записываются по новому source hash; PASS исходного PR #16 не зачтён исправлению автоматически.

До исправления новый delayed-hydration test воспроизвёл enabled input в отдельном WebKit на прежнем web image (контрольный RED). Первый Compose-контроль не дошёл до браузера из-за unexpected EOF у Go proxy; он не считается результатом теста. Recovery fixture ждёт completed именно нового export ID и после restore выбирает в admin list именно эту запись; наличие CSV от другого сценария не требуется.

В [PR run 38167905201](https://github.com/elnemir/p2p-files/actions/runs/38167905201) проявилась другая гонка: contract observer оставлял requestfinished listener после finish, а новые response promises могли отклониться уже при закрытии контекста. Теперь finish сначала синхронно снимает listeners, затем дожидается всех ранее принятых ответов. Rejection handler присоединяется сразу; ошибка capture или JSON остаётся failure, а не игнорируется. Static resources и artifact content отфильтровываются до обращения к response. Тела/секреты ошибок не записываются. Четыре Node fault/lifecycle tests входят в make check через test:contracts; все четыре были RED на прежнем helper и PASS после исправления. Browser matrix остаётся 25 cases, retries=0 и прежние timeout.

Traceability: FR-012 связан с delayed-hydration public-page case; OPS-003 сохраняет recovery case; NFR-003 включает lifecycle/fault tests наблюдателя. Env/config/migrations/contracts API не менялись. На новом source обязательные gates выполняются заново; результат фиксируется в session.

Первый прогон новой квалификации выявил в подготовке recovery fixture потерю response body при logout/navigation. USER и ADMIN теперь используют отдельные browser contexts: USER создаёт export → observer завершает захват/валидацию → context закрывается → ADMIN работает в собственной сессии. Payload ответа считывается до компиляции schema, уменьшая окно потери browser resource. Capture failures не подавляются; test проверяет все принятые observer ответы и точный export ID после восстановления. Неуспешный прогон source 11a18ce7 сохранён отдельно; его первые пять PASS не переносятся автоматически на новый hash.

Итог локального исправления: все 13 gates PASS на source `8390f72c6ac08927e609dc9eccfb78d93ebc7fbbdae8363c47e449cf50feb210`; 25/25 browser, 8/8 Node contract/lifecycle tests, без retries/baseline updates. Точные raw records — `.cache/consolidation/hydration-gates/`; GitHub CI проверяется отдельно на commit этого source.

Код доставлен через PR #17 в main `548359d9d722641a5a66acd53afd8702d98876c6`: [development CI 38172130775](https://github.com/elnemir/p2p-files/actions/runs/38172130775) подтвердил все 13 matching gates для merge ref `83cd5aa67fd44f8b6ec60a5b41f57362641c99e8`, [security CI 38172130794](https://github.com/elnemir/p2p-files/actions/runs/38172130794) — четыре CodeQL категории без findings/errors/warnings. Исторические main/PR failures остаются evidence расследования, успешный новый run относится только к исправленному source 8390f72c.
