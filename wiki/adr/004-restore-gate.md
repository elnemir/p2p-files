# ADR-004: обязательный backup перед restore

Статус: принято 2026-10-10. Requirements: OPS-003.

## Контекст и проблема
Operations содержал аварийный override, противоречащий обязательному pre-restore backup в requirements и decisions.

## Решение
Убрать неподтверждённый override. Приложение и CLI не начинают destructive restore, если backup текущей БД не создан и не проверен. При невозможности backup операция завершается ошибкой; аварийное восстановление не объявляется поддержанным обходом этого контракта.

## Поток
Maintenance gate → остановка writers/jobs → проверенный encrypted pre-restore backup → restore → session epoch invalidation → audit/crypto/storage reconciliation → отчёт проблем → явный ADMIN reopen. Recipient links/API tokens могут воскреснуть из backup: gate требует их проверки и нужного revoke/rotation. Orphans автоматически не публикуются.

## Ограничения и проверка
Ни running transfer, ни старая сессия не продолжаются после restore. Недоступный BackupProvider блокирует restore. Нужны тесты backup failure, incomplete backup, KEK mismatch, resurrected link, storage missing и reopen до reconciliation. R-01 не считается принятым только из-за наличия gate.
