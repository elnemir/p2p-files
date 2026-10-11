# Разработка, воспроизведение и evidence

## Контекст
FileX имеет рабочий local development data plane: владельцы создают и публикуют передачи, получатели явно скачивают файлы, worker доставляет события и формирует отчёты. Локальные проверки выполнены на macOS arm64/Docker Desktop; актуальный статус прогона и публикации в Git — [session](session.md). Push/PR запускает `development-checks` в GitHub Actions: зависимости → базовые проверки и сборки → PostgreSQL integration → security → Chromium E2E → TLS ingress. Workflow не выполняет deployment; результаты remote CI подтверждаются отдельно. Production и реальные внешние SMTP/S3 accounts не использовались.

## Воспроизведение

```sh
npm ci
make check
make integration-docker
make security
PLAYWRIGHT_BROWSERS_PATH="$PWD/.cache/browsers" npx playwright install chromium
make e2e
make ingress
```

`make check`: gofmt/vet, TypeScript strict, Go race tests, contract tests, Markdown/traceability, OpenAPI/AsyncAPI validation, Go и Next.js build. Без `FILEX_TEST_DATABASE_URL` обычный Go suite пропускает PG tests; отдельный `make test-integration` требует этот URL. `integration-docker` создаёт disposable PostgreSQL с random credentials, запускает все `TestIntegration*` с race и удаляет только свой container. Каждому test соответствует отдельная schema.

`make e2e` создаёт отдельный Compose project, random KEK/password и USER/ADMIN accounts, проверяет приложение через Traefik на loopback 8090, перезапускает только свои API/worker в restore drill, удаляет только свой project/volumes. Порт должен быть свободен. Скриншоты/скачанные synthetic artifacts — `.cache/evidence`. Failure traces могут содержать тестовую сессию и не публикуются. Docker images/build cache остаются локально. `scripts/smoke-platform.py` отдельно проверяет PG outage → generic 503.

## Локальный запуск

```sh
python3 scripts/init-dev.py
make up
curl --fail http://localhost:8090/api/v1/status
```

`.env` создаётся mode 0600 и не перезаписывается. `make down` останавливает этот dev stack, сохраняя volumes. Initial ADMIN создаётся через `filex-cli user create`: JSON `{email,password,role}` подаётся на stdin внутри API container, например из файла mode 0600 через `docker compose --env-file .env -f deploy/compose.yaml exec -T api /usr/local/bin/filex-cli user create < /absolute/private-user.json`. Удалить файл после provisioning; не помещать пароль в shell history. Раздел ADMIN интерфейса требует повторного ввода пароля каждые пять минут.

## Configuration и ограничения

| Параметр | Поведение |
|---|---|
| APP_ENV | Только development/test; production fail closed до фазы 7. |
| DATABASE_URL / MASTER_KEK | Bootstrap; KEK — base64 32 random bytes, persisted sentinel запрещает неверный ключ. |
| HTTP_ADDR / INTERNAL_ADDR | Defaults 127.0.0.1:8080 и :9090; internal listener не публикуется Traefik. |
| DEV_PAYLOAD_ROOT / DEV_ARTIFACT_ROOT / DEV_BACKUP_ROOT | Только локальное bootstrap provisioning отдельных storage purposes; Compose использует отдельные volumes. |
| Settings v1 в PG | publicOrigin localhost:8090; session 43200 s; max TTL 604800 s; maxFiles 10000; maxPathBytes 1024. Draft→validate→apply с expected revision, rollback через прежнюю validated revision. |
| File policy | disabled/allow/deny по server-sniffed MIME; scanner — optional Go interface, внешний антивирус не настроен. |
| File / artifact / report | File ≤1 TiB, upload chunk ≤4 MiB, append attempts ≤2^20. Artifact ≤64 MiB, default TTL 24 h; ADMIN export TTL 60…604800 s. Report snapshot ≤16 MiB. |
| Export | Snapshot ≤50000 rows/16 MiB, DB statement ≤10 s. Global active jobs 2/per owner 1, priority aging 5 min. |
| Pools и streams | PG pool 10 + runtime-lock pool до 24 на процесс, statement/idle tx 10 s, lock 3 s; up to 8 concurrent content/upload/finalize/forensic operations, Argon2 semaphore 4. |
| HTTP | Header/read/idle 5/60/30 s; response stream guards каждые 32 KiB, write deadline 2 s, writer lease 10 s. |

Это инженерные bounds, не SLA. Go toolchain 1.26.9 берётся в project cache, системный Go не меняется. Runtime libraries фиксируются `go.sum` и `package-lock.json`; `.env`, caches и test artifacts исключены из Git. Redis развёрнут как ephemeral service, correctness/queues сейчас полностью в PG, потери Redis безопасны.

## Набор доказательств

- Go unit/race: passwords, settings, paths/fuzz seeds, AEAD vectors/tamper, ZIP64/ranges, trusted signatures, SMTP TLS/STARTTLS downgrade rejection, S3 TLS/conditional Put/Get/Delete.
- 32 PG integration/race cases: identity/settings; atomic publish/owner isolation/outbox rollback; leases/concurrent acquire/revoke; freeze/expiry/integrity; migration/rotation; tus/forensic; webhook dedup/retry/DLQ; tree/admin guards; report privacy/hashes/trash; export snapshot/retry/bulk/authorization; QR invalidation and key budgets; WS origin/revocation; backup/restore/pre-backup failure/drain/rollback, KEK rewrap/fresh DB, crypto containment, audit/ops checkpoint evidence, schedules/privacy, ADMIN exports/TTL, read-only upgrade check; trusted proxy/allowlist/effective IP, privileged request generation fencing и audited host reset.
- Playwright Chromium через proxy: interrupted PATCH→HEAD resume, publish, mobile recipient ZIP, report PDF/JSON verification, CSV export, settings dark theme, public help RU/EN/status/internal isolation; ADMIN backup→restore с pre-backup→restart→reauth→reconcile→reopen, checkpoint download, schedules, чужой export/TTL. JSON responses сравниваются с OpenAPI. Axe WCAG tags и screenshots: desktop, 390px и 360px.
- Contracts: 107 runtime operations, OpenAPI 3.1.1/AsyncAPI 3.0, versioned schemas, four contract/privacy/coverage tests. Generated files проверяются `--check`.
- Earlier bounded path fuzz: 284267 executions, 2 workers, около 6 s. Это smoke fuzz, не исчерпывающий security proof.
- npm production audit: zero vulnerabilities. Govulncheck: zero affected symbols/imported packages; module-only GO-2026-5932 относится к неимпортируемому `x/crypto/openpgp` (Argon2id использует другую часть модуля, исправления advisory нет). Не скрывать это как «ни одного advisory».

Итог выполнения этих gates записывается в session после завершения команд, а не выводится из списка тестов.

## Неподтверждённое и дальнейшая работа
Настоящая доставка во внешний mailbox и совместимость конкретного S3 provider проверяются при его подключении. Full Safari/Firefox/mobile browser matrix, ручная WCAG 2.2 AA приёмка, production load/DR, отдельный frontend lint policy, SBOM/license/container/secret scans и independent security review — фаза 7. Backup/restore/reconciliation реализованы фазой 6, [runbook](operations-recovery.md); local fixture evidence не заменяет production DR. `APP_ENV=production` сохраняет запрет запуска. Зелёный development CI не закрывает эти release gates.

## Исторический итог фаз 1–5 (2026-10-11)

| Gate | Результат |
|---|---|
| make check | PASS: gofmt/vet, TS, unit/race, 4 contract tests, 40 Markdown/32 requirements, 93-operation OpenAPI/AsyncAPI, Go/Next build. |
| make integration-docker | PASS: 18 API + 5 repository tests с race; последний HTTP suite 61.1 s, repository 2.7 s. |
| make e2e | PASS: 2 Chromium scenarios, 14.6 s; QR artifact download включён; no axe violations/overflow/response contract failures на проверенных страницах. |
| Visual | PDF A4/embedded Cyrillic/long paths/hashes и mobile recipient/dark settings просмотрены. |
| make security | PASS: npm production 0, govulncheck affected symbols/packages 0; module-only openpgp advisory описан выше. |
| GitHub Actions | PASS: [development-checks #38125813423](https://github.com/elnemir/p2p-files/actions/runs/38125813423) для `3af2f0c`, Ubuntu runner; check/build, PG integration, security и Chromium E2E. |
| Workspace | Whitespace 0, source files 178, migrations 13; статус commit/push и remote CI — в session. |

На момент закрытия этапов 1–5 в requirement matrix было 22 verified/9 partial/1 planned. Это исторический результат; актуальная матрица после фазы 6 приведена ниже.

## Итог фазы 6 (2026-10-11)

| Gate | Подтверждённый результат |
|---|---|
| `make check` | PASS после последних code changes: gofmt/vet, TypeScript strict, Go unit/race, 4 contract tests, 42 Markdown/32 requirements, 107-operation OpenAPI/AsyncAPI, Go/Next.js build. Лог: `.cache/evidence/phase6-check.log`. |
| Полный PostgreSQL suite | PASS: 26 HTTP + 5 repository integration/race tests; HTTP 89.4 s, repository 3.4 s. Восемь сценариев фазы 6 добавлены к прежним 23. |
| Затронутые PG сценарии | PASS: после согласованного crypto-health snapshot/signing-pair checks повторены 8 сценариев (38.0 s). Последний targeted run recovery/ADMIN exports — 2 PASS, HTTP 12.6 s: concurrent restore rejection, stale reconciliation после capability revoke, role/reauth/TTL и Unicode reason bounds. Лог: `.cache/evidence/phase6-final-pg.log`. |
| `make e2e` | PASS: 3 Chromium scenarios, 29.1 s. Полный restore/restart/relogin/reopen, mandatory pre-backup и ADMIN export download проверены через Traefik. Axe violations, horizontal overflow и JSON response contract errors на проверенных страницах — 0. |
| Visual | Security incident PDF получен из integration HTTP response, отрендерен Poppler и просмотрен: A4, читаемые Cyrillic/hash/JSON без clipping. Restore/reopened mobile screenshots 390px просмотрены; notices исправлены и повторно проверены. |
| `make security` | PASS: npm production vulnerabilities 0; govulncheck affected symbols/imported packages 0. Неимпортируемый module-only openpgp advisory описан выше. |
| Review/contracts/wiki | ADR-006, threat/design review, migrations 0014–0016, operational/incident runbooks, CLI, matrix и session синхронизированы. 27 requirements verified по local evidence; 5 partial остаются фазой 7. |
| Публикация | Фаза 6 опубликована как `02c53d9`, [GitHub Actions #38131827674](https://github.com/elnemir/p2p-files/actions/runs/38131827674) success: build/check, PostgreSQL, security, Chromium E2E. Production deployment отсутствует. |

Unit/race archive checks включают wrong KEK, byte tamper, truncation, swapped/missing chunks, incomplete descriptor, malformed/out-of-order snapshot records, Local pagination и TLS S3 roundtrip/inventory pagination. PG drills дополнительно проверяют drain активного writer, rollback failed import, невозврат KEK usage reservations, rewrap/чистую БД с новым KEK, audit и ops ledger corruption/checkpoints, schedule concurrency и critical crypto containment при невозможности записать report.

Для точечного повторения: `python3 scripts/test-integration.py 'TestIntegrationRecoveryBackupFaultAndReopen'`. Это фильтр тестов, не замена полного suite для новой общей регрессии. Логи, snapshots, скачанные synthetic artifacts и browser traces сохраняются только в ignored `.cache`; scoped integration/E2E scripts удалили свои containers/volumes. Tests не обращались к production или реальным SMTP/S3 accounts.

Фаза 6 закрыта по локальным development gates. Следующий этап — release acceptance фазы 7, включая production topology, реальные storage/provider настройки, внешнее хранение checkpoints/KEK, full browser/manual WCAG и load/DR/security/SBOM. Локальный архив ограничен 1 GiB и совпадающей схемой; этот результат не обещает production RPO/RTO.

## Первый срез фазы 7: ingress/network policy (2026-10-11)

[Runbook и ограничения](production-ingress.md). Реализованы typed PG policy, доверенный single-hop proxy, эффективный IP для login rate limit, CIDR/HTTPS/Host guards, settings generation fencing и host reset. Generator production profile не запускает контейнеры; APP_ENV=production остаётся закрытым release guard.

| Gate | Результат текущего среза |
|---|---|
| `make check` | PASS: unit/race (включая malformed/spoofed proxy/CIDR), vet, TS, 4 contract tests, 3 generator tests, Go/Next build, 44 Markdown/32 requirements, 107 API operations. `.cache/evidence/phase7-check.log`. |
| `make integration-docker` | PASS: 27 HTTP + 5 repository = 32 cases с race; HTTP 97.4 s/repository 6.2 s. Новый network scenario проверяет CAS rollback, отмену privileged request, mandatory restart, IP bucket, secure cookie, reset/audit. `.cache/evidence/phase7-all-pg.log`. |
| `make e2e` | PASS: прежние 3 Chromium scenarios, 28.6 s; пользовательская передача/отчёты/QR/export и ADMIN restore/restart/reopen. Axe/overflow/response-contract errors на проверенных страницах — 0. `.cache/evidence/phase7-e2e.log`. |
| `make ingress` | PASS: настоящий pinned Traefik, тестовая CA с полной проверкой; TLS 1.2 success, сервер отвергает TLS 1.1, неизвестная CA и чужой Host отвергаются, redirect/HSTS/Secure cookie, private routes, forwarded spoof rejection, CIDR deny, audited host reset и закрытый gate после restart. `.cache/evidence/phase7-ingress.log`. |
| `make security` | PASS: npm production 0; govulncheck 0 affected symbols/imported packages, прежний module-only openpgp advisory сохраняется. `.cache/evidence/phase7-security.log`. |
| Публикация и CI | Первый срез опубликован как `50de642`; [GitHub Actions #38133772690](https://github.com/elnemir/p2p-files/actions/runs/38133772690) completed/success: check/build, PG integration, security, Chromium E2E, TLS ingress. Snapshot — `.cache/evidence/phase7-ci.json`. Последующая фиксация этого evidence меняет только документацию. |

Integration выявила необходимость `.yaml` extension для file provider и literal backend IP в proxy network: Docker DNS при общей egress мог выбрать другой интерфейс. Эти ошибки исправлены и повторный TLS drill прошёл. Test CA содержит явные X.509 extensions для строгой проверки современными TLS clients; проверки сертификата не отключались.

Локальные fixtures используют random secrets и собственные projects, удаляют только свои containers/volumes, не контактируют с public ACME/production/provider accounts. GitHub CI теперь включает `make ingress`; результат публикации текущего среза хранится в session. Вся фаза 7 не закрыта: controlled upgrade, full browser/manual WCAG, load/DR, independent security и SBOM/container/secret/license gates остаются следующими шагами.
