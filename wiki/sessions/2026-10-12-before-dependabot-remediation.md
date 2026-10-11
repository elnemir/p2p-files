# Session State

## Task
Объединить ветки FileX, сохранить проверенные настройки GitHub/Code scanning и подготовленную озвучку тура. Пользователь отложил реальный ElevenLabs синтез.

## Status
В main перенесены восемь совместимых Dependabot обновлений через [PR #16](https://github.com/elnemir/p2p-files/pull/16), затем исправления публичной формы и browser observer через [PR #17](https://github.com/elnemir/p2p-files/pull/17). Заключительное изменение — [PR #18](https://github.com/elnemir/p2p-files/pull/18): подтверждённые данные draft до backup/publish, точные ID после restore и итоговые записи wiki. Состояние merge и проверки конкретного head доступны непосредственно в PR; post-merge runs — в [main Actions](https://github.com/elnemir/p2p-files/actions?query=branch%3Amain).

Текущий source: `2a8380c6a3d34503a7117e842c46a0c2bb049108ff2bb8369e449146f852b8ab`. Все 13 local gates PASS, 25/25 browser cases на пяти Linux профилях; unexpected/flaky/skipped=0. Records проверены на неизменный source. Merge допускается только после зелёного exact-head CI и CodeQL; локальный PASS не заменяет CI. Скачанные CI artifacts и свежие анализы сохраняются в `.cache/consolidation/fixture-pr-evidence/`, `fixture-pr-gates.json`, `fixture-pr-codeql.json`; отдельная main проверка — в `fixture-main-evidence/`, `fixture-main-gates.json`, `fixture-main-codeql.json`.

Первый PR #18 run [38174057287](https://github.com/elnemir/p2p-files/actions/runs/38174057287) FAILED: mobile WebKit отправил пустой title до backup, restore вернул тот же ID/value. Точный механизм редкого сбоя ввода не установлен; 35 локальных повторов прежнего ввода PASS. Новый общий fixture вводит название с клавиатуры и подтверждает DOM/PATCH/GET/heading до продолжения. Product UI и restore не менялись. Причина и границы проверки — [testing](../testing.md#подтверждённый-draft-перед-backup-и-publish).

Пять Dependabot PR остаются открытыми с конкретными блокерами и не считаются объединёнными. Реальный ElevenLabs синтез отложен пользователем после HTTP 402. Production deployment и внешняя приёмка не выполнялись.

## Completed steps
- В PR #18 добавлен общий `scripts/e2e/transfers.ts`: создание → keyboard input → точный PATCH → независимый GET → UI heading; recovery проверяет сохранённый и поздний ID. Локальная квалификация нового source: все 13 gates PASS. Wiki, changelog и traceability согласованы.
- Сохранены изменения прежних человеческих веток; восемь superseded Dependabot PR закрыты после переноса изменений. Учёт всех 13 предложений и основания — [branch consolidation](../branch-consolidation.md).
- Обновлены пять Actions SHA pins, Alpine 3.24, Go 1.27.2, React types 19.3.0. Node 24 и Redis 7.2 сохранены. Для Redis 8 подготовлен отдельный build/scan review с исправленными OpenSSL pins и 0 findings; runtime/лицензия не приняты.
- Публичная verify форма ждёт client initialization; delayed-hydration regression проверяет недоступность SSR controls, ID из fragment, один фактический API request и запрет пустого submit. Recovery создаёт собственный USER CSV в отдельном context и после ADMIN restore выбирает тот же ID. Observer снимает listeners до drain, читает body до schema compilation, фиксирует capture errors без unhandled rejection.
- Контрольные RED на прежней реализации подтверждены. Исторический кандидат #17: 25/25 browser cases на пяти Linux профилях, Node contract/lifecycle tests 8/8, PG/race, fuzz, TLS ingress, controlled upgrade, compatibility и пять verifier targets PASS. Performance 32 MiB / 2000 rows — compact, не release-size/external acceptance.
- Main CodeQL отдельно подтверждён для `548359d`: четыре свежих анализа без findings/error/warning и 0 open alerts; readback сохранён рядом с PR evidence.
- Npm vulnerabilities 0; Go reachable 0; source и пять images без HIGH/CRITICAL; history/worktree secrets scan чист. CodeQL PR: четыре языка, 0 findings/error/warning; open alerts 0. GitHub audit: matchesPolicy=true, main protection не ослаблена.
- ElevenLabs generator/renderer и 8 unit/fault tests включены в #16. Ключ хранится вне Git в `~/.config/filex/elevenlabs.env` (0600; каталог 0700), приложению/CI не передаётся. Первый синтез отклонён HTTP 402; по решению пользователя новых synthesis requests нет. Публичный тур сохраняет Milena, 19 глав, 44 PNG и 325,94 s; реальная Ariana ещё не принята.
- Wiki/session/traceability/changelog обновлены. Причины двух неуспешных CI runs и промежуточной fixture-гонки сохранены в [testing](../testing.md) и [истории квалификации](../sessions/2026-10-11-consolidation-qualification.md). Исторические FAILED не переименованы в PASS.

## Next steps
- Redis #3: owner/reviewer выбирает допустимую лицензию; затем перенести проверенные pins и пройти полные gates.
- ESLint #9/#11: дождаться/подобрать поддерживаемую комбинацию с JSX-a11y без force/legacy-peer-deps и потери проверок.
- Node #5/#10: согласованно обновлять поддерживаемые runtime/types/images/CI; Node 25 EOL не включать как промежуточный шаг.
- ElevenLabs: только после нового сообщения пользователя о доступной квоте — sample с явным --retry-rejected, прослушивание, остальные главы через кэш, offline render и media/browser QA.
- Внешняя приёмка продолжается по [карточке окружения](../environment-acceptance.md) и [плану](../implementation-plan.md).

## Decisions
- Merge только с зелёными required checks exact head; старое evidence не подтверждает новый source. Markdown-фиксация измерений не меняет binary source hash, но проходит docs-check и обычный CI.
- Пять отложенных PR не выдаются за merged; лицензии и residual risks не принимаются автоматически.
- Новых реальных TTS запросов, покупок/пополнений, расширения scope ключа и production deployment нет.

## Context
[Индекс](../index.md), [объединение](../branch-consolidation.md), [проверки](../testing.md), [repository governance](../repository-governance.md), [озвучка](../tour-maintenance.md).

Текущая квалификация: `.cache/consolidation/fixture-gates/summary.json`, `fixture-gates/`, `title-probe.log`, `fixture-repository-audit.json`; GitHub audit matchesPolicy=true, differences=[].

Ignored evidence: `.cache/consolidation/hydration-gates/`, `hydration-pr-evidence/`, `hydration-pr-gates.json`, `hydration-pr-codeql.json`, `hydration-main-codeql.json`. Контрольный RED: `hydration-control/`, `contracts-before.log`; промежуточный FAILED: `hydration-first-attempt/`. CI artifacts сохраняются 14 дней и не заменяют долговременный release package.

## Risks / blockers
- Пять открытых Dependabot PR: лицензии Redis 8, Node 25 EOL, ESLint/a11y peers и Node types/runtime.
- ElevenLabs HTTP 402: реальное аудио и качество Ariana не проверены; subscription scope не расширялся.
- Окружение не подготовлено: 14 pending external decisions / 5 partial MUST. APP_ENV=production guard сохранён, release/deploy не объявлены разрешёнными.
