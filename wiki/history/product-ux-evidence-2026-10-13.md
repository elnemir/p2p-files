# Проверка удобства USER/ADMIN — 2026-10-13

## Контекст и граница
Проверяется локальный UX срез [ADR-015](../adr/015-product-ux.md) в `codex/product-ux`, поверх `c221369a6dc5d795c836d11ccdaeb39ee97ca3e9`. Обзор до/после, потоки и снимки — [product UX](../product-ux.md). Изменены интерфейс, read-only editor endpoint, встраивание стандартной IANA database, контракты и тесты. Новых зависимостей, migrations и env нет.

Source SHA-256: `509dd43094935c4761e1f398b783361a1f527cb0cc497494c0ff7a48086da93a`. `scripts/acceptance.py` включает код, contracts, тесты и изображения; исключает Markdown wiki/README/changelog, чтобы запись измерений не меняла измеряемый кандидат. Working tree ещё не закоммичен. Это локальная квалификация изменений, не release acceptance, CI или production deployment.

## Окружение и воспроизведение
Host macOS, Go 1.27.2, Node 26.11.1 из локального toolchain. Браузеры работают в закреплённом Linux Playwright image; каждый профиль получает отдельный disposable Compose с настоящим API/worker/PostgreSQL. PG integration использует отдельный временный контейнер, rclone задан через `FILEX_RCLONE_PATH`. `caffeinate -i` предотвращает idle sleep только во время процесса проверки, системные настройки не меняет.

```sh
export PATH="$PWD/.cache/node26-review/node-v26.11.1-darwin-arm64/bin:$PATH"
export FILEX_RCLONE_PATH="$PWD/.cache/webdav-tools/rclone"
caffeinate -i python3 scripts/acceptance.py run check
caffeinate -i python3 scripts/acceptance.py run integration
caffeinate -i python3 scripts/acceptance.py run dependencies
caffeinate -i python3 scripts/acceptance.py run source
caffeinate -i python3 scripts/acceptance.py run compatibility
caffeinate -i python3 scripts/acceptance.py run images
caffeinate -i python3 scripts/acceptance.py run browser
caffeinate -i python3 scripts/acceptance.py run licenses
caffeinate -i python3 scripts/acceptance.py run fuzz
caffeinate -i python3 scripts/acceptance.py run ingress
caffeinate -i python3 scripts/acceptance.py run upgrade
caffeinate -i python3 scripts/acceptance.py run performance
caffeinate -i python3 scripts/acceptance.py run verifier
```

## Результаты
Все 13 gates PASS на указанном неизменном source: перед разрешёнными commit/push/merge дополнительно выполнены licenses/fuzz/ingress/upgrade/performance/verifier. Финальный PG/race выполнен отдельно от браузеров и прошёл 62/62 без skips; полный browser pack — 60/60. Записи каждого gate находятся в `.cache/evidence/gates/<name>.json`, логи — `.cache/product-ux/gate-<name>.log`. PASS ниже означает exit 0 и `sourceUnchanged=true` для указанного source, а не исторический результат другой ветки.

| Gate | Статус | Проверяемое поведение |
|---|---|---|
| check | PASS | Go vet/staticcheck/race, TypeScript/ESLint/format, контрактные и toolchain tests, docs, generated types и production build |
| integration | PASS: 62/62, без skips | Полный PG/race набор, owner/scope/editor compatibility и существующие серверные инварианты |
| dependencies | PASS | npm audit 0; govulncheck 0 affected symbols/imported packages |
| source | PASS | Gitleaks history/worktree без секретов; Trivy HIGH/CRITICAL=0 |
| compatibility | PASS | Прежние strict DTO/operations сохранены; новый editor DTO разрешён как addition |
| images | PASS | Пять runtime images и SBOM; HIGH/CRITICAL=0 во всех пяти |
| browser | PASS: 60/60 | Полная матрица пяти профилей, без snapshot update и retries |
| licenses | PASS | 434 npm packages, 37 Go libraries; 16 позиций требуют owner review, inventory не означает принятия |
| fuzz | PASS | Пять полноценных кампаний по 15 секунд: path, FXP1, Range, WebDAV path/XML/If |
| ingress | PASS | Реальный TLS/test CA/Traefik, network/Host/cookie/WebDAV/fencing; без public ACME |
| upgrade | PASS | Разные baseline/candidate images, verified backup, schema 16→19, startup faults, явный reopen и one-time |
| performance | PASS | Локальный smoke 32 MiB / 2000 rows, fault/recovery и ZIP; не release-size |
| verifier | PASS | Пять cross-build targets и checksums; без публикации |

Полная browser matrix завершилась одним непрерывным run без `FILEX_E2E_PROJECTS` и `FILEX_UPDATE_VISUALS`: Chromium 12/12, Firefox 12/12, WebKit 12/12, mobile Chromium 12/12, mobile WebKit 12/12. В каждом JSON report `unexpected=flaky=skipped=0`; retries=0. Длительность gate 864.5 s, timestamps всех пяти reports находятся внутри этого run. Это pinned Linux engines и мобильная эмуляция, не приёмка реальных Safari/iOS/Android устройств.

Длительности остальных gates: check 60.2 s; integration 197.8 s; dependencies 6.6 s; source 6.0 s; compatibility 0.1 s; images 52.2 s; licenses 3.8 s; fuzz 88.5 s; ingress 19.8 s; upgrade 56.2 s; performance 38.0 s; verifier 5.3 s. После записи итоговой wiki отдельно повторены docs/link check и `git diff --check`. GitHub configuration audit `20261013T201841.458196Z` подтвердил `matchesPolicy=true`, `differences=[]`; настройки защиты не изменялись.

## Состав проверенных образов

| Image | Локальный immutable image ID |
|---|---|
| api | `sha256:43ab8d27fd22df01234acf3cde6195368ed46797638247555a58b6c60fc48970` |
| web | `sha256:ec1b7eda1294dd11a1b58a21ecf45271ab38b771e63c99bcdef5bffdce4316cf` |
| postgres | `sha256:87078dab1173fbea3fc541dd7a821160c32dde940458b746d08b20d331830ccd` |
| redis | `sha256:38272b23d66afa993187a836865b7b629ab4845c7c9726121eec83c8b2c0b44b` |
| traefik | `sha256:39fa4cd0a446673c6be8dcba4c2a79d767d7e486defdca0d00eeb34a3be69766` |

## Что проверяется в новом UX
- Название сохраняется перед review вместе с прежним message; изменение файла другой сессией закрывает review. Удаление можно отменить Escape без DELETE.
- Presets/slider/точное число используют действующий TTL policy; отдельный сценарий применяет 90 секунд и проверяет настоящий publish с этим значением.
- Запоздалый POST search не подменяет новый фильтр; timezone/quiet hours работают в минимальном runtime image. Неверный пароль reauth оставляет действующую сессию, отзыв cookie убирает приватный интерфейс прямо из настроек.
- Settings v1 проходит draft → validate → explicit CAS apply. Параллельный apply не теряет ввод; невалидный expert JSON остаётся доступен для исправления; неизменённая network policy не вызывает ложное предупреждение о закрытии сервиса.
- Recipient получает настоящий ZIP, после полного consume лишается старых controls; revoke и natural expiry дают тот же privacy-safe экран. Новый token имеет приоритет над старой cookie. Сетевая ошибка списка и отмена клиентом запуска до content GET сохраняют доступ и resume. Это не тест обрыва посередине TCP stream: Range/interrupted-stream/atomic ZIP инварианты отдельно проверяются PG integration.
- Удержанный старый ответ после terminal не возвращает прежние файлы. Axe и reflow проверяются на 320/390 px; 10 RU-light/EN-dark baseline PNG просмотрены до запуска gate без update.

## Найденные проблемы и исправления
Подготовительные browser runs обнаружили составные accessible labels, обрезание длинной ADMIN подписи, отсутствие IANA database в Alpine и подмену новой ссылки существующей recipient cookie. Исправлены поля/мобильная сетка, добавлен стандартный `time/tzdata` в API/worker/CLI и generation-safe fragment navigation. Compatibility gate обнаружил, что добавление полей к прежнему strict `Transfer` недопустимо: вместо ослабления gate введён самостоятельный `/editor` и сохранён прежний DTO.

Один подготовительный Firefox recovery run остановился на reconciliation; закрытый стенд вызвал последующие 503. Диагностика сохранена в `.cache/product-ux/matrix-first`; точная причина задержки не установлена. Новый отдельный подготовительный Firefox pack прошёл полностью, без исключения assertions или увеличения recovery timeout. Этот повтор не доказывает устранение причины единичной задержки; итоговая матрица записывается отдельно.

Первоначальная попытка прервать native download через Playwright route не перехватывала browser download и завершилась отказом теста. Тест теперь явно отменяет одну клиентскую навигацию до content GET и затем возобновляет настоящую серверную reservation через UI; эта граница не выдаётся за транспортный fault.

Дополнительный Chromium run: 11/12, единственный отказ — strict locator нашёл одновременно alert приложения и служебный Next.js route announcer. Проверка сообщения ограничена `main`; assertion, реальный HTTP 401 и проверка сохранённой сессии остаются. На предшествующем source `7dd71675` PG/race прошёл 62/62 без skips. После изменения селектора все выбранные gates выполняются заново на текущем source; старые записи сохранены в `.cache/product-ux/pre-final-selector`.

Итоговый source `76de9fc2` прошёл все 62 PG/race tests и все новые Chromium UX cases. На первом browser run этого source отдельный keyboard/error-recovery case не дождался ошибочного POST login за штатные 5 секунд: trace показывает запрос без ответа до закрытия контекста, UI оставался «Входим…». Поля и отправка были корректны. В это время параллельно выполнялся PG/race; точная причина задержки не доказана. Диагностика — `.cache/product-ux/parallel-login-delay`. Полная browser matrix на этом предшествующем source повторялась на чистых стендах без других тяжёлых проверок, без изменения source/timeouts/retries/assertions; последующие изменения и окончательный результат описаны отдельно.

Финальное review расширило session generation guard с фонового refresh на общий owner API adapter: поздний ответ ручного открытия тоже не должен восстановить прежние данные после смены аккаунта. Browser scenario дополнен удержанным editor response и входом другим аккаунтом без reload. Перед этой правкой отдельный Chromium pack 12/12 прошёл, Firefox runner остановлен намеренно для изменения исходников; partial matrix не засчитывается. Предшествующие записи сохранены в `.cache/product-ux/pre-session-response-guard`. Окончательный source получил новый полный набор выбранных gates; результаты приведены выше.

Регрессия смены аккаунта обнаружила оставшийся `busy` от старого действия: сессия уже закрыта, но новая форма входа ждёт старый запрос. Добавлен action generation, сброс busy при session end и запрет старому finally завершать новое действие. Отдельный Chromium regression подтвердил сценарий: другой аккаунт входит до доставки старого editor response, прежняя передача не появляется.

В том же debug pack branding PUT вернул 200, но ответ с contract observer пришёл через 6–8 секунд и прежний UI expect истёк раньше. Это суммарное время browser/proxy/validation/server, не измерение API latency. В branding и keyboard negative-login tests теперь сначала явно проверяется настоящий HTTP результат (200/409/401), затем прежние DOM assertions. Общие test timeout, UI expect timeout и retries=0 сохранены; API failure не может маскироваться оптимистичным UI. Debug pack 11/12 не считается итоговой матрицей; диагностика — `.cache/product-ux/session-guard-branding-delay`.

## Ограничения и следующие шаги
Сканеры сохраняют прежний UNKNOWN `GO-2026-5932` для `x/crypto/openpgp`, без исключений. `go list -deps ./cmd/...` не содержит openpgp; это согласуется с govulncheck FileX, но не является binary reachability review стороннего Traefik.

Обновлены четыре UX снимка в wiki; прежний озвученный MP4 остаётся исторической записью от 2026-10-12. Реальные устройства/assistive technology, Windows WebDAV, конкретные SSO tenants и внешняя инфраструктура не квалифицированы этой задачей. Все прежние внешние blockers и production guard сохранены. Все 13 локальных gates проверены для текущего source; GitHub CI проверяется отдельно на PR и main. Полный release profile, решения владельца и внешняя приёмка из этого результата не следуют.
