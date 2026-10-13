# FileX

[![CI](https://github.com/elnemir/p2p-files/actions/workflows/ci.yml/badge.svg)](https://github.com/elnemir/p2p-files/actions/workflows/ci.yml) [![Security](https://github.com/elnemir/p2p-files/actions/workflows/security.yml/badge.svg)](https://github.com/elnemir/p2p-files/actions/workflows/security.yml)

Приватная передача файлов: tus upload, неизменяемая публикация, ссылки с PIN/password, одноразовая выдача файла или ZIP, зашифрованные Local/S3 хранилища, уведомления, отчёты, exports и управляемое восстановление. Интерфейс RU/EN, light/dark/system; [настраиваемый брендинг](wiki/white-label.md): название, логотипы/favicon, палитры и стили иконок через ADMIN. [Корпоративный вход SAML/OIDC/LDAP](wiki/federated-auth.md) с настраиваемыми провайдерами, регистрацией USER при первом входе и явной привязкой аккаунта.

Вся документация — в **[wiki](wiki/index.md)**. [Текущее состояние](wiki/current-state.md) отделяет реализованное и проверенное от оставшейся приёмки. Фазы 1–6 и технический набор проверок фазы 7 реализованы; внешнее окружение не подготовлено, production запуск блокируется.

Для знакомства с интерфейсом: **[скрин-тур и видео](wiki/screen-tour.md)** · [интерактивная галерея](wiki/tour/index.html).

```sh
npm ci
python3 scripts/init-dev.py
make up
```

Локальный адрес: http://localhost:8090. Необходимые инструменты, первый ADMIN и остановка без удаления данных — [локальный запуск](wiki/development.md). Перед изменениями прочитать [AGENTS](AGENTS.md), [архитектуру](wiki/architecture.md), [план дальнейшей разработки](wiki/implementation-plan.md) и [workflow](wiki/development-workflow.md).

[Требования](wiki/requirements.md) · [OpenAPI](contracts/openapi.json) · [матрица](wiki/test-matrix.md) · [CLI](wiki/cli.md) · [проверки](wiki/testing.md) · [backup/restore](wiki/operations-recovery.md) · [внешняя приёмка](wiki/environment-acceptance.md).

PostgreSQL — источник истины, Redis — ephemeral. Код proprietary; лицензии зависимостей ещё требуют review. Production deployment выполняется только по отдельной команде владельца.

[Правила участия](CONTRIBUTING.md) · [Сообщить об уязвимости](SECURITY.md) · [Настройка репозитория](wiki/repository-governance.md).

Нативный WebDAV: [подключение macOS/Linux/Windows и статус проверки](wiki/webdav-upload.md). Windows ожидает стенда. [Скрин-тур и единый MP4](wiki/screen-tour.md) включают этот workflow.
