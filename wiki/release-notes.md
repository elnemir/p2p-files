# FileX — кандидат фазы 7, не опубликованный релиз

## Изменения
Технический объём фазы 7 включает HTTPS/ACME configuration generator и trusted proxy/CIDR policy, controlled upgrade/drain, строгий migration prefix, типы UI из OpenAPI, staticcheck/ESLint/formatting, compatibility diff, fuzz campaigns, browser/visual/axe matrix, load/fault/DR harness, SBOM/license/secret/container scans и source-bound acceptance gate.

Runtime images очищены от уязвимых компонентов: web package managers удалены, PostgreSQL gosu заменён su-exec, OS libraries и Traefik Go/x/net исправлены. Production configuration теперь требует immutable digest каждого из пяти проверенных образов. Один API image используется API/worker/CLI/verifier.

## Совместимость и обновление
HTTP v1, AsyncAPI и versioned JSON Schemas не меняют существующий контракт. Новых SQL migrations в этом завершающем срезе нет: текущая схема содержит 16 migrations. Контроль префикса и image upgrade описаны в [runbook](https://github.com/elnemir/p2p-files/blob/main/wiki/controlled-upgrade.md); после DDL не предполагается безопасный автоматический downgrade. Key/payload/backup formats и одноразовая выдача сохранены.

## CLI review bundle
`scripts/release_bundle.py` создаёт `.cache/evidence/release/`: пять платформенных `filex-verify`, `SHA256SUMS`, Go build metadata, manifest с commit/source SHA-256 и статусом `unpublished_review_bundle`. Все бинарники cross-built, это не заявление о запуске на всех пяти OS/architectures. По умолчанию verifier проверяет signed metadata; `--verify-files` также читает и хэширует payload. Trusted PEM и fingerprint получают независимым каналом.

Пример: `filex-verify --public-key trusted.pem --fingerprint <sha256-spki> --verify-files transfer.zip`. Проверка собственных SHA256SUMS bundle не заменяет независимую аутентификацию поставщика.

## Приёмка и ограничения
Результаты — [история evidence](https://github.com/elnemir/p2p-files/blob/main/wiki/history/development-evidence-2026-10-11.md), актуальное состояние — [session](https://github.com/elnemir/p2p-files/blob/main/wiki/session.md). Владелец подтвердил, что внешнее окружение ещё не подготовлено. Реальный host/DNS/ACME, SMTP/S3, external backups/KEK/checkpoints, target load/DR, настоящие browser/device/assistive technology, independent security, license decisions и R-01–06 остаются открытыми. `make release-check` должен блокировать публикацию, пока они не закрыты. Production startup guard сохранён, автоматической публикации/tag/deployment нет.
