# Release checklist — фаза 7

## Контекст
На 2026-10-11 технические local gates выполнены для кандидата 7dce226/source dbcbbf3b774c888d388781933cc3e7e502796eee499a474c6213cdec94566705; полная release приёмка заблокирована внешними условиями. Это список конкретных проверок, а не разрешение production запуска. Evidence: [история проверок](history/development-evidence-2026-10-11.md), [session](session.md), [release engineering](release-engineering.md).

## Исторически выполненные технические проверки

Отметки ниже относятся к этому кандидату. Для нового release повторить все gates по [testing](testing.md); перенос tooling/CLI paths меняет hash и не наследует прежний PASS.
- [x] Все 32 MUST отражены в requirement → implementation → test; 27 verified/5 partial имеют объяснённую release границу, требования не исключены.
- [x] Внутренние architecture/threat/crypto review и ADR актуальны.
- [x] Go vet/staticcheck, TS strict/ESLint, formatting, history/worktree secrets, dependency scans.
- [x] Unit/race, 36 PostgreSQL integration tests, три настоящие fuzz campaigns и fault injection.
- [x] 25 E2E scenarios: Chromium/Firefox/WebKit и две mobile emulations, axe/keyboard/error/320px/themes, 10 reviewed pixel baselines без автообновления в gate.
- [x] Fresh/prefix migrations, controlled image upgrade/startup rejection, OpenAPI/AsyncAPI/JSON Schema compatibility и generated type sync.
- [x] TLS ingress/private routes/trusted proxy/CIDR/reset; тестовая CA, public ACME не вызывался.
- [x] 2050 MiB payload / 20000 synthetic rows: resume, Local→TLS S3, rotation, Redis loss, concurrent acquire, worker SIGKILL/retry, export/VACUUM, backup/restore/reconcile/reopen, полный ZIP SHA-256/audit, 0 OOM.
- [x] Пять итоговых Docker images: 0 HIGH/CRITICAL; JSON/CycloneDX SBOM, npm/Go/font license inventory подготовлены.
- [x] CHANGELOG, release notes, docs links, five-platform CLI verifier review bundle/SHA256SUMS. Бинарники cross-built; запуск на каждой целевой OS не заявляется.
- [x] Source-bound gate records, completeness/freshness/negative tests и fail-closed `make release-check`.

## Открытая внешняя приёмка
- [ ] Все 32 MUST имеют verified release evidence; все 14 внешних решений reviewer/owner с evidence внесены в `contracts/release-acceptance.json`.
- [ ] VPS/IPv4/DNS/ACME issuance+renewal, firewall/SSH/resource limits и host backups. Окружение ещё не подготовлено — подтверждено владельцем.
- [ ] Реальные SMTP/S3, independent backup/KEK/checkpoint storage, target load/DR/RPO/RTO.
- [ ] Настоящие Safari/mobile devices, ручной WCAG/assistive-technology review.
- [ ] Независимый security review, triage UNKNOWN advisories и явное решение по R-01–06.
- [ ] Владелец рассмотрел [конкретные third-party licenses](third-party-review.md), OS/container/font obligations; inventory не означает approval.
- [ ] Перед фактическим релизом повторены свежие gates, `make release-check` PASS; подготовлены сохраняемые release evidence. Tag/GitHub Release только после этого, production deployment — по отдельной команде.

## Допуск окончательного кандидата

- [ ] DEV-01 определил и реализовал reviewed механизм допуска: существующий безусловный production guard не исчезает после release-check.
- [ ] При любом изменении code/config/contracts/rules/images после review повторена квалификация окончательного source hash; human evidence относится к этому же scope.
- [ ] DEV-03/04: определён состав поставки/лицензия проекта и сохранён независимый долговременный пакет evidence; одних 14-дневных CI artifacts недостаточно.

Зависимости и роли — [implementation plan](implementation-plan.md), протоколы — [external acceptance](environment-acceptance.md).
