# ADR-003: bounded materialization экспортов

Статус: реализовано 2026-10-11, FR-013–014; production load gate/R-03 остаётся открыт.

## Контекст и проблема
Пагинация live rows после закрытия MVCC не сохраняет snapshot значений. В то же время нельзя удерживать DB snapshot на весь срок внешнего S3 upload или обещать произвольно большой экспорт с короткой transaction.

## Решение и поток
Worker claim фиксирует fence и проверяет current owner auth_epoch/role. Одна REPEATABLE READ transaction (statement ≤10 s) копирует разрешённые **значения** в export_rows. Limits: 50000 rows/16 MiB; превышение откатывает материализацию и даёт snapshot_limit_exceeded. snapshotAt — transaction start, не enqueue time.

После commit staging immutable. Следующая fenced стадия читает его, потоково формирует CSV/JSON и ZIP/GZIP (auto gzip при snapshot >1 MiB), пишет encrypted ArtifactStorage chunks и сверяет записанный ciphertext чтением. Ready artifact descriptor фиксирует total bytes/SHA-256. Финальная DB transaction проверяет fence/lease/authorization, связывает artifact с job и удаляет staging. Так PG+encrypted ArtifactStorage обеспечивают гибридную модель без второго промежуточного encrypted snapshot artifact; это явное уточнение первоначального baseline, исключающее лишнее дублирование данных.

## Crash, retry и cancel
Crash до materialization commit → новый snapshot при следующем запуске. Crash после commit → те же значения и snapshotAt, временные incomplete artifacts удаляются/пересобираются. Failed job допускает retry только при полном retained staging; возрастное cleanup через 24 h удаляет failed staging. Cancel increments fence, старый worker не может завершить job; удаляются staging и временные artifacts обычным cleanup. Completed artifacts TTL 24 h. Authority change запрещает snapshot/commit, ADMIN priority aging не отменяет owner authorization.

## Пример и доказательства
Изменить title после materialization, затем crash/retry rendering: export содержит старый title и исходный snapshotAt. `TestIntegrationExportSnapshotRetryAndBulk` проверяет этот сценарий; `TestIntegrationQRInvalidationAndExportAuthorization` отменяет queued ADMIN export после demotion. Browser проверяет фактический CSV artifact download.

## Ограничения
Limits — инженерные bounds, не SLA; load tuning входит в фазу 7. SQL staging не encrypted отдельным envelope и защищается DB/host boundary, в отличие от внешних artifacts. Restore отменяет незавершённые exports и инвалидирует их leases/fences: старый staging не возобновляется автоматически. Это реализовано фазой 6 и проверено recovery drills; completed artifacts проходят отдельную проверку доступности и SHA/AEAD, [runbook](../operations-recovery.md). Нельзя подменять snapshot live pagination при превышении bounds.
