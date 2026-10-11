# Разработка FileX

Обязательные правила — [AGENTS](AGENTS.md). Начать с [wiki](wiki/index.md), [состояния](wiki/current-state.md) и [сессии](wiki/session.md); выбрать задачу из [плана](wiki/implementation-plan.md).

[Developer workflow](wiki/development-workflow.md) содержит порядок изменения кода/контрактов, шаблон задачи/PR и Definition of Done. [Локальный запуск](wiki/development.md), [карта компонентов](wiki/repository-map.md), [конфигурация](wiki/configuration.md), [проверки](wiki/testing.md) позволяют воспроизвести работу без истории чата.

Conventional Commits; applied migrations и generated files вручную не менять. После каждого шага обновлять тематическую wiki и session. Обязательные gates не отключать. Все проверки записывать с областью действия: unit, PostgreSQL, браузер, CI, внешний provider и production acceptance различаются. CI не выполняет deployment.

Изменения проходят через `codex/<topic>` и PR в main с обязательным CI и squash merge. [Настройка репозитория](wiki/repository-governance.md) описывает required checks, CODEOWNERS, Dependabot, GitHub settings и их проверку. Об уязвимостях сообщать по [SECURITY.md](SECURITY.md), без публичных секретов и пользовательских данных.
