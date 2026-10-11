# FileX: база знаний и вход в разработку

FileX — приватная передача файлов: владелец публикует неизменяемый набор, получатель явно получает одноразовую серверную выдачу файла или ZIP. PostgreSQL хранит права и состояние, Local/S3 — зашифрованные объекты. Пользовательские и административные данные разделены. Организационная tenancy и billing в продукт не добавлены.

Вся проектная документация находится в `wiki/`. Машиночитаемые API/event/schema/traceability остаются в [contracts](../contracts/); код и генераторы не переносились. Корневые README/CONTRIBUTING/MEMORY служат короткими указателями, [AGENTS](../AGENTS.md) задаёт обязательные правила. Актуализация: 2026-10-11.

## С чего начать

1. [Текущее состояние и границы проверки](current-state.md), [текущая сессия](session.md).
2. [Требования](requirements.md), [архитектура](architecture.md), [решения](decisions.md).
3. [Дорожная карта](roadmap.md), [задачи продолжения и зависимости](implementation-plan.md).
4. [Локальный запуск](development.md), [карта репозитория](repository-map.md), [правила выполнения задачи](development-workflow.md).
5. [Конфигурация и ограничения](configuration.md), [проверки](testing.md), [генерируемая матрица](test-matrix.md).
6. [Скрин-тур и видеотур по системе](screen-tour.md), [интерактивная галерея](tour/index.html), [как обновлять запись](tour-maintenance.md).
7. [Настройки GitHub, защита main, CI/security и шаблоны](repository-governance.md), [учёт объединения веток и блокеров](branch-consolidation.md).

Фазы 1–6 реализованы; технический набор проверок фазы 7 выполнен для кандидата `7dce226`. Выпуск ещё не принят: 27 из 32 MUST имеют локальный статус verified, 5 partial; 14 внешних решений pending. Окружение не подготовлено по подтверждению владельца. `APP_ENV=production` отклоняется кодом; зелёный CI не снимает эту блокировку.

## Продукт и реализация

| Вопрос | Документ |
|---|---|
| Какие гарантии обязаны сохраняться | [Требования](requirements.md), [решения](decisions.md), [state machines](state-machines.md) |
| Как устроены данные и выдача | [Архитектура](architecture.md), [модель данных](data-model.md), [ядро](core-design.md), [file delivery](file-delivery.md) |
| Как защищены ключи и объекты | [Формат FXP1](encryption-format.md), [управление ключами](key-management.md), [security](security.md), [модель угроз](threat-model.md), [внутренний review](design-review.md) |
| Как использовать интерфейсы | [API примеры](api-examples.md), [CLI и offline verifier](cli.md), [OpenAPI](../contracts/openapi.json), [AsyncAPI](../contracts/asyncapi.json), [JSON Schemas](../contracts/schemas/) |
| Что реализовано в ранних фазах | [Срез 1–5](implementation-1-5.md), [история evidence 1–7](history/development-evidence-2026-10-11.md) |

## Эксплуатация и выпуск

| Вопрос | Документ |
|---|---|
| Bootstrap, резервирование, метрики | [Эксплуатация](operations.md), [backup/restore/reconciliation](operations-recovery.md) |
| TLS, сеть, обновление | [Hardening](hardening.md), [ingress и network policy](production-ingress.md), [controlled upgrade](controlled-upgrade.md) |
| Инциденты и ограничения доверия | [Incident response](incident-response.md), [реестр рисков](risk-register.md) |
| Подготовка целевого окружения | [План внешней приёмки и карточка окружения](environment-acceptance.md) |
| Что нужно проверить перед релизом | [Критерии](acceptance-criteria.md), [release checklist](release-checklist.md), [evidence pipeline](release-engineering.md), [нагрузка/DR](performance-testing.md) |
| Лицензии и состав поставки | [Third-party review](third-party-review.md), [release notes](release-notes.md), [changelog](../CHANGELOG.md) |

## ADR

- [001: completion/Range/ZIP](adr/001-download-completion.md), [002: freeze/expiry/revoke](adr/002-retention-precedence.md), [003: export snapshot](adr/003-export-snapshot.md).
- [004: обязательный pre-restore backup](adr/004-restore-gate.md), [005: криптография](adr/005-cryptography.md), [006: recovery control plane](adr/006-operations-recovery.md).
- [007: ingress/network trust](adr/007-ingress-and-network-policy.md), [008: controlled upgrade](adr/008-controlled-upgrade.md), [009: release evidence](adr/009-release-evidence.md).

## Как поддерживать знания

Требование → ADR/контракт → код → тест → evidence → актуальный статус. После каждого шага обновляются тематическая страница и session; новое измерение содержит источник и ограничения. [Аудит переноса и устранённых расхождений](documentation-audit.md) объясняет изменения документации. [Первоначальный аудит](history/documentation-audit-2026-10-10.md) и [сессия завершения фазы 7](sessions/2026-10-11-phase7.md) — исторические записи, не план новых работ.

Исходное интервью до вопроса 978 отсутствует: проверить полноту переноса интервью невозможно. Все 32 зафиксированных MUST сохранены; новые продуктовые требования, SLA, разрешения на production и принятие рисков не выводятся из истории автоматически.
