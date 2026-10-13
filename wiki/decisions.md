# Решения и вопросы, требующие отдельного согласования

## Контекст

Решения уточняют [32 MUST](requirements.md), не исключают их. Исходное интервью отсутствует; даты ADR относятся к текущему циклу разработки. Статус implementation не означает independent review или принятие residual risk.

## Действующие продуктовые правила

- Позднее решение имеет приоритет: после publish payload полностью immutable; раннее разрешение добавлять файлы отменено.
- QR bundle содержит секретные URLs в QR и mapping email; это чувствительный временный encrypted artifact, инвалидируется при изменении links.
- `integrity_failed`: при обычном TTL объект удерживается для диагностики; при revoke/block/delete аккаунта удаляется; при transfer/file freeze сохраняется до решения ADMIN.
- Restore: API tokens и recipient links восстанавливаются по backup (риск resurrection), но Web/recipient sessions инвалидируются, data plane блокируется до reconciliation и явного ADMIN approval.
- Доступ ADMIN к чужим exports/forensic не уведомляет владельца; действия фиксируются в tamper-evident audit.
- CLI имеет host/container trust boundary без дополнительного admin password и без обязательных интерактивных destructive confirmation flags.
- Промежуточный export snapshot должен сохранять **значения**, а не только IDs; гибридная модель должна гарантировать consistent view и crash safety.
- Обязательный backup PostgreSQL перед restore, но не автоматически перед startup migrations; restore-test вручную, без напоминаний.
- Отчёты имеют canonical report hash, document content hash (без собственного поля) и отдельный artifact SHA-256; не смешивать эти понятия.
- Ed25519 только SHA256SUMS; подписанный manifest хранится в ZIP без отдельного meta.json; ключ в PEM/SPKI, fingerprint SHA-256(DER/SPKI), сверка через независимый канал.
- Публичная `/verify-report` принимает reportId/revision/canonical hash, не подтверждает admin reports, при trash/purge — neutral not found.
- Нет IPv6 requirement, нет PWA, нет сторонней аналитики, нет внешнего proxy, нет auto production deploy.

## Реализованные design decisions

| Решение | Реализация / граница |
|---|---|
| [ADR-001](adr/001-download-completion.md) | Server-observed Range coverage/one-time commit, deterministic atomic ZIP; fsync получателя не доказуем |
| [ADR-002](adr/002-retention-precedence.md) | Lifecycle отдельно от hold; hard deadline/revoke не отменяются freeze |
| [ADR-003](adr/003-export-snapshot.md) | Bounded materialized PG values → encrypted final artifact, retry сохраняет committed snapshot |
| [ADR-004](adr/004-restore-gate.md) | Mandatory verified pre-restore backup без аварийного override |
| [ADR-005](adr/005-cryptography.md) | FXP1/HKDF/AAD, per-attempt chunk identity, durable key budgets, independent vectors |
| [ADR-006](adr/006-operations-recovery.md) | Неоткатываемый ops control plane, generation/restart, matching schema, KEK recovery и fresh-instance key |
| [ADR-007](adr/007-ingress-and-network-policy.md) | Single trusted Traefik hop, IPv4 /32, CIDR/HTTPS/Host, closed gate при network change |
| [ADR-008](adr/008-controlled-upgrade.md) | Strict migration prefix, drain/stop/verified backup, exclusive DDL, explicit reopen |
| [ADR-009](adr/009-release-evidence.md) | Source-bound gates и пять scanned images; human decisions отдельно от scanner inventory |

Design вопросы completion, export snapshot, partial delivery и crypto isolation получили реализацию и [внутренний review](design-review.md). Capacity/independent trust и окончательная приёмка остаются открытыми. Это не повод заново спрашивать уже принятое решение о baseline поведении.

## Открытые решения и необходимые входы

| Вопрос | Кто/когда решает | До решения |
|---|---|---|
| Как квалифицировать target runtime и затем разрешать production при безусловном guard | DEV-01: инженер готовит ADR, security reviewer/владелец рассматривают | Guard не снимается, development не используется как production обход |
| Целевые host/domain/SMTP/S3/backups/escrow, access/retention | Оператор, карточка [окружения](environment-acceptance.md) | Environment not_prepared по ответу владельца |
| Capacity/RPO/RTO и допустимые operational bounds | Владелец/оператор до target measurement | Локальные цифры не SLA и не критерий pass произвольной инфраструктуры |
| Third-party obligations и proprietary LICENSE/правообладатель | DEV-03 + EXT-07, владелец/reviewer | Inventory не разрешает поставку, текст LICENSE не выдумывается |
| R-01–06 и независимый security/device/AT review | Назначенные reviewer/владелец | Все 14 decisions pending, 5 MUST partial |
| Хранение release evidence после удаления CI artifacts | DEV-04 + оператор | Локальная .cache и 14 дней CI не являются бессрочным архивом |

Практические карточки — [план](implementation-plan.md). Новые варианты механизма/провайдера можно исследовать локально, но это не назначение чужого аккаунта, утверждение рисков или разрешение внешнего воздействия. Production deployment всегда требует отдельной команды владельца.

## Дополнение владельца: native WebDAV
Работа с файлами одного owner draft из штатных Windows/macOS/Linux клиентов разрешена прямым запросом 2026-10-12. [ADR-011](adr/011-native-webdav.md) заменяет upload-only ограничения ADR-010; published immutable/recipient one-time, USER/ADMIN, crypto и production boundaries остаются прежними. Владелец подтвердил отсутствие Windows-стенда: сценарий подготовлен, фактический статус — ожидает стенда. [Текущее руководство](webdav-upload.md), [сессия](session.md).

## Дополнение владельца: Node 26
2026-10-13 владелец прямо разрешил довести Node 24→26 PR #22 до merge. [ADR-012](adr/012-node26-runtime.md) заменяет прежнее откладывание до LTS: runtime на дату review Current, все manifest/types/CI/images меняются согласованно и получают свежие gates. Это не принятие внешних release risks и не разрешение production deployment.

## Дополнение владельца: white labeling
2026-10-13: настраиваемый бренд всей установки без заранее заданных assets; [ADR-013](adr/013-white-label.md). Это не организационная tenancy и не переименование технических протоколов.

## Дополнение владельца: SAML/OIDC/LDAP
2026-10-13 разрешена реализация корпоративного входа/регистрации, подтверждены Keycloak, Entra ID/AD, Google Workspace и другие IdP/LDAP. [ADR-014](adr/014-federated-auth.md): провайдеры настраиваются ADMIN, JIT только USER, no implicit email linking; local password fallback сохраняется. Конкретные tenant credentials не предоставлены.
