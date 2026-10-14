# Session State

## Task
Реализовать платное офлайн-лицензирование FileX и закрытую веб-панель генератора: subscription/perpetual, отдельный vendor Ed25519, лимиты включённых пользователей, размера передачи и функций; утверждено владельцем 2026-10-14.

## Status
Локальная реализация и заключительная проверка завершены. Изменения не закоммичены, не отправлены и не развёрнуты. Production guard FileX сохранён.

## Completed steps
- До кода подготовлены модель угроз, состояния, crypto/architecture review и ADR-016; прочитаны AGENTS и обязательный wiki context.
- Strict JWS/Ed25519 verifier: exact bytes, algorithm/type/key allowlist, обязательные поля с точным регистром, отсутствие duplicate/unknown keys, installation binding и monotonic revisions.
- Migration 0020 и PostgreSQL guards: user seats, transfer reservations/chunks/replacement/rename, publish, WebDAV, federation, branding, write tokens и новые export jobs. Expiry сохраняет старую recipient delivery/TTL/one-time/revoke/cleanup.
- ADMIN status/preview/import с reauth/CSRF/epoch CAS; host CLI с maintenance shared lock. Bootstrap допускает recovery на прежней схеме; pending migrations требуют update entitlement до DDL.
- Отдельные issuer OIDC MFA/allowlist/roles, web UI с preview/issue/history/download/renewal/future start, durable outbox и изолированный signer. DB roles и immutable history проверены. Private seed не входит в customer image или web API.
- Generated customer/issuer OpenAPI, issuer AsyncAPI, license schema; FR-017/SEC-008 traceability. Error enum v1 сохранён. Добавлены runbook, keygen/sign-release tooling, отдельные issuer Dockerfiles/Compose и signed qualification fixtures.
- Реальным backup/restore проверено сохранение installation ID, epoch, high revision и max observed time; старая лицензия после restore отклоняется.
- Визуально проверены issuer desktop/mobile и customer mobile. Исправлены title, узкая таблица, отступы импорта и человекочитаемые состояния/функции/даты UTC.

## Decisions
Подтверждённые модель/ключи/expiry/limits не требуют нового согласования. PostgreSQL authoritative, Redis не источник прав; license ключ не шифрует данные. Trial/grace и сетевой activation не введены. Пользователи при downgrade не удаляются; новые места/увеличение передачи блокируются. Будущая лицензия сохраняется pending и заменяет текущую по notBefore.

## Validation
Baseline HEAD: `4215c0a7953eca30422b9681cd33789f9993ebe1`. Финальный source hash (штатный scripts/acceptance.py, без Markdown evidence): `94cd90bc4595599ca880ad7596c478292f76f0ee1e4a11e33f86a3f420346cbc`.

- `make check` PASS на финальном hash 94cd90bc, sourceUnchanged=true, exitCode=0; record `.cache/evidence/gates/check.json`, лог `.cache/licensing-check-accepted.log`.
- Полный PG suite PASS (`.cache/licensing-integration-final.log`); последующие изменения verifier/bootstrap/error mapping прошли повторный licensing/issuer/migration набор (`.cache/licensing-pg-verified.log`, `.cache/licensing-pg-final.log`). Финальные изменения после этих прогонов — только browser test/traceability, копирайтинг и CSS.
- Отдельный TestIntegrationLicenseHighWaterSurvivesRestore PASS (`.cache/licensing-restore.log`). Подмена, quota races, last user race, signer audit fault, OIDC weak/stale/unknown/nonce/CSRF, DB role isolation и повреждённая лицензия при startup — PASS.
- Финальные 13/13 Chromium (1.8 минуты) плюс issuer workflow PASS (`.cache/licensing-browser-accepted.log`). Customer mobile screenshot просмотрен: отступы, локализованные состояния/функции и UTC даты корректны; overflow/axe PASS. Issuer browser использует настоящий API/signer и синтетический TLS IdP, не реальный MFA провайдер.
- Fuzz: шесть bounded campaigns PASS; после строгого регистра отдельно FuzzClaims 15 секунд / 480424 входа PASS. Примитивы Ed25519 стандартные; независимая Node crypto подпись проверена customer browser import.
- v1 compatibility, generated contracts/types и docs-check PASS: 99 Markdown, 34 requirements. `git diff --check` PASS.
- npm audit: 0; govulncheck: 0 вызываемых уязвимостей. Source scan: no leaks, 0 HIGH/CRITICAL, один UNKNOWN GO-2026-5932 в неиспользуемом уязвимом пути x/crypto. Все семь runtime images (включая issuer/issuer-web) scan: 0 HIGH/CRITICAL (`.cache/licensing-images.log`). Image scan относится к зафиксированным в отчёте digest, не является release qualification финального кандидата после последующих правок.

Скриншоты: `.cache/licensing-issuer-browser/`, `.cache/evidence/browser/chromium/license-mobile.png`. Логи и ключи fixtures находятся в игнорируемой `.cache`; реальные vendor secrets не создавались.

## Next steps
Авторизованная локальная реализация завершена. Для production отдельно нужны реальный IdP/MFA, vendor/release keys и escrow, внешние DEV/EXT/REL acceptance decisions, независимый security review и явная команда deployment. Полные пять browser profiles, ingress/upgrade/performance/release bundle на окончательном релизном кандидате в этой задаче не объявлены PASS.

## Context
[Модель](../licensing.md), [ADR-016](../adr/016-offline-licensing.md), [runbook](../licensing-operations.md), [карта кода](../repository-map.md), [matrix](../test-matrix.md). Предыдущая сессия: [архив](../sessions/2026-10-14-before-licensing.md). Изменены internal/licensing, internal/issuer, shared domain guards, bootstrap/CLI, apps/web и apps/issuer, migration/contracts/deploy/test harnesses.

## Risks / blockers
Офлайн подпись защищает от подмены файла, но не от root-патча binary, клона всей VM/БД и полного rollback. Мгновенный offline revoke невозможен. Эти остаточные риски не приняты за владельца. CRM/платежи/revocation bundles/migration grants/clock recovery grants/web key rotation не реализованы и явно отделены от текущего scope в runbook. Исходные внешние release blockers сохраняются; независимый криптоаудит не выполнен.
