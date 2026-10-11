# Реестр остаточных рисков

## Контекст

R-01–06 входят в решение о выпуске. Наличие mitigation и локальных tests не означает принятия риска. На 2026-10-11 все шесть записей pending в [release-acceptance.json](../contracts/release-acceptance.json); ИИ-агент не переводит их в accepted от имени владельца. Полнота технической реализации — [current state](current-state.md).

## Риски, действующие меры и оставшаяся проверка

| ID | Риск | Реализованный контроль | Что нужно владельцу для решения |
|---|---|---|---|
| R-01 | Backup восстанавливает ранее отозванные API tokens/recipient links | Closed restore gate, invalidated sessions, generation/restart, reconciliation, массовый revoke-capabilities и fresh report/hash для reopen | DR на целевой инфраструктуре и протокол проверки восстановленных capabilities; [recovery](operations-recovery.md) |
| R-02 | HTTP completion не доказывает сохранение файла получателем; crash до commit допускает повтор bytes | Explicit acquire, PG coverage/fencing, atomic consume, guarded writes и точная семантика report | Принять предел протокола и формулировку server-observed; [ADR-001](adr/001-download-completion.md) |
| R-03 | Snapshot export держит MVCC/DB ресурсы и может превысить допустимые bounds | Materialized values, 50000 rows/16 MiB, bounded transaction, lease retry/cleanup | Target load/VACUUM/latency measurements и согласованный capacity profile; [ADR-003](adr/003-export-snapshot.md) |
| R-04 | Бессрочные holds/audit/key/security metadata увеличивают storage | Hold-aware cleanup, inventory/metrics, явный ADMIN review; automatic deletion forensic data не вводится | Политика capacity/retention/действий оператора при pressure и ответственность за независимые копии |
| R-05 | Компрометация class DEK затрагивает класс данных | Wrapped separate classes, HKDF contexts, per-object identity, rotation/key budgets | Независимый escrow/access/incident review; rotation не отзывает уже скопированные ключи; [keys](key-management.md) |
| R-06 | Host CLI доверен без дополнительного ADMIN password | Host/container boundary, reason/audit, mandatory pre-backup/schema/key/reopen guards | Host least privilege/SSH/operator audit и принятие последствий host compromise; [CLI](cli.md) |

## Поток решения

Инженер готовит фактические tests/limits/findings → независимый reviewer оценивает ограничения → владелец принимает или требует изменение → решение и ссылка на evidence попадают в соответствующий decision ID → traceability/release readiness проверяются заново. Шаблон протокола и роли — [external acceptance](environment-acceptance.md), карточка EXT-07 — [план](implementation-plan.md).

Пример: успешный restore и закрытый gate подтверждают mitigation R-01; они не доказывают, что владелец согласился с resurrection. Без его решения R-01 остаётся pending даже после зелёного теста. Отказ принять риск приводит к доработке/отказу выпуска, не к молчаливому исключению требования.

## Ограничения

Это реестр известных согласованных категорий, не утверждение об отсутствии других findings. Новый security finding сохраняется отдельно с affected commit, severity и reproducer; HIGH/CRITICAL не выпускается. Нельзя выдумывать SLA, юридическое решение или принятый риск по наличию ссылки на runbook.
