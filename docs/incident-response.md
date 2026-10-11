# Incident response runbook — обязательные сценарии

> Нормативный scope следующих фаз 6–7. Этот документ не является отчётом о выполнении; текущее evidence и ограничения — [development](../wiki/development.md), порядок работы — [roadmap](../wiki/roadmap.md).

Сценарии: утечка API token, admin compromise, KEK compromise, missing/degraded Local/S3, stuck deletion, audit-chain corruption, PostgreSQL/Redis loss, webhook/SMTP outage, integrity_failed/storage_missing, orphan snapshots. Для каждого агент должен добавить detection, immediate containment, affected scope, evidence preservation, recovery steps, validation, communication, audit references и rollback/limitations.

Audit corruption: critical incident, restricted admin operations, investigate, establish new checkpoint с комментарием без переписывания старой цепочки; защищённый PDF/JSON incident report бессрочно. Forensic access всегда re-auth/reason/audit.
