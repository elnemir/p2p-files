# Разработка, воспроизведение и evidence

> Историческая запись: статусы, команды и формулировки ниже относятся к указанному этапу. Актуальные [состояние](../current-state.md), [план](../implementation-plan.md), [сессия](../session.md) и [инструкции разработки](../development.md) имеют приоритет для продолжения.

## Контекст
FileX имеет рабочий local development data plane: владельцы создают и публикуют передачи, получатели явно скачивают файлы, worker доставляет события и формирует отчёты. Локальные проверки выполнены на macOS arm64/Docker Desktop; актуальный статус прогона и публикации в Git — [session](../sessions/2026-10-11-phase7.md). Push/PR запускает `development-checks` в GitHub Actions: functional job (static/unit/race/contracts/build/PG/fuzz/five-profile browser/TLS/upgrade/load/verifier) и supply-chain job (source/secret/license/container/SBOM). Workflow не выполняет deployment; результаты remote CI подтверждаются отдельно. Production и реальные внешние SMTP/S3 accounts не использовались.

## Воспроизведение

```sh
npm ci
make check
make integration-docker
make security
make e2e
make ingress
make upgrade-drill
```

`make check`: gofmt/vet, TypeScript strict, Go race tests, contract tests, Markdown/traceability, OpenAPI/AsyncAPI validation, Go и Next.js build. Без `FILEX_TEST_DATABASE_URL` обычный Go suite пропускает PG tests; отдельный `make test-integration` требует этот URL. `integration-docker` создаёт disposable PostgreSQL с random credentials, запускает все `TestIntegration*` с race и удаляет только свой container. Каждому test соответствует отдельная schema.

`make e2e` создаёт отдельный Compose project, random KEK/password и USER/ADMIN accounts, проверяет приложение через Traefik на loopback 8090, перезапускает только свои API/worker в restore drill, удаляет только свой project/volumes. Порт должен быть свободен. Скриншоты/скачанные synthetic artifacts — `.cache/evidence`. Failure traces могут содержать тестовую сессию и не публикуются. Docker images/build cache остаются локально. `scripts/smoke-platform.py` отдельно проверяет PG outage → generic 503.

`make upgrade-drill`: закреплённый baseline Git commit → отдельный image; текущие файлы → другой candidate image. Изолированный Compose/random loopback port проверяет prepare/drain, stop, verified backup, deliberate startup failures, image replacement, reconciliation/reopen и сохранность payload/одноразовой выдачи. Timing/границы: [runbook](../controlled-upgrade.md). Требуется Git history с baseline; CI fetch-depth=0. Production инфраструктура не используется.

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
- 36 PG integration/race cases: identity/settings; atomic publish/owner isolation/outbox rollback; leases/concurrent acquire/revoke; freeze/expiry/integrity; migration/rotation; tus/forensic; webhook dedup/retry/DLQ; tree/admin guards; report privacy/hashes/trash; export snapshot/retry/bulk/authorization; QR invalidation and key budgets; WS origin/revocation; backup/restore/pre-backup failure/drain/rollback, KEK rewrap/fresh DB, crypto containment, audit/ops checkpoint evidence, schedules/privacy, ADMIN exports/TTL, read-only upgrade check; trusted proxy/allowlist/effective IP, privileged request generation fencing и audited host reset; strict migration prefix/concurrent startup/DDL rollback и upgrade prepare drain/timeout/reopen.
- Playwright Chromium через proxy: interrupted PATCH→HEAD resume, publish, mobile recipient ZIP, report PDF/JSON verification, CSV export, settings dark theme, public help RU/EN/status/internal isolation; ADMIN backup→restore с pre-backup→restart→reauth→reconcile→reopen, checkpoint download, schedules, чужой export/TTL. JSON responses сравниваются с OpenAPI. Axe WCAG tags и screenshots: desktop, 390px и 360px.
- Contracts: 107 runtime operations, OpenAPI 3.1.1/AsyncAPI 3.0, versioned schemas, four contract/privacy/coverage tests. Generated files проверяются `--check`.
- Earlier bounded path fuzz: 284267 executions, 2 workers, около 6 s. Это smoke fuzz, не исчерпывающий security proof.
- npm production audit: zero vulnerabilities. Govulncheck: zero affected symbols/imported packages; module-only GO-2026-5932 относится к неимпортируемому `x/crypto/openpgp` (Argon2id использует другую часть модуля, исправления advisory нет). Не скрывать это как «ни одного advisory».

Итог выполнения этих gates записывается в session после завершения команд, а не выводится из списка тестов.

## Неподтверждённое и дальнейшая работа
Настоящая доставка во внешний mailbox и совместимость конкретного S3 provider проверяются при его подключении. Автоматическая Chromium/Firefox/WebKit/mobile-emulation matrix, frontend lint, fuzz, local load/DR и SBOM/license/container/secret gates реализованы фазой 7; полный текущий отчёт — [release engineering](../release-engineering.md). Настоящие Safari/mobile devices, ручная WCAG 2.2 AA/assistive technology, target load/DR и independent security review остаются внешней приёмкой. Окружение не подготовлено по подтверждению владельца от 2026-10-11. Backup/restore/reconciliation реализованы фазой 6, [runbook](../operations-recovery.md); local fixture evidence не заменяет production DR. `APP_ENV=production` сохраняет запрет запуска. Зелёный development CI не закрывает эти release gates.

## Controlled upgrade: результат 2026-10-11

[ADR-008](../adr/008-controlled-upgrade.md), [runbook](../controlled-upgrade.md). Общий strict migration plan исправляет acceptance неизвестного ledger при startup; host prepare закрывает gate и дожидается старого поколения. Новых env/migrations/HTTP операций нет.

| Gate | Фактический результат |
|---|---|
| `make check` | PASS: gofmt/vet, Go unit/race, TS strict, 4 contracts, 3 generator tests, 46 Markdown/32 requirements, Go/Next builds. `.cache/evidence/phase7-upgrade-check.log`. |
| `make integration-docker` | PASS: 36 top-level tests (2 database + 29 HTTP + 5 repository), с race; database 5.700 s, HTTP 104.576 s, repository 4.247 s. Реальный SQL prefix 15→16, concurrent startup, unknown/checksum/hole, active runtime/DDL timeout, failed multi-DDL rollback, prepare drain/reopen и timeout/retry. `.cache/evidence/phase7-upgrade-all-pg.log`. |
| `make upgrade-drill` | PASS: baseline `50de642` → candidate modified sources на HEAD `31876b8`, реальные разные image IDs сохранены. Schema 16 остаётся неизменной. Verified backup, prepare/drain, unknown migration/wrong KEK startup failures, replacement, closed restart, reconciliation/reopen, retained sessions/payload SHA и ровно одно завершение. `.cache/evidence/phase7-upgrade-drill.log`, `.cache/evidence/phase7-upgrade.json`. |
| Измерение | Window 6.489 s, sampled unavailable 5.878 s, 29 неуспешных samples с интервалом 200 ms. Время включает backup и injected failures на маленьком локальном fixture; не production SLA/RTO. |
| `make e2e` | PASS: 3 Chromium scenarios, 28.6 s; transfer/report/export и operational restore/restart/reconcile/reopen. На проверенных страницах axe/overflow/response contract errors — 0. `.cache/evidence/phase7-upgrade-e2e.log`. |
| `make ingress` | PASS: strict CA, TLS versions/redirect/HSTS, proxy spoof/CIDR, Secure cookies и host reset/fencing. `.cache/evidence/phase7-upgrade-ingress.log`. |
| `make security` | PASS: npm production 0, govulncheck affected symbols/imported packages 0; прежний module-only openpgp advisory сохраняется. `.cache/evidence/phase7-upgrade-security.log`. |

Fixtures завершили scoped cleanup; `docker ps -a --filter name=filex-` не показал оставшихся контейнеров. Логи содержат synthetic fixture evidence и остаются ignored. В CI добавлен `make upgrade-drill` и fetch Git history для pinned baseline. Commit `dda4587` опубликован; [CI #38135958275](https://github.com/elnemir/p2p-files/actions/runs/38135958275) completed/success всех gates, включая upgrade. Remote fixture использовал exact clean candidate commit: window 6.251 s, sampled unavailable 5.542 s, 28 samples. Snapshot/log/JSON — `.cache/evidence/phase7-upgrade-ci.json`, `phase7-upgrade-ci.log`, `phase7-upgrade-ci-drill.json`. Эта последующая фиксация publication evidence меняет только документацию. Фаза 7 в целом не закрыта.

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

[Runbook и ограничения](../production-ingress.md). Реализованы typed PG policy, доверенный single-hop proxy, эффективный IP для login rate limit, CIDR/HTTPS/Host guards, settings generation fencing и host reset. Generator production profile не запускает контейнеры; APP_ENV=production остаётся закрытым release guard.

| Gate | Результат текущего среза |
|---|---|
| `make check` | PASS: unit/race (включая malformed/spoofed proxy/CIDR), vet, TS, 4 contract tests, 3 generator tests, Go/Next build, 44 Markdown/32 requirements, 107 API operations. `.cache/evidence/phase7-check.log`. |
| `make integration-docker` | PASS: 27 HTTP + 5 repository = 32 cases с race; HTTP 97.4 s/repository 6.2 s. Новый network scenario проверяет CAS rollback, отмену privileged request, mandatory restart, IP bucket, secure cookie, reset/audit. `.cache/evidence/phase7-all-pg.log`. |
| `make e2e` | PASS: прежние 3 Chromium scenarios, 28.6 s; пользовательская передача/отчёты/QR/export и ADMIN restore/restart/reopen. Axe/overflow/response-contract errors на проверенных страницах — 0. `.cache/evidence/phase7-e2e.log`. |
| `make ingress` | PASS: настоящий pinned Traefik, тестовая CA с полной проверкой; TLS 1.2 success, сервер отвергает TLS 1.1, неизвестная CA и чужой Host отвергаются, redirect/HSTS/Secure cookie, private routes, forwarded spoof rejection, CIDR deny, audited host reset и закрытый gate после restart. `.cache/evidence/phase7-ingress.log`. |
| `make security` | PASS: npm production 0; govulncheck 0 affected symbols/imported packages, прежний module-only openpgp advisory сохраняется. `.cache/evidence/phase7-security.log`. |
| Публикация и CI | Первый срез опубликован как `50de642`; [GitHub Actions #38133772690](https://github.com/elnemir/p2p-files/actions/runs/38133772690) completed/success: check/build, PG integration, security, Chromium E2E, TLS ingress. Snapshot — `.cache/evidence/phase7-ci.json`. Последующая фиксация этого evidence меняет только документацию. |

Integration выявила необходимость `.yaml` extension для file provider и literal backend IP в proxy network: Docker DNS при общей egress мог выбрать другой интерфейс. Эти ошибки исправлены и повторный TLS drill прошёл. Test CA содержит явные X.509 extensions для строгой проверки современными TLS clients; проверки сертификата не отключались.

Локальные fixtures используют random secrets и собственные projects, удаляют только свои containers/volumes, не контактируют с public ACME/production/provider accounts. Этот ingress срез добавил `make ingress` в CI; его publication evidence хранится в session. Controlled upgrade выполнен следующим срезом ниже. Full browser/manual WCAG, load/DR, independent security и SBOM/container/secret/license gates остаются открытыми.


## Завершающий технический проход фазы 7 (2026-10-11)
Новый набор проверок и способ воспроизведения: [release engineering](../release-engineering.md), [нагрузочный профиль](../performance-testing.md), [license review](../third-party-review.md). Приведённые выше таблицы относятся к ранним commits, а не к текущему кандидату. Финальные source-bound записи сохраняются в `.cache/evidence/gates/`; текущий статус каждого gate — [session](../sessions/2026-10-11-phase7.md).

Browser harness использует один immutable официальный Playwright Linux image и отдельный Compose project для каждого из пяти browser/device profiles. Host runner управляет restart только своих API/worker; browser server не получает Docker socket. 25 сценариев первого полного прогона прошли: 40.2/49.4/55.6/39.3/50.6 s по Chromium/Firefox/WebKit/mobile Chromium/mobile WebKit. Далее gate повторяется с неизменяемыми baseline images. Desktop/light, desktop/dark, long filename, mobile help и mobile restore review просмотрены. Проверки настоящего Safari/iOS/Android и assistive technology не подменяются эмуляцией. macOS native Firefox не запустился (profile error); поэтому воспроизводимая матрица использует Linux container и на macOS, и на CI.

Source scan охватывает все lockfile dependencies; Trivy images — пять итоговых images. Первоначальные HIGH/CRITICAL устранены без severity exceptions. Остались module advisories UNKNOWN для x/crypto/openpgp, отражённые в отчётах; reachable приложение проверяется отдельно govulncheck. License inventory: 443 npm entries, 24 Go runtime libraries, bundled font SHA, 16 записей для отдельного owner review. Это inventory completeness, не license acceptance.


### Полный локальный load/DR: PASS
Source-bound профиль 2050 MiB и 20000 synthetic transfers прошёл: один файл 2149580800 bytes, 4 MiB chunks, concurrency 8, Local + TLS S3 fixture. Docker VM: Linux aarch64, 8 CPU, 4108632064 bytes RAM, kernel 6.12.76-linuxkit, overlay2. Browser matrix выполнялась параллельно; это общий development host, не изолированный benchmark.

| Этап | Измерено |
|---|---|
| tus upload + finalize | 38.441 s, 53.329 MiB/s |
| Local → TLS S3 | 50.557 s |
| Payload key rotation | 65.664 s |
| Worker SIGKILL + injected lease expiry + export/VACUUM | 5.442 s, 20001 CSV rows, 0 idle snapshots |
| Backup + verify | 1.149 s |
| Restore/restart/reconcile/reopen | 29.458 s, 1 file / 1540 objects |
| ZIP download + full SHA-256 check | 116.739 s, 17.560 MiB/s |
| 100 status requests, concurrency 8 | p50 4.272 ms, p95 20.477 ms, max 25.090 ms |

1022 container samples; sampled peak RAM: API 184.9 MiB, worker 46.8, PostgreSQL 135.8, Traefik 22.2, web 36.0, Redis 3.8, S3 fixture 189.3. Sampled peak CPU: API 107.68%, worker 106.42%, PostgreSQL 98.33%, S3 fixture 111.78% (100% соответствует одному CPU). Raw CPU/RAM/IO records: `.cache/evidence/performance/results-2050mib-20000rows.json`. OOM не было, SHA-256 исходного и восстановленного ZIP payload совпал, repeated lease rejected, audit verified, старые сессии отменены, до reviewed reopen data plane закрыт. Все 15 assertions PASS.

Эти времена не являются production SLA/RPO/RTO. В частности, принудительное истечение lease не измеряет реальную трёхминутную паузу crash recovery; backup исключает payload bytes, которые сверяются отдельно из storage. Реальные provider/host/escrow/DR acceptance остаются заблокированными отсутствием окружения.


### Публикация и CI на GitHub
Implementation commit `7dce226f1dffa7241267e3546371ad7b2665c423` отправлен в origin/main. [GitHub Actions #38144319522](https://github.com/elnemir/p2p-files/actions/runs/38144319522) completed/success обоих jobs. Все 13 CI records имеют matching commit/source hash `dbcbbf3b774c888d388781933cc3e7e502796eee499a474c6213cdec94566705`, совпадающий с локальным кандидатом. Артефакты сохранены в `.cache/evidence/ci-38144319522/{functional,supply-chain}`.

CI подтвердил code/build/PG/race/security/fuzz, 25 browser scenarios с теми же Linux pixel baselines (0 skipped/failed/flaky), TLS/upgrade, compact load/DR и verifier bundle. Пять контейнеров Linux amd64 также имеют 0 HIGH/CRITICAL. CI browser tests заняли 38.2/45.1/50.8/38.5/48.2 s, весь browser gate вместе с первой сборкой runtime images — 827.9 s. CI выполняет 32 MiB/2000 rows; это не замена измеренному локальному 2050 MiB/20000 rows профилю выше. Последующая фиксация publication evidence меняет только wiki; source hash и реализация сохранены. Внешняя приёмка, лицензии и R-01–06 открыты, production deployment отсутствует.
