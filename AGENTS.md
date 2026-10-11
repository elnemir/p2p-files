# AGENTS.md — правила разработки FileX

## Порядок работы
1. Прочитать wiki/index.md, wiki/session.md, MEMORY.md, wiki/requirements.md, wiki/architecture.md, wiki/decisions.md и wiki/implementation-plan.md; затем относящиеся к задаче страницы из wiki/repository-map.md.
2. До основной реализации создать threat model, state machines, OpenAPI 3.1, AsyncAPI, JSON Schemas, ADR и провести architecture consistency review и crypto design review.
3. Реализовывать вертикальными срезами; каждый инвариант подтверждать негативными, concurrency и fault-injection тестами.
4. Вести матрицу requirement → implementation → test; при неразрешимом противоречии остановить затронутую работу и спросить владельца.
5. Запускать все обязательные проверки до commit/PR/release; не отключать gates ради прохождения CI.

## DO NOT CHANGE WITHOUT APPROVAL
- Стек: Go, Next.js/TypeScript, PostgreSQL, Redis, Traefik, Docker Compose, Local/S3.
- PostgreSQL authoritative; Redis ephemeral; transactional outbox, distributed leases/fencing.
- Одноразовая выдача и удаление, immutable payload после publish, атомарный Download all, strict revoke/expiry, freeze/forensic semantics.
- Разделение USER/ADMIN privacy, tamper-evident audit, crypto/key handling и report verification.
- Proprietary license, ключевые бизнес-инварианты и запрет самостоятельного production deployment.

## Правила качества
Нельзя скрывать TODO/mock в production, игнорировать ошибки, логировать секреты, тела запросов или raw query. Не изобретать криптографические primitives. Generated code вручную не править. Комментарии в коде — английские и объясняют причины; внутренняя проектная документация — русская; OpenAPI/AsyncAPI descriptions — английские. JSON поля camelCase, enum snake_case, PostgreSQL snake_case.

## Автономность
Разрешены локальные commits (Conventional Commits), push, PR, merge при зелёных gates, tag и GitHub Release. Production deploy только по отдельной явной команде владельца. Major dependency upgrade допустим с review/тестами; смена основного стека, инвариантов, принятие residual risks или исключение требований — только с одобрения.

## Definition of Done
Код, migration, тесты, security review, docs, OpenAPI/AsyncAPI/schema sync, changelog, test matrix. Нет открытых critical/high findings.


## Знания и состояние сессии
`wiki/` — единственный каталог проектной документации; `wiki/index.md` — точка входа. Требования, архитектура, ADR, runbooks, планы и evidence находятся здесь. Машиночитаемые контракты и traceability остаются в `contracts/`; исходники генераторов — в `scripts/`. `wiki/test-matrix.md` генерируется, вручную не редактируется. Исторические записи находятся в `wiki/history/` и `wiki/sessions/`, текущий статус — в `wiki/current-state.md` и `wiki/session.md`.

Приоритет: явная задача пользователя → эти правила → wiki → существующие соглашения кода → инженерное решение. Расхождение кода и wiki расследовать и отражать в документации, не объявлять автоматически правильной одну из сторон. Не изменять бизнес-инварианты ради совпадения текста с кодом.

После каждого шага обновлять соответствующую wiki-страницу и `wiki/session.md`: задача, статус, выполненное, следующее, решения, контекст, риски. Сохранять сессию также перед рискованной операцией и перед ответом. Документировать контекст, проблему, решение, поток данных, примеры и ограничения так, чтобы инженер без истории чата мог продолжить работу. Сохранять чужие изменения. Не объявлять этап закрытым по одному каркасу или по пропущенным тестам. Измерения всегда связывать с commit/source hash и границей проверки; исторический PASS не означает PASS текущего diff. Порядок изменений и шаблон задачи: `wiki/development-workflow.md`.
