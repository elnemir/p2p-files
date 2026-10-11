# Приёмка MVP

- Все MUST требования имеют реализацию и подтверждающие тесты; исключения только после явного согласования владельца.
- One-time semantics и ZIP atomicity проходят concurrency/fault injection; Redis loss не допускает повторной выдачи.
- Restore+reconciliation, backup/key recovery, audit verification, deletion, crypto Range/rotation проходят документированные тесты.
- UI RU/EN, light/dark, keyboard/mobile/WCAG, browser matrix, error flows проверены.
- OpenAPI/AsyncAPI/schema/codegen sync; migrations upgrade path; observability и security alerts.
- Нет открытых critical/high security findings; medium/low residual risks явно согласованы владельцем.
- Пройден docs/release-checklist.md; production deployment не выполняется автоматически.

Машинная проверка полноты: `make release-check`, [процесс и внешние блокеры](../wiki/release-engineering.md). Зелёный compact CI не заменяет полный локальный load profile или решения reviewer/owner. На 2026-10-11 окружение отсутствует по подтверждению владельца; это blocker, не waiver.
