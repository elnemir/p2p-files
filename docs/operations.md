# Эксплуатация и DR

## Контекст
Фаза 6 реализует encrypted BackupProvider Local/S3, restore через ADMIN UI/host CLI, обязательный проверенный pre-backup, post-restore reconciliation и explicit reopen. Подробный воспроизводимый поток, alerts, данные и ограничения: [runbook](../wiki/operations-recovery.md), [ADR-006](../wiki/adr/006-operations-recovery.md), [CLI](cli.md).

## Bootstrap и изменения
DB connection, MASTER_KEK, APP_ENV — bootstrap env. Dev Compose автоматически provision отдельный `DEV_BACKUP_ROOT=/data/backups` и `backup_data` volume рядом с payload/artifact volumes. S3 secrets подаются ADMIN UI или CLI stdin и хранятся encrypted. `APP_ENV=production` запрещён до release gates фазы 7.

Migrations под checksum/advisory lock применяются при startup. `upgrade check` только читает migration compatibility; он не применяет SQL и не заменяет backup. Перед upgrade оператор создаёт backup и планирует downtime. Автоматический pre-migration backup отсутствует; pre-restore backup обязателен без override ([ADR-004](../wiki/adr/004-restore-gate.md)).

## Восстановление и эксплуатационный эффект
Backup → verified descriptor → close gate/drain → verified pre-backup → atomic import/session invalidation/new generation → restart всех API/worker → audit/crypto/storage reconciliation → reviewed capabilities/quarantine/orphans → explicit ADMIN reopen. Ошибка не открывает сервис автоматически. Redis не резервируется, payload/artifact bytes не возвращаются из DB backup.

Optional расписания: backup (по умолчанию выключено), audit+crypto (3600 s), storage (86400 s). Mandatory cleanup/outbox jobs сохраняются, но обычный worker data plane приостанавливается на время recovery. Security incidents/checkpoints/reports бессрочные и не попадают в owner notifications. Internal health/readiness/metrics доступны только внутри deployment; public status не раскрывает topology.

## Ограничения
Manual restore-test выполняется по runbook, без scheduled reminder. Локальный drill не доказывает production RPO/RTO; такие значения фиксируют измерением на целевой инфраструктуре. Production TLS/ACME/trusted-proxy/CIDR, update/deployment acceptance, полные load/DR/security gates остаются фазой 7. Production deploy требует отдельного поручения владельца.
