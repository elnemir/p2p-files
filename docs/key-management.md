# Управление ключами

## Реализованный поток
Bootstrap MASTER_KEK = base64 32 random bytes вне PG. Persisted sentinel выявляет неверный KEK. Class DEKs payload/artifacts/configuration/signing/backups хранятся wrapped AES-GCM в key_versions. Active version одна на класс, новая rotation применяется к новым записям сразу. Старые versions не удаляются автоматически.

ADMIN `POST /api/v1/admin/keys/rotate {class}` с re-auth запускает background rotation для payload/artifacts/configuration; PATCH job paused/running/cancelled регулирует оставшиеся объекты. Copy/decrypt/verify/re-encrypt выполняется под fencing; pointer swap атомарен, исходная копия удаляется после grace period. Frozen payload и live artifact credentials задерживают migration; fairness сохраняет прогресс остальных jobs. Signing rotation фиксирует новый public key сразу и хранит historical public keys бессрочно.

## Crypto bounds и recovery
[ADR-005](../wiki/adr/005-cryptography.md) и [binary format](encryption-format.md) определяют HKDF/AAD/salts/nonce и лимиты. Root-key seal_count и KEK wrap budget резервируются до encryption, предел 2^20; исчерпание блокирует записи, но не чтение. Миграция 0013 исчерпывает budget старых DEK с неизвестным числом attempts; нужно создать новые active versions. Отмена re-encryption не возвращает старый active key и не удаляет уже сменённые references.

Public fingerprint — SHA-256 DER/SPKI lowercase hex, сверять независимо от ZIP. Signing seed — отдельный class key, forensic никогда не экспортирует key material. KEK нельзя сохранять вместе с DB backup или выводить в logs.

## KEK recovery и BackupProvider
`filex-cli kek rewrap` с новым 32-byte base64 KEK на stdin выполняет verified pre-backup, exclusive maintenance и атомарную замену wraps. Ключи данных/audit остаются прежними; env всех API/worker нужно обновить и перезапустить, затем выполнить reconciliation/reopen. Исторические backups сохраняют старый KEK: для verify/import/restore используется recoveryKek stdin. Старые keys нельзя удалять, пока нужны соответствующие backup descriptors.

Restore резервирует KEK seals до import и сохраняет maximum known usage; неудачная попытка не возвращает бюджет. Восстановленные non-signing DEKs становятся exhausted, создаются четыре свежих active class keys. Чистая БД требует новый текущий KEK и отдельно исторический recoveryKek; reuse старого master с неизвестным high-water mark отвергается. Audit key был ранее HKDF-derived: первый startup сохраняет этот же ключ encrypted, сохраняя существующий audit. Operations journal использует отдельный persisted encrypted key.

Periodic audit job проверяет также wrapped keys, encrypted references и соответствие signing seeds/public keys. Критическая crypto ошибка закрывает gate даже при невозможности записать encrypted report. [Runbook](../wiki/operations-recovery.md), [ADR-006](../wiki/adr/006-operations-recovery.md). Ротация не заменяет incident response при host compromise; production/escrow/DR приёмка остаётся фазой 7.
