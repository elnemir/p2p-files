# Encryption/storage format v1

## Контекст
Данные хранятся только в encrypted chunks Local/S3. PostgreSQL хранит описание объектов, salt/key ID, plaintext ranges и контрольные суммы. Полные plaintext файлы не создаются в `/tmp`. [ADR-005](adr/005-cryptography.md) объясняет trust boundaries и retry safety.

## Small envelope
`01 || nonce[12] || ciphertext[N] || tag[16]`, AES-256-GCM через Go `NewGCMWithRandomNonce`. AAD — проверенный context; wrong context/key, truncation и extra bytes отклоняются. Это формат KEK wrapping, configuration, report snapshots и chunks ArtifactStorage. Для root keys действует durable reservation 2^20 сообщений; для artifact subkey максимум 16.

## Payload FXP1
Один immutable storage object содержит header ниже, затем ciphertext ровно length bytes и GCM tag[16]. Все integer — unsigned big-endian. Header целиком является AAD.

| Offset | Размер | Содержимое |
|---|---|---|
| 0 | 4 | ASCII FXP1 |
| 4 | 1 | class=1 (payload) |
| 5 | 36 | chunk UUID, lowercase ASCII |
| 41 | 36 | file UUID |
| 77 | 36 | class key version UUID |
| 113 | 32 | file salt |
| 145 | 8 | полный plaintext size файла |
| 153 | 4 | maximum chunk size = 4194304 |
| 157 | 8 | plaintext offset |
| 165 | 4 | plaintext length этого chunk |
| 169 | 8 | index = persisted upload fencing epoch |
| 177 | length + 16 | ciphertext + authentication tag |

File key = HKDF-SHA256(DEK, salt, `filex/payload/object/v1/<file UUID>`, 32). Chunk key = HKDF-SHA256(file key, empty salt, `filex/payload/chunk/v1/<chunk UUID>`, 32). Nonce = four zero bytes + index uint64. Каждый write attempt получает новый UUID/subkey, даже если предыдущий offset commit не состоялся. Rotation получает новый file salt. Migration между providers копирует существующий формат без изменения crypto identity.

## Чтение и ограничения
Ожидаемый header строится из DB metadata и сравнивается byte-for-byte до decrypt. Максимум файла 1 TiB, chunk 4 MiB, 2^20 попыток append; пустой файл не имеет chunks. Длины проверяются до allocation. HTTP Range расшифровывает целиком только пересекаемые chunks, затем отдаёт нужные bytes. Помимо AEAD проверяется chunk SHA-256. Отсутствующий/повреждённый object закрывает выдачу; файл получает storage_missing/integrity_failed.

Memory ограничена размером chunk для bytes; metadata ZIP/миграции пропорциональна числу записей и ограничена настройками/лимитом upload. В API одновременно до восьми upload/finalize/content/forensic потоков, каждый response write ограничен 32 KiB и deadline 2 секунды. Лимит не является production performance SLA.

## Воспроизводимость
[Test vector](../internal/cryptobox/testdata/payload-v1.json) создан Node/OpenSSL независимо от Go implementation. `TestPayloadBinaryVectorAndBindings` проверяет точные bytes, HKDF, header tamper, truncation, extra data и retry identity. [Код](../internal/cryptobox/payload.go). Backward read обещается начиная с этого первого versioned release format; ранние экспериментальные локальные envelopes payload не мигрируются автоматически.

## Неизвестная длина WebDAV
FXP1 неизменен и аутентифицирует размер. Chunked PUT без length сначала пишет скрытый provisional file с фиксированной длиной 1 TiB. После EOF новый file ID/salt связывает chunks с фактическим размером: bounded decrypt/re-encrypt, затем обычные SHA/MIME/scanner и atomic swap. Provisional ciphertext никогда не публикуется; plaintext временные файлы не создаются. Известный Content-Length/X-Expected-Entity-Length обходится одним проходом. Дополнительные storage/CPU затраты и lease/cleanup — [ADR-011](adr/011-native-webdav.md).
