# ADR-012: согласованный переход на Node 26

Статус: принят владельцем 2026-10-13 для разработки и сборок; квалификация кандидата и merge фиксируются в [session](../session.md). Production deployment этим решением не разрешён.

## Контекст и проблема
Node исполняет Next.js web server, сборку приложения и JavaScript-инструменты проверок. [PR #22](https://github.com/elnemir/p2p-files/pull/22) первоначально менял только Docker FROM с 24 на 26. `.nvmrc`, npm engine, типы и reviewed policy оставались на 24: clean Docker install закономерно завершался EBADENGINE. Отключение engine/peer checks скрывает несовместимость вместо её устранения.

Владелец прямо запросил довести PR Node 24→26 до merge. Это заменяет прежнее решение отложить 26 до LTS. По [upstream schedule](https://github.com/nodejs/Release/blob/main/schedule.json) Node 26 на дату review — **Current**, LTS начинается 2026-10-28, security support заканчивается 2029-04-30. [26.11.1](https://nodejs.org/en/blog/release/v26.11.1) — выбранный patch для локальной квалификации и pinned Docker image.

## Решение и поток данных
Reviewed policy → `.nvmrc` и root `engines.node` → clean npm install с lockfile → Node types того же major → одинаковые pinned builder/runtime images → проверки графа/сборки/браузеров/upgrade/security → exact-commit GitHub checks → merge.

Используется Node 26, root engine `26.x`, `@types/node@26.6.5`; CI читает `.nvmrc`. `.npmrc` сохраняет `engine-strict=true` и `strict-peer-deps=true`. `npm ls --all`, соответствие major и digest pins остаются обязательными. Lockfile обновляется npm, не вручную. Next.js, React, TypeScript, ESLint и Redis не обновляются попутно.

В `.github/dependency-policy.json` `channel` означает канал **на дату `reviewedAt`**, а `ltsFrom` содержит подтверждённую upstream дату LTS. Validator проверяет календарные даты, review не из будущего, LTS раньше EOL и соответствие канала дате review. Поэтому Current не переименовывается в LTS до срока, а наступление 28 октября не ломает уже проведённый review. EOL проверяется относительно времени запуска каждого gate. Новый major требует согласования всех файлов и нового review; metadata не заменяет фактическую проверку upstream и security scans.

## Пример
Для review 2026-10-13, `ltsFrom=2026-10-28`, `securityUntil=2029-04-30` допустим `channel=Current`; `channel=LTS` отвергается. Для нового review от 2026-10-28 допустим LTS. Старый Current review остаётся валиден после перехода, пока runtime поддерживается. Начиная с 2029-04-30 gate отказывает независимо от прежнего зелёного CI.

## Architecture consistency / threat / crypto review до реализации
Первоначальная граница изменения — JavaScript runtime/toolchain. PG authoritative, Redis ephemeral, FXP1, keys, one-time/immutable payload и USER/ADMIN separation сохраняются. При квалификации дополнительно исправлен idle operations worker, который без задания мог удерживать lock и отклонять валидный reopen; [уточнение ADR-006](006-operations-recovery.md) и PG RED/GREEN подтверждают сохранность recovery guards. Runtime Go не обновляется. State machines, OpenAPI 3.1, AsyncAPI и JSON Schemas не получают новых состояний/событий/операций. Угрозы: несовпадение dev/CI/runtime, типы отсутствующих API, неподдерживаемый runtime, уязвимые transitive/image packages. Контроли: strict install, negative policy tests, build/browser/ingress/upgrade, dependency/source/image scans и CodeQL без suppressions. [Модель угроз](../threat-model.md) и [crypto design review](../design-review.md) остаются применимыми.

## Проверка, эксплуатация и ограничения
Локально используется официальный macOS ARM64 архив 26.11.1 с SHA-256 из upstream release; глобальная установка Node не меняется. Linux Docker runtime проверяется отдельно. Playwright browser-server image остаётся закреплённым образом версии Playwright; host test driver и web application работают на Node 26. Версия runtime browser-server не является версией приложения.

Обязательны clean `npm ci`, full dependency graph, negative policy tests, все 13 local gates и required PR checks на новом source. Старые Node 24 результаты хранятся как история. Фото/видео не переснимаются без изменения UI; исходный capture source остаётся указанным в manifest. Windows WebDAV ожидает стенда, внешние release blockers и production guard сохраняются. Rollback — согласованный revert всего изменения runtime/types/policy/lockfile через проверенный PR, а не замена одного Docker FROM; миграций БД в этом ADR нет.
