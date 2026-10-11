# AGENTS.md — правила разработки FileX

## Порядок работы
1. Прочитать wiki/index.md, wiki/session.md, MEMORY.md, docs/requirements.md, docs/architecture.md, docs/decisions.md и docs/implementation-plan.md.
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
`wiki/index.md` — каноническая точка входа в знания проекта; `docs/` содержит нормативные продуктовые контракты и связан ссылками с wiki. После каждого шага обновлять соответствующую wiki-страницу и `wiki/session.md`: задача, статус, выполненное, следующее, решения, контекст, риски. Документировать контекст, проблему, решение, поток данных, примеры и ограничения. Сохранять чужие изменения. Не объявлять этап закрытым по одному каркасу или по пропущенным тестам.
