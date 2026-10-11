# Реестр рисков (начальный)

| ID | Риск | Мера | Статус |
|---|---|---|---|
| R-01 | Restore может восстановить ранее отозванные API tokens/access links | post-restore reconciliation, явный ADMIN gate, operator warning | На рассмотрении, не accepted |
| R-02 | HTTP completion не доказывает сохранение файла на устройстве | точная формулировка отчёта, network tests | На рассмотрении |
| R-03 | Долгие export snapshots удерживают MVCC и раздувают БД | bounded transaction + materialized hybrid snapshot | На рассмотрении |
| R-04 | Бессрочные frozen payload/reports приводят к storage pressure | monitoring/capacity guard, manual admin review | На рассмотрении |
| R-05 | Per-class DEK увеличивает blast radius компрометации | KEK protection, context isolation, rotation | На рассмотрении |
| R-06 | CLI host access без дополнительного re-auth повышает риск при host compromise | hardening, least-privileged host access, audit | На рассмотрении |

ИИ-агент не вправе самостоятельно переводить риски в `accepted`.
