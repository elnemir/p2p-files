# CLI: реализованные команды и следующий этап

## Контекст
`filex-cli` работает внутри доверенной границы host/container, без дополнительного ADMIN password. Bootstrap env совпадает с API. CLI не предназначен для публичного HTTP доступа; stdin с секретами нельзя журналировать.

| Команда | Вход → результат |
|---|---|
| `filex-cli --version` | dev binary version |
| `filex-cli config validate` | env → validation, без соединения с DB |
| `filex-cli migrate` | locked/checksummed migrations → KEK sentinel → typed settings check |
| `filex-cli user create` | JSON stdin `{email,password,role}` → user UUID; role=user/admin |
| `filex-cli storage local /absolute/root` | register/probe payload Local provider → provider UUID |

CLI exit 0 — success, 2 — invalid command/config/failure. `user create` не принимает пароль в arguments. В контейнере binary находится в `/usr/local/bin/`. Рабочие `filex-api` и `filex-worker` запускаются без subcommands; `--version` доступен всем binaries.

## Offline verifier

```sh
filex-verify --public-key trusted.pem transfer.zip
filex-verify --public-key trusted.pem --fingerprint <sha256-DER-SPKI> --verify-files transfer.zip
```

По умолчанию: ZIP structure + sorted SHA256SUMS + Ed25519 signature + соответствие embedded key независимо полученному trusted PEM. Payload bytes не читаются. `--verify-files` дополнительно сверяет каждый файл по SHA-256. Fingerprint lowercase hex; несовпадение всегда ошибка. Embedded public key сам по себе не является trust anchor.

Success JSON: `{"status":"verified","manifestFiles":2,"filesChecked":false}`. Exit 0 — success, 1 — verification failure, 2 — usage/key file error. Default success утверждает подлинность manifest, не целостность payload на диске.

## Фаза 6
Не реализованы backup create/restore, storage reconciliation CLI, audit checkpoints, KEK re-wrap, upgrade check, network-policy reset, общий `--json` и shell completion. HTTP ADMIN storage/rotation/forensic/audit verify работают отдельно; наличие endpoint не означает наличие одноимённой CLI команды. [Key management](key-management.md), [restore gate](../wiki/adr/004-restore-gate.md).
