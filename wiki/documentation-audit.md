# Аудит документации FileX

> Исторический аудит исходного комплекта. Актуальная реализация и устранённые расхождения: [implementation-1-5](implementation-1-5.md), [design-review](design-review.md), [session](session.md).

Исторический срез до исправлений. Последующий прогресс: [roadmap](roadmap.md) и [session](session.md). Утверждения об отсутствии кода ниже относятся только к моменту исходного аудита.

Дата: 2026-10-10. Область: весь каталог проекта, кроме внутренних данных Git и служебного `.DS_Store`.

## Контекст и метод проверки

FileX описан как приватный B2B/SaaS-сервис одноразовой передачи файлов. В репозитории находится проектная спецификация, а не работающее приложение. Прочитаны полностью все 22 исходных Markdown-документа: пять в корне и семнадцать в `docs/`. Проверены файловая структура, Git status, список отслеживаемых файлов и история последних коммитов. Исходного кода, манифестов зависимостей, миграций, контрактов API, CI и исполняемых тестов не обнаружено.

До аудита каталога `wiki/`, файлов `AGENTS.md` и `agents.md` не было. Локальные инструкции находятся в `AGENT.md`. Для этого аудита применены инструкции пользователя и прочитан `AGENT.md`; переименование или перенос исходных документов не выполнялись. `docs/` содержит предметную спецификацию, эта страница — её анализ, а не замену согласованных решений.

## Карта прочитанных материалов

| Документ | Содержание и степень готовности |
|---|---|
| [README](../README.md) | Назначение продукта, порядок чтения, честный статус specification-only. |
| [AGENT](../AGENT.md) | Стек, запреты, workflow, reviews и Definition of Done. |
| [MEMORY](../MEMORY.md) | Заявленное состояние интервью и последние решения; не доказательство реализации. |
| [CONTRIBUTING](../CONTRIBUTING.md) | Правила commits/PR и требования к будущему Makefile. |
| [CHANGELOG](../CHANGELOG.md) | Только initial documentation, реализованных релизов нет. |
| [Requirements](../docs/requirements.md) | 32 верхнеуровневых требования: 16 FR, 7 SEC, 5 OPS, 4 NFR. |
| [Architecture](../docs/architecture.md) | Компоненты, источники истины, базовые upload/download flows. |
| [Decisions](../docs/decisions.md) | Поздние решения и список вопросов для design review; отдельных ADR нет. |
| [Implementation plan](../docs/implementation-plan.md) | Этапы 0–7 и качественные gates. |
| [State machines](../docs/state-machines.md) | Обзор состояний; прямо требует полных transition tables. |
| [Security](../docs/security.md) | Активы, границы доверия и требуемые контроли; не законченная threat model. |
| [Encryption format](../docs/encryption-format.md) | Требования к будущему формату; framing, параметры и vectors ещё не определены. |
| [Key management](../docs/key-management.md) | Жизненный цикл ключей; подробные recovery-процедуры предстоит написать. |
| [Operations](../docs/operations.md) | Bootstrap, restore gate, scheduler и внутренние health endpoints. |
| [CLI](../docs/cli.md) | Область ответственности двух CLI; синтаксис и exit codes ещё предстоит определить. |
| [Incident response](../docs/incident-response.md) | Перечень инцидентов и обязательных разделов будущих runbooks. |
| [Hardening](../docs/hardening.md) | Базовые требования к TLS, контейнерам, сети и секретам. |
| [Risk register](../docs/risk-register.md) | Шесть рисков, ни один не принят владельцем. |
| [Test matrix](../docs/test-matrix.md) | Начальные группы проверок; нет test IDs, путей и результатов. Есть ошибки соответствия FR. |
| [Performance testing](../docs/performance-testing.md) | Требуемые сценарии и измерения; числовых результатов нет. |
| [Acceptance criteria](../docs/acceptance-criteria.md) | Приёмка широкого набора требований; явного выделения MVP из общей области нет. |
| [Release checklist](../docs/release-checklist.md) | Будущие release gates; не свидетельство пройденных проверок. |

## Архитектура и поток данных

Фиксированный стек: Go, Next.js/TypeScript, PostgreSQL, Redis, Traefik, Docker Compose, Local/S3. Развёртывание рассчитано на один VPS и single origin. Предусмотрены Go binaries `filex-api`, `filex-worker`, `filex-cli`, `filex-verify`; scheduler встроен в worker.

PostgreSQL хранит бизнес-состояние, leases/fencing, jobs, расписания, audit и transactional outbox. Redis используется для временных очередей и кэша: его потеря не должна разрешать повторную выдачу. Хранилища разделены на несколько payload providers, один активный encrypted ArtifactStorage и отдельный encrypted BackupProvider. Backup БД не включает payload.

Основной поток:

1. USER создаёт draft и загружает файлы через tus; сервер завершает загрузку и вычисляет SHA-256.
2. Publish атомарно фиксирует manifest, ссылки и outbox; TTL отсчитывается от `publishedAt`. Файлы, имена и пути становятся immutable.
3. Получатель явно инициирует скачивание: GET/HEAD страницы не расходуют выдачу. API получает PostgreSQL lease с fencing и передаёт файл либо ZIP.
4. После признанного сервером успешного завершения API атомарно выполняет logical delete; worker удаляет физические копии асинхронно. При неуспехе Download all не удаляет ни один файл операции.
5. Отчёты и exports выполняются отдельными механизмами; большие exports требуют согласованного snapshot значений и encrypted intermediate artifact.
6. После restore сессии инвалидируются, выполняется reconciliation, data plane открывается только после явного ADMIN approval.

USER не должен видеть идентификацию скачивавших получателей, IP и UA. ADMIN получает расширенные audit/forensic возможности. Это ограничение касается не только UI, но и API, журналов, событий и exports.

## Проблема: документация задаёт направление, но ещё не закрывает contract freeze

Сильные стороны: явно закреплены authoritative storage, immutable publish, logical/physical deletion, privacy boundary, restore gate и запрет автоматического production deployment. Документы честно отделяют запланированное от реализованного. Сложные риски HTTP completion, token resurrection и MVCC snapshot уже отмечены авторами.

Основной пробел — большинство критических контрактов изложено краткими требованиями, а не однозначными алгоритмами и проверяемыми переходами. Этап 0 плана нельзя считать завершённым по наличию этих файлов.

### A-01. Ошибки трассировки требований — подтверждено

В `test-matrix.md` есть `FR-016–017`, хотя последнее функциональное требование — `FR-016`. Cursor pagination и bulk selection относятся к `FR-015`; locale/a11y — к `FR-001`. Report revision/hash и trash verification относятся к `FR-012`, но находятся в группе `FR-013–015`. Группа `FR-010–012` не перечисляет соответствующие проверки отчётов. `FR-001` вообще отсутствует как ID в матрице.

Рекомендация: отдельная строка на requirement, устойчивые test IDs, статус `planned/not implemented`, затем actual paths и результаты после реализации. Исправление исходной матрицы в этом аудите не выполнялось.

### A-02. Успех одноразового скачивания — обязательный design blocker

`FR-006–007`, architecture и decisions признают Range/resume и пределы HTTP completion, но не определяют точный момент consume/commit. Нужны правила покрытия байтов несколькими Range, повторного Range, параллельных запросов одной сессии, таймаута lease, crash после отправки последнего байта и до DB commit, а также reverse-proxy buffering.

Пример: сервер отправил тело, но упал до logical delete. Документация пока не задаёт точный допустимый результат повторного запроса. PostgreSQL fencing защищает запись состояния, но само по себе не доказывает прекращение передачи старым обработчиком.

Нужен ADR с наблюдаемой гарантией, протоколом завершения и fault/concurrency cases. Нельзя обещать подтверждённое сохранение файла на диске получателя.

### A-03. Freeze, revoke и expiry — неполная приоритетность

Описаны удержание frozen payload, пауза оставшегося TTL при transfer freeze и продолжающийся TTL при file freeze. Не определены все переходы при одновременном freeze/revoke/expiry, воздействие freeze на уже открытый поток и взаимодействие paused TTL с absolute max TTL.

Пример: transfer заморожен перед absolute max TTL и разморожен после него. Нельзя самостоятельно выбрать продление или немедленный expiry. Нужны transition tables с guards, precedence и разделением запрета доступа и удержания объекта.

### A-04. Download all — неоднозначный набор файлов

`FR-007` говорит обо всех оставшихся файлах, state machines — обо всех доступных. Freeze одного файла явно блокирует Download all, но судьба операции при файле, уже leased другой сессией, не определена столь же явно. Нужно зафиксировать правила для leased/integrity_failed/storage_missing, конкуренцию individual/ZIP, snapshot membership и точку атомарного commit.

### A-05. Snapshot больших exports — механизм отсутствует

Согласованный snapshot значений, bounded DB transaction и crash-safe reuse заданы как обязательные свойства, но способ их совместного обеспечения не выбран. Простая пагинация после закрытия транзакции не сохраняет snapshot изменяемых данных.

Нужен ADR: что именно материализуется внутри согласованного чтения, как фиксируется completeness, как проверяется artifact после crash и когда удаляются промежуточные данные. Memory/MVCC/disk bounds должны быть измерены. Это уже признанный авторами риск R-03, а не новое обнаруженное нарушение реализации.

### A-06. Pre-restore backup — неоднозначное исключение

`OPS-003` и decisions требуют обязательный backup перед restore. Operations добавляет «с аварийным override», не задавая его условий и контроля. Нужна единая формулировка: разрешён ли override, кто его запускает, какие условия и audit обязательны. Аудит не принимает это исключение от имени владельца.

### A-07. Криптографические спецификации ещё не готовы

Отсутствуют binary format, nonce construction, AAD encoding, chunk/range mapping, лимиты, truncation rules и deterministic vectors. Для checksum manifest нужны однозначные encoding/canonicalization и правила имён файлов. Хранение public keys определено, а полный lifecycle/key compatibility для restore ещё требует детализации.

`filex-verify` по явному решению проверяет подпись checksum manifest, но не хеширует payload автоматически. Поэтому его успех нельзя представлять как подтверждение целостности всех распакованных файлов; документация и UX будущего CLI должны сохранять эту границу.

### A-08. Прочие контракты, обязательные до соответствующих этапов

- Нет OpenAPI 3.1, AsyncAPI, JSON Schemas, ERD/схемы БД и отдельных ADR.
- Не задана полная матрица USER/ADMIN/recipient/API scopes; организационная tenancy не описана. Само обозначение B2B/SaaS не разрешает придумывать tenant model.
- Не определены форматы ошибок, idempotency contracts, cursor semantics, полный набор WebSocket событий и webhook signing/replay contract.
- Canonical report hash, content hash и artifact hash разграничены словами, но нет формальной canonicalization и примеров verification.
- Tamper-evident audit описан как требование: нужны chain serialization, concurrency model, checkpoint trust и политика восстановления после повреждения. Наличие слова tamper-evident не означает защиту от переписывания всей БД привилегированным атакующим.
- Incident/key/restore runbooks пока являются заданиями на написание процедур. Нет проверенных команд, exit codes или результатов учений.
- Числовые defaults, ограничения размеров/concurrency и retention требуют отдельного реестра решений; SLA/RPO/RTO нельзя выдавать за измеренные.
- Для proprietary проекта пока нет файла лицензии. Добавлять юридический текст автоматически по результатам аудита не следует.

### A-09. Полнота исходного интервью не проверяема

README/MEMORY ссылаются на интервью до вопроса 978 и приоритет поздних решений. Самого интервью, хронологии решений или mapping к ответам в каталоге нет. Можно проверить согласованность текущего комплекта, но нельзя доказать, что все ответы интервью перенесены без потерь.

### A-10. Состояние Git и источников знания

В момент проверки Git отслеживал только `README.md` из initial commit `8fb83dc`; README изменён локально, остальные исходные документы неотслеживаемые. Таким образом, наличие документов на диске не означает, что они сохранены в истории репозитория. Commit/push в рамках аудита не выполнялись.

Инструкции пользователя называют `wiki/` канонической базой, тогда как README направляет в `docs/`, а инструкции лежат в `AGENT.md`. Рекомендуется определить единый entrypoint и явную связь каталогов. Массовый перенос сейчас не нужен: он затруднит сравнение исходного комплекта.

## Рекомендуемое продолжение

1. Нормализовать entrypoint документации, requirement IDs и матрицу трассировки, сохранив согласованные бизнес-инварианты.
2. Завершить этап 0: ADR по download completion/Range, freeze/expiry precedence, ZIP concurrency, export snapshot, restore override; threat model и crypto review.
3. Создать transition tables, data model, API/event schemas и негативные сценарии на каждый инвариант.
4. Зафиксировать границы MVP: текущая приёмка требует весь широкий набор MUST, отдельный урезанный scope не согласован.
5. Только после design gate переходить к platform и вертикальным срезам из implementation plan. Production deploy требует отдельного поручения.

## Изменения этого аудита, проверка и ограничения

Добавлены только `wiki/documentation-audit.md` и `wiki/session.md`; исходные требования и решения сохранены. Конфигурация/env, приложение и инфраструктура не менялись, операционного воздействия нет. Исходные документы рассматриваются как план, а не evidence прохождения gates. Исполняемые тесты отсутствуют; функциональные, crypto, security, performance и live-проверки не проводились. Проверка документационных ссылок и whitespace относится только к качеству документации.
