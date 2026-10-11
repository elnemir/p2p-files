# CLI FileX

## Контекст и безопасный ввод
`filex-cli` работает в доверенной границе host/container и использует bootstrap env API. Дополнительный ADMIN password и интерактивные destructive confirmation flags не требуются. JSON stdin с секретами нельзя журналировать; KEK/password/S3 secrets не передаются в argv. Все operational commands возвращают JSON; `--json` включает его также у legacy provisioning/config commands. Shell completion выводит shell source. Exit 0 — success, 2 — invalid input/config/failure.

В контейнере binary: `/usr/local/bin/filex-cli`. Для JSON input использовать файл mode 0600 или секретный stdin pipe; ниже приведены только схемы, не реальные secrets. После использования удалить временный secret input согласно host policy.

## Команды и поток

| Команда | JSON stdin / аргументы → результат |
|---|---|
| `--version` | dev binary version |
| `config validate` | env → validation без соединения с DB |
| `config ingress-check` | read-only active settings → proxy/HTTPS readiness; не применяет migrations и не одобряет production release |
| `settings show` | read-only active `{revision,active,validated,value}` |
| `settings apply` | `{expectedRevision,value}` → draft/validate/CAS apply полного settings v1; network/origin change закрывает gate и меняет generation |
| `network reset` | `{reason}` → новая revision без client allowlist; proxy/HTTPS/origin сохранены, требуется restart/reconcile/reopen |
| `upgrade check` | env/DB read-only → `{applied,pending,compatible}`; unknown/checksum mismatch/пропуск внутри migration prefix отклоняются, SQL не применяется |
| `upgrade prepare` | `{reason}` → close gate/new generation/drain/audit → `{generation,mode,drained,requiresRestart}`; matching schema без pending SQL, без migrations; timeout сохраняет closed gate |
| `migrate` | locked/checksummed migrations → KEK sentinel → settings |
| `user create` | `{email,password,role}` → user UUID; role=user/admin |
| `storage local /absolute/root` | probe/register payload Local → provider UUID |
| `storage backup-local /absolute/root` | probe/register BackupProvider Local → provider UUID |
| `storage s3` | `{purpose,priority,s3:{endpoint,region,bucket,accessKey,secretKey,sessionToken?}}` → encrypted/probed provider UUID |
| `backup create` | `{providerId,reason}` → verified backup job ID |
| `backup verify` | `{backupId,recoveryKek?}` → `{id,bytes,sha256,verified}`; полное чтение/проверка, без import |
| `backup import` | `{backupId,providerId,reason,recoveryKek?}` → восстановить catalog entry из существующего verified descriptor |
| `backup restore` | `{backupId,providerId,reason,recoveryKek?}` → drain/pre-backup/atomic restore, requires restart и reconciliation |
| `recovery close` | `{reason}` → durable maintenance gate |
| `recovery reconcile` | `{reason}` → job ID; report ID/hash в result этого job |
| `recovery report` | `{reconciliationId}` → immutable JSON findings |
| `recovery revoke-capabilities` | `{reason}` → revoke all recipient links/API tokens при закрытом gate, invalidates reconciliation |
| `recovery reopen` | `{reconciliationId,hash,reason,acknowledgeCapabilities:true}` → guarded explicit reopen |
| `audit verify` | без stdin → verify app/operations audit; corruption creates critical incident/gate |
| `audit evidence` | без stdin → evidenceHash обоих текущих журналов |
| `audit checkpoint` | `{kind,reason,evidenceHash?}`; kind=regular/recovery; recovery требует закрытый gate и matching evidenceHash |
| `kek rewrap` | `{newKek,providerId,reason}` → verified pre-backup/atomic rewrap; требует свежий независимо сгенерированный 32-byte base64 KEK |
| `operations status` | без stdin → state/jobs (последние 100) |
| `operations run` | без stdin → выполнить одно pending job; lock предотвращает параллельное исполнение |
| `operations schedules` | без stdin → интервалы/nextAt/provider |
| `operations schedule` | `{kind,intervalSeconds,providerId?}`; kind=backup/audit/storage, 0 либо 60…2592000 |
| `incidents list` | без stdin → последние 100 private incidents |
| `incidents update` | `{id,state,reason}`; state=acknowledged/resolved, resolved требует устранить audit/crypto failure |
| `incidents report` | `{id,revision?}` → immutable JSON и SHA-256; revision=0/latest |
| `completion bash` / `completion zsh` | shell completion source |

`providerId` у restore обозначает **назначение предварительной копии**, source provider сохраняется в encrypted backup catalog. При восстановлении исторического backup `recoveryKek` относится к нему; текущий MASTER_KEK остаётся целевым. Исторические KEK необходимы для старых descriptors после rewrap.

Пример JSON reopen после проверки отчёта (UUID/hash заменить фактически полученными):

```json
{"reconciliationId":"00000000-0000-4000-8000-000000000001","hash":"0000000000000000000000000000000000000000000000000000000000000000","reason":"Проверены restored capabilities и storage findings","acknowledgeCapabilities":true}
```

Этот пример не пройдёт guard без действительного matching report. Причина — 5…2000 символов; один JSON object, unknown fields/trailing JSON отклоняются. Operations job failures возвращаются без SQL/secret details; смотреть protected operations state/errorCode. Host CLI не обходит pre-backup, schema, key-budget или reopen guards.

## Recovery на чистой БД и ограничения
Matching schema/binary + новый MASTER_KEK → provision backup root/bucket → import с historical recoveryKek → restore → restart → reconcile → read report → reopen. Старый KEK на новой instance ID не допускается из-за неизвестного usage budget. Смена env без rewrap существующей БД блокируется sentinel. [Runbook](../wiki/operations-recovery.md), [ключи](key-management.md).

Network reset реализован для application CIDR allowlist: [ingress runbook](../wiki/production-ingress.md). Он не меняет host firewall, proxy trust или TLS. Для восстановления неправильного trusted proxy/origin используется host settings apply. Upgrade check проверяет только schema compatibility binary. Upgrade prepare использует operations runner/runtime locks, требует отсутствия queued/running operations и не удерживает постоянный deployment lock; затем необходим полный stop старых instances, verified backup и [controlled upgrade runbook](../wiki/controlled-upgrade.md). Production deployment и автоматическое скачивание/обновление images не выполняются.

## Offline verifier

```sh
filex-verify --public-key trusted.pem transfer.zip
filex-verify --public-key trusted.pem --fingerprint <sha256-DER-SPKI> --verify-files transfer.zip
```

По умолчанию: ZIP structure + sorted SHA256SUMS + Ed25519 signature + соответствие embedded key независимо полученному trusted PEM. Payload bytes не читаются. `--verify-files` дополнительно сверяет каждый файл по SHA-256. Fingerprint lowercase hex; несовпадение всегда ошибка. Embedded public key сам по себе не является trust anchor.

Success JSON: `{"status":"verified","manifestFiles":2,"filesChecked":false}`. Exit 0 — success, 1 — verification failure, 2 — usage/key file error. Default success утверждает подлинность manifest, не целостность payload на диске.
