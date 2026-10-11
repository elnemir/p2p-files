# Session State

## Task
Опубликовать завершённую фазу 6 (commit/push), отдельно проверить remote CI и перейти к реализации фазы 7. Фаза 6 завершена локально 2026-10-11; production deployment не запрошен.

## Status
**Фазы 1–6 закрыты по локальным development gates.** Фаза 6 реализована и проверена; документация, миграции, контракты и traceability обновлены. В матрице 27 verified/5 partial/0 planned, все 32 MUST сохранены. Оставшиеся partial относятся к release acceptance фазы 7. `APP_ENV=production` по-прежнему отклоняется.

Изменения фазы 6 находятся в текущем working tree ветки `main`; commit/push и remote CI для них не выполнялись. Предыдущие этапы 1–5 опубликованы как `3af2f0c` (`feat: implement FileX development phases 1-5`), publication evidence — `703c2ea`. Их [development-checks #38125813423](https://github.com/elnemir/p2p-files/actions/runs/38125813423) прошёл; это исторический CI, не проверка текущего diff. Production deployment не выполнялся.

## Completed steps
- Восстановлены правила и контекст wiki/requirements, проведены architecture/crypto review, зафиксирован [ADR-006](adr/006-operations-recovery.md).
- Реализованы отдельный encrypted BackupProvider Local/S3, bounded logical PostgreSQL archive, descriptor-last/readback verification и импорт backup catalog для новой БД.
- Restore закрывает durable gate, дожидается старых writers, создаёт обязательный проверенный pre-backup, атомарно импортирует application rows и сохраняет неоткатываемый `ops_*` control plane. Ошибка import откатывает данные, но не KEK usage reservations.
- Session/credential/lease invalidation, новая runtime generation, отмена незавершённых upload/export/delivery и pause maintenance jobs запрещают старому процессу продолжить выдачу после restore.
- Reconciliation проверяет audit/crypto, SHA/AEAD файлов и artifacts, inventory/orphans и resurrected capabilities. Reopen привязан к текущему report/hash/generation/возрасту и явному подтверждению ADMIN; revoke capabilities инвалидирует предыдущий отчёт.
- KEK rewrap сохраняет plaintext audit identity; fresh-instance recovery требует новый master KEK и historical recoveryKEK через stdin. Восстановленные non-signing class DEKs получают exhausted budget, новые записи используют свежие ключи.
- Incidents имеют immutable encrypted JSON revisions/PDF, HMAC checkpoints обоих журналов сохраняют historical evidence. Critical containment коммитится до попытки report encryption.
- Добавлены private ADMIN operations UI, schedules/alerts, aggregate internal metrics, host CLI recovery/backup/keys/incidents/checkpoints/completion и read-only upgrade check.
- Отдельные ADMIN export credentials/TTL требуют reason/audit и current role/reauth при каждом Range guard, не уведомляют USER и не воскрешают истёкший artifact.
- Migrations 0014–0016, 107 OpenAPI operations, request/response schemas и traceability синхронизированы. Operational reason принимает 5…2000 Unicode-символов; расписание — 0 либо 60…2592000 секунд.
- `make check` PASS: gofmt/vet, TS strict, unit/race, 4 contract tests, 42 Markdown/32 requirements, contracts validation, Go и Next.js build. Последний лог `.cache/evidence/phase6-check.log`.
- Полный PG/race suite PASS: 26 HTTP + 5 repository tests (89.4/3.4 s). После crypto-health изменения повторены все 8 phase-6 tests (38.0 s); последний targeted run recovery/ADMIN exports PASS (12.6 s), `.cache/evidence/phase6-final-pg.log`.
- Chromium E2E PASS: 3 scenarios (29.1 s), включая backup→restore/pre-backup→restart→reauth→reconciliation→explicit reopen→checkpoint→schedule→ADMIN export/TTL. На проверенных страницах 0 axe violations/overflow/response contract failures.
- Security PASS: npm production vulnerabilities 0; govulncheck affected packages/symbols 0. Есть module-only GO-2026-5932 в неимпортируемом x/crypto/openpgp; это не reachable finding и не повод заявлять отсутствие всех advisories.
- Security incident PDF и mobile restore/reopened screenshots 390px просмотрены; clipping/overflow не обнаружены. Scoped fixtures удалили свои контейнеры и volumes; secrets/caches/traces не добавлялись в Git.

## Next steps
1. Выполняется публикация фазы 6 по прямой команде владельца: проверка diff/index, Conventional Commit, push в существующий origin и отдельная проверка remote CI. Локальный green не является remote CI evidence.
2. Фаза 7: production TLS/ACME/trusted proxy/CIDR, controlled upgrade и зависимый network-policy reset; actual provider/escrow/checkpoint/DR acceptance.
3. Full Safari/Firefox/mobile browser matrix, manual WCAG, load/DR, SBOM/license/container/secret scans и independent security review; затем release review всех 32 MUST и residual risks с владельцем.
4. Реальный deployment только по отдельной явной команде пользователя.

## Decisions
- Stack и доменные инварианты сохранены: PostgreSQL authoritative, Redis ephemeral, immutable publish, one-time delivery, USER/ADMIN privacy. Никакие требования или residual risks не исключены/не приняты автоматически.
- FileX backup — logical encrypted archive, не pg_dump; payload/artifacts/roles/extensions/Redis/KEK не входят. Exact schema fingerprint и mandatory verified pre-backup без override обязательны.
- Runtime shared/exclusive locks и persisted generation охватывают HTTP/worker side effects; CLI берёт runner lock до queue, чтобы worker не перехватил transient recoveryKEK.
- `ops_*` history/gate не откатывается. Application audit key хранится encrypted и сохраняется при rewrap. Исторические backup descriptors требуют исторических KEK.
- Orphans не публикуются и не удаляются автоматически; missing/tampered bytes карантинируются. Capability resurrection проверяется явно перед reopen, administrative mutations требуют новой сверки.
- HMAC checkpoint сам по себе не доказывает отсутствие полного host+DB+key rollback; внешние anchors/escrow остаются операторской и release границей. Ed25519 подписывает только ZIP checksum manifest.

## Context
[Вход](index.md), [roadmap](roadmap.md), [эксплуатация](operations-recovery.md), [development/evidence](development.md), [design review](design-review.md), [матрица](../docs/test-matrix.md), [CLI](../docs/cli.md), [incident runbook](../docs/incident-response.md). Изменены Go cmd/internal/SQL, Next.js components, contracts/scripts, Compose/Dockerfile API и документация. Новые основные пакеты: operations, maintenance; migrations 0014–0016. Чужие изменения не сбрасывались; working tree перед фазой 6 был чистым.

## Risks / blockers
Блокировок для локального закрытия фазы 6 нет. Production topology, real SMTP/S3 accounts, full DR/load/a11y/browser/security и внешнее хранение KEK/checkpoints не подтверждены. Архив ≤1 GiB, record ≤64 MiB, inventory ≤1 млн объектов, findings ≤10 тыс.; worker/CLI execution bounds 5/15 минут — не SLA/RPO/RTO. Backup без payload не восстанавливает физически удалённые bytes; потерянный historical KEK не восстанавливается кодом. Исходное интервью отсутствует. R-01–06 не приняты.

## Перед публикацией фазы 6
Состав working tree относится к завершённой фазе 6; текущая ветка main, origin — существующий elnemir/p2p-files. Workflow development-checks выполняет только проверки, без deployment. Локальные gates подтверждены предыдущим шагом, включая последние Unicode/restore tests. Перед commit повторяются документационные/contract/whitespace проверки; credentials/caches/traces исключены из Git. Фаза 7 начинается с production ingress/trusted proxy и release evidence; инфраструктурная приёмка не считается выполненной заранее.
