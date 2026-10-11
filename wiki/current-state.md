# Текущее состояние FileX

## Контекст и основание

Срез 2026-10-11 сверяет документацию с исходниками, Makefile, workflow, 16 SQL migrations, OpenAPI и машиночитаемой traceability. Базовый HEAD до переноса документации — `21aaa00c34ddbe7511ca950f1aacec2454989047`; реализация фазы 7 — `7dce226f1dffa7241267e3546371ad7b2665c423`. CI исходного документационного HEAD также завершился успешно: [#38146011717](https://github.com/elnemir/p2p-files/actions/runs/38146011717), оба jobs. Это проверка исходного HEAD; проверки самого переноса записаны в [сессии переноса](sessions/2026-10-11-documentation-migration.md).

Для знакомства с работающим интерфейсом подготовлен [скрин-тур и видеотур](screen-tour.md). Съёмка выполняется на отдельном локальном стенде с синтетическими данными; она не меняет статусы внешней приёмки. Сценарий и инструменты пересъёмки описаны в [руководстве](tour-maintenance.md).

Наблюдение тура DEV-05: после полного ZIP сервер переводит передачу в completed и отклоняет recipient API, но refresh уже открытой страницы может оставить устаревший ready-список рядом с общим сообщением об отказе. Исправление клиентского терминального состояния описано в [плане](implementation-plan.md); серверная защита повторной выдачи действует.

## Реализовано

- Платформа, identity/USER/ADMIN/scopes, настройки, audit, tus/дерево/publish, Local/S3, одноразовая Range/ZIP выдача, freeze/forensic, crypto/rotation.
- SMTP/webhooks/outbox, уведомления, immutable отчёты, exports/QR/artifacts, saved views/bulk.
- Backup/restore с обязательной проверенной предварительной копией, generation fencing, reconciliation/reopen, KEK recovery, incidents/checkpoints и эксплуатационные UI/CLI.
- HTTPS/ACME configuration generator, proxy/CIDR guards, controlled upgrade и технические проверки фазы 7: static analysis, compatibility, fuzz, browser/visual/a11y, load/DR, security/SBOM/licenses, source-bound readiness и verifier review bundle.

Карта компонентов — [repository map](repository-map.md), потоки — [архитектура](architecture.md). Redis развёрнут как ephemeral сервис; authoritative очереди и correctness реализованы в PostgreSQL. Внешний scanner — точка расширения Go, действующий антивирусный сервис не настроен.

## Проверено и не подтверждено

| Область | Подтверждённая граница | Что ещё требуется |
|---|---|---|
| Требования/контракты | 32 MUST; 27 verified/5 partial; 107 HTTP операций; 16 migrations; 9 ADR | Все partial закрыть фактической приёмкой без исключения MUST |
| Кандидат 7dce226 | 13 local gates и [CI #38144319522](https://github.com/elnemir/p2p-files/actions/runs/38144319522) PASS; 36 PG/race tests | Новый diff требует своих проверок; старый PASS не переносится на новый source hash |
| Браузеры | 25 scenarios, 5 Linux engine/emulation profiles, 10 visual baselines, axe/keyboard/320px | Реальные Safari/mobile/assistive technology и ручная WCAG приёмка |
| Load/DR | 2050 MiB/20000 synthetic rows, 15 assertions, ZIP SHA, 0 OOM, local TLS S3 | Целевой VPS/провайдер, естественное ожидание crash lease, внешнее восстановление и измерение RPO/RTO |
| Security | На указанном кандидате source и 5 runtime images: 0 HIGH/CRITICAL; 3 fuzz campaigns | Свежий scan, независимый review и triage видимого module-only UNKNOWN advisory |
| Лицензии | Inventory: 443 npm entries, 24 runtime Go libraries, font SHA, 16 записей для review | Решение владельца по поставке/обязательствам, OS/image/fonts и лицензии самого проекта |
| Выпуск/эксплуатация | Генератор и unpublished verifier bundle существуют | Нет опубликованного release, принятого production окружения или разрешения deploy |

Подробные измерения и hashes — [история evidence](history/development-evidence-2026-10-11.md), [сессия фазы 7](sessions/2026-10-11-phase7.md). Это датированные результаты. `.cache/evidence` не входит в Git, CI хранит artifacts 14 дней; ссылка в wiki не гарантирует бессрочную доступность raw logs.

## Пять требований с незакрытой частью

| ID | Что не закрыто | Следующая работа |
|---|---|---|
| FR-001 | Реальные устройства и ручная accessibility приёмка | EXT-05 |
| SEC-005 | Public DNS/ACME/host и независимая проверка | EXT-01, EXT-06 |
| OPS-005 | Приёмка целевого deployment и безусловный production guard | DEV-01, EXT-01, REL-01 |
| NFR-003 | Полный пакет release evidence | EXT-01–06, REL-01 |
| NFR-004 | Лицензии, независимый review и R-01–06 | DEV-03, EXT-06–07 |

ID задач раскрыты в [implementation plan](implementation-plan.md); точная связь с кодом/тестами генерируется в [matrix](test-matrix.md). Не менять partial на verified только по green CI или существованию runbook.

## Изменено, добавлено, требует доработки

**Изменено:** 20 документов перенесены в wiki; entrypoints, generators и CLI ссылаются на новый путь. Исторические измерения и первоначальный аудит отделены от текущих инструкций. Исправлены устаревшие указания о фазе 7, количестве E2E и необходимости устанавливать браузер на host.

**Добавлено:** карта компонентов, конфигурационный справочник, developer workflow, план задач с зависимостями, внешняя acceptance matrix и шаблоны записи результатов. Продуктовые требования и runtime поведение не расширялись.

**Требует доработки:** reviewed механизм production допуска, сопровождение ESLint/CI/image pins, решение лицензирования и сохранение release evidence вне 14-дневных CI artifacts. Это backlog, не скрытая реализация.

## Блокеры и ограничения

Владелец подтвердил: «Окружение ещё не подготовлено; зафиксировать внешние блокеры». Все 14 записей [release-acceptance.json](../contracts/release-acceptance.json) pending. Адреса, аккаунты, сроки, ответственные лица и SLA не придуманы. [План приёмки](environment-acceptance.md) позволяет подготовить их без повторного проектирования.

`make release-check` только проверяет полноту, не включает production. `internal/config/config.go` принимает только development/test. Проектирование замены этого guard — DEV-01; запуск реальной инфраструктуры и production deploy требуют отдельной команды владельца. Подмена APP_ENV на development на production сервере не является закрытием задачи.
