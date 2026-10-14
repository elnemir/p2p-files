# Changelog

## Unreleased — Офлайн-лицензирование

- Subscription/perpetual, отдельная Ed25519 vendor подпись, ADMIN preview/import и host CLI; migration 0020 сохраняет license identity/revision/clock в operations control plane.
- Общие PG guards ограничивают enabled users, transfer bytes и DAV/SSO/branding; expiry сохраняет старую recipient выдачу, one-time/TTL/freeze и recovery.
- Отдельные закрытая Next.js панель, Go OIDC MFA API и signer, immutable/idempotent issuance с transactional outbox. Private key не входит в клиентскую поставку и web API.
- Подписанные release metadata ограничивают обновления до migration; ephemeral signed fixtures используются в локальных проверках. [Эксплуатация и ограничения](wiki/licensing-operations.md), текущие результаты — wiki/session.md.

## Unreleased — Полный тур, 2026-10-14

- Пересняты 71 скриншот и 26 видеоглав актуального USER/ADMIN: настройки draft/validate/apply, история и восстановление отчёта, формы OIDC/SAML/LDAP, настоящий локальный SAML-вход/регистрация USER, white labeling и публичный мобильный бренд.
- Единый MP4 с голосом ElevenLabs Liza, главами и русскими субтитрами; 8 новых фрагментов и 18 проверенных неизменённых записей. Галерея и Markdown согласованы с manifest и независимым планом съёмки.
- Неполная съёмка и устаревший narration cache отклоняются до замены поставки; добавлены негативные проверки. PNG сначала сохраняются в приватный staging. Обновлены обзорные страницы wiki, инструкции и границы внешней приёмки; runtime/API/env/migrations не изменены.

## Unreleased — Удобство USER/ADMIN

- Навигация кабинета и тематические ADMIN разделы, reauth на месте, иконки общего бренда, компактные действия файлов и понятные empty/error/loading состояния.
- Срок публикации: presets/ползунок/точный ввод по действующему лимиту установки, клиентская проверка доступа, сохранение названия перед review без потери message; отдельный owner GET /editor возвращает message/maxTtlSeconds без изменения прежнего DTO.
- Settings v1: обычная форма, выбор сроков/MIME policy, expert JSON, явные validate/apply/CAS и откат; введённое сохраняется при ошибке и конфликте.
- Timezone/quiet hours и SMTP TLS/port selectors; privacy-safe terminal recipient view очищает устаревшие download controls, временные ошибки сохраняют resume.
- Встроенная IANA database в API/worker/CLI исправляет отклонение часовых поясов минимальным runtime image. Отказ самой сессии очищает локальные данные во всех разделах; неверный пароль reauth сохраняет вход, поздние ответы не возвращают старую сессию.
- E2E сохранения quiet hours ждёт подтверждённый PUT=204 перед проверочным GET, исключая гонку тестового чтения без изменения runtime или ослабления assertions.
- [UX review, инструкции и ограничения](wiki/product-ux.md), сквозные negative/race/a11y regression tests. Статус текущих проверок — wiki/session.md.

## Unreleased — SAML/OIDC/LDAP

- ADMIN-настройки внешних identity providers, USER-only регистрация при первом входе, явная привязка и повторная авторизация.
- Migration 0019: PostgreSQL CAS/replay guards, encrypted provider secrets и browser-bound SSO flows; отзыв provider sessions при обновлении конфигурации.
- OIDC code/PKCE/nonce/signature, SAML signed SP-initiated responses, LDAP TLS/service-search/user-bind. Локальные роли/disabled и ADMIN fallback сохранены.
- [Инструкции Keycloak, Entra ID/AD, Google Workspace и generic IdP/LDAP](wiki/federated-auth.md); внешняя приёмка tenants отдельно от локальных fixtures.

## Unreleased — White labeling

- ADMIN-редактор общего бренда: название, RU/EN описания, подпись, светлая/тёмная палитры, default theme, три стиля иконок, логотипы/favicon, preview/apply/reset.
- Migration 0018, bounded raster normalization, contrast validation, PostgreSQL revision CAS и атомарный audit; публичный DTO и SSR до JavaScript.
- Бренд в интерфейсах, уведомлениях, SMTP и новых PDF; immutable presentation snapshot сохраняет старые отчёты. Настройки включены в backup/restore.
- Negative/concurrency/fault и browser coverage; [инструкция](wiki/white-label.md).

## Node 26 — 2026-10-13

- Согласованный переход на Node 26.11.1 в pinned Docker builder/runtime, `.nvmrc`/CI и npm engine; `@types/node` 26.6.5 с обновлённым npm lockfile.
- Policy отражает Current на дату review и upstream даты LTS/EOL; guards проверяют календарь, соответствие канала, runtime/types и полный peer graph. Strict install, immutable pins и security gates сохраняются.
- Устранён воспроизводимый конфликт idle operations worker с корректным reopen: пустой tick больше не захватывает runner lock; PG concurrency/fault tests сохраняют pending/crash/generation guards.
- Новый runtime получает отдельные source-bound checks; прежние Node 24 результаты сохранены как история. Native WebDAV/windows-pending и туры из PR #23 объединены в main.

## Native WebDAV — 2026-10-12

- WebDAV LOCK: upper bound Timeout явно проверяется до преобразования в `int`; сохранён предел 3600 s и добавлены проверки границ 32-bit integer без scanner suppression.
- Поддержка штатных macOS/Linux DAV backend, инструкции Windows (ожидает стенда); owner draft GET/Range, atomic overwrite, PG LOCK/UNLOCK/If, persisted properties/timestamps.
- Migration 0017: hidden encrypted stages/lease cleanup и durable locks, guards REST/tus/bulk/publish, bounded metadata. Неизвестная длина PUT без изменения FXP1.
- Обновлены wiki, 48 кадров/20 глав тура с новой озвученной главой WebDAV.
- E2E проверяет JSON до передачи настоящего upstream-ответа браузеру: навигация больше не теряет body для contract validation; добавлен управляемый HTTP regression, матрица расширена до 35 cases.
- Дополнительная загрузка в черновик из WebDAV-клиента: отдельные отзываемые credentials, XML browse, MKCOL/PUT/MOVE/DELETE и UI подключения/обновления/отзыва.
- PUT принимает известный или неизвестный размер, использует bounded encrypted chunks, server checksum/MIME/scanner и auth guards на транзакционных границах. Общие upload/cancel/tree операции строже соблюдают freeze.
- Рекурсивные операции учитывают case-fold варианты имени папки; XML-ответ ограничен 8 MiB, metadata view — 10 000 nodes, все DAV methods делят 8 stream slots.
- Отдельный protocol contract/credential schema, ADR-011, threat/state review, negative/concurrency/fault/fuzz/client/browser проверки. COPY и WebDAV resume не предоставляются. Production guard сохраняется.

Опубликованных releases пока нет. Формат Keep a Changelog.

## Unreleased — этапы 1–6 и техническая реализация фазы 7

### Added
- Локальный ElevenLabs TTS generator с лимитом текста, кэшем, character timings и защитой от повторного платного запроса; renderer принимает проверенное внешнее audio offline. Отдельный импорт MP3 из Chrome проверяет полный набор глав, исходный текст и audio SHA-256; приблизительные субтитры явно отделены от API character alignment. Статус новой озвучки Liza и API-ограничения — в wiki/session.md.
- Воспроизводимая настройка GitHub/main/Actions, CODEOWNERS, Issue/PR templates, SECURITY policy, Dependabot для четырёх ecosystems, CodeQL/dependency review, repository validation и общие editor/Git/Node defaults. Применение и CI подтверждаются в session.
- Скрин-тур по локальному интерфейсу, записанный видеотур с русской озвучкой/субтитрами, автономная галерея и воспроизводимый сценарий на отдельном Compose-стенде. Внешняя приёмка не подменяется демонстрацией.
- Единая wiki: current state, developer workflow, карта компонентов, configuration/testing reference, backlog с зависимостями и полная внешняя acceptance matrix. Исторические результаты отделены от текущих инструкций.
- Staticcheck, ESLint/hooks/JSX-a11y, Prettier и generated OpenAPI UI types; conservative breaking diff для OpenAPI/AsyncAPI/JSON Schemas.
- Pinned Trivy/Gitleaks, source/history/worktree/container scans, CycloneDX SBOM и third-party license inventory с отдельным owner review.
- Пять browser/device profiles в pinned Linux container, 25 E2E scenarios, reviewed visual baselines, keyboard/error/320px/theme checks и bounded fuzz campaigns.
- Multi-GB tus/Local→TLS S3/rotation/Redis-loss concurrency, large-export SIGKILL/retry/VACUUM, backup/restore/reconcile/reopen/ZIP hash performance harness.
- Source-bound release evidence, fail-closed readiness, пять cross-built verifier targets/checksums и unpublished release review bundle.
- Фаза 7.1: audited host upgrade prepare, generation/drain, отдельный controlled image replacement/downtime drill с verified backup, startup faults и explicit reopen.
- Фаза 7.1: trusted proxy/CIDR network policy в PG settings, effective IP для login rate limit, runtime fencing и host reset при ошибочной сетевой конфигурации.
- Генератор HTTPS/ACME Compose с immutable images, non-root/private networks и isolated TLS acceptance harness; production startup guard сохранён.
- Go API/worker/CLI/verifier, Next.js RU/EN owner/recipient UI, themes/mobile layouts, local Compose/Traefik, internal health/readiness/metrics и CI.
- Argon2id/session/CSRF/scoped tokens, account epoch/history, privileged reauthentication, typed settings revisions и HMAC audit.
- tus/HEAD resume, Local/S3 encrypted chunks, server finalization/policy/scanner, draft tree/virtual folders/atomic repath, immutable publish, PIN/links/QR.
- Fenced one-time Range/file/atomic ZIP64 delivery, server checksum manifest/Ed25519/PEM verifier, logical/physical cleanup, freeze/forensic/check/delete.
- FXP1 authenticated header/HKDF, independent binary vectors, durable key usage budgets, reference-preserving class rotation and storage migration.
- SMTP STARTTLS/SMTPS templates, outbox/retries/DLQ, HMAC/SSRF/replay webhook controls, WebSocket/polling/digests/quiet hours/cross-tab notifications.
- Immutable PDF/CSV/JSON reports with three hash layers, public USER verify, trash/history, encrypted ArtifactStorage, Range credentials, bounded export snapshots, private views/bulk jobs.
- Зашифрованный BackupProvider Local/S3, проверка архивов, atomic restore с обязательной предварительной копией, durable maintenance gate, инвалидация сессий и runtime generation.
- Audit/crypto/storage reconciliation, quarantine missing/tampered данных, orphan inventory и явный ADMIN reopen после проверки восстановленных links/API tokens.
- KEK rewrap/recovery, восстановление в новую БД с новым master key, бессрочные encrypted incident reports PDF/JSON, exportable checkpoints обоих журналов.
- Operational UI/CLI, distributed schedules, private alerts/aggregate metrics, read-only upgrade preflight, отдельный ADMIN доступ к чужим exports и управление TTL с reason/audit.
- OpenAPI: 107 операций; traceability: 32 требования; unit/race, 36 PG integration tests, 25 browser E2E, TLS ingress и controlled upgrade drills, axe/visual и response contract validation.

### Fixed
- ESLint 10.12.0 и @eslint/js 10.0.1 работают с JSX-a11y-x 0.2.0 без peer bypass; 31 прежнее accessibility-правило проверено behavioral fixtures. Node types обновлены до 24.19.2. CI проверяет installed peer tree, версии runtime/types/config и сроки поддержки Node 24/Redis 7.2; lint updates объединены в Dependabot group.
- Browser fixtures подтверждают название draft в DOM, PATCH и отдельном GET до backup/publish; recovery сравнивает точные ID сохранённой и поздней передачи. Сбой ввода больше не маскируется как ошибка восстановления.
- Публичная проверка отчёта разрешает ввод/submit после инициализации fragment ID; delayed-hydration регрессия подтверждает один рабочий клик. Recovery E2E готовит собственный CSV, а observer контрактов завершает обработку до закрытия browser context и явно фиксирует capture failures.
- В main через PR #16 объединены восемь Dependabot обновлений: пять Actions по SHA, Alpine 3.24, Go 1.27.2 и React types 19.3.0; пять несовместимых/лицензионно не принятых обновлений учтены отдельно в wiki. Статус merge и проверок — в session.
- Upgrade harness удерживает download connection до подтверждённого PG commit Range/completion, устраняя гонку client Content-Length против server transaction. Строгие 401 и single journal проверки сохранены; отсутствующий commit приводит к ошибке.
- Галерея тура принимает только локальные PNG paths перед src/href; тестовый S3 fixture строит имя файла из канонического UUID. Добавлены негативные browser/wire проверки после сигналов первого CodeQL анализа, без исключения файлов из scan.
- 20 страниц проектной документации перенесены в wiki; ссылки, AGENTS/entrypoints, CLI help, docs checker, matrix generator и release bundle используют канонический путь. Устаревшие статусы фаз/проверок исправлены без принятия внешних blockers.
- Vulnerable npm tooling удалён из web runtime; PostgreSQL gosu заменён su-exec; исправлены zlib/nghttp2/OpenSSL и Go/x/net в инфраструктурных images. Generator требует пять проверенных immutable digests.
- Skip link переводит keyboard focus в main; UI использует точные типы API и привязывает recovery acknowledgement к просмотренному report.
- Startup и read-only preflight используют strict непрерывный migration prefix; unknown/checksum/hole отклоняются до SQL. Pending DDL требует closed gate/exclusive runtime lock и атомарно инвалидирует прежнюю generation/reconciliation; bootstrap не удерживает shared lock при ожидании exclusive.
- Повторяющиеся test IDs OPS-005 устранены; docs gate проверяет уникальность всех traceability test IDs.
- Restore ждёт завершения старых writers, не откатывает KEK usage reservations и отклоняет параллельный import; устаревшая reconciliation не разрешает reopen.
- Критическая crypto ошибка закрывает доступ до попытки записи encrypted incident report; повреждённая историческая audit prefix остаётся проверяемой после recovery checkpoint.
- Кириллица в operational reason проверяется по символам; mobile restore/reconciliation notices помещаются на 390px.
- QR archives now encrypted with recipient mapping and transactional invalidation on link changes.
- Public report verification hides ADMIN/trashed/purged revisions; verifier defaults to signed metadata only.
- Atomic recursive moves, stale upload/maintenance fencing, fair frozen-job scheduling, quiet-hours delivery starvation and admin artifact/job authorization.
- Export cancellation persists even when claim finds no remaining eligible jobs.
- Mobile logo wrapping, PDF Cyrillic/date/state rendering, build loop fail-fast and contract observer cancellation handling.
- Documentation inconsistencies in requirement mapping, restore override, initial scaffold status and implemented contracts.

### Remaining
- Внешнее окружение не подготовлено (решение владельца 2026-10-11): real host/DNS/ACME/SMTP/S3/backups/KEK/checkpoints и target load/DR acceptance заблокированы. Реальные browser/device/assistive-technology, независимый review, лицензии и R-01–06 остаются открытыми. Production bootstrap отклоняется.
- Local fixtures не доказывают совместимость реальных SMTP/S3 accounts. CI реализации фазы 7 подтверждён для 7dce226; статус последующих изменений — в session. Production deployment не выполнялся, residual risks не приняты. См. [историю evidence](wiki/history/development-evidence-2026-10-11.md) и [session](wiki/session.md).

## First-run web setup — 2026-10-14

- FileX `/setup`: временный host token, первый ADMIN и monotonic completion latch.
- Issuer `/issuer/setup`: OIDC/MFA/subject, encrypted immutable configuration, отдельный config key; vendor seed остаётся signer-only.
- Responsive формы, bounded contracts, race/fault/restart проверки; [runbook](wiki/first-run-setup.md). Production guard сохранён.
