# ADR-016: автономная коммерческая лицензия и панель издателя

Статус: **accepted; реализация локально**, 2026-10-14. Владелец явно утвердил отдельный vendor Ed25519 профиль, expiry policy и ограничения users/bytes/features. Проверки и границы — [session](../session.md), эксплуатация — [руководство](../licensing-operations.md).

## Контекст и проблема

FileX поставляется автономно; доступность внешнего сервера не может определять право текущей работы. Нужно ограничить использование и исключить простое редактирование license claims. Исходный SEC-004/ADR-005 ограничивает Ed25519 ZIP checksum manifest, поэтому новый класс подписи нельзя внедрить неявно.

## Предлагаемое решение

Отдельная vendor панель выдаёт immutable JWS с vendor Ed25519 подписью. Заказчик хранит только доверенные public keys и файл лицензии. Vendor key никогда не совпадает с instance ZIP signing key. Подписанные kind/installationId/revision/dates/limits/features проверяются offline, права применяются общими PostgreSQL guards. Подписка ограничивает срок новых платных действий; perpetual сохраняет право работы разрешённого релиза и отдельно ограничивает обновления.

Утверждённое расширение SEC-004: ZIP signing остаётся только SHA256SUMS; отдельные vendor ключи используются для license/release metadata, вне клиентского keyring. [Полная спецификация](../licensing.md) содержит формат, threat model, states, issuer workflow, recovery, тестовую матрицу и открытые решения.

## Поток и пример

Installation request → ручной импорт в vendor panel → approved immutable issuance → isolated signer → license.jws → ADMIN preview/reauth/CAS apply → PG epoch/domain checks. Продление той же установки сохраняет entitlementId и повышает revision; неверный импорт не заменяет активную лицензию.

## Альтернативы и ограничения

- HMAC с секретом в продукте позволяет извлечь ключ и выпускать лицензии; отклонён как базовая модель.
- Online activation противоречит автономности; не используется.
- MAC/IP binding нестабилен и подделывается; случайный installation ID честно даёт только software binding.
- TPM/HSM у заказчика повышает стоимость копирования, требует аппаратной поддержки и отдельного approval; не baseline.
- Обфускация не является криптографической гарантией и не заменяет signature/enforcement.

Без доверенной внешней точки/аппаратного состояния невозможно надёжно обнаружить клон, полный rollback или выключить удалённую offline установку. Предлагается сохранять safety/recovery и прежнюю выдачу после expiry; эта политика утверждена владельцем. Независимый security review остаётся отдельным gate.

## Следствия и реализация

Customer runtime env не управляют trust; public keys/release metadata встроены при сборке. Реализованы отдельные vendor secret storage/deployment, customer control-plane migration, verifier, schema sources, UI/API/CLI, shared quota guards и signed release metadata. Контракты создаются до реализации; новая логика не снимает production guard и не изменяет payload encryption/retention. Карточки LIC-01–06 и критерии — в спецификации; реализация и локальные проверки не заменяют независимый security review и release qualification.

## Подтверждение владельца — 2026-10-14
Владелец явно утвердил отдельный Ed25519 ключ издателя, политику истечения и ограничения по пользователям, размеру передачи и функциям; поручил реализацию лицензирования и закрытой веб-панели. Это снимает ранее указанный блокер согласования crypto/license policy. Реализация начата; внешний review, остаточные риски и production deployment автоматически не приняты.
