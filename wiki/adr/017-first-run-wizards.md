# ADR-017: мастера настройки запущенных контейнеров

## Контекст и проблема
Пользователь запросил веб-настройку уже запущенных контейнеров, без установки Docker и запуска Compose из HTTP. Первый ADMIN сейчас требует CLI; портал требует ручного OIDC env.

## Решение
FileX /setup создаёт первого ADMIN. Портал /issuer/setup сохраняет OIDC client, MFA ACR и стабильный subject первого security_admin. Origin задаёт оператор контейнера. БД, KEK, storage и vendor seed остаются host bootstrap; веб не меняет trust roots, не получает Docker socket и не генерирует signing seed.
Оба мастера opt-in: SHA-256 случайного 32-byte токена и абсолютное время истечения передаются контейнеру; браузер предъявляет токен только заголовком. Exact Origin, bounded JSON, отсутствие CORS, no-store. Завершение FileX атомарно с ADMIN/password history/audit; ops_setup закрывает повторный запуск даже после удаления users и штатного restore. CLI создание пользователя также закрывает мастер. Existing users закрывают его при migration.
Портал сохраняет одну неизменяемую encrypted config в PostgreSQL; отдельный ISSUER_CONFIG_KEY (32-byte base64) хранится вне БД и signer. Используется существующий cryptobox AES-GCM с отдельным AAD; signing key не переиспользуется. Runtime читает config при старте/первом запросе, далее immutable cache; разные replicas видят одну строку. Env-configured портал сохраняет прежний режим без мастера.

## Состояния и поток
Disabled (нет токена/истёк) или pending → validate → transaction → complete. Ошибка/rollback оставляет pending. Конкурирующие FileX операции сериализованы ops_setup; issuer INSERT unique singleton даёт одного победителя. Complete не возвращается в pending через HTTP. Потеря ответа после commit определяется чтением status. Никаких background jobs или новых AsyncAPI событий: synchronous DB transactions; существующий signing outbox неизменен.
Вход FileX: email/password → hash + license bootstrap guard → ADMIN → обычный вход и раздел Лицензия. Вход issuer: HTTPS IdP/client secret/ACR/sub → discovery с timeout → AEAD → immutable config → обычный OIDC MFA вход. Discovery не доказывает MFA: первый оператор проходит настоящий login; ошибка subject/ACR требует host recovery, не публичного reset.

## Threat model и reviews
Угрозы: первый посетитель захватывает систему, CSRF, replay, гонка двух администраторов, crash после commit, read-only DB dump, подмена origin, повторное открытие после restore. Контрмеры: host-delivered случайный token, expiry/constant-time hash comparison, exact Origin, atomic durable latch, immutable issuer config, separate AEAD key. DB/host root и полный rollback VM остаются доверенной границей; browser XSS компрометирует вводимые secrets, поэтому не сохранять токен в URL/localStorage/logs.
Architecture consistency review: PostgreSQL authoritative, без Redis state и side effects в Docker, existing licence/recovery/network guards сохранены. Crypto design review: standard SHA-256 token digest и существующий AES-GCM; без нового primitive или reuse Ed25519 seed. Setup token не шифровальный ключ; оба значения независимы. API никогда не возвращает password/clientSecret/config key. Public status содержит только state.

## Ограничения и проверка
FileX production guard не снимается. TLS/loopback tunnel готовит host оператор. Реальный MFA и escrow не подтверждаются мастером. Проверки: negative auth/origin/expiry/body, concurrent completion, transaction fault rollback, restart/readback, wrong key, browser flows/mobile. Контракты создаёт scripts/setup_contracts.py; internal review не заменяет внешний security audit.
