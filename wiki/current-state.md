# Текущее состояние FileX

## Контекст и проверяемая граница
Срез 2026-10-14: UX USER/ADMIN доставлен через [PR #26](https://github.com/elnemir/p2p-files/pull/26), main `28a6b4a0aac2ed23c7699ec35e468f59255949b7`. Все 13 локальных gates, PR CI и main CI PASS; 62/62 PG/race и 60/60 browser. [История доставки](sessions/2026-10-14-product-ux-delivered.md) и [evidence](history/product-ux-evidence-2026-10-13.md) фиксируют точный проверенный источник. Фото/видеотур и документация обновлены на этом runtime; новый media/tooling diff прошёл отдельный check и media QA — [session](session.md).

SSO доставлен через [PR #25](https://github.com/elnemir/p2p-files/pull/25), white labeling — [PR #24](https://github.com/elnemir/p2p-files/pull/24). Фазы 1–6 и техническая часть 7 исторически квалифицированы; внешние release blockers сохраняются. Исторические результаты относятся к своим источникам: новый media/tooling diff не получает прежний repository source fingerprint автоматически.

## Реализовано
- [UX USER/ADMIN](product-ux.md): навигация и иконки, шаги публикации, пресеты/ползунок/точный срок, формы ADMIN Settings с CAS, timezone/quiet hours, подтверждения и защита смены сессии. DEV-05 закрыт локально: recipient terminal/revoke/expiry/resume имеют проверенные переходы.
- [SAML/OIDC/LDAP](federated-auth.md): провайдеры в ADMIN, корпоративный вход, USER-only регистрация, явная привязка/reauth, encrypted secrets, provider revoke, recovery/rotation.
- [White labeling](white-label.md): общий ADMIN preview/apply/reset, имя, палитры, логотипы/favicon, три стиля иконок; SSR, SMTP, immutable PDF presentation. Все 13 локальных gates PASS на source из исторического [evidence](history/white-label-evidence-2026-10-13.md).
- Identity/USER/ADMIN/scopes, settings/audit, tus/tree/publish, Local/S3, one-time Range/ZIP, freeze/forensic, crypto/rotation.
- SMTP/webhooks/outbox, уведомления, immutable reports, exports/QR/artifacts, saved views/bulk.
- Backup/restore с verified pre-backup, generation/drain/reconciliation/reopen, KEK recovery, checkpoints, operations UI/CLI.
- TLS/ACME configuration, ingress policy, controlled upgrade, contracts/types, scanners/fuzz/browser/load/DR и fail-closed release evidence.
- [Native WebDAV](webdav-upload.md): owner draft чтение, условная атомарная замена, PG locks/If, metadata, encrypted staging/cleanup, инструкции трёх ОС. Publish/recipient semantics сохранены.

Контракт: **128 REST/tus операций + WebDAV wire profile; 21 migrations; 17 ADR**. Redis ephemeral, PG authoritative. Scanner — расширяемый Go interface, внешний антивирусный сервис не настроен. Карта — [repository map](repository-map.md), потоки — [архитектура](architecture.md).

## Проверено и не подтверждено
| Область | Текущая граница | Что ещё требуется |
|---|---|---|
| UX USER/ADMIN | 13/13 local gates, 62 PG/race, 60 browser cases, 10 reviewed visual baselines; [evidence](history/product-ux-evidence-2026-10-13.md) | PR #26 merged, main CI PASS; реальные устройства/AT и внешняя приёмка отдельно |
| SAML/OIDC/LDAP | 13/13 local gates, 61 PG/race, 45 browser cases; protocol TLS/signature, JIT/link/reauth/replay/revoke/restore/rotation | PR #25 merged и main CI PASS; конкретные tenants/DC отдельно; локальные fixtures не означают внешнюю приёмку |
| White labeling | 13/13 local gates, 56 PG/race и 40 browser cases; SSR/ETag/assets/CAS/audit/restore/PDF/SMTP | PR #24 merged, PR checks PASS; реальные устройства и независимая приёмка не заявлены |
| Требования | 35 MUST, 27 verified/8 partial в traceability | Полная release acceptance, не автоматическая смена partial по green CI |
| Native WebDAV | macOS mount_webdav и настоящий Linux GVfs: полный файловый цикл, multi-chunk hash, server-created GET; 15 профильных HTTP/PG/race tests | Windows **ожидает стенда**; Office/KDE/davfs2 не квалифицированы |
| WebDAV regression | 13 локальных gates и PR #23/main CI PASS; 51 integration и 35 browser cases; macOS/GVfs повторно PASS в [исторической квалификации](sessions/2026-10-13-webdav-merged.md) | Performance smoke; неповторившаяся задержка WebKit сохранена в диагностике |
| Node 26 | Runtime/types/CI/policy, clean install и 57 toolchain tests PASS; все 13 local gates, 53 integration и 35 browser PASS на source из [архива](sessions/2026-10-13-node26-delivered.md); PR #22 и main CI PASS | Для нового diff требуется новая квалификация; это не production acceptance |
| Фото/видео | [71 кадр и MP4 7:56 / 26 глав](screen-tour.md) от 2026-10-14, ElevenLabs Liza; полный актуальный UX, SSO и branding | Локальный SAML signer; OIDC/LDAP — формы; реальные providers/devices отдельно. [Media evidence](history/tour-refresh-evidence-2026-10-14.md) |
| Release/production | Внешнего окружения нет по ответу владельца; APP_ENV=production отклоняется кодом | 14 решений owner/reviewer, реальные host/providers/escrow/DR/devices/security/licenses |

Тур переснят 2026-10-14 на runtime main `28a6b4a`: 18 неизменённых фрагментов Liza сверены по exact text/hash, восемь записаны заново. [Воспроизведение и media QA](tour-maintenance.md). В видео показаны новый terminal recipient после ZIP, отказ повторного входа, настройки draft/validate/apply, отчёт trash/restore, корпоративная регистрация USER и реальное применение бренда. Это отдельная локальная демонстрация, а не повторная квалификация всех 13 release gates.

## Пять требований с незакрытой частью

| ID | Что не закрыто | Следующая работа |
|---|---|---|
| FR-001 | Реальные устройства и ручная accessibility приёмка | EXT-05 |
| SEC-005 | Public DNS/ACME/host и независимая проверка | EXT-01, EXT-06 |
| OPS-005 | Приёмка целевого deployment и безусловный production guard | DEV-01, EXT-01, REL-01 |
| NFR-003 | Полный пакет release evidence | EXT-01–06, REL-01 |
| NFR-004 | Лицензии, независимый review и R-01–06 | DEV-03, EXT-06–07 |

ID задач раскрыты в [implementation plan](implementation-plan.md); точная связь с кодом/тестами генерируется в [matrix](test-matrix.md). Не менять partial на verified только по green CI или существованию runbook.

## Блокеры и ограничения

Владелец подтвердил: «Окружение ещё не подготовлено; зафиксировать внешние блокеры». Все 14 записей [release-acceptance.json](../contracts/release-acceptance.json) pending. Адреса, аккаунты, сроки, ответственные лица и SLA не придуманы. [План приёмки](environment-acceptance.md) позволяет подготовить их без повторного проектирования.

`make release-check` только проверяет полноту, не включает production. `internal/config/config.go` принимает только development/test. Проектирование замены этого guard — DEV-01; запуск реальной инфраструктуры и production deploy требуют отдельной команды владельца. Подмена APP_ENV на development на production сервере не является закрытием задачи.

## Лицензирование — локальный diff 2026-10-14

Утверждённая модель subscription/perpetual реализована: клиент проверяет JWS отдельным vendor Ed25519 public key без сети; PostgreSQL сериализует import/revision/epoch, user и transfer quota. Истечение закрывает новые платные операции, сохраняя lifecycle существующих передач. Панель издателя, её PostgreSQL и signer развёртываются отдельно; private key отсутствует в клиентских образах и web API. FR-017/SEC-008 добавлены к исходным 32 требованиям. [Runbook](licensing-operations.md) описывает конфигурацию, поток данных, проверки и ограничения; прежние внешние release blockers сохраняются.

Локальная проверка licensing завершена на source `94cd90bc4595599ca880ad7596c478292f76f0ee1e4a11e33f86a3f420346cbc`: recorded make check PASS; 13 Chromium + issuer browser PASS; PG negative/concurrency/fault/recovery/restore проверки и scans описаны с границами в [session](session.md). Это не production допуск и не закрытие внешних release blockers.

## Веб-мастера первого запуска — 2026-10-14

[FileX /setup и issuer /issuer/setup](first-run-setup.md) настраивают уже запущенные контейнеры. Временный host capability, atomic ADMIN completion, immutable encrypted issuer OIDC config; private signing key в веб не передаётся. FR-018: локальный scope реализован, external MFA/TLS/security acceptance отдельно. Актуальный hash/checks — [session](session.md).
