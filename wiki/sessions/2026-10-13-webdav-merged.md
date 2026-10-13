# История завершения native WebDAV и туров

Это сохранённая запись до финального merge; формулировки о незавершённых PR ниже относятся к тому моменту. Итог 2026-10-13: [PR #23](https://github.com/elnemir/p2p-files/pull/23) merged, squash `1ce65ab5b15d21adf434fb7f5e1b72567e322938`. PR head `dfd541584e79ce1563982cdb9785dfbb783b0109`; development run `38267535217`, security run `38267535211` SUCCESS. Все 13 скачанных CI records PASS/unchanged на source `016a9793d2c8010bf877a1436065e19d747e519d18c5d56aaf1036578579bbe1`; четыре CodeQL анализа merge ref без findings/error/warning. Finding #12 устранён новым анализом без dismiss/suppression. Состояние main CI и продолжение Node 26 — в [текущей сессии](../session.md).

# Session State

## Task
Завершить поддержку штатных WebDAV-клиентов macOS/Linux/Windows, актуализировать wiki, фото- и видеотуры. Windows по прямому решению владельца **ожидает стенда**. Ветка `codex/webdav-upload`, база `28ab4f36e15a0b5e052b23e0f058bdda7a1c9d12`; исходный незакоммиченный срез сохранён: [upload inbox](../sessions/2026-10-12-webdav-inbox.md).

Продолжение 2026-10-12: владелец явно запросил commit и push завершённого среза в GitHub. Scope включает реализацию, contracts, тесты, wiki и обновлённые туры.

Текущий шаг по новой команде владельца: объединить завершённую WebDAV-ветку с main через PR и обязательные checks, затем синхронизировать локальный main и удалить только объединённую topic-ветку. При fetch обнаружен новый независимый Dependabot PR #22; его не включать с красными gates и несогласованным Node major.

## Status
Слияние [PR #23](https://github.com/elnemir/p2p-files/pull/23) выполняется. Первый CodeQL run завершил jobs, но обнаружил HIGH `go/incorrect-integer-conversion` ([alert #12](https://github.com/elnemir/p2p-files/security/code-scanning/12)) в `davTimeout`. Исходный `int(min(n,3600))` уже ограничивал значение; overflow не воспроизведён. Upper bound теперь выражен явным условием до conversion с тем же wire behavior. Добавлены 20 timeout boundary cases и traceability; новый source требует новых проверок, старые 13 PASS ниже — история предыдущего кандидата. Finding не dismiss/suppress; merge ждёт чистого повторного CodeQL. Локально `go test -race ./internal/httpapi -run '^TestWebDAV' -count=1` и полный `make check` PASS; новый source `016a9793d2c8010bf877a1436065e19d747e519d18c5d56aaf1036578579bbe1`, raw log `.cache/webdav-merge/check-timeout-bound.log`.

Реализация, wiki и туры завершены и отправлены в `origin/codex/webdav-upload`: [42d0e34](https://github.com/elnemir/p2p-files/commit/42d0e347f11310a9f7e600f8aef165f75cc83fd2), `feat(webdav): support native clients and refresh product tours`. Remote SHA подтверждён через `git ls-remote`. Все 13 локальных gates и реальные macOS/GVfs прошли на одном source; перед commit проверено совпадение всех 13 records с неизменёнными файлами, возраст менее часа. Windows ожидает стенда. Main/production не изменялись.

После первого push GitHub Actions не содержал запусков для `42d0e34`: feature push без PR не является CI-триггером. По новой команде готовится PR из `codex/webdav-upload` в `main`; CI и merge ещё не завершены. Исходный head доставки — `325286bcbfa51e0695df2bbec297232885e999f8`; base — `28ab4f36e15a0b5e052b23e0f058bdda7a1c9d12`. Runtime source после staging — `ab17d1f1a09a0cdc86bbcd8a7841587379299592103ddba68f6fc23019ea05a5`; последующие изменения этого шага пока только Markdown.

Инвентаризация веток: main, WebDAV и `dependabot/docker/deploy/node-26-alpine` ([PR #22](https://github.com/elnemir/p2p-files/pull/22), head `6bb2e372b32d585a07ab16e3bd967351a279f131`). В #22 меняются только два FROM на Node 26; `.nvmrc`, engine/types/policy остаются на 24. [CI 38260976979](https://github.com/elnemir/p2p-files/actions/runs/38260976979) имеет failed repository-policy/checks/supply-chain. Предложение остаётся открытым для отдельной согласованной квалификации; guard не отключается ради merge. Подробнее — [учёт объединения](../branch-consolidation.md).

Окончательный полный source SHA-256: `768864c97a397db92a18a908948b577e4b3bc32fb7a3388e6110843aceafab18`. Квалификация завершена 2026-10-12 17:39 MSK. Gate records: `.cache/evidence/gates`; логи: `.cache/webdav-native-final-*.log`. Исторические PASS не подставляются под этот hash.

## Итоговая локальная квалификация

Все записи ниже — source `768864c97a397db92a18a908948b577e4b3bc32fb7a3388e6110843aceafab18`, 2026-10-12; `sourceUnchanged=true`, exitCode=0.

| Gate | Результат / время | Граница |
|---|---|---|
| check | PASS / 39.7 s | gofmt/vet/staticcheck, TS/lint/format, Go race, Node/Python/contract/docs checks и сборки |
| integration | PASS / 160.3 s | 51/51 TestIntegration, 15 WebDAV, реальный rclone включён; пропусков нет |
| browser | PASS / 373.1 s | 35/35: Chromium, Firefox, WebKit и два мобильных профиля; skipped/unexpected/flaky=0 в итоговом прогоне |
| ingress | PASS / 15.7 s | TLS/router/auth boundary: GET bytes, HEAD, Range, overwrite/read, revoke и network negatives |
| upgrade | PASS / 49.1 s | baseline 28ab4f3/schema 16 → schema 17; backups, downgrade/startup faults, retained session/payload, reopen |
| fuzz | PASS / 90.2 s | 5 настоящих campaigns по 15 s: path, FXP1, Range, WebDAV XML/path и If |
| verifier | PASS / 4.8 s | 5 cross-built targets, checksums и локальный unpublished review bundle |
| images | PASS / 9.3 s | 5 образов, 0 HIGH/CRITICAL; UNKNOWN finding сохранён в inventory |
| performance | PASS / 38.5 s | smoke 32 MiB/2000 rows; Local→synthetic TLS S3, rotation, crash/retry, DR, one-time ZIP; не release-size |
| compatibility | PASS / 0.1 s | консервативный compatibility check существующего baseline |
| licenses | PASS / 3.7 s | inventory/SBOM сформированы; не заменяет решение владельца по лицензиям |
| dependencies | PASS / 5.8 s | npm audit: 0; govulncheck: 0 reachable/0 imported-package findings, 1 module-level advisory |
| source | PASS / 3.1 s | Trivy 0 HIGH/CRITICAL; Gitleaks history/worktree: 0 leaks; UNKNOWN сохранён |

Raw records: `.cache/evidence/gates/*.json`, логи `.cache/webdav-native-final-*.log`; не переписывались под новый hash. Нагрузка — локальный smoke, historical 2050 MiB/20000 rows не переносится на этот кандидат. `readiness` ожидаемо BLOCKED: необходимы большой профиль, 14 внешних решений, 5 partial MUST и clean Git; это не ошибка WebDAV и не разрешение production.

Security inventory видимо сохраняет `GO-2026-5932` (UNKNOWN, x/crypto/openpgp): source v0.58.0, два binary locations API v0.58.0 и Traefik v0.57.0. FileX govulncheck не нашёл затронутых импортированных packages/symbols; этот вывод не объявляется binary reachability review стороннего Traefik. Исключения scanner не добавлялись. [Границы third-party review](../third-party-review.md).

## Реальные клиенты на том же source
| Клиент | Результат | Содержимое |
|---|---|---|
| macOS 27.0.1 / mount_webdav | PASS: mount, mkdir, upload, server-created GET/read, replace, rename, delete, rmdir, unmount | 5 700 000 bytes; SHA-256 `755a378a57a6e937804d1641edf10af177e820448a519b427863b95e30a51359` |
| Linux GNOME GVfs 1.54.4-0ubuntu1~24.04.4 / GLib 2.80.0 | PASS: тот же цикл, изолированный Docker/D-Bus backend | 5 600 000 bytes; SHA-256 `d7eb5dff92552706b38c125ae7edf490f519d42a7d51efb4c5b0290a5a7ef3ff` |
| Windows WebClient | **Ожидает стенда** | PowerShell-сценарий и ручная матрица подготовлены; запуск не выдумывается |

Logs/checksums: `.cache/webdav-native-{macos,linux}.log`, `.cache/webdav-native-clients.json`. Cleanup assertion обоих clients PASS; после прогона временных тестовых контейнеров и WebDAV mounts не осталось. Finder/GNOME desktop clicks и Windows не входят в эти измерения.

## Completed steps
- Прочитаны обязательные AGENTS/wiki; до реализации создан [ADR-011](../adr/011-native-webdav.md), актуализированы threat/state/architecture/crypto review. Прямой запрос владельца расширяет доступ только к owner draft.
- Migration 0017 добавляет PG exclusive locks, свойства/времена и hidden encrypted stages. Реализованы GET/HEAD/Range, conditional PUT/atomic overwrite, unknown-length encrypted rewrap, LOCK/UNLOCK/If, PROPPATCH и MOVE с сохранением метаданных.
- Сохранены immutable publish, recipient one-time, USER/ADMIN privacy, FXP1 и общий recovery/freeze/credential guard. REST/tus/tree/bulk/publish учитывают DAV locks и stages; worker очищает expired stages с учётом hold.
- Добавлены ограничения XML/namespace/locks/properties, включая aggregate metadata budget; исправлены root-property inheritance, parent-first locks и LOCK-null под locked parent. Последний regression был RED (423 вместо 201), затем GREEN после сохранения родительского If token.
- Реальные mount_webdav macOS 27.0.1 и Linux GVfs 1.54.4/GLib 2.80.0 повторно прошли полный файловый цикл, multi-chunk hash, чтение server-created fixture и unmount на окончательном source.
- Подготовлен Windows PowerShell-сценарий, инструкции подключения и матрица приёмки. Ни Windows-клиент, ни Office/KDE/davfs2 не объявлены проверенными.
- Обновлены канонические wiki, ADR, contracts/types/traceability и changelog. Исторический аудит сохранён отдельно. В credential operation устранено устаревшее название upload-only; генераторы выполнены, generated files вручную не правились.
- Обновлены 48 скриншотов, 20 глав, 79 cues, единый MP4 5:56 с Liza. Все 48 кадров и ключевые видеокадры просмотрены; gallery/axe/playback, 54 checksum entries и полный video decode PASS. Инструкция в панели больше не привязана к Mac/8090.
- Новая глава WebDAV ранее синтезирована через Chrome один раз (454 символа); 19 неизменных глав переиспользованы. Последняя пересъёмка и сборка полностью offline относительно ElevenLabs, дополнительных TTS-запросов нет.
- Устранена потеря API response body при навигации в E2E: один реальный upstream JSON response проверяется до fulfill, без повторов mutations. Binary tus остаётся на браузерном транспорте. Добавлены 6 Node cases и реальный HTTP race test; полная матрица 35 cases прошла на окончательном source. Неповторившаяся задержка WebKit зафиксирована отдельно ниже.
- TLS fixture обновлён с прежнего GET=403 на точные проверки native GET/HEAD/Range/overwrite; он прошёл отдельно. Upgrade 16→17 прошёл с проверкой pre-DDL backup, отказом старой schema backup после DDL, fresh backup, downgrade и retained payload/session. Все перечисленные gates повторно прошли на окончательном source.

## Media evidence
Съёмка: `2026-10-12T13:58:18.286Z`, synthetic локальный Compose, Chromium Linux. Application scope SHA-256: `aa0afcdedbd0d1f2dfebca3a2b84fd44598ac4a87dce6cb83f4182ab2c7acd3d`; после съёмки изменён только generated TS comment о названии credential operation, UI/runtime не менялись. Этот scope отличается от полного repository source hash.

[MP4](../tour/filex-tour.mp4): 355,96 s, 9 955 451 bytes, H.264 1440×1080/25fps + AAC mono 48kHz. SHA-256: `8457a74f51773997816ef587e6a06f2ca757ab9bf5621e11fa7bf9efc58e133a`. Loudness −16,8 LUFS, LRA 2,9 LU, peak −1,0 dBFS. [Галерея](../tour/index.html), [скрин-тур](../screen-tour.md), [воспроизведение](../tour-maintenance.md). Raw evidence: `.cache/webdav-native-tour-{capture,render,qa,decode,volume}.log`, `.cache/tour/qa/screens-*.png`, `.cache/tour/qa/native-video-*.png`.

## Next steps
1. На будущем Windows-стенде выполнить подготовленный сценарий с trusted HTTPS; записать Windows/WebClient build, лимиты и exact server source.
2. При повторении задержки E2E auth/me сопоставить browser trace с upstream/DB и ресурсами. Не увеличивать timeout без диагностики и не считать единичный успешный повтор исправлением причины.
3. Дождаться required checks WebDAV PR на точном head, проверить CodeQL findings/conversations, выполнить squash merge и отдельно проверить main CI. Любое новое изменение кода/контрактов/медиа требует актуальных gates.
4. Внешний release путь продолжить по [плану](../implementation-plan.md) и [карточке окружения](../environment-acceptance.md).

## Decisions
- Draft доступ: отдельный Basic credential с TTL/отзывом, только свой owner/transfer. Перед publish закрыть файлы и отключить сетевую папку.
- PG authoritative, parent-first transactions; административное containment сильнее DAV lock. Redis/in-memory locks не добавлялись.
- Неизвестный размер PUT требует двух проходов зашифрованных данных; plaintext spool и новые crypto primitives не добавлялись.
- Windows pending по ответу владельца, HTTP fixtures не заменяют native acceptance. Тур демонстрирует веб-панель, не запуск Windows.
- Source-bound gates выполняются последовательно: параллельные тяжёлые прогоны ранее приводили к host pressure/timeouts. Assertions, retries и timeout gates ради PASS не ослаблялись.

## Context
[WebDAV](../webdav-upload.md), [ADR-011](../adr/011-native-webdav.md), [testing](../testing.md), [current state](../current-state.md), [documentation audit](../documentation-audit.md), [tour maintenance](../tour-maintenance.md). Продукт: 108 REST/tus operations + WebDAV wire profile, 17 migrations, 11 ADR.

## Подготовка доставки
Remote `main` проверен перед commit: `28ab4f36e15a0b5e052b23e0f058bdda7a1c9d12`; удалённой ветки `codex/webdav-upload` и PR для неё пока нет. Локальный snapshot `.cache/webdav-commit-delivery/pre-stage-source.json` связывает каждое проверенное содержимое с исходным source hash. При staging удалений source hash изменился на `ab17d1f1a09a0cdc86bbcd8a7841587379299592103ddba68f6fc23019ea05a5` из-за 39 уже отсутствующих старых PNG: алгоритм `acceptance.source_hash()` до staging включает их имена с маркером `deleted`, после staging исключает. Побайтовая сверка всех 336 остальных source entries PASS; gate records не переписывались под новый hash. Это изменение Git inventory, а не новый запуск тестов. Readback: `.cache/webdav-commit-delivery/staged-verification.json`. Перед commit повторно проверены 78 Markdown, traceability всех 32 MUST, отсутствие битых ссылок/whitespace и все 54 checksum тура.

## Risks / blockers
- Windows native acceptance ожидает стенда; Office/KDE/davfs2 не квалифицированы.
- Внешнее окружение не подготовлено, 5 из 32 MUST partial, 14 human decisions pending. APP_ENV=production по-прежнему отклоняется; deploy требует отдельной команды.
- Performance default profile (32 MiB/2000 rows) — smoke, не release-size measurement и не target SLA.
- DEV-05 (устаревшие controls открытой recipient page после completed) сохранён как отдельный существующий backlog; сервер запрещает повторную выдачу.

## Диагностика промежуточного прогона
Ingress/check PASS на окончательном source. Один WebKit operations case встретил незавершённый auth/me после login=200; причина задержки не установлена, trace сохранён приватно. Полный browser gate повторён на новом disposable окружении без изменения source/assertions/timeout/retries и прошёл. Причина промежуточной задержки не установлена; детали в testing.

Повторный полный browser PASS на том же source: 35/35, skipped/unexpected/flaky=0, без изменения timeout/retries/assertions. Ресурсный журнал `.cache/webdav-native-browser-resources.jsonl` завершён; задержка не повторилась, её первопричина не объявляется устранённой. Gate metadata предыдущего отказа и trace сохранены в `.cache/webdav-native-prerun/`. Integration/upgrade/fuzz/security/native затем завершены успешно на том же source.
