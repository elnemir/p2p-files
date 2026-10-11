# Дорожная карта разработки

## Контекст и статус

Исходная нумерация 0–7 сохранена. Все 32 MUST входят в MVP; отдельный урезанный scope не согласован. Фаза закрывается по работающему вертикальному сценарию, проверкам и синхронной документации. Development completion, выпуск и production deployment — разные события.

## Реализованные фазы

| Фаза | Результат | Статус на 2026-10-11 |
|---|---|---|
| 0. Контрактная база | Requirements, ADR, threat model, state machines, OpenAPI/AsyncAPI/schemas, traceability | Реализовано; документы актуализируются с каждым изменением |
| 1. Платформа | Go/Next.js/Compose, migrations, auth/roles/CSRF, settings, audit, CLI и CI | Закрыта по локальным development gates |
| 2. Передачи | tus/resume/tree, Local/S3, server finalize, publish/links/PIN, leases/Range/cleanup | Закрыта по локальным development gates |
| 3. ZIP/crypto | Atomic ZIP64/Ed25519, verifier, FXP1, rotation/migration, freeze/forensic | Закрыта по локальным development gates |
| 4. Коммуникации | SMTP TLS/templates, outbox/retry/DLQ, HMAC webhooks, WS/poll/digest | Закрыта по локальным development gates |
| 5. Отчёты | Immutable reports/three hashes, verify/trash, exports/artifacts/QR, saved views/bulk | Закрыта по локальным development gates |
| 6. Эксплуатация | Backup/restore/pre-backup, reconciliation/reopen, KEK recovery, incidents/checkpoints, UI/CLI | Закрыта по локальным development gates |
| 7. Приёмка | Ingress/upgrade, quality/security/browser/fuzz/load/DR gates и release evidence | Технический набор реализован и проверен на 7dce226; полная release приёмка открыта |

Исторические результаты — [evidence](history/development-evidence-2026-10-11.md); текущая граница — [current state](current-state.md). 27 MUST verified локально и 5 partial не означают 32 release-accepted MUST.

## Порядок продолжения фазы 7

Это рабочая декомпозиция оставшегося допуска и сопровождения, а не изменение продуктового scope.

| Шаг | Вход | Выход / условие завершения |
|---|---|---|
| 7.4. Подготовка кандидата | Текущий код и данная wiki | DEV-01 design, DEV-02 сопровождение, DEV-03 пакет license review, DEV-04 evidence retention; локальные проверки зелёные |
| 7.5. Подготовка окружения | Назначенные оператором VPS/домен/аккаунты/escrow и явная команда на внешние действия | Заполненная карточка окружения, EXT-01–03 с фактическими протоколами |
| 7.6. Независимая и целевая приёмка | Согласованный кандидат, изолированное целевое окружение, reviewer | EXT-04–07: load/DR, реальные устройства/AT, security, лицензии и R-01–06; найденные дефекты исправлены и проверены |
| 7.7. Допуск и выпуск | Все внешние evidence, reviewed DEV-01 implementation | REL-01: свежие 13 gates, полная traceability, clean commit, release-check PASS, сохранённый release package |
| Production deployment | Принятый выпуск и отдельное поручение владельца | Контролируемый rollout/проверка/rollback plan по фактическому окружению; эта задача сейчас не поручена |

DEV-01 сначала определяет безопасный путь тестового допуска и финального production допуска: существующий безусловный guard иначе создаёт цикл «нельзя проверить production до release, release нельзя принять до проверки». Проектное решение и его review должны предшествовать любому включению режима. До них generator/TLS fixture не считаются работающим production deployment.

## Что можно начать без внешнего окружения

Приоритет: проектирование DEV-01 и пакет DEV-03; затем DEV-02 и DEV-04. Независимый reviewer может начать разбор кода/форматов без VPS, но его финальные выводы должны учитывать выбранную поставку. Конкретные карточки, зависимости, тесты и ответственные роли — [implementation plan](implementation-plan.md).

При записи [тура](screen-tour.md) выявлен DEV-05: уже открытая страница получателя сохраняет устаревший список после completed/401. Сервер повторную выдачу запрещает; отдельный локальный шаг должен сделать терминальное состояние UI явным, сохранив privacy и resume. Визуальные материалы показывают действующее поведение, а не будущий исправленный интерфейс.

Новые бизнес-функции после MVP не согласованы. Multi-tenant organizations, billing, PWA, IPv6 или замена стека не добавлены в план как обязательные задачи. Обнаруженный дефект фиксируется отдельной карточкой с requirement ID и фактическим воспроизведением.
