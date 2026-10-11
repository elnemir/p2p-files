# Разработка, воспроизведение и evidence

## Контекст
FileX имеет рабочий local development data plane: владельцы создают и публикуют передачи, получатели явно скачивают файлы, worker доставляет события и формирует отчёты. Локальные проверки выполнены на macOS arm64/Docker Desktop; актуальный статус прогона и публикации в Git — [session](session.md). Push/PR запускает `development-checks` в GitHub Actions: зависимости → базовые проверки и сборки → PostgreSQL integration → security → Chromium E2E. Workflow не выполняет deployment; результаты remote CI подтверждаются отдельно. Production и реальные внешние SMTP/S3 accounts не использовались.

## Воспроизведение

```sh
npm ci
make check
make integration-docker
make security
PLAYWRIGHT_BROWSERS_PATH="$PWD/.cache/browsers" npx playwright install chromium
make e2e
```

`make check`: gofmt/vet, TypeScript strict, Go race tests, contract tests, Markdown/traceability, OpenAPI/AsyncAPI validation, Go и Next.js build. Без `FILEX_TEST_DATABASE_URL` обычный Go suite пропускает PG tests; отдельный `make test-integration` требует этот URL. `integration-docker` создаёт disposable PostgreSQL с random credentials, запускает все `TestIntegration*` с race и удаляет только свой container. Каждому test соответствует отдельная schema.

`make e2e` создаёт отдельный Compose project, random KEK/password и пользователя browser@example.test, проверяет приложение через Traefik на loopback 8090, удаляет только свой project/volumes. Порт должен быть свободен. Скриншоты/скачанные synthetic artifacts — `.cache/evidence`. Failure traces могут содержать тестовую сессию и не публикуются. Docker images/build cache остаются локально. `scripts/smoke-platform.py` отдельно проверяет PG outage → generic 503.

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
| APP_ENV | Только development/test; production fail closed до 6–7. |
| DATABASE_URL / MASTER_KEK | Bootstrap; KEK — base64 32 random bytes, persisted sentinel запрещает неверный ключ. |
| HTTP_ADDR / INTERNAL_ADDR | Defaults 127.0.0.1:8080 и :9090; internal listener не публикуется Traefik. |
| DEV_PAYLOAD_ROOT / DEV_ARTIFACT_ROOT | Только локальное bootstrap provisioning отдельных storage purposes; Compose использует отдельные volumes. |
| Settings v1 в PG | publicOrigin localhost:8090; session 43200 s; max TTL 604800 s; maxFiles 10000; maxPathBytes 1024. Draft→validate→apply с expected revision, rollback через прежнюю validated revision. |
| File policy | disabled/allow/deny по server-sniffed MIME; scanner — optional Go interface, внешний антивирус не настроен. |
| File / artifact / report | File ≤1 TiB, upload chunk ≤4 MiB, append attempts ≤2^20. Artifact ≤64 MiB, TTL 24 h. Report snapshot ≤16 MiB. |
| Export | Snapshot ≤50000 rows/16 MiB, DB statement ≤10 s. Global active jobs 2/per owner 1, priority aging 5 min. |
| Pools и streams | PG pool 10, statement/idle tx 10 s, lock 3 s; up to 8 concurrent content/upload/finalize/forensic operations, Argon2 semaphore 4. |
| HTTP | Header/read/idle 5/60/30 s; response stream guards каждые 32 KiB, write deadline 2 s, writer lease 10 s. |

Это инженерные bounds, не SLA. Go toolchain 1.26.9 берётся в project cache, системный Go не меняется. Runtime libraries фиксируются `go.sum` и `package-lock.json`; `.env`, caches и test artifacts исключены из Git. Redis развёрнут как ephemeral service, correctness/queues сейчас полностью в PG, потери Redis безопасны.

## Набор доказательств

- Go unit/race: passwords, settings, paths/fuzz seeds, AEAD vectors/tamper, ZIP64/ranges, trusted signatures, SMTP TLS/STARTTLS downgrade rejection, S3 TLS/conditional Put/Get/Delete.
- 23 PG integration/race cases: identity/settings; atomic publish/owner isolation/outbox rollback; leases/concurrent acquire/revoke; freeze/expiry/integrity; migration/rotation; tus/forensic; webhook dedup/retry/DLQ; tree/admin guards; report privacy/hashes/trash; export snapshot/retry/bulk/authorization; QR invalidation and key budgets; WS origin/revocation.
- Playwright Chromium через proxy: interrupted PATCH→HEAD resume, publish, mobile recipient ZIP, report PDF/JSON verification, CSV export, settings dark theme, public help RU/EN/status/internal isolation. JSON responses сравниваются с OpenAPI. Axe WCAG tags и screenshots: desktop, 390px и 360px.
- Contracts: 93 runtime operations, OpenAPI 3.1.1/AsyncAPI 3.0, versioned schemas, four contract/privacy/coverage tests. Generated files проверяются `--check`.
- Earlier bounded path fuzz: 284267 executions, 2 workers, около 6 s. Это smoke fuzz, не исчерпывающий security proof.
- npm production audit: zero vulnerabilities. Govulncheck: zero affected symbols/imported packages; module-only GO-2026-5932 относится к неимпортируемому `x/crypto/openpgp` (Argon2id использует другую часть модуля, исправления advisory нет). Не скрывать это как «ни одного advisory».

Итог выполнения этих gates записывается в session после завершения команд, а не выводится из списка тестов.

## Неподтверждённое и дальнейшая работа
Настоящая доставка во внешний mailbox и совместимость конкретного S3 provider проверяются при его подключении. Full Safari/Firefox/mobile browser matrix, ручная WCAG 2.2 AA приёмка, production load/DR, отдельный frontend lint policy, SBOM/license/container/secret scans и independent security review — фаза 7. Backup/restore/reconciliation — фаза 6. `APP_ENV=production` сохраняет запрет запуска. Зелёный development CI не закрывает эти release gates.

## Итог прогона 2026-10-11

| Gate | Результат |
|---|---|
| make check | PASS: gofmt/vet, TS, unit/race, 4 contract tests, 40 Markdown/32 requirements, 93-operation OpenAPI/AsyncAPI, Go/Next build. |
| make integration-docker | PASS: 18 API + 5 repository tests с race; последний HTTP suite 61.1 s, repository 2.7 s. |
| make e2e | PASS: 2 Chromium scenarios, 14.6 s; QR artifact download включён; no axe violations/overflow/response contract failures на проверенных страницах. |
| Visual | PDF A4/embedded Cyrillic/long paths/hashes и mobile recipient/dark settings просмотрены. |
| make security | PASS: npm production 0, govulncheck affected symbols/packages 0; module-only openpgp advisory описан выше. |
| Workspace | Whitespace 0, source files 178, migrations 13; статус commit/push и remote CI — в session. |

Этапы 1–5 закрыты. В requirement matrix 22 verified/9 partial/1 planned: оставшийся cross-phase scope не исключён и не принят как residual risk.
