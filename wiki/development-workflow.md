# Workflow разработки и передача задачи

## Контекст

Правила [AGENTS](../AGENTS.md) обязательны; wiki задаёт знания и контракты, код и тесты подтверждают реализацию. Workflow нужен, чтобы следующий инженер мог продолжить без чтения чата и чтобы документация не отставала от проверенного поведения.

## Порядок работы

1. Прочитать index/session/requirements/architecture/decisions/implementation plan, проверить Git status и сохранить чужие изменения. Определить requirement IDs, компоненты, инварианты и неизвестные входы.
2. Воспроизвести проблему или описать наблюдаемое до/после. До архитектурного изменения обновить ADR, state machines, threat/design review и контракты; согласовать затрагивание защищённых AGENTS инвариантов.
3. Реализовать минимальный вертикальный срез: данные/guards → API/worker → UI/CLI → negative/concurrency/fault tests по риску. Не вводить production stub или параллельный механизм.
4. Обновить source контракты и codegen; новые migrations добавлять, применённые не редактировать. Сохранить privacy/redaction/reauth и fail-closed.
5. Выполнить относящиеся [gates](testing.md), записать command/commit/hash/окружение/профиль/результат. Skipped тест — не PASS. Обновить тематическую wiki, traceability, changelog и session.
6. Review diff/whitespace/generated files; commit Conventional Commits и push в разрешённом scope после обязательных gates. Отдельно проверить exact-commit CI. Не считать push deploy или незавершённый CI успехом.

Ветки по умолчанию `codex/<short-topic>`. Task/PR должен описывать окончательное поведение, не историю промежуточных подходов. [План](implementation-plan.md) фиксирует зависимости; отсутствие внешнего окружения не мешает готовить конкретный локальный результат.

Main принимает PR через squash merge после required checks и разрешения review conversations. При одном владельце второе approval не требуется; независимый security/release review остаётся отдельным условием. Формы задач, PR template и воспроизводимые GitHub settings описаны в [repository governance](repository-governance.md).

## Источники и генерация

```sh
python3 scripts/build_contracts.py
npm run generate:types
python3 scripts/traceability.py
make contracts
make docs-check
```

Редактируются build_contracts.py и contracts/traceability.json; generated contracts/api-schema/test-matrix — только результат генератора. `apps/web/lib/api-types.ts` содержит пользовательские aliases к generated schema, это отдельный рукописный слой. Совместимость v1 проверяется к закреплённому baseline; major/baseline review документируется отдельно.

Traceability verified означает существующую реализацию и привязанные тесты в проверенной области. Для пяти partial нужен ещё внешний scope. Смена status требует достаточных результатов, не только ссылки на будущий runbook.

## Шаблон рабочей карточки

```text
ID / requirement IDs:
Контекст и воспроизводимый триггер:
Проблема и наблюдаемое до/после:
Scope и затрагиваемые инварианты:
Компоненты / данные / contracts / migrations:
Вход → guards/обработка → выход; ошибки и retry:
Зависимости / неизвестные входы / ответственная роль:
Критерии приёмки (позитивные + negative/concurrency/fault):
План проверки и evidence:
Env/config/операционное влияние и восстановление:
Wiki/ADR/changelog/matrix:
Статус, выполненное, следующий шаг, блокеры:
```

Пример: DEV-01 / OPS-005 → production config сейчас безусловно отвергается → сначала ADR допуска, затем reviewed implementation; default deny, invalid evidence rejection и exact-candidate retest обязательны. Разрешение deploy не выводится из карточки.

## Definition of Done и review

- Требуемое поведение реализовано end-to-end; новые ошибки имеют понятный контракт, важные негативные/конкурентные сценарии проверены.
- Contracts/types/migrations согласованы; generated outputs актуальны; критичные security findings не скрыты.
- Code/config flow, примеры, ограничения, operational impact и follow-ups понятны без чтения исходника.
- Wiki, session, matrix и changelog обновлены. Есть фактическое evidence с границами; внешние blockers названы явно.
- Если задача требует внешнего подтверждения, она остаётся partial/blocked до фактического решения, даже когда локальная часть готова.

## Сессия и исторические данные

`wiki/session.md` обновляется после каждого development step, перед рискованной операцией и перед ответом: Task, Status, Completed steps, Next steps, Decisions, Context, Risks / blockers. При смене большой задачи завершённый журнал сохраняется в `wiki/sessions/<date>-<topic>.md`, ссылки становятся относительными к новой папке. Current state остаётся коротким каноническим срезом; подробные датированные результаты — в history.

Первоначальная спецификация не переписывается задним числом как принятый production scope. При несовпадении docs/code фиксировать «реализовано», «изменено», «добавлено», «не подтверждено», «требует доработки» с конкретным основанием. Правила, юридическое принятие риска и независимый review не заменяются автоматически заполненным JSON.
