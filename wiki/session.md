# Session State

## Task
Реализовать авторизацию и регистрацию через SAML 2.0, OpenID Connect и LDAP с настройками ADMIN. Владелец указал Keycloak, Microsoft Entra ID/AD, Google Workspace и другие IdP/LDAP.

## Status
Локальная реализация и квалификация завершены в `codex/federated-auth` от main `73bf030`. **13/13 gates PASS** на неизменном source SHA-256 `70ada47d3ef41da8300ef649282149af202abb915aebfbdf9b0699cc0a3c80fb`. Владелец 2026-10-13 разрешил commit, push и merge. Начата доставка: source и все 13 gate records повторно сверены, origin/main остаётся 73bf030; посторонних изменений нет. GitHub CI и merge ещё ожидаются; production deployment не выполняется. [Подробное evidence](history/federated-auth-evidence-2026-10-13.md). Предыдущая доставка: [white labeling](sessions/2026-10-13-white-label-delivered.md).

## Completed steps
- Прочитаны AGENTS, нормативная wiki и auth/session/keyring/recovery. Подготовлены ADR-014, threat/state machines, architecture consistency/crypto review, OpenAPI/JSON Schema до основной реализации; AsyncAPI не расширяется auth событиями.
- Migration 0019: providers/revision/encrypted secrets, provider-subject bindings, browser-bound expiring flows, provider sessions. OIDC Code+PKCE/nonce, SAML signed SP flow, LDAP TLS/search/bind.
- USER-only JIT, запрет auto-link по email, explicit fresh-account link, same-identity reauth, local role/disabled/auth_epoch authority, CAS и provider session revoke. Локальный ADMIN fallback сохранён.
- ADMIN UI, corporate login/link/reauth, JSON contract, strict config parsing, trusted origins/CIDR/CA и bounded outbound. Recovery/reconcile/rotation/cleanup включают новые данные.
- 61 PG/race cases без skips, 45 browser cases в пяти профилях без skipped/unexpected/flaky, protocol TLS/signature negatives, concurrency и audit fault PASS. Upgrade 16→19, ingress, fuzz, compatibility, verifier и performance smoke PASS.
- Найденные AAD, connection-pool, select accessibility, migration fixture, saved-session reload и long-name overflow проблемы исправлены; RED evidence сохранено. Screenshot с длинным именем визуально просмотрен.
- README, профильная wiki, current state, contracts/types/schema, traceability, changelog, security/license review и operational guide актуализированы.

## Next steps
- Выполнить разрешённые commit/push, создать PR и проверить exact-head CI/security; squash merge только после зелёных checks. Затем сверить main и post-merge workflows.
- При получении параметров организаций настроить провайдеры по [руководству](federated-auth.md), выполнить внешнюю приёмку claims/TLS/clock/certificate rotation/disabled account/emergency login.
- Сохранять прежние release blockers и отдельное требование production deployment команды владельца.

## Decisions
PG authoritative; протокольные библиотеки и существующий configuration DEK, без новых crypto primitives. Роли FileX не импортируются. Секреты write-only. Provider authority immutable, обновление отзывается через revision. Browser completion атомарен с JIT/session/audit. Обычный OIDC login без auth_time допустим без fresh reauth; privileged reauth требует свежий auth_time. SAML request signing/encryption/SLO, SCIM и group mapping вне текущего профиля.

## Context
[SSO guide](federated-auth.md), [ADR-014](adr/014-federated-auth.md), [review](design-review.md), [evidence](history/federated-auth-evidence-2026-10-13.md), [workflow](development-workflow.md). Raw logs `.cache/identity/`; все 13 records `.cache/evidence/gates/`. Performance — 32 MiB / 2000 rows smoke, не release-load.

## Risks / blockers
Реальные tenant/DC, endpoints/metadata/CA и тестовые аккаунты не предоставлены: локальные fixtures не означают внешнюю приёмку. GO-2026-5932 UNKNOWN остаётся видимым; FileX govulncheck 0 affected symbols/imported packages, сторонний Traefik binary отдельно не квалифицирован. License inventory не равен owner acceptance. Windows WebDAV ожидает стенда. Production guard и прежние внешние release blockers сохранены; фото/видеотуры этой задачей не пересняты.
