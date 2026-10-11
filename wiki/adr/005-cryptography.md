# ADR-005: криптографические контексты и формат payload

Статус: реализовано, внутренний review 2026-10-11. SEC-003–004. Production допуск по-прежнему зависит от фазы 7; это не независимый криптоаудит.

## Контекст и проблема
Class DEK не используется как универсальный ключ всех данных. KEK находится вне PostgreSQL; разные классы имеют отдельные версии. Помимо защиты ciphertext, необходимо аутентифицировать длины, положение чанка, идентичность файла и версию ключа. Повтор PATCH после сбоя и восстановление старой БД не должны повторять пару key/nonce с другим plaintext.

## Решение
Малые секреты и encrypted report snapshot используют стандартный AES-256-GCM envelope: `01 || random nonce[12] || ciphertext || tag[16]`. AAD — ASCII `filex/<class>/v1/<safe identifier>`; только проверенные identifiers. На root DEK допускается 2^20 encryption attempts, предварительная атомарная reservation в PostgreSQL не возвращается при ошибке. На KEK также резервируется wrap budget 2^20. Исчерпание блокирует новые записи до rotation/recovery. Старые незасчитанные DEK при миграции 0013 получают исчерпанный budget; чтение сохраняется, перед новыми записями нужна rotation. Старые ключи автоматически не удаляются.

Payload использует fixed header `FXP1` и AES-256-GCM с nonce `00000000 || uint64_be(index)`. Формат offsets — [спецификация](../encryption-format.md). File subkey = HKDF-SHA256(class DEK, random file salt, file ID context). Для каждой попытки записи генерируется новый chunk UUID, от которого выводится отдельный chunk subkey. Это уточнение baseline от 2026-10-10 необходимо для tus retries/DB restore: одного file salt и повторяемого offset недостаточно. AAD включает весь header с chunk/file/key IDs, salt, file length, chunk limit, offset, length и index. UUID попытки попадает в DB только вместе с offset commit. Неуспешный ciphertext остаётся лишь в управляемой очереди cleanup. Copy migration сохраняет crypto identity; rotation меняет class key и file salt и проверяет исходный SHA-256.

Artifact получает собственные UUID/salt/HKDF subkey, максимум 64 MiB/16 chunks. Его random nonce envelope поэтому имеет не более 16 записей на subkey. Artifact rotation всегда использует новый salt. Snapshot reports относится к counted root-key envelopes, а не к этому исключению.

## Поток
KEK → unwrap class DEK → derive object keys → authenticate header/chunk → verify server checksum → guarded stream. Неверный KEK при startup блокирует запуск. Локальная порча payload переводит файл в integrity_failed; сервис целиком не отключается. Signing seed хранится как отдельный wrapped class key. Ed25519 подписывает только bytes SHA256SUMS.txt; PEM/SPKI и SHA-256(DER/SPKI) позволяют независимую проверку доверия.

## Пример и evidence
[Node/OpenSSL vector](../../internal/cryptobox/testdata/payload-v1.json), `TestPayloadBinaryVectorAndBindings`, `TestEnvelopeBinaryVector`, RFC8032 vector и PG `TestIntegrationPayloadHeaderAndKeyBudget`. Тесты меняют каждый byte, header fields, truncation/trailing bytes и проверяют отдельный subkey для retry. [Verifier](../../cmd/filex-verify/main.go) проверяет signed manifest по умолчанию; `--verify-files` явно включает чтение payload.

## Ограничения
FXP1 — первый зафиксированный payload format до первого релиза. Экспериментальные ciphertext ранних локальных срезов не являются поддерживаемым release format; нельзя открывать такой storage новым binary без отдельной миграции. Restore/reconciliation и KEK rewrap/recovery CLI реализованы фазой 6; key budgets и fresh-instance recovery описаны в [ADR-006](006-operations-recovery.md). Не выводить ключи/секреты в logs. Randomness, UUID и независимые backup KEK остаются частью trust model. Нормативные primitives: [Go cipher](https://pkg.go.dev/crypto/cipher), [RFC5869](https://www.rfc-editor.org/rfc/rfc5869).
