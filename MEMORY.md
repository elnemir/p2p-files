# MEMORY.md — контекст репозитория

Текущая сессия: [wiki/session.md](wiki/session.md). Фазы/gates: [wiki/roadmap.md](wiki/roadmap.md), implementation/evidence: [wiki/development.md](wiki/development.md). Исходное интервью 1–978 в репозитории отсутствует; решения текущего цикла не приписываются интервью.

2026-10-11: реализованы вертикальные сценарии этапов 1–5: platform/auth/settings, tus/Local/S3/one-time delivery, ZIP/crypto/forensic, communications, reports/artifacts/exports/QR/bulk. Финальные команды и результаты сохраняются в session. Фазы 6–7 и production acceptance открыты, APP_ENV=production отклоняется.

## DO NOT CHANGE WITHOUT APPROVAL
- Go + Next.js + PostgreSQL + Redis + Traefik + Docker Compose, single origin.
- Draft → publish → active; после publish файлы/пути/имена immutable.
- TTL от publishedAt; hard absolute max TTL, PostgreSQL leases/fencing, strict revoke, atomic Download all.
- USER без recipient correlation/IP/UA, ADMIN forensic/report audit отдельно.
- Encrypted artifacts/backups, KEK/class DEK, Ed25519 только SHA256SUMS.
- Production deploy только по отдельной команде; R-01–06 не принимать самостоятельно.

## Важные решения для продолжения
- [ADR-001–005](wiki/index.md): server-observed completion, retention precedence, bounded value snapshots, mandatory pre-restore backup, FXP1 crypto boundaries.
- Snapshot export: PG materialized values → verified encrypted final artifact; crash/retry reuse committed values/time; failed staging cleanup 24 h. Не live pagination.
- QR — encrypted artifact с mapping email и link invalidation. Public verification только active USER. Private report tombstones сохраняются, публично не подтверждаются.
- ADMIN операции чужих exports/TTL и расширенный эксплуатационный CLI требуют отдельного phase-6 интерфейса/audit; текущие owner-scoped artifacts не выдаются за такой интерфейс. Исходное разрешение доступа ADMIN к чужим exports без отдельного re-auth не применять как обход current report/forensic re-auth guards; policy согласуется при реализации этого отдельного сценария.
- Limits/defaults — инженерные bounds, не SLA/RPO/RTO. Все requirement statuses проверять в generated matrix, не по историческому тексту.
