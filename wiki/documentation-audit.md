# Аудит и актуализация документации

## Контекст и проблема

Дата: 2026-10-11. Запрос: перенести весь комплект из docs в wiki, сверить его с реализованным проектом и подготовить продолжение разработки. Исходный clean HEAD `21aaa00`; прочитаны все 50 Markdown страниц корня/docs/wiki, 9 ADR, актуальные contracts, config/bootstrap, Makefile/CI и связанные generators. Первоначальный [аудит 2026-10-10](history/documentation-audit-2026-10-10.md) сохранён как исторический материал.

До переноса нормативные документы находились отдельно от wiki. Корневой MEMORY описывал начало фазы 7, CONTRIBUTING требовал host Chromium, несколько страниц предлагали ещё реализовать уже прошедшие проверки. Исторические числа и текущие инструкции смешивались. CLI, docs checker, matrix generator и release bundle содержали пути к старому каталогу.

## Решение и карта переноса

Все 20 страниц перенесены без изменения имён в wiki: acceptance-criteria, api-examples, architecture, cli, decisions, encryption-format, hardening, implementation-plan, incident-response, key-management, operations, performance-testing, release-checklist, release-notes, requirements, risk-register, security, state-machines, test-matrix, third-party-review (расширение `.md`). Дублирующий каталог удалён; forwarding copies не создавались.

| Было | Стало / назначение |
|---|---|
| Продуктовые документы в отдельном каталоге | wiki/*.md, единая навигация index |
| wiki/development.md со смешанной хронологией | Текущая инструкция development + history/development-evidence-2026-10-11.md |
| wiki/documentation-audit.md о specification-only | Этот актуальный аудит + history/documentation-audit-2026-10-10.md |
| Предыдущий wiki/session.md | sessions/2026-10-11-phase7.md + новый рабочий session |
| README/CONTRIBUTING/MEMORY | Короткие актуальные указатели и правила без копирования журнала |
| Hardcoded tooling paths | check_docs/traceability/release_bundle/CLI используют wiki |

Поток знаний: requirements/ADR → source contracts → implementation/tests → generated matrix → dated evidence → current state/session → следующие задачи. Контракты остались в contracts, генераторы в scripts, migration/runtime поведение не менялось.

## Сверка baseline → текущее состояние

| Классификация | Что подтверждено / исправлено |
|---|---|
| Реализовано | Фазы 1–6, 9 ADR, 16 migrations, 107 операций; technical phase 7 на 7dce226; 32 requirement IDs сохранены |
| Изменено | Canonical wiki paths/entrypoints; current state отделён от истории; production guard описан как безусловный кодовый отказ |
| Добавлено | Карта кода, config reference, developer workflow/шаблон задачи, testing/evidence guide, backlog с зависимостями, external acceptance matrix/шаблон протокола |
| Не подтверждено | Реальные host/DNS/ACME/SMTP/S3/escrow/DR, устройства/AT, independent review и решения владельца |
| Требует доработки | DEV-01 production admission, DEV-02 tooling maintenance, DEV-03 license package/текст лицензии владельца, DEV-04 evidence retention |

Ранние расхождения A-01–08 исходного аудита получили traceability, ADR-001–009, state/data/security/crypto/recovery contracts и тесты. A-09 (исходное интервью отсутствует) остаётся ограничением provenance; полнота интервью не заявляется. A-10 (разрозненные источники/неотслеживаемые документы) устранён предыдущими commits и этим каноническим переносом. Старые рекомендации архива не становятся новым backlog автоматически.

## Сохранённые ограничения

Все 32 MUST и 27 verified/5 partial остаются; ни один pending risk/license decision не принят. Замена source paths не подтверждает новый полный runtime acceptance. AGENTS/scripts/CLI меняют source fingerprint, поэтому 13 прежних PASS относятся к историческому кандидату, а не автоматически к текущему diff. Старые gate records сохранены локально в `.cache/evidence/documentation-migration/baseline-gates`.

## Проверка и продолжение

PASS: 59 Markdown, все 32 requirement statements сохранены дословно, все 20 страниц перенесены, broken links/unreachable wiki pages отсутствуют. Отрицательные пробы подтвердили отказ при прежнем каталоге и orphan wiki page. make check, compatibility, source/secret scan и сборка пяти verifier targets прошли на одном source hash; release notes и checksums bundle сверены. Окончательные записи — [session](session.md). Полные PG/browser/load/DR/image gates в этом документационном шаге не повторялись, исторические PASS не переносятся на новый hash.

Практический следующий шаг — [implementation plan](implementation-plan.md). [Environment acceptance](environment-acceptance.md) содержит все 14 decision IDs и необходимые входы; новая подготовка не выдумывает адреса, секреты, SLA или разрешение production deploy.
