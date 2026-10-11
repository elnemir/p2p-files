# Session State

## Task
Продолжить фазу 7.1: реализовать controlled upgrade/downtime drill после опубликованного HTTPS/network-policy среза. Проверить migration compatibility на startup, drain старого поколения, сохранность данных и явное reopening. Production deployment не запрошен.

## Status
Срез controlled upgrade завершён локально: [ADR-008](adr/008-controlled-upgrade.md), [runbook](controlled-upgrade.md). Реализованы общий strict migration plan, startup rejection unknown/checksum/hole, exclusive pending DDL и host upgrade prepare с generation/drain. PASS: make check/security, 36 PG/race tests, 3 Chromium scenarios, TLS и upgrade drills. Реальный SQL prefix 15→16 и rollback проверены отдельно от same-schema Compose replacement. Fixture window 6.489 s, sampled unavailable 5.878 s (29 samples), не SLA/RTO. Готовится отдельный commit/push; remote CI этого среза ещё не запускался. Исторические результаты ниже относятся к опубликованным commits.

**Фазы 1–6 закрыты по локальным development gates.** Фаза 6 реализована и проверена; документация, миграции, контракты и traceability обновлены. В матрице 27 verified/5 partial/0 planned, все 32 MUST сохранены. Оставшиеся partial относятся к release acceptance фазы 7. `APP_ENV=production` по-прежнему отклоняется.

Фаза 6 опубликована коммитом `02c53d9f384f378dd6a279b2d9f7d2e2432b82aa` в origin/main; SHA подтверждён. [CI #38131827674](https://github.com/elnemir/p2p-files/actions/runs/38131827674) — success, включая build/PG/security/Chromium E2E. Первый срез фазы 7 (HTTPS ingress/network policy) реализован, прошёл локальные gates и опубликован отдельным коммитом `50de642f2bf9e2c58e5e3ae0e92f92903cfa634e`; SHA origin/main подтверждён после push. Его [CI #38133772690](https://github.com/elnemir/p2p-files/actions/runs/38133772690) — completed/success, включая новый TLS ingress gate. Этот результат относится к указанному implementation commit; последующая фиксация publication evidence меняет только документацию. Вся фаза 7 остаётся открытой. Предыдущие этапы 1–5 опубликованы как `3af2f0c` (`feat: implement FileX development phases 1-5`), publication evidence — `703c2ea`. Их [development-checks #38125813423](https://github.com/elnemir/p2p-files/actions/runs/38125813423) прошёл; это исторический CI, не проверка текущего diff. Production deployment не выполнялся.

## Completed steps
- Controlled upgrade: host prepare без migrations; reason/queued-job/stale-runtime guards, audited generation/drain и timeout fail closed. Startup требует strict prefix; pending DDL выполняется под exclusive lock и атомарно меняет generation/reconciliation. CLI/ADR/runbook/state machine/traceability синхронизированы. Duplicate test IDs OPS-005 исправлены; docs gate проверяет уникальность.
- Полная регрессия upgrade среза: 36 PG/race (2 database/29 HTTP/5 repository, 5.700/104.576/4.247 s), 3 Chromium (28.6 s), make check (46 Markdown/32 MUST/107 API operations), security, TLS и image upgrade PASS. Все disposable projects удалены. Evidence `.cache/evidence/phase7-upgrade-*`, точные границы — [development](development.md).
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
1. Опубликовать проверенный controlled upgrade/downtime срез и подтвердить remote CI. Затем локальная фаза 7.2: static analysis/SBOM/license/container/secret scans. Фаза 6 и ingress срез опубликованы, оба remote CI — success; CI нового upgrade среза ещё не выполнялся.
2. Actual ACME/DNS/host/provider/escrow/checkpoint/DR acceptance после согласованной инфраструктурной подготовки.
3. Full Safari/Firefox/mobile browser matrix, manual WCAG, load/DR, SBOM/license/container/secret scans и independent security review; затем release review всех 32 MUST и residual risks с владельцем.
4. Реальный deployment только по отдельной явной команде пользователя.

## Decisions
- Controlled upgrade использует existing recovery gate/generation/runner locks. Prepare не применяет SQL и не заменяет полный stop старых instances. Backup/verify выполняются явно до startup migrations; автоматический downgrade/reopen отсутствует. DDL и ledger/fencing атомарны; crypto primitives/backup format сохранены.
- Stack и доменные инварианты сохранены: PostgreSQL authoritative, Redis ephemeral, immutable publish, one-time delivery, USER/ADMIN privacy. Никакие требования или residual risks не исключены/не приняты автоматически.
- FileX backup — logical encrypted archive, не pg_dump; payload/artifacts/roles/extensions/Redis/KEK не входят. Exact schema fingerprint и mandatory verified pre-backup без override обязательны.
- Runtime shared/exclusive locks и persisted generation охватывают HTTP/worker side effects; CLI берёт runner lock до queue, чтобы worker не перехватил transient recoveryKEK.
- `ops_*` history/gate не откатывается. Application audit key хранится encrypted и сохраняется при rewrap. Исторические backup descriptors требуют исторических KEK.
- Orphans не публикуются и не удаляются автоматически; missing/tampered bytes карантинируются. Capability resurrection проверяется явно перед reopen, administrative mutations требуют новой сверки.
- HMAC checkpoint сам по себе не доказывает отсутствие полного host+DB+key rollback; внешние anchors/escrow остаются операторской и release границей. Ed25519 подписывает только ZIP checksum manifest.

## Context
[Controlled upgrade](controlled-upgrade.md): изменены cmd/filex-cli, internal/database/bootstrap/maintenance/operations, Go PG tests, scripts/test-upgrade.py, Makefile/CI, contracts/traceability и связанные docs/wiki. Application schema остаётся 16 migrations, OpenAPI 107 operations. Изменений UI/crypto primitives нет. Рабочее дерево при начале шага было чистым на `31876b8`; все текущие изменения относятся к этому срезу.

[Вход](index.md), [roadmap](roadmap.md), [эксплуатация](operations-recovery.md), [development/evidence](development.md), [design review](design-review.md), [матрица](../docs/test-matrix.md), [CLI](../docs/cli.md), [incident runbook](../docs/incident-response.md). Изменены Go cmd/internal/SQL, Next.js components, contracts/scripts, Compose/Dockerfile API и документация. Новые основные пакеты: operations, maintenance; migrations 0014–0016. Чужие изменения не сбрасывались; working tree перед фазой 6 был чистым.

## Risks / blockers
Блокировок для локального закрытия фазы 6 нет. Production topology, real SMTP/S3 accounts, full DR/load/a11y/browser/security и внешнее хранение KEK/checkpoints не подтверждены. Архив ≤1 GiB, record ≤64 MiB, inventory ≤1 млн объектов, findings ≤10 тыс.; worker/CLI execution bounds 5/15 минут — не SLA/RPO/RTO. Backup без payload не восстанавливает физически удалённые bytes; потерянный historical KEK не восстанавливается кодом. Исходное интервью отсутствует. R-01–06 не приняты.

## Итог публикации фазы 6 и первого среза фазы 7
- Фаза 6: commit `02c53d9f384f378dd6a279b2d9f7d2e2432b82aa` отправлен в origin/main; local/remote SHA подтверждены. CI #38131827674 — completed/success.
- Фаза 7.1, первый срез: commit `50de642f2bf9e2c58e5e3ae0e92f92903cfa634e` отправлен в origin/main; local/remote SHA подтверждены после push. CI #38133772690 — completed/success: check/build, PostgreSQL integration, security, Chromium E2E и TLS ingress. Snapshot — `.cache/evidence/phase7-ci.json` (ignored); эта запись publication evidence меняет только документацию.
- [ADR-007](adr/007-ingress-and-network-policy.md), [production ingress](production-ingress.md): typed PG networkPolicy, точные доверенные proxy /32, single-valued XFF/XFP, effective IP для allowlist/login bucket, requireProxy/HTTPS/Host и bounded CIDR validation.
- Network policy/publicOrigin changes атомарно меняют active settings, закрывают gate, инвалидируют reconciliation и меняют generation. Проверена отмена privileged request; restart API/worker обязателен. Host network reset снимает только client allowlist, сохраняет transport trust, создаёт revision/audit и не открывает gate.
- CLI settings show и config ingress-check read-only, без migrations; settings apply использует draft/validate/CAS. Production ingress-check не одобряет release. API contracts остаются 107 operations, новая networkPolicy совместима с прежними settings v1.
- Standalone production config generator использует pinned base images и требует immutable API/web digests, private networks, non-root/read-only containers, TLS/ACME, strict SNI, HSTS и только 80/443 снаружи. Не выполняет deployment. Runtime APP_ENV=production остаётся запрещён.
- Проверки: make check PASS (44 Markdown, 32 requirements, Go/Next builds, unit/race, 4 contracts, 3 generator tests); полный PG/race PASS 32 tests (97.4 s HTTP/6.2 s repository); Chromium PASS 3 tests (28.6 s); make ingress PASS; make security PASS (npm 0, reachable govulncheck 0, прежний module-only advisory не скрыт).
- TLS fixture подтвердил TLS 1.2, отказ сервера TLS 1.1, strict CA, HTTP redirect/HSTS/Secure cookie, неизвестный Host/private routes, spoofed header rejection, CIDR deny, host reset и закрытый gate. Исправлены file-provider extension и неоднозначный service DNS: Traefik обращается по literal proxy-network IP.
- Evidence logs — ignored `.cache/evidence/phase7-*.log`. Тестовые projects/volumes удалены scoped scripts. Настоящие ACME/DNS/host/production mutations не выполнялись. Новые исходники: config/network, networkpolicy, httpapi/network, settings/CLI, production generator/TLS harness; новая SQL migration не потребовалась.
- Release phase 7 ещё не закрыта. Остаточные risks не приняты, все 32 MUST сохранены. Следующие gates перечислены выше.
