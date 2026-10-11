# State machines и guards

## Контекст
Таблица описывает persisted states текущей реализации. Upload использует состояние `files`, а streaming/resuming — условия `downloads` lease/coverage, не выдуманные дополнительные enum. Сроки определяет DB time; hold независим от lifecycle. Порядок mutation locks parent transfer → file; неуказанный переход запрещён.

| Объект | Переход / guard |
|---|---|
| Transfer | draft → active: owner/revision/all ready/nonempty, manifest+links+outbox атомарно. active → completed при последнем полном consume. active → expired при ordinary/hard deadline. Revoke → revoked с access_epoch++. |
| Transfer hold | frozen flag; active ordinary TTL приостанавливается, absolute deadline остаётся. Unfreeze не возвращает revoked/completed/expired к active. |
| File upload | uploading → finalizing при exact offset=size; ready только после server SHA/CRC/MIME/scanner; rejected при policy failure. Draft cancel → cancelled с cleanup. |
| File delivery | ready → leased при acquire; stale lease позволяет новое acquire вернуть ready и зарезервировать заново. leased → deleting только при полном commit всей reservation. |
| File integrity | ready/leased/finalizing → integrity_failed/storage_missing при чтении; выдача fail closed. ADMIN check восстанавливает ready только при совпадении оригинального SHA и допустимом active parent. |
| File cleanup | deleting → deleted после idempotent physical removal всех chunks; hold/write lease блокируют удаление. Empty files завершаются без object jobs. |
| Download | leased → completed; interrupted/resume остаётся leased с coverage. lease_until/epochs/parent/session/file state являются authoritative guards; expired lease не даёт нового stream/commit. |
| Artifact | writing → ready после encrypted writes/read-back и descriptor commit; writing timeout/ready TTL/purge/QR invalidation → deleting → deleted. |
| Artifact credential | new → leased при однократном claim; coverage completion → completed; expiry/session/authorization/source-state прекращают доступ. |
| Report | active ↔ trashed → purged; purged terminal, content удаляется, private hash/history остаются. Public verify видит только active USER. |
| Report render | pending → running → ready/failed; stale running lease позволяет новый fence, failed/expired outputs можно regenerate из того же snapshot. |
| Export | queued → snapshotting → materialized → rendering → completed; ошибка → failed, owner cancel/authority loss → cancelled. Lease crash возвращает queued либо materialized в зависимости от committed snapshot. Retry failed сохраняет committed values/time, если staging complete. |
| Delivery | pending → sending → delivered; временная ошибка → pending/backoff, 8 attempts → dead; manual retry dead/delivered → pending. Invalid recipient link/config → cancelled. |
| Migration/rotation | paused ↔ running → completed/cancelled; active key при cancel не откатывается. Внешняя ошибка не даёт pointer swap, следующая попытка повторяет безопасно. |
| Recovery gate | open → maintenance при restore queue/явном close → restoring после verified pre-backup → reconcile после atomic import/new generation → open только matching fresh report/hash/generation + ADMIN review. Critical failure → blocked. Restart не открывает gate. |
| Operations job | queued → running под единственным schema-scoped advisory runner lock → completed/failed. Обнаруженный после crash running становится failed/interrupted; destructive replay не автоматический. |
| Controlled upgrade | prepare: closed gate + новая generation + audit → exclusive drain → полный stop/verified backup → locked atomic migrations/new generation при DDL → restart с closed gate → fresh reconciliation → explicit reopen. Timeout/failed startup не открывают доступ. |
| Security incident | open → acknowledged → resolved; каждая ревизия encrypted report сохраняется. Audit/crypto resolution требует успешной проверки. Critical unresolved блокирует reopen. |
| Bulk | paused (dry-run selection) → queued → running → completed; cancel → cancelled. Items pending → done/conflict; выполненные элементы не откатываются. |

## Примеры и порядок удержания
Download all атомарно резервирует все оставшиеся published files; один busy/error/frozen member даёт conflict, partial consume запрещён. Ordinary TTL оставляет integrity_failed для диагностики. Revoke удаляет retained content, но physical hold сильнее cleanup и сохраняется до ADMIN решения. [ADR-001](adr/001-download-completion.md), [ADR-002](adr/002-retention-precedence.md).

## Ограничения
Lifecycle guard проверяется непосредственно при request/commit, а не только scheduler. Поэтому задержка worker не продлевает recipient права. DB state completed означает server-observed coverage, не доказательство сохранения получателем. Restore не отменяет фактическое удаление payload; восстановленные ссылки/API tokens удерживаются закрытым gate до review. [Runbook](operations-recovery.md). Проверяемые transitions связаны с [матрицей](test-matrix.md).
