# Объединение веток 2026-10-11

## Контекст и задача

Пользователь поручил объединить ветки после настройки GitHub и тура. Исходная main — `e70af757c7d501a8e2972c0f906d3e16e2500839`. В удалённом репозитории были 13 открытых Dependabot PR; локально оставались две ранее squash-merged ветки и незакоммиченная актуализация wiki. Объединение не должно отключать защиту main, peer dependency checks, security gates или принимать лицензионные обязательства без review.

## Решение и поток изменений

`origin/*` после fetch → сравнение patch и результата прежних squash merges → перенос совместимых изменений в `codex/consolidate-branches-and-narration` → local gates с единым source hash → PR и обязательные exact-head checks → squash merge → отдельная проверка main. Номер PR и итоговые SHA/измерения записываются в [session](session.md). До подтверждённого merge изменения ниже являются кандидатом, а не состоянием main.

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
| [#3](https://github.com/elnemir/p2p-files/pull/3) | Redis 8.10 | Не включён: image не собирается с прежними OpenSSL apk pins; Redis 8 меняет набор лицензий, нужен [owner review](third-party-review.md) |
| [#5](https://github.com/elnemir/p2p-files/pull/5) | Node 25 image | Не включён: версия EOL с 2026-06-01, исходный PR имеет HIGH/CRITICAL image failure и browser failure; поддерживаемый Node 24 сохранён |
| [#9](https://github.com/elnemir/p2p-files/pull/9) | @eslint/js 10.0.1 | Не включён: переход на ESLint 10 должен быть согласован со всем lint toolchain |
| [#10](https://github.com/elnemir/p2p-files/pull/10) | @types/node 26.6.4 | Не включён: major типов не соответствует используемому Node 24; исходный PR зелёный, причина решения — согласованность runtime/types |
| [#11](https://github.com/elnemir/p2p-files/pull/11) | eslint 10.12.0 | Не включён: eslint-plugin-jsx-a11y 6.10.2 декларирует peers только до ESLint 9; обход force/legacy-peer-deps запрещён |

Жизненный цикл Node проверен по [официальной таблице](https://nodejs.org/en/about/previous-releases) и [schedule.json](https://github.com/nodejs/Release/blob/main/schedule.json). Go выбран по [официальным downloads](https://go.dev/dl/). Peer ranges проверены через registry metadata `npm view eslint-plugin-jsx-a11y@6.10.2 peerDependencies --json`. Эти сведения относятся к 2026-10-11 и требуют повторной проверки при продолжении.

## Проверки и ограничения

Изменены CI Actions, Go/Alpine build/runtime images, React types, локальные инструменты озвучки и wiki. Требуются `npm ci`, базовый check, PG integration, browser matrix, ingress/upgrade/performance, fuzz/verifier и dependency/source/license/image scans. Запуски выполняются через [acceptance wrapper](testing.md); старые зелёные PR не доказывают совместимость общего кандидата. Реальный ElevenLabs синтез отложен пользователем после HTTP 402; offline тесты не доказывают качество голоса.

Снимок веток и patch-файлы находятся в ignored `.cache/consolidation/`; это локальные диагностические материалы, не release artifacts. Восемь superseded PR можно закрыть после успешного общего merge; пять нерешённых PR остаются открытыми и не считаются слитыми. Удаление веток допустимо только после проверки сохранности их полезного содержимого.

## Продолжение

- Redis: получить решение о лицензиях, затем исправить pins и пройти функциональные/security gates; до этого остаётся 7.2.
- ESLint: подобрать совместимый a11y-плагин/поддерживаемую комбинацию с прежним покрытием правил и проверить без peer bypass.
- Node: обновлять runtime, .nvmrc, types, images и CI согласованно в поддерживаемую ветку; Node 25 не использовать как промежуточный шаг.
- Озвучка: после отдельного сообщения о доступной квоте продолжить по [инструкции](tour-maintenance.md#озвучка-elevenlabs).

Production deployment и принятие 14 внешних решений не входят в объединение веток.

## Результат локальной квалификации

Все 13 [gates](testing.md) PASS на source `0cc2cee4b1f8cc10fb1a780a28fd1fd42f8272d466b83b13af72f3a58646c722`. Npm audit: 0 vulnerabilities; source и пять фактических image IDs: 0 HIGH/CRITICAL; secrets scan чист. Пройдены PG integration, все 25 browser scenarios, TLS ingress, distinct baseline/candidate upgrade, fuzz и пять verifier targets. Performance выполнен с 32 MiB / 2000 rows; этот компактный локальный профиль не закрывает release-size/external acceptance.

В момент измерения HEAD был `e70af75` с рабочим diff; source hash связывает результат именно с объединённым кандидатом. `.cache/consolidation/local-gates.json` содержит сводку, `.cache/evidence/gates/` — исходные records. GitHub exact-head CI и результат merge проверяются отдельно; их последняя привязка находится в session и соответствующем PR.
