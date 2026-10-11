# Эксплуатация и DR

> Нормативный scope следующих фаз 6–7. Этот документ не является отчётом о выполнении; текущее evidence и ограничения — [development](../wiki/development.md), порядок работы — [roadmap](../wiki/roadmap.md).

Bootstrap: DB connection, master KEK, APP_ENV и Traefik domain/ACME в deployment env; остальные настройки в PostgreSQL. Auto migrations at startup под distributed lock, без автоматического pre-migration backup; backup перед upgrade — ответственность runbook. DB backups encrypted, без payload, BackupProvider Local/S3; restore Admin UI и CLI, перед restore обязательный проверенный pre-restore backup; при ошибке backup restore блокируется, аварийного override нет ([ADR-004](../wiki/adr/004-restore-gate.md)).

После restore: инвалидировать Web/recipient sessions, проверить audit/crypto, reconciliation PostgreSQL↔Storage; missing files fail-closed, orphan objects не восстанавливаются автоматически; до явного ADMIN approval пользовательский data plane закрыт. Redis не резервируется.

Периодические jobs: mandatory expiry/deletion/outbox cleanup, configurable integrity checks, backup, audit verification, update check, notifications. PostgreSQL scheduler state/leases/fencing, idempotent workers. `/health`/`/ready`/`/metrics` только internal, публичная `/status` без topology и истории.

Manual restore-test по runbook, без scheduled test/reminder. RPO/RTO не выдумывать; фиксировать измеренные значения.
