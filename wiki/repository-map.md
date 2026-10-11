# Карта кода и точек изменения

## Контекст и проблема

Страница связывает задачу с существующим компонентом, тестами и wiki. Это позволяет продолжать вертикальные срезы без параллельной архитектуры. Источник истины требований — [requirements](requirements.md); конкретные test IDs генерируются в [matrix](test-matrix.md).

## Исполняемые точки входа

| Путь | Ответственность |
|---|---|
| [cmd/filex-api](../cmd/filex-api/) | Bootstrap, HTTP API и отдельный internal listener |
| [cmd/filex-worker](../cmd/filex-worker/) | Background processing, scheduler, generation guards |
| [cmd/filex-cli](../cmd/filex-cli/) | Доверенная host/container граница, provisioning/settings/операции, JSON stdin |
| [cmd/filex-verify](../cmd/filex-verify/) | Offline проверка ZIP manifest/signature; payload SHA только с --verify-files |
| [apps/web/app](../apps/web/app/) | Next.js страницы owner/recipient/help/status/verify; layout/styles |
| [apps/web/components](../apps/web/components/) | FileX, OperationsPanel и AdminExportsPanel; RU/EN, themes, пользовательские workflows |

## Доменные компоненты

| Изменение | Код и данные | Проверки / документация |
|---|---|---|
| Auth/roles/scopes/CSRF | [auth](../internal/auth/), [httpapi/app.go](../internal/httpapi/app.go), identity migrations | app/closure integration, [security](security.md) |
| Typed settings/ingress | [config](../internal/config/), [settings](../internal/settings/), [networkpolicy](../internal/networkpolicy/), httpapi/network.go | config/network unit и network integration, [ingress](production-ingress.md) |
| Draft/publish/tree/retention | [domain](../internal/domain/), [transfer](../internal/transfer/), httpapi/transfers/tree/organization | repository/retention/closure integration, [core design](core-design.md), [states](state-machines.md) |
| Upload/storage/download | [files](../internal/files/), [storage](../internal/storage/), httpapi/recipients/transfers/adminfiles/forensic | files/migration/provenance integration, [file delivery](file-delivery.md) |
| Crypto/ZIP/verifier | [cryptobox](../internal/cryptobox/), [keyring](../internal/keyring/), [bundle](../internal/bundle/), [verify](../internal/verify/) | vectors/tamper/fuzz/Range/rotation, [FXP1](encryption-format.md), [keys](key-management.md) |
| Reports/artifacts/exports | [reports](../internal/reports/), [artifacts](../internal/artifacts/), [exports](../internal/exports/), httpapi/reports/artifacts/exports/qr | reports/provenance/closure integration, [ADR-003](adr/003-export-snapshot.md) |
| Notifications/outbound | [communications](../internal/communications/), httpapi/communications, [worker](../internal/worker/) | SMTP TLS/SSRF/replay, communications/websocket integration |
| Operations/recovery/audit | [operations](../internal/operations/), [maintenance](../internal/maintenance/), [audit](../internal/audit/), httpapi/operations/adminexports | operations/upgrade integration, [recovery](operations-recovery.md), [incidents](incident-response.md) |
| Schema/bootstrap | [database](../internal/database/), [migrations](../internal/database/migrations/), [bootstrap](../internal/bootstrap/) | migration prefix/DDL rollback/race, [data model](data-model.md), [upgrade](controlled-upgrade.md) |

`httpapi/organization.go` обслуживает личную организацию списков/папок/views/bulk, а не организационную tenancy. Имена пакетов не должны становиться придуманным продуктовым scope.

## Контракты, инструменты и поставка

| Источник | Производный результат / потребитель |
|---|---|
| [scripts/build_contracts.py](../scripts/build_contracts.py) | contracts/openapi.json, asyncapi.json, schemas; затем `npm run generate:types` → apps/web/lib/api-schema.ts |
| [contracts/traceability.json](../contracts/traceability.json) | `python3 scripts/traceability.py` → wiki/test-matrix.md |
| [contracts/release-acceptance.json](../contracts/release-acceptance.json) | Pending/accepted решения reviewer/owner → release readiness; не приложение |
| [scripts/tour](../scripts/tour/) | Изолированный capture → реальные PNG/видеозаписи → Milena или narrate.py (ElevenLabs cache) → озвученный MP4, manifest, галерея и [скрин-тур](screen-tour.md); [пересъёмка и QA](tour-maintenance.md) |
| [scripts/acceptance.py](../scripts/acceptance.py), [Makefile](../Makefile) | 13 source-bound gates → .cache/evidence/gates; [testing](testing.md) |
| [deploy](../deploy/), [production_config.py](../scripts/production_config.py) | Пять runtime images, dev Compose и offline production config; no deployment |
| [CI](../.github/workflows/ci.yml), [security](../.github/workflows/security.yml) | Functional/supply-chain/repository-policy, CodeQL/dependency review; artifacts 14 дней |
| [.github/repository.json](../.github/repository.json), [github_repository.py](../scripts/github_repository.py) | Desired GitHub settings → audit/apply/readback; [правила репозитория](repository-governance.md) |
| [scripts/e2e](../scripts/e2e/), [playwright.config.ts](../playwright.config.ts) | 25 browser scenarios, 10 Linux baselines; review перед заменой изображений |

## Поток изменения и пример

Для исправления export после демоции ADMIN: FR-013/014 → ADR-003 и guards → exports worker + HTTP authorization → migration только при необходимости → PG race/negative case → UI error/retry flow → contracts/types при изменении ответа → wiki/matrix/evidence. Одного изменения кнопки в UI недостаточно, если worker продолжает читать чужие данные.

Сначала продлить существующий механизм и его тест. Новый сервис/очередь/ключевой формат требует отдельного ADR и согласования затронутых инвариантов. Комментарии к коду английские, внутренняя wiki русская; schemas/contract descriptions английские.
