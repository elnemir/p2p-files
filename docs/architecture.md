# Архитектура

```mermaid
flowchart LR
U[USER / Recipient / ADMIN] --> T[Traefik HTTPS ACME]
T --> W[Next.js Web]
T --> A[Go API]
W --> A
A --> P[(PostgreSQL authoritative)]
A --> R[(Redis ephemeral)]
A --> S[Payload Local/S3]
K[Go worker + scheduler] --> P
K --> R
K --> S
K --> F[Encrypted ArtifactStorage Local/S3]
K --> B[Encrypted BackupProvider Local/S3]
```

- `filex-api`, `filex-worker`, `filex-cli`, `filex-verify` — отдельные Go binaries. Scheduler встроен в worker.
- API stateless относительно бизнес-состояния; PostgreSQL authoritative для transfer/file states, download leases/fencing, schedules, outbox, job metadata, audit. Redis — очереди/кэш, его потеря безопасна.
- Payload StorageProvider поддерживает несколько Local/S3, priority/fallback; ArtifactStorage — один активный Local/S3; BackupProvider — отдельный Local/S3. Payload не включён в DB backup.
- Environment: DB connection, master KEK, APP_ENV, Traefik/ACME deployment bootstrap; остальные application settings — typed/versioned PostgreSQL с draft, validate/test, atomic apply и rollback.
- API: REST `/api/v1`, tus upload, HTTP Range download, WebSocket real-time, outbound HMAC webhooks. Одна OpenAPI 3.1 + AsyncAPI.

```mermaid
sequenceDiagram
participant U as USER
participant API as Go API
participant DB as PostgreSQL
participant S as Storage
U->>API: create draft / tus upload
API->>S: stream chunks
API->>S: finalize + SHA-256
API->>DB: file ready
U->>API: review + publish
API->>DB: atomic published manifest, links, outbox
DB-->>U: active + manifestHash
```

```mermaid
sequenceDiagram
participant R as Recipient
participant API as Go API
participant DB as PostgreSQL
participant S as Storage
participant W as Worker
R->>API: explicit download action
API->>DB: acquire lease + fencing
API->>S: stream file / ZIP
API->>DB: verify lease + atomic logical delete on full success
DB->>W: outbox delete job
W->>S: physical delete
W->>DB: deleted
```

**Критическое ограничение:** успешная передача по HTTP не доказывает сохранение файла получателем. Системный статус означает завершённую выдачу со стороны сервера в пределах наблюдаемого протокола.


## Реализация и источники деталей
[Модель данных](../wiki/data-model.md), [core contracts](../wiki/core-design.md), [фазы](../wiki/roadmap.md). Этапы 1–5 подключают repository к защищённым HTTP routes, UI и worker. Диаграмма выше содержит целевую production архитектуру: ACME/TLS и BackupProvider пока остаются фазами 6–7; локальный Compose использует loopback HTTP. Redis развёрнут, но correctness/очереди реализованы в PG. Internal health endpoints находятся на отдельном listener. Реальное состояние и ограничения: [development](../wiki/development.md).
