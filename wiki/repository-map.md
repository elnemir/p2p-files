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
| [apps/web/components](../apps/web/components/) | FileX, AdminPanel, ApplicationSettings, Controls, WebDAVPanel, OperationsPanel и AdminExportsPanel; RU/EN, themes, пользовательские workflows |

| [cmd/filex-issuer](../cmd/filex-issuer/), [cmd/filex-license-signer](../cmd/filex-license-signer/), [apps/issuer](../apps/issuer/) | Отдельный vendor контур: OIDC MFA, выпуск/история/продление, private key только у signer |

## Доменные компоненты

| Изменение | Код и данные | Проверки / документация |
|---|---|---|
| Офлайн-лицензирование | [licensing](../internal/licensing/), [issuer](../internal/issuer/), migration 0020, LicensePanel | [Модель](licensing.md), [runbook](licensing-operations.md); quota/race/fault/crypto/role/browser tests |
| Auth/roles/scopes/CSRF | [auth](../internal/auth/), [httpapi/app.go](../internal/httpapi/app.go), identity migrations | app/closure integration, [security](security.md) |
| Typed settings/ingress | [config](../internal/config/), [settings](../internal/settings/), [networkpolicy](../internal/networkpolicy/), httpapi/network.go | config/network unit и network integration, [ingress](production-ingress.md) |
| Draft/publish/tree/retention | [domain](../internal/domain/), [transfer](../internal/transfer/), httpapi/transfers/tree/organization | repository/retention/closure integration, [core design](core-design.md), [states](state-machines.md) |
| Upload/storage/download | [files](../internal/files/), [storage](../internal/storage/), httpapi/recipients/transfers/adminfiles/forensic | files/migration/provenance integration, [file delivery](file-delivery.md) |
| WebDAV upload/credentials/XML | [webdav.go](../internal/httpapi/webdav.go), [tree adapter](../internal/httpapi/webdav_tree.go), [XML](../internal/httpapi/webdav_xml.go), [WebDAVPanel](../apps/web/components/WebDAVPanel.tsx) | PG/race/fault/client/browser tests, [контракт и подключение](webdav-upload.md), [ADR-011](adr/011-native-webdav.md), [native client checks](../scripts/webdav/) |
| Crypto/ZIP/verifier | [cryptobox](../internal/cryptobox/), [keyring](../internal/keyring/), [bundle](../internal/bundle/), [verify](../internal/verify/) | vectors/tamper/fuzz/Range/rotation, [FXP1](encryption-format.md), [keys](key-management.md) |
| Reports/artifacts/exports | [reports](../internal/reports/), [artifacts](../internal/artifacts/), [exports](../internal/exports/), httpapi/reports/artifacts/exports/qr | reports/provenance/closure integration, [ADR-003](adr/003-export-snapshot.md) |
| Notifications/outbound | [communications](../internal/communications/), httpapi/communications, [worker](../internal/worker/) | SMTP TLS/SSRF/replay, communications/websocket integration |
| Operations/recovery/audit | [operations](../internal/operations/), [maintenance](../internal/maintenance/), [audit](../internal/audit/), httpapi/operations/adminexports | operations/upgrade integration, [recovery](operations-recovery.md), [incidents](incident-response.md) |
| Schema/bootstrap | [database](../internal/database/), [migrations](../internal/database/migrations/), [bootstrap](../internal/bootstrap/) | migration prefix/DDL rollback/race, [data model](data-model.md), [upgrade](controlled-upgrade.md) |

`httpapi/organization.go` обслуживает личную организацию списков/папок/views/bulk, а не организационную tenancy. Имена пакетов не должны становиться придуманным продуктовым scope.

## Контракты, инструменты и поставка

| Источник | Производный результат / потребитель |
|---|---|
| [scripts/build_contracts.py](../scripts/build_contracts.py) | contracts/openapi.json, webdav.json, asyncapi.json, schemas; затем `npm run generate:types` → apps/web/lib/api-schema.ts |
| [contracts/traceability.json](../contracts/traceability.json) | `python3 scripts/traceability.py` → wiki/test-matrix.md |
| [contracts/release-acceptance.json](../contracts/release-acceptance.json) | Pending/accepted решения reviewer/owner → release readiness; не приложение |
| [scripts/tour](../scripts/tour/) | Изолированный capture → plan.json/validate.py проверяют полноту → точный текст и SHA-256 кеша ElevenLabs Liza → MP4/PNG/VTT/manifest/галерея и [скрин-тур](screen-tour.md). Source templates и [правила пересъёмки/QA](tour-maintenance.md); API cache и локальная Milena остаются отдельными вариантами |
| [scripts/acceptance.py](../scripts/acceptance.py), [Makefile](../Makefile) | 13 source-bound gates → .cache/evidence/gates; [testing](testing.md) |
| [deploy](../deploy/), [production_config.py](../scripts/production_config.py) | Пять runtime images, dev Compose и offline production config; no deployment |
| [.github/dependency-policy.json](../.github/dependency-policy.json), [check_dependencies.mjs](../scripts/check_dependencies.mjs) | Reviewed Node/Redis timeline → engine/types/images/installed graph guard; [ADR-012](adr/012-node26-runtime.md), [negative tests](../scripts/dependencies.test.mjs) |
| [CI](../.github/workflows/ci.yml), [security](../.github/workflows/security.yml) | Functional/supply-chain/repository-policy, CodeQL/dependency review; artifacts 14 дней |
| [.github/repository.json](../.github/repository.json), [github_repository.py](../scripts/github_repository.py) | Desired GitHub settings → audit/apply/readback; [правила репозитория](repository-governance.md) |
| [scripts/e2e](../scripts/e2e/), [playwright.config.ts](../playwright.config.ts) | 60 browser scenarios, 10 Linux baselines; review перед заменой изображений |

## Поток изменения и пример

Для исправления export после демоции ADMIN: FR-013/014 → ADR-003 и guards → exports worker + HTTP authorization → migration только при необходимости → PG race/negative case → UI error/retry flow → contracts/types при изменении ответа → wiki/matrix/evidence. Одного изменения кнопки в UI недостаточно, если worker продолжает читать чужие данные.

Сначала продлить существующий механизм и его тест. Новый сервис/очередь/ключевой формат требует отдельного ADR и согласования затронутых инвариантов. Комментарии к коду английские, внутренняя wiki русская; schemas/contract descriptions английские.

## Новые точки native WebDAV
[webdav_put.go](../internal/httpapi/webdav_put.go) управляет staged swap; [webdav_read.go](../internal/httpapi/webdav_read.go) — owner streaming guards; [conditions](../internal/httpapi/webdav_conditions.go), [locks](../internal/httpapi/webdav_locks.go), [properties](../internal/httpapi/webdav_properties.go) — wire methods. Общие PG guards находятся в [transfer/dav.go](../internal/transfer/dav.go), expiry cleanup — [files/staging.go](../internal/files/staging.go), schema — [0017](../internal/database/migrations/0017_native_webdav.sql). Любое изменение guard проверять также на REST/tus/bulk/publish, а не только DAV route.

## Брендинг установки

[internal/branding](../internal/branding/) — defaults, validation, revision CAS/audit; [branding API](../internal/httpapi/branding.go) — публичный DTO и ADMIN routes. [BrandingPanel](../apps/web/components/BrandingPanel.tsx) — редактор, [BrandingProvider](../apps/web/components/BrandingProvider.tsx) — runtime context; SSR читает внутренний API. Migration 0018 и reports/communications интегрируют snapshots и письма. Инструкция и границы — [white labeling](white-label.md).

## SAML/OIDC/LDAP
[internal/identity](../internal/identity/) — config, encrypted provider/flow store, protocol adapters, bounded outbound TLS. [HTTP routes](../internal/httpapi/identity.go), [IdentityLogin](../apps/web/components/IdentityLogin.tsx), [ADMIN editor](../apps/web/components/IdentityProvidersPanel.tsx). [0019](../internal/database/migrations/0019_federated_auth.sql), [browser fixture](../scripts/identity-fixture/main.go), [browser test](../scripts/e2e/identity.spec.ts); [инструкция](federated-auth.md).

## UX основных сценариев
[ADR-015](adr/015-product-ux.md), [руководство](product-ux.md): общие controls/confirmation, sidebar, typed Settings v1, recipient terminal state. Проверки — [product-ux.spec.ts](../scripts/e2e/product-ux.spec.ts).

## Веб-настройка первого запуска

[Мастера запущенных контейнеров](first-run-setup.md): одноразовый ADMIN FileX и encrypted OIDC bootstrap портала; [ADR-017](adr/017-first-run-wizards.md). Docker/DB/ключи готовит host; production guard сохранён.
