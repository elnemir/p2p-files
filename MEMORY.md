# MEMORY.md — контекст репозитория

Текущая сессия: [wiki/session.md](wiki/session.md). Фазы/gates: [wiki/roadmap.md](wiki/roadmap.md), implementation/evidence: [wiki/development.md](wiki/development.md). Исходное интервью 1–978 в репозитории отсутствует; решения текущего цикла не приписываются интервью.

2026-10-11: этапы 1–6 закрыты по локальным development gates. К platform/auth/settings, tus/Local/S3/one-time delivery, ZIP/crypto/forensic, communications и reports/artifacts/exports/QR/bulk добавлены encrypted backup/restore, mandatory pre-backup, reconciliation/reopen, KEK recovery/rewrap, incidents/checkpoints, operational UI/CLI и ADMIN exports. Проверки: make check/security, 31 PG integration/race и 3 Chromium E2E. Точное evidence и Git state — в session. Фаза 7 и production acceptance открыты, APP_ENV=production отклоняется.

## DO NOT CHANGE WITHOUT APPROVAL
- Go + Next.js + PostgreSQL + Redis + Traefik + Docker Compose, single origin.
- Draft → publish → active; после publish файлы/пути/имена immutable.
- TTL от publishedAt; hard absolute max TTL, PostgreSQL leases/fencing, strict revoke, atomic Download all.
- USER без recipient correlation/IP/UA, ADMIN forensic/report audit отдельно.
- Encrypted artifacts/backups, KEK/class DEK, Ed25519 только SHA256SUMS.
- Production deploy только по отдельной команде; R-01–06 не принимать самостоятельно.

## Важные решения для продолжения
- [ADR-001–006](wiki/index.md): server-observed completion, retention precedence, bounded value snapshots, mandatory pre-restore backup, FXP1 crypto boundaries, durable recovery control plane.
- Snapshot export: PG materialized values → verified encrypted final artifact; crash/retry reuse committed values/time; failed staging cleanup 24 h. Не live pagination.
- QR — encrypted artifact с mapping email и link invalidation. Public verification только active USER. Private report tombstones сохраняются, публично не подтверждаются.
- ADMIN операции чужих exports/TTL реализованы отдельными routes/UI с reason/audit, без USER notifications. ADMIN credential привязана к session и проверяет role/reauth на каждом Range guard; обычные owner-scoped credentials не открывают чужие artifacts.
- Restore меняет generation, инвалидирует sessions, останавливает незавершённые jobs и требует restart API/worker. `ops_*` control plane не откатывается. Reopen — только после свежей reconciliation и проверки resurrected links/API tokens; orphan bytes не публикуются и не удаляются.
- Restore на новую instance требует новый master KEK и исторический recoveryKEK через stdin. Старые backups сохраняют старый KEK; plaintext audit key при rewrap не меняется. Root seal budgets не откатываются при import failure; восстановленные non-signing keys получают exhausted budget.
- Все operational подробности и команды — [runbook](wiki/operations-recovery.md), [CLI](docs/cli.md). Начата фаза 7.1: [HTTPS/proxy/CIDR и host reset](wiki/production-ingress.md). Network/origin changes требуют restart API/worker и reconciliation/reopen; APP_ENV=production всё ещё отклоняется.
- Limits/defaults — инженерные bounds, не SLA/RPO/RTO. Все requirement statuses проверять в generated matrix, не по историческому тексту.

Фаза 6 опубликована как `02c53d9`, remote CI #38131827674 success. Первый срез фазы 7 опубликован как `50de642`: HTTPS ingress/network policy, host reset, 32 PG tests и 3 Chromium scenarios, TLS harness. Его remote CI #38133772690 — success, включая make ingress; точное evidence — в session. Реализация среза не означает закрытие всей release фазы. Controlled upgrade/downtime опубликован как `dda4587`, CI #38135958275 success (включая upgrade drill): strict migration prefix на startup, host prepare/drain, closed restart и explicit reopen. Локально PASS 36 PG, 3 Chromium, TLS/upgrade drills и check/security; измеренное fixture окно 6.489 s не является SLA. См. [runbook](wiki/controlled-upgrade.md). Следующий локальный шаг 7.2 — static analysis/SBOM/license/container/secret scans, затем browser/a11y/load/DR.
