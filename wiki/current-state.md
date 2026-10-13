# Текущее состояние FileX

## Контекст и проверяемая граница
Срез 2026-10-12: рабочая ветка `codex/webdav-upload`, baseline `28ab4f3`. Фазы 1–6 реализованы, технический набор фазы 7 ранее квалифицирован на `7dce226`. Его измерения сохранены в [истории](history/development-evidence-2026-10-11.md), результаты нового diff — в [session](session.md). Наличие кода, локальный тест, реальный клиент, CI и production приёмка — разные границы.

## Реализовано
- Identity/USER/ADMIN/scopes, settings/audit, tus/tree/publish, Local/S3, one-time Range/ZIP, freeze/forensic, crypto/rotation.
- SMTP/webhooks/outbox, уведомления, immutable reports, exports/QR/artifacts, saved views/bulk.
- Backup/restore с verified pre-backup, generation/drain/reconciliation/reopen, KEK recovery, checkpoints, operations UI/CLI.
- TLS/ACME configuration, ingress policy, controlled upgrade, contracts/types, scanners/fuzz/browser/load/DR и fail-closed release evidence.
- [Native WebDAV](webdav-upload.md): owner draft чтение, условная атомарная замена, PG locks/If, metadata, encrypted staging/cleanup, инструкции трёх ОС. Publish/recipient semantics сохранены.

Контракт: **108 REST/tus операций + WebDAV wire profile; 17 migrations; 11 ADR**. Redis ephemeral, PG authoritative. Scanner — расширяемый Go interface, внешний антивирусный сервис не настроен. Карта — [repository map](repository-map.md), потоки — [архитектура](architecture.md).

## Проверено и не подтверждено
| Область | Текущая граница | Что ещё требуется |
|---|---|---|
| Требования | 32 MUST, 27 verified/5 partial в traceability | Полная release acceptance, не автоматическая смена partial по green CI |
| Native WebDAV | macOS mount_webdav и настоящий Linux GVfs: полный файловый цикл, multi-chunk hash, server-created GET; 15 профильных HTTP/PG/race tests | Windows **ожидает стенда**; Office/KDE/davfs2 не квалифицированы |
| Общая регрессия | 13 локальных gates PASS на source из session, 51 integration и 35 browser cases; macOS/GVfs повторно PASS | Performance smoke; неповторившаяся задержка WebKit сохранена в диагностике; CI после доставки отдельно |
| Фото/видео | [48 кадров, 20 глав, Liza, единый MP4 5:56](screen-tour.md) | Это локальный synthetic Compose, не внешняя приёмка или запись Windows |
| Release/production | Внешнего окружения нет по ответу владельца; APP_ENV=production отклоняется кодом | 14 решений owner/reviewer, реальные host/providers/escrow/DR/devices/security/licenses |

Тур переснят 2026-10-12; неизменившаяся озвучка 19 глав переиспользована по exact text/hash, добавлена одна глава WebDAV. [Воспроизведение и media QA](tour-maintenance.md). Наблюдение DEV-05 остаётся: после completed сервер запрещает повтор, но refresh открытого recipient UI может сохранить старый список с общей ошибкой; [отдельный backlog](implementation-plan.md) не закрыт этой задачей.

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
