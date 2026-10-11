# Session State

## Task
Настроить и подтвердить работу Code scanning в GitHub после полной настройки репозитория FileX; сохранить диагностику Advanced setup в wiki.

## Status
2026-10-11: Code scanning настроен через CodeQL Advanced setup и повторно проверен. [Ручной анализ 38161371347](https://github.com/elnemir/p2p-files/actions/runs/38161371347) завершился SUCCESS: Go, JS/TS, Python, Actions и aggregate `security-checks` прошли; четыре свежих анализа без findings/errors/warnings, open alerts на main = 0. Проверенный `main` — `e70af757c7d501a8e2972c0f906d3e16e2500839`; повторное чтение head подтвердило тот же SHA. Изменения этого шага — только локальная документация, commit/push не выполнялись.

Техническая настройка репозитория завершена ранее: [PR #1](https://github.com/elnemir/p2p-files/pull/1) и [PR #15](https://github.com/elnemir/p2p-files/pull/15) merged. После merge [development-checks](https://github.com/elnemir/p2p-files/actions/runs/38159399627) и [security-analysis](https://github.com/elnemir/p2p-files/actions/runs/38159399474) завершились SUCCESS; все 13 gate records PASS/sourceUnchanged=true на source `00865e70215913b3571a35a1d569f12ec421f8ec3e5dc735607126face7a4f7f`. Этот полный CI не запускался заново для текущего документационного diff.

## Completed steps
- GitHub API подтвердил: `security.yml` active; ruleset `24884645` active для main, без bypass actors, CodeQL HIGH/CRITICAL и errors блокируют merge. Четыре новых анализа CodeQL 2.27.2 на `e70af75` загружены после начала ручного run `38161371347` и завершились без результатов, ошибок или предупреждений; open alerts = 0. Evidence: `.cache/code-scanning/2026-10-11-run-38161371347.json`. Независимый audit PASS без drift: `.cache/repository-setup/20261011T175300.163216Z/result.json`.
- В [governance](../repository-governance.md#code-scanning-режим-результаты-и-проверка) описаны Advanced/Default setup, расписание, поток результатов, повторный запуск и критерии проверки. `default-setup: not-configured` не трактуется как отсутствие Advanced анализа; переключение режима не выполнялось.
- Локальные `make docs-check` и `git diff --check` PASS для документационного diff: 66 Markdown-файлов, все 32 требования трассированы, неработающих локальных ссылок нет. Worktree содержит изменения только `wiki/repository-governance.md` и `wiki/session.md`.
- Добавлены versioned GitHub policy и audit/apply CLI, формы Issues/PR, CODEOWNERS, Dependabot для npm/Go/Docker/Actions, editor/Git/Node defaults, SECURITY.md и документация сопровождения. [Состав и поток](../repository-governance.md).
- Применены main protection для admins, обязательные `checks`, `supply-chain`, `repository-policy`, `security-checks`, squash-only и линейная история; force push/delete запрещены. Approvals = 0 при единственном владельце; независимый review этим не заявляется.
- Actions используют read token, GitHub-owned actions и SHA pins; внешние fork требуют approval. Secret scanning/push protection включены. CodeQL выполняет Go, JS/TS, Python, Actions; ruleset **active**, HIGH/CRITICAL и errors блокируют merge. Первый bootstrap сохранял новое правило disabled только до появления анализов main; теперь оно активно.
- Final apply/readback PASS: `.cache/repository-setup/20261011T163025.333552Z/result.json`; независимые audit PASS без drift: `.cache/repository-setup/20261011T163600.430835Z/result.json` и повтор после публикации follow-up `.cache/repository-setup/20261011T165821.370695Z/result.json`. Visibility, collaborators, secrets и production не менялись. CODEOWNERS API errors = 0.
- Локально установлены `pull.ff=only`, `fetch.prune=true`, `push.default=simple`; прежние значения сохранены в `.cache/repository-setup/git-local-before.json`. Глобальный Git config не менялся.
- PR #1 head `0a9f384ad9ab0aa9c32b985709a621a23add3795`: development-checks [38153575839](https://github.com/elnemir/p2p-files/actions/runs/38153575839) и security-analysis [38153575851](https://github.com/elnemir/p2p-files/actions/runs/38153575851) SUCCESS, все 10 checks зелёные. Все 13 gate records скачаны, PASS/sourceUnchanged=true на одном source `56f73f7bba5b90c99787eebb720d234cce67a8c888356921299d199f45e1eca3`.
- Сигналы CodeQL в галерее устранены проверкой локального PNG path перед src/href. QA: 7 негативных URL сценариев, 50 checksums, 44 скриншота, 19 глав, desktop/mobile accessibility и playback PASS. Артефакты тура сохранены.
- Первый main CodeQL выявил 5 path-injection сигналов в S3 fixture. Имя файла теперь строится из разобранного canonical UUID; malformed/noncanonical inputs отклоняются до filesystem. Добавлены unit и loopback wire tests. На PR #15 head `f23caaf` CodeQL четырёх категорий, platform result и dependency review PASS; API PR ref вернул 0 открытых alerts. Исключения scanner не добавлялись.
- Повторный main development-checks [38155133870](https://github.com/elnemir/p2p-files/actions/runs/38155133870) обнаружил race upgrade harness: после чтения Content-Length соединение закрывалось до PG completion, повтор получал 409 вместо ожидаемого 401. Исправлен только тестовый клиент: connection закрывается после PG barrier Range/completion. Неуспешный commit/timeout/неверный HTTP status приводят к ошибке; 401 и ровно одна journal запись по-прежнему обязательны. [Разбор и воспроизведение](../controlled-upgrade.md).
- Финальные локальные `check`, `source`, `upgrade` PASS/sourceUnchanged=true на source `00865e70215913b3571a35a1d569f12ec421f8ec3e5dc735607126face7a4f7f`; время 21,39 / 3,13 / 22,92 s. Четыре regression tests harness PASS. Source HIGH/CRITICAL=0, Gitleaks history/worktree без находок. Логи `.cache/repository-setup/{upgrade-check,upgrade-source,final-upgrade}.log`.
- Полный локальный upgrade drill: разные baseline/candidate images, schema 16, backup verified, migration/KEK startup faults rejected, explicit reopen, retained sessions/payload и отказ повторной выдачи. Измерено 6,924 s maintenance / 6,268 s sampled unavailable только на этом локальном прогоне; это не SLA. Изолированный project и volumes удалены автоматически.
- Реальный compact TLS/load/recovery с canonical S3 fixture PASS до добавления traceability, source `49b2b6036e139503a4c6b6b4b848af20f5bb346881b672e6c577f93c6bbd6b3d`: 32 MiB / 2000 rows, Local→S3, rotation/crash/retry/backup/restore/reconcile/reopen. Это историческое доказательство fixture, не финальный release-size load gate.
- NFR-003 implementation/test mappings и generated matrix дополнены; 32 требования и их acceptance status не менялись. Dependabot уже создал первые PR; major обновления не включены автоматически.

## Next steps
- Code scanning автоматически выполняется для PR, push в main и по понедельникам 04:23 UTC; findings и status проверять по инструкции в governance. Текущая документация подготовлена локально в `codex/code-scanning-verification`; перед её commit/PR выполнить обязательные checks по workflow. CI evidence сохраняется 14 дней, поэтому для release нужен DEV-04.
- Продолжить [DEV-01–05 и EXT/REL](../implementation-plan.md): production admission design, toolchain review, licenses, долговременное evidence и recipient terminal UI.
- После выбора правообладателя/условий владельцем оформить proprietary LICENSE в DEV-03. Техническая настройка не принимает это решение.

## Decisions
- Сохранён CodeQL Advanced setup из Git; второй scanner/workflow не создаётся. Изменения текущего шага касаются только wiki и session в ветке `codex/code-scanning-verification`.
- Существующие стек, PG authority, server-observed one-time completion, crypto/privacy и release guard сохраняются. Продуктовые handlers в repository setup не менялись.
- Ветки `codex/*`, PR и обязательные проверки; merge без admin bypass. Настройки воспроизводятся через versioned baseline и API-readback.
- LICENSE не генерируется из предположений. Production deploy, secrets, публикация Pages и платная внешняя озвучка не входят в эту задачу.

## Context
[Индекс](../index.md), [workflow](../development-workflow.md), [тесты](../testing.md), [governance](../repository-governance.md), [план](../implementation-plan.md), [сессия тура](../sessions/2026-10-11-product-tour.md).

## Risks / blockers
- Внешнее окружение не подготовлено по решению владельца: 14 решений pending / 5 MUST partial. Настройка GitHub не означает release acceptance; APP_ENV=production по-прежнему отклоняется.
- Независимый security/crypto review, target load/DR, реальная browser/AT matrix и юридические решения не заменяются CI/SAST.
