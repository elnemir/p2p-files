# Управление ключами

## Реализованный поток
Bootstrap MASTER_KEK = base64 32 random bytes вне PG. Persisted sentinel выявляет неверный KEK. Class DEKs payload/artifacts/configuration/signing/backups хранятся wrapped AES-GCM в key_versions. Active version одна на класс, новая rotation применяется к новым записям сразу. Старые versions не удаляются автоматически.

ADMIN `POST /api/v1/admin/keys/rotate {class}` с re-auth запускает background rotation для payload/artifacts/configuration; PATCH job paused/running/cancelled регулирует оставшиеся объекты. Copy/decrypt/verify/re-encrypt выполняется под fencing; pointer swap атомарен, исходная копия удаляется после grace period. Frozen payload и live artifact credentials задерживают migration; fairness сохраняет прогресс остальных jobs. Signing rotation фиксирует новый public key сразу и хранит historical public keys бессрочно.

## Crypto bounds и recovery
[ADR-005](../wiki/adr/005-cryptography.md) и [binary format](encryption-format.md) определяют HKDF/AAD/salts/nonce и лимиты. Root-key seal_count и KEK wrap budget резервируются до encryption, предел 2^20; исчерпание блокирует записи, но не чтение. Миграция 0013 исчерпывает budget старых DEK с неизвестным числом attempts; нужно создать новые active versions. Отмена re-encryption не возвращает старый active key и не удаляет уже сменённые references.

Public fingerprint — SHA-256 DER/SPKI lowercase hex, сверять независимо от ZIP. Signing seed — отдельный class key, forensic никогда не экспортирует key material. KEK нельзя сохранять вместе с DB backup или выводить в logs.

## Следующий этап
KEK re-wrap/recovery CLI, encrypted BackupProvider, reconciliation после restore и periodic crypto-health — фаза 6. Namespace backups зарезервирован, рабочие backup jobs не реализованы. Нельзя подменять KEK env в существующей БД: unwrap/sentinel fail closed. Ротация не заменяет incident response при host compromise.
