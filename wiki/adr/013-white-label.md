# ADR-013: настраиваемый брендинг установки

## Контекст и проблема
2026-10-13 владелец запросил полный white labeling и подтвердил отсутствие фиксированных названия/цветов. FileX — single-origin установка без tenancy. Сейчас бренд зашит в интерфейс, browser notifications, SMTP и PDF; переключение light/dark не меняет палитру. Settings v1 и JSON отчётов имеют строгие действующие контракты.

## Решение до реализации
Отдельный typed `branding_revisions` в PostgreSQL следует существующим versioned settings/CAS/audit patterns; не расширяет strict Settings v1. Общий бренд содержит productName, taglineRu/taglineEn, footerText, defaultTheme (system/light/dark), iconStyle (outline/rounded/duotone), две палитры и PNG/JPEG logoLight/logoDark/favicon. Исходный бренд FileX действует при отсутствии ревизий.

ADMIN редактирует локальный черновик, видит изолированный preview обоих режимов, применяет одним PUT с expectedRevision. В одной transaction выполняются current ADMIN/session/reauth guards, advisory lock, сравнение revision, insert и audit. Conflict не затирает чужую запись. Сброс создаёт новую ревизию исходного оформления; отмена только убирает локальный draft. Внешние URLs, CSS/JS/SVG/HTML не принимаются. Logo dark может использовать light fallback; пустой logo показывает название.

Публичный GET возвращает только предназначенный для публикации branding, не app settings/actor/history. SSR получает тот же DTO через внутренний GET /branding на существующем internal listener; URL задаётся оператором через FILEX_BRANDING_API_URL, не Host/request input. No-store и bounded timeout; при недоступности показывается нейтральное оформление без ложного названия заказчика. Клиент обновляет branding через same-origin GET, периодически и при возврате фокуса; применение в текущей/соседней вкладке инициирует немедленное обновление. Личная light/dark/system preference остаётся override общего defaultTheme.

## Данные и контракты
ADMIN input → strict bounded JSON → name/color/contrast/raster validation → canonical PNG re-encode → revision CAS + audit → public DTO → SSR/context/CSS variables/icons/logo/favicon/title. OpenAPI получает только новые GET /branding, GET /admin/branding и PUT /admin/branding, самостоятельную schema branding-v1. AsyncAPI/доменные события передачи не меняются: branding refresh использует HTTP, не recipient/user event channel.

Новые PDF фиксируют presentation snapshot отдельно в `reports.branding` и `security_incident_reports.branding` в transaction создания. Render использует сохранённые productName/logoLight/light palette, не текущий бренд. Старые строки с NULL сохраняют FileX renderer. Canonical report JSON/hash/privacy остаются прежними, artifact hash подтверждает PDF bytes. Branding public/nonsecret, существующие encrypted snapshot/ArtifactStorage/crypto formats не меняются. SMTP использует актуальный brand при фактической попытке доставки, включая display name отправителя; адрес/transport задаются прежним SMTP config. Уже доставленные письма не переписываются. Технические cookies/protocol headers/CLI identifiers не переименовываются: это совместимость интеграций, не пользовательское название продукта.

## State machines
UI clean → editing → preview → saving → clean; invalid/error → editing; 409 → conflict с явной загрузкой актуальной версии, без automatic overwrite. Reset → editing(defaults) → saving. PG revision N → новая ревизия только после commit (sequence допускает пропуски после rollback); audit failure/invalid asset/stale revision не меняют public state. Report branding фиксируется при создании и не меняется при regenerate/restore.

## Threat model и review
Stored XSS/CSS injection закрываются plain text + строгим hex и отсутствием arbitrary markup. PNG/JPEG проверяются по bytes, dimension/pixel/size budget, decode/re-encode удаляет посторонние chunks/metadata. Upload ≤128 KiB каждый, logo ≤1024×512, favicon ≤256×256, request ≤600 KiB. Цвета проверяются на contrast текста/акцента ≥4.5:1 и границ ≥3:1 в обоих режимах. Public branding не содержит secret URLs/actors и предназначен для anonymous login/recipient/help/status/verify. ADMIN cookie/CSRF/fresh reauth сохраняются, bearer запрещён; current session/role сверяются под transaction locks. Никакого server fetch logo URL/SSRF. Resource bounds действуют до full decode.

Architecture consistency: PG authority, общий origin, backup schema discovery, существующий runtime recovery guard и tamper-evident audit сохраняются. Migration 0018 additive, controlled upgrade остаётся обязательным. Crypto review: нет новых primitives/ключей/AAD, report JSON/manifest signatures/recipient one-time не затронуты. Это внутренний review, не независимый security acceptance.

## Приёмка и ограничения
Тесты: invalid names/CSS/raster/dimensions/contrast; USER/bearer/CSRF/expired reauth отказ; concurrent CAS; audit fault rollback; backup/restore; historical report rendering; branded SMTP; пять browser profiles, SSR до hydration, все public modes, preview/apply/reset/conflict, logo/favicon/icons, long name/320px, axe/light/dark/system и reviewed visual baselines. Документация и media предыдущих версий сохраняются как история; новая UI evidence отдельно.

Пример: ADMIN задаёт «Acme Share», индиго для light и светлый лавандовый для dark, rounded icons, загружает два логотипа и favicon, проверяет preview, затем применяет. Вкладки и новые письма получают новое имя; ранее выпущенный PDF остаётся исходным. Production guard и Windows-native pending не снимаются.
