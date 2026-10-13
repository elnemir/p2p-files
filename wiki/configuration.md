# Конфигурация и инженерные ограничения

## Контекст и поток

Bootstrap env нужен для подключения к PG, KEK и listeners. Прикладные настройки живут в versioned PostgreSQL, provider secrets зашифрованы. Поток: env → config.Load → migrations → KEK sentinel → active settings → guards/worker. Источники: [config](../internal/config/config.go), [bootstrap](../internal/bootstrap/bootstrap.go), [settings store](../internal/settings/store.go), [dev Compose](../deploy/compose.yaml). Числа ниже — проверяемые bounds/defaults, не SLA.

## Bootstrap и инструменты

| Имя | Назначение / default | Изменение |
|---|---|---|
| APP_ENV | Только development или test; пустое/production отвергаются | Production допуск требует DEV-01, не переключения env |
| DATABASE_URL | Обязательный postgres/postgresql URL с user/host/database | Секретный bootstrap; не логировать |
| MASTER_KEK | Обязательные base64 32 random bytes | Existing DB: только rewrap runbook; случайная замена блокируется sentinel |
| HTTP_ADDR | 127.0.0.1:8080; dev Compose 0.0.0.0:8080 внутри container | Не публиковать API в обход Traefik |
| INTERNAL_ADDR | 127.0.0.1:9090; dev Compose 0.0.0.0:9090 внутри container | Отдельный listener, public proxy его не маршрутизирует |
| DEV_PAYLOAD_ROOT / DEV_ARTIFACT_ROOT / DEV_BACKUP_ROOT | Optional auto-provision только APP_ENV=development; Compose /data/payload, /data/artifacts, /data/backups | Разные purposes/volumes; в production generator DEV_* отсутствуют |
| POSTGRES_PASSWORD | Вход Compose для PG и DATABASE_URL | init-dev.py создаёт случайное значение, это не typed application setting |
| FILEX_TEST_DATABASE_URL | Только test runner | Использовать disposable database, не пользовательскую/production |
| FILEX_E2E_PROJECTS / FILEX_UPDATE_VISUALS | Только browser harness; фильтр / явное обновление baselines | Фильтр не полный gate; baseline update запрещён acceptance wrapper |
| FILEX_PERF_MIB / FILEX_PERF_ROWS / FILEX_FUZZ_SECONDS | Только test profiles | [Пределы и запись evidence](testing.md) |

Redis URL не является bootstrap setting приложения: текущие очереди/leases работают через PG. `.env` предназначен Compose; `go run` сам его не читает. `make migrate` требует уже переданные APP_ENV/DATABASE_URL/MASTER_KEK и запускает startup migrations; CLI внутри dev container получает их из Compose.

## Settings v1

Пример полного начального значения без secrets:

```json
{
  "version": 1,
  "publicOrigin": "http://localhost:8090",
  "sessionTtlSeconds": 43200,
  "transferMaxTtlSeconds": 604800,
  "maxFiles": 10000,
  "maxPathBytes": 1024,
  "fileTypeMode": "disabled",
  "fileTypes": [],
  "bannerRu": "",
  "bannerEn": "",
  "networkPolicy": {}
}
```

| Поле | Валидация / поведение |
|---|---|
| version | Ровно 1; unknown fields/trailing JSON отвергаются |
| publicOrigin | HTTPS origin без user/path/query/fragment; HTTP только localhost/loopback; пустое → localhost:8090 |
| sessionTtlSeconds | 300…604800; 0 → 43200 |
| transferMaxTtlSeconds | 1…31536000; initial settings 604800 |
| maxFiles / maxPathBytes | 1…100000 / 1…4096; initial settings 10000 / 1024 |
| bannerRu / bannerEn | Каждая строка ≤2000 bytes |
| fileTypeMode / fileTypes | disabled/allow/deny или пустой disabled; allow требует непустой список; ≤100 MIME entries |
| networkPolicy | ≤16 canonical IPv4 /32 trusted peers; ≤128 client CIDRs; requireHttps требует requireProxy и HTTPS origin |

Читать active revision → сохранить полный value → draft/validate → CAS apply с expectedRevision. Rollback применяет прежнюю validated revision, не переписывает историю. Смена origin/network policy закрывает gate, меняет generation и требует restart/reconcile/reopen. Пустой client allowlist разрешает все IP, сохраняя proxy/HTTPS guards. Полный [ingress runbook](production-ingress.md).

## Отдельные настройки и секреты

SMTP/webhook configuration и storage providers не входят в объект Settings v1: для них предусмотрены собственные ADMIN routes/CLI, encrypted secrets и audit. Payload providers могут быть множественными; ArtifactStorage — один активный; BackupProvider отдельный. Endpoint SMTP/S3 задаётся доверенным ADMIN, webhook URL проходит отдельный SSRF policy. Нет plaintext SMTP/HTTP S3 как production варианта.

Schedules также отдельные: backup default off, audit+crypto 3600 s, storage 86400 s; 0 отключает optional schedule, иначе 60…2592000 s. Mandatory cleanup не отключается этим параметром. [Operations](operations.md), [CLI](cli.md).

## Фиксированные пределы реализации

| Объект | Bound / следствие |
|---|---|
| Payload | ≤1 TiB, chunk ≤4 MiB, ≤2^20 append attempts; memory для bytes по chunk |
| Artifact / report snapshot | Artifact ≤64 MiB/16 chunks; snapshot отчёта ≤16 MiB |
| Export | Snapshot ≤50000 rows/16 MiB в bounded transaction; global 2 / owner 1; staging cleanup 24 h |
| Ключи | Root DEK/KEK seals ≤2^20 с durable reservations; исчерпание блокирует новую запись |
| Потоки | До 8 concurrent upload/finalize/content/forensic; 32 KiB guarded write, deadline 2 s; Argon2 semaphore 4 |
| PG pools | Основной 10 + runtime-lock до 24 connections на процесс; statement/idle transaction 10 s, lock 3 s; cleanup имеет scoped override |
| Backup/recovery | Archive ≤1 GiB, record ≤64 MiB, inventory ≤1M, findings ≤10000; worker operation ≤5 min, CLI ≤15 min |
| Re-auth/reopen | ADMIN fresh re-auth 5 min; reconciliation ≤15 min и matching generation/hash |

Изменение лимита требует исследования памяти/locks/faults и соответствующих tests, не только замены числа. Подробные форматы — [FXP1](encryption-format.md), [snapshot](adr/003-export-snapshot.md), [recovery](operations-recovery.md). Production CPU/RAM/storage targets определяются по [целевой приёмке](environment-acceptance.md).

## WebDAV native profile
Новых env/config keys или открытых портов нет. Credential выдаётся из draft после reauth на 60 s–30 days. Протокольные бюджеты фиксированы: 4 MiB chunk, 1 TiB file, 5 min stage lease, lock ≤3600 s и ≤256 active/draft, If 8 KiB, owner XML 4 KiB, properties 32 KiB/256 на ресурс и 4 MiB/draft, XML 64 KiB/depth 16/response 8 MiB, namespace 10 000, общие 8 stream slots. [Подключение/ошибки](webdav-upload.md). Клиент Windows может накладывать меньший лимит файла; изменение реестра не выполняется автоматически. Production guard остаётся закрытым.
