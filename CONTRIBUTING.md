# Разработка

Начать с [AGENTS.md](AGENTS.md), [wiki/index.md](wiki/index.md) и [wiki/session.md](wiki/session.md). Conventional Commits; PR содержит поведение, FR/SEC/OPS/NFR IDs, миграции, риски и фактически выполненные проверки.

`make check` — локальный базовый gate; `make integration-docker` — обязательные PostgreSQL/race tests на disposable БД; `make security` — dependency scans. Эти команды не заменяют [release checklist](docs/release-checklist.md). `make e2e` проверяет реальные transfer/report/export/QR сценарии через disposable Compose/Traefik, OpenAPI response contracts и axe; нужен установленный Playwright Chromium и свободный порт 8090.

Source API contracts: `scripts/build_contracts.py`; после изменения запускать generator и проверки `make contracts`. Source traceability: `contracts/traceability.json`; матрицу обновлять через `contracts/traceability.json` и `python3 scripts/traceability.py`. Применённые SQL migrations не изменять. Generated files — только через source/codegen. После каждого шага обновлять wiki с контекстом, проблемой, решением, data flow, примерами, ограничениями и результатами проверки.
