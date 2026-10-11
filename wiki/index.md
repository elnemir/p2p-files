# FileX: база знаний

## Контекст
FileX — приватная передача файлов с одноразовым серверным завершением выдачи. Продуктовые требования находятся в `docs/`; wiki связывает их с решениями, кодом и доказательствами проверки. Это единая точка входа, а не второй набор конкурирующих требований.

## Порядок работы
1. [Правила](../AGENTS.md) и [сессия](session.md).
2. [Требования](../docs/requirements.md), [архитектура](../docs/architecture.md), [решения](../docs/decisions.md).
3. [Фазы и gates](roadmap.md), [матрица требований](../docs/test-matrix.md).
4. [Дизайн ядра](core-design.md), [модель данных](data-model.md), [модель угроз](threat-model.md), [design review](design-review.md).
5. [Разработка и проверки](development.md), [исходный аудит](documentation-audit.md).

## Решения
- [ADR-001: одноразовая выдача, Range, ZIP](adr/001-download-completion.md).
- [ADR-002: freeze, expiry, revoke](adr/002-retention-precedence.md).
- [ADR-003: согласованный export snapshot](adr/003-export-snapshot.md).
- [ADR-004: restore и backup](adr/004-restore-gate.md).
- [ADR-005: криптографические контексты](adr/005-cryptography.md).

## Контракты
[OpenAPI](../contracts/openapi.json), [AsyncAPI](../contracts/asyncapi.json), [JSON Schemas](../contracts/schemas/event-v1.json). OpenAPI содержит 93 реализованные операции; generator и route coverage проверяются CI. Planned фазы 6–7 описаны в roadmap, не выдаются за runtime endpoints.

[Реализация 1–5](implementation-1-5.md), [file delivery](file-delivery.md).

## Ограничения
Полный MVP включает все 32 требования. Начальные фазы не являются самостоятельным production release. Источник интервью до вопроса 978 отсутствует; новые решения этого цикла фиксируются явно, а не приписываются интервью. Числа в development settings — инженерные defaults, не SLA.
