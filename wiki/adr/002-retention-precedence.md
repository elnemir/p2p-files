# ADR-002: доступ и удержание

Статус: реализовано и проверено; принято 2026-10-10. Requirements: FR-004, FR-007, SEC-006–007.

## Контекст и проблема
Freeze удерживает payload, но не отменяет revocation. Одно поле состояния не должно смешивать доступ, lifecycle и retention.

## Решение и поток
Хранить lifecycle (`draft/active/completed/expired/revoked`) и hold отдельно. Freeze не превращает terminal lifecycle в active. Порядок проверок: restore gate/account block → revoked → absolute expiry → ordinary expiry → transfer/file hold → file integrity/storage → lease. Все deny conditions действуют независимо; hold предотвращает physical deletion, а не восстанавливает доступ.

Transfer freeze сохраняет `remainingTtl` на момент freeze и блокирует новые и текущие streams. `absoluteExpiresAt = publishedAt + maxTTL` никогда не сдвигается. При unfreeze обычный expiry становится `min(now + remainingTtl, absoluteExpiresAt)`. Если absolute expiry уже прошёл или revoke произошёл во время hold, transfer остаётся terminal. Повтор freeze не увеличивает remainingTtl. File freeze не останавливает transfer TTL и блокирует весь ZIP.

Expiry логически закрывает доступ. Worker удаляет только объекты без hold; integrity_failed при обычном expiry сохраняется для расследования. Revoke/block/account delete удаляют integrity_failed, если hold отсутствует. После снятия hold terminal transfer не открывается: объект попадает в соответствующий cleanup. ADMIN force delete — отдельный privileged/re-auth/reason/audit workflow, не обход guard в обычном worker.

## Пример
TTL 24 часа, freeze через 6 часов: remaining 18 часов. Unfreeze через неделю не даёт 18 часов, если absolute max TTL уже истёк. Payload сохранён для ADMIN до снятия hold, recipient доступа не получает.

## Проверки и ограничения
Требуются таблицы interleavings freeze/revoke/expiry/complete и clock boundary tests. Удержание бессрочно; доступ ограничен hard deadline. Числовые значения примера не являются настройками по умолчанию.
