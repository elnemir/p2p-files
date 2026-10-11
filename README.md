# FileX

Приватная передача файлов: tus upload, immutable publish, ссылки с PIN/password, одноразовая выдача файла или ZIP, зашифрованные Local/S3 хранилища, уведомления, отчёты и exports. Интерфейс RU/EN, light/dark/system, адаптивные страницы владельца и получателя.

**Этапы 1–5 закрыты по локальным development gates.** Подробности: [состояние сессии](wiki/session.md), [путь разработки](wiki/roadmap.md), [воспроизведение](wiki/development.md). Эксплуатационные backup/restore и release acceptance — этапы 6–7; production запуск сейчас блокируется.

```sh
npm ci
python3 scripts/init-dev.py
make up
```

Открыть http://localhost:8090. Создание первого ADMIN через CLI описано в [development](wiki/development.md). Для проверок: `make check`, `make integration-docker`, `make security`, `make e2e`. Integration/E2E создают и удаляют только собственные тестовые containers; browser tests требуют Chromium и свободный loopback порт 8090.

Перед изменениями читать [AGENTS.md](AGENTS.md) и [wiki](wiki/index.md). [Требования](docs/requirements.md), [OpenAPI](contracts/openapi.json), [AsyncAPI](contracts/asyncapi.json), [матрица requirement→test](docs/test-matrix.md), [CLI](docs/cli.md). PostgreSQL — источник истины, Redis ephemeral. Код proprietary; production deployment требует отдельной явной команды владельца.
