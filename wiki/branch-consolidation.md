# Объединение веток и результаты проверок

## WebDAV: объединение 2026-10-12

**Контекст:** команда владельца после commit/push разрешает объединить завершённую `codex/webdav-upload` с main. Head до записи этого шага — `325286b`, main — `28ab4f3`. Изменения включают owner draft WebDAV, migration 0017, contracts/tests/wiki и 48 PNG/20 глав MP4. Windows остаётся ожидающей стенда; macOS mount_webdav/Linux GVfs имеют локальное evidence.

**Поток:** неизменённый runtime source `ab17d1f1a09a0cdc86bbcd8a7841587379299592103ddba68f6fc23019ea05a5` → PR → required checks/CodeQL для фактического head → squash merge → main CI и сверка дерева → удаление только принятой topic-ветки. Связь исходных 13 локальных gates с этим Git inventory описана в [session](session.md); records не переписываются. Внешняя приёмка и production отдельно.

**Новая независимая ветка:** [Dependabot #22](https://github.com/elnemir/p2p-files/pull/22) / `6bb2e37` меняет только два Node FROM на 26. [Development run 38260976979](https://github.com/elnemir/p2p-files/actions/runs/38260976979) имеет failed repository-policy/checks/supply-chain; `.nvmrc`, engines, Node types и reviewed policy остаются на 24. Это не допустимый кандидат для merge. [Upstream schedule](https://github.com/nodejs/Release/blob/main/schedule.json), повторно проверенный 2026-10-12, назначает Node 26 LTS на 2026-10-28, а Node 24 получает security support до 2028-04-30. Сохраняется согласованная поддерживаемая ветка; #22 остаётся открытым, scanner/engine/LTS guards не меняются. Следующий шаг для Node 26 — синхронная квалификация runtime/types/CI после LTS, как описано в [dependency remediation](dependency-remediation.md).

## Видеотур: объединение 2026-10-12

**Контекст:** после отдельного commit/push озвученный тур находится в `codex/tour-liza`, тогда как main ещё содержит прежнее видео. По новой команде владельца проверены все локальные и удалённые ветки: main `5096726` и tour `5166594`, открытых PR нет. Три коммита тура включают готовый MP4, безопасный offline-импорт аудио из Chrome, новые негативные тесты, обновлённые assets и документацию.

**Поток:** весь diff topic → [PR #21](https://github.com/elnemir/p2p-files/pull/21) в main → обязательные checks и CodeQL на текущем PR head → squash merge → отдельный main CI/readback → сравнение дерева и SHA-256 видео → удаление объединённой ветки. Входной source SHA-256 — `3e179312df400d012e686e129bc88f1ace7961d8e7866dbc030eb328933adcfe`; локальные check/source/gallery и предыдущий manual CI описаны в [session](session.md). PR хранит фактические merge SHA/state и собственные проверки; старый manual PASS не заменяет их.

**Результат для пользователя:** `wiki/tour/filex-tour.mp4` в main содержит все 19 глав, голос Liza и вшитые субтитры; файл пригоден для самостоятельной передачи. Контракты, env, миграции и runtime приложения не меняются. Исходные MP3/ключи/локальные logs остаются вне Git. Фактические PR/merge/main результаты фиксируются в session; внешняя приёмка фазы 7 не закрывается этим объединением.

## Контекст и задача

Пользователь поручил объединить ветки после настройки GitHub и тура. Исходная main — `e70af757c7d501a8e2972c0f906d3e16e2500839`. В удалённом репозитории были 13 открытых Dependabot PR; локально оставались две ранее squash-merged ветки и незакоммиченная актуализация wiki. Объединение не должно отключать защиту main, peer dependency checks, security gates или принимать лицензионные обязательства без review.

## Решение и поток изменений

`origin/*` после fetch → сравнение patch и результата прежних squash merges → перенос совместимых изменений в `codex/consolidate-branches-and-narration` → local gates с единым source hash → PR и обязательные exact-head checks → squash merge → отдельная проверка main. Результат: [PR #16](https://github.com/elnemir/p2p-files/pull/16) слит 2026-10-11, основной code commit main — `2e1d2e4b17cffb59fec53eca55b96c45b1847e43`. Source hash и результаты проверок приведены ниже; отдельный итог проверки main фиксируется в [session](session.md).

`codex/repository-setup` (`0a9f384`) имеет пустой diff к результату PR #1 (`db02ab2`). `codex/repository-verification` (`a2a1af8`) имеет пустой diff к результату PR #15 (`e70af75`). `codex/code-scanning-verification` указывает на `e70af75`; его незакоммиченная wiki сохранена в новом кандидате. Повторное наложение старых commits не требуется.

## Учёт всех Dependabot PR

| PR | Предложение | Решение и основание |
|---|---|---|
| [#2](https://github.com/elnemir/p2p-files/pull/2) | upload-artifact 7.0.2 | Перенесён полный SHA pin; retention/always/upload hidden evidence сохранены |
| [#4](https://github.com/elnemir/p2p-files/pull/4) | checkout 7.0.1 | Перенесён SHA pin; persist-credentials=false и полная история для scans сохранены |
| [#6](https://github.com/elnemir/p2p-files/pull/6) | Alpine 3.24 | Перенесён immutable digest для API и Traefik; обязателен новый image scan |
| [#7](https://github.com/elnemir/p2p-files/pull/7) | Go 1.27.0 builder | Заменён на исправленный Go 1.27.2, согласованы go.mod и два builder digest; исходный PR имел HIGH/CRITICAL image failure |
| [#8](https://github.com/elnemir/p2p-files/pull/8) | React/React DOM types 19.3.0 | Перенесены обе версии, lockfile согласован с точными specs package.json |
| [#12](https://github.com/elnemir/p2p-files/pull/12) | setup-node 7.1.0 | Перенесён SHA pin; runtime остаётся Node 24 по .nvmrc |
| [#13](https://github.com/elnemir/p2p-files/pull/13) | dependency-review-action 5.0.0 | Перенесён SHA pin; HIGH fail threshold сохранён |
| [#14](https://github.com/elnemir/p2p-files/pull/14) | setup-go 7.0.0 | Перенесён SHA pin; версия берётся из go.mod |
| [#3](https://github.com/elnemir/p2p-files/pull/3) | Redis 8.10 | Не включён: Redis 8 меняет набор лицензий, нужен [owner review](third-party-review.md). Сбой OpenSSL pins исправлен в отдельном review-образе; в runtime остаётся 7.2 |
| [#5](https://github.com/elnemir/p2p-files/pull/5) | Node 25 image | Не включён: версия EOL с 2026-06-01, исходный PR имеет HIGH/CRITICAL image failure и browser failure; поддерживаемый Node 24 сохранён |
| [#9](https://github.com/elnemir/p2p-files/pull/9) | @eslint/js 10.0.1 | Не включён: переход на ESLint 10 должен быть согласован со всем lint toolchain |
| [#10](https://github.com/elnemir/p2p-files/pull/10) | @types/node 26.6.4 | Не включён: major типов не соответствует используемому Node 24; исходный PR зелёный, причина решения — согласованность runtime/types |
| [#11](https://github.com/elnemir/p2p-files/pull/11) | eslint 10.12.0 | Не включён: eslint-plugin-jsx-a11y 6.10.2 декларирует peers только до ESLint 9; обход force/legacy-peer-deps запрещён |

Жизненный цикл Node проверен по [официальной таблице](https://nodejs.org/en/about/previous-releases) и [schedule.json](https://github.com/nodejs/Release/blob/main/schedule.json). Go выбран по [официальным downloads](https://go.dev/dl/). Peer ranges проверены через registry metadata `npm view eslint-plugin-jsx-a11y@6.10.2 peerDependencies --json`. Эти сведения относятся к 2026-10-11 и требуют повторной проверки при продолжении.

## Проверки и ограничения

Изменены CI Actions, Go/Alpine build/runtime images, React types, локальные инструменты озвучки и wiki. Требуются `npm ci`, базовый check, PG integration, browser matrix, ingress/upgrade/performance, fuzz/verifier и dependency/source/license/image scans. Запуски выполняются через [acceptance wrapper](testing.md); старые зелёные PR не доказывают совместимость общего кандидата. При объединении веток реальный ElevenLabs синтез был отложен пользователем после HTTP 402. Новая попытка с Liza 2026-10-12 также получила HTTP 402; актуальное состояние и команды — в [обслуживании тура](tour-maintenance.md). Offline тесты не доказывают качество голоса.

Снимок веток и patch-файлы находятся в ignored `.cache/consolidation/`; это локальные диагностические материалы, не release artifacts. Восемь superseded PR закрыты после общего merge; их ветки удалены после сравнения точного SHA. Четыре локальные рабочие ветки удалены после проверки tree/diff к соответствующим squash-результатам. Пять нерешённых PR остаются открытыми и не считаются слитыми. Dependabot автоматически перебазировал их на новую main; это не отменяет описанных ограничений.

## Продолжение

Состояние пяти отложенных PR выше — исторический результат PR #16. Их последующий разбор и исправления ведутся в [dependency remediation](dependency-remediation.md); актуальная квалификация — [session](session.md).

- Redis: получить решение о лицензиях, затем перенести проверенные pins из review и пройти полные функциональные/security gates; до этого остаётся 7.2.
- ESLint: подобрать совместимый a11y-плагин/поддерживаемую комбинацию с прежним покрытием правил и проверить без peer bypass.
- Node: обновлять runtime, .nvmrc, types, images и CI согласованно в поддерживаемую ветку; Node 25 не использовать как промежуточный шаг.
- Озвучка: 2026-10-12 все 19 глав Liza получены через Chrome, локальный MP4 пересобран. Готовый кэш не требует повторного платного синтеза; отдельное API-ограничение и воспроизведение описаны в [инструкции](tour-maintenance.md#озвучка-elevenlabs).

Production deployment и принятие 14 внешних решений не входят в объединение веток.

## Результат локальной квалификации

Все 13 [gates](testing.md) PASS на source `0cc2cee4b1f8cc10fb1a780a28fd1fd42f8272d466b83b13af72f3a58646c722`. Npm audit: 0 vulnerabilities; source и пять фактических image IDs: 0 HIGH/CRITICAL; secrets scan чист. Пройдены PG integration, все 25 browser scenarios, TLS ingress, distinct baseline/candidate upgrade, fuzz и пять verifier targets. Performance выполнен с 32 MiB / 2000 rows; этот компактный локальный профиль не закрывает release-size/external acceptance.

В момент измерения HEAD был `e70af75` с рабочим diff; source hash связывает результат именно с объединённым кандидатом. `.cache/consolidation/local-gates.json` содержит сводку, `.cache/evidence/gates/` — исходные records. GitHub exact-head CI [PR #16](https://github.com/elnemir/p2p-files/pull/16) также PASS: все 13 скачанных gate records имеют этот source и merge ref `86043498494df2a5e79f7de6899907dc14b1851e`. Обе группы CI artifacts сверены; четыре CodeQL анализа и open alerts чисты. Отдельные post-merge main runs приведены в session.

## Отдельный review Redis 8

Исходный #3 пытался установить libcrypto3/libssl3 3.3.7-r2 поверх base, которому нужен 3.5.8-r0. В отдельном ignored `.cache/consolidation/redis8-review/Dockerfile` сохранены base digest `sha256:3811787313eba226a2ef38658c6ccb91cd5e110edc89c37767de373120a0e5a0`, zlib 1.3.2-r1 и оба OpenSSL pins 3.5.8-r0. Образ `sha256:2d6b62c16a496ca9cd93da252efd0735a8f5d28ade047b8cc592e2a208d69459` успешно собран; `redis-server --version` сообщает 8.10.2, Trivy — 0 findings. JSON scan, CycloneDX SBOM и result.json находятся рядом.

Это локальная техническая подготовка к решению, а не включение Redis 8 в FileX. Проверен build/scan/version; полная функциональная/DR приёмка Redis 8 не выполнялась. Владелец/reviewer ещё должен оценить RSALv2/SSPLv1/AGPLv3 и выбранные обязательства. В основном PR/main сохранён Redis 7.2.

## Исправление после проверки main

Main development run 38167045971 завершился failure: ранний клик в публичной проверке отчёта до client initialization, затем отсутствие чужого CSV в recovery test. Прежний PR PASS не исключил гонку. Повторный PR run 38167905201 обнаружил listener, остававшийся активным после завершения contract observer. [PR #17](https://github.com/elnemir/p2p-files/pull/17) включает исправление готовности формы, собственный recovery export fixture и корректное завершение observer, а также итоговые записи объединения. Причина, data flow, регрессии и ограничения описаны в [testing](testing.md); актуальная квалификация нового source — в [session](session.md). Gates/retries/timeouts не ослабляются.

Квалификация исправления локально завершена: source `8390f72c6ac08927e609dc9eccfb78d93ebc7fbbdae8363c47e449cf50feb210`, 13 gates PASS, 25 browser cases, пять images без HIGH/CRITICAL. PR/main результаты этого кода публикуются отдельно, исторические результаты #16 не переименованы.

PR #17 слит 2026-10-11T20:48:40Z без bypass; code commit main `548359d9d722641a5a66acd53afd8702d98876c6`. Все 13 скачанных PR CI records совпадают с source 8390f72c и merge ref `83cd5aa67fd44f8b6ec60a5b41f57362641c99e8`; development [38172130775](https://github.com/elnemir/p2p-files/actions/runs/38172130775) и security [38172130794](https://github.com/elnemir/p2p-files/actions/runs/38172130794) SUCCESS. Четыре CodeQL категории без findings/error/warning. Ветка `codex/consolidation-results` удалена после полного tree comparison с main. Итоговая запись документации не меняет этот binary source hash; отдельные main runs и сохранённая история — в [session](session.md).

[PR #18](https://github.com/elnemir/p2p-files/pull/18) дополнительно подтверждает данные UI fixture до backup/publish и сравнивает ID после восстановления. Первый development run 38174057287 выявил пустой title ещё в исходном PATCH; данные restore соответствовали backup. Точный механизм редкого сбоя ввода не установлен. Подробнее — [testing](testing.md#подтверждённый-draft-перед-backup-и-publish); актуальные source/evidence — [session](session.md).

Заключительный fixture source `2a8380c6a3d34503a7117e842c46a0c2bb049108ff2bb8369e449146f852b8ab` прошёл все 13 local gates, включая 25 browser cases. Его CI и post-merge evidence проверяются отдельно; результаты `8390f72c` выше остаются историческими.
