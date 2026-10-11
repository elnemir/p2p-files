# FileX

Приватная передача файлов: tus upload, immutable publish, ссылки с PIN/password, одноразовая выдача файла или ZIP, зашифрованные Local/S3 хранилища, уведомления, отчёты и exports. Интерфейс RU/EN, light/dark/system, адаптивные страницы владельца и получателя.

**Этапы 1–6 закрыты по локальным development gates.** Добавлены зашифрованные резервные копии Local/S3, обязательный backup перед restore, сверка данных и явное открытие доступа, KEK recovery, инциденты и эксплуатационные UI/CLI. Подробности: [состояние сессии](wiki/session.md), [путь разработки](wiki/roadmap.md), [воспроизведение](wiki/development.md), [восстановление](wiki/operations-recovery.md). Начата [фаза 7.1: HTTPS ingress и network policy](wiki/production-ingress.md); вся release acceptance фазы 7 остаётся открытой, production запуск блокируется.

```sh
npm ci
python3 scripts/init-dev.py
make up
```

Открыть http://localhost:8090. Создание первого ADMIN через CLI описано в [development](wiki/development.md). Для проверок: `make check`, `make integration-docker`, `make security`, `make e2e`, `make ingress`, `make upgrade-drill`. [Controlled upgrade](wiki/controlled-upgrade.md) проверяет image replacement, startup faults и downtime. Fixtures создают и удаляют только собственные тестовые containers; browser tests требуют Chromium и свободный loopback порт 8090.

Перед изменениями читать [AGENTS.md](AGENTS.md) и [wiki](wiki/index.md). [Требования](docs/requirements.md), [OpenAPI](contracts/openapi.json), [AsyncAPI](contracts/asyncapi.json), [матрица requirement→test](docs/test-matrix.md), [CLI](docs/cli.md). PostgreSQL — источник истины, Redis ephemeral. Код proprietary; production deployment требует отдельной явной команды владельца.
