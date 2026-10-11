# Эксплуатация и DR

## Контекст
Фаза 6 реализует encrypted BackupProvider Local/S3, restore через ADMIN UI/host CLI, обязательный проверенный pre-backup, post-restore reconciliation и explicit reopen. Подробный воспроизводимый поток, alerts, данные и ограничения: [runbook](operations-recovery.md), [ADR-006](adr/006-operations-recovery.md), [CLI](cli.md).

## Bootstrap и изменения
DB connection, MASTER_KEK, APP_ENV — bootstrap env. Dev Compose автоматически provision отдельный `DEV_BACKUP_ROOT=/data/backups` и `backup_data` volume рядом с payload/artifact volumes. S3 secrets подаются ADMIN UI или CLI stdin и хранятся encrypted. `APP_ENV=production` безусловно запрещён текущим кодом; изменение допуска выделено в DEV-01 [плана](implementation-plan.md).

Migrations при startup проверяют strict checksum/prefix/unknown history и сериализуются advisory lock. Pending SQL существующей instance требует closed gate и exclusive runtime lock; generation/reconciliation меняются атомарно с DDL. `upgrade check` только читает compatibility. `upgrade prepare` закрывает gate, fence старого поколения и ждёт drain без миграций. Затем оператор останавливает все старые instances, создаёт/проверяет backup и меняет image. После restart обязательны reconciliation/explicit reopen: [runbook](controlled-upgrade.md). Автоматический pre-migration backup отсутствует; pre-restore backup обязателен без override ([ADR-004](adr/004-restore-gate.md)).

## Восстановление и эксплуатационный эффект
Backup → verified descriptor → close gate/drain → verified pre-backup → atomic import/session invalidation/new generation → restart всех API/worker → audit/crypto/storage reconciliation → reviewed capabilities/quarantine/orphans → explicit ADMIN reopen. Ошибка не открывает сервис автоматически. Redis не резервируется, payload/artifact bytes не возвращаются из DB backup.

Optional расписания: backup (по умолчанию выключено), audit+crypto (3600 s), storage (86400 s). Mandatory cleanup/outbox jobs сохраняются, но обычный worker data plane приостанавливается на время recovery. Security incidents/checkpoints/reports бессрочные и не попадают в owner notifications. Internal health/readiness/metrics доступны только внутри deployment; public status не раскрывает topology.

## Ограничения
Manual restore-test выполняется по runbook, без scheduled reminder. Локальный drill не доказывает production RPO/RTO; такие значения фиксируют измерением на целевой инфраструктуре. TLS/proxy/CIDR, upgrade, локальные load/DR/security gates реализованы; public ACME/host/providers, target DR и независимый review остаются внешней приёмкой [по протоколу](environment-acceptance.md). Production deploy требует отдельного поручения владельца.
