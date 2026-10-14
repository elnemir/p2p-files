# Evidence: полный фото- и видеотур, 2026-10-14

## Контекст и проблема
Запись 2026-10-12 предшествовала UX, white labeling и корпоративной аутентификации. Задача — показать действующую систему целиком, добавить ElevenLabs и синхронизировать wiki. Изменены только tour tooling, media и документация; runtime/API/env/migrations сохранены.

## Источник и поток данных
Локальный runtime main `28a6b4a0aac2ed23c7699ec35e468f59255949b7`; applicationSourceSHA256 `76574e49e37a5fdfd1bbd7e935c9e22691222070a912f1260ca058e3d53bcc42`. Capture plan SHA-256 `beaf9b9053a80f8727c8c2ae5d2a730bf1adbeffd600d1c7f63ec13181e753a8`. Application fingerprint не включает документацию и генераторы; scope записан в manifest. Полная квалификация UX до этой задачи — [PR #26 / архив](../sessions/2026-10-14-product-ux-delivered.md).

`capture.py` → disposable Compose и Linux Chromium → приватные raw recordings/PNG → reviewed plan validation → exact-text/hash narration → MP4/VTT/manifest/галерея/Markdown → media/browser/docs QA. Съёмка начата `2026-10-14T09:16:08.379Z`; все 71 кадр и 26 глав получены одним завершённым capture, 16 assertions PASS. Project/volumes/signer после съёмки остановлены и удалены.

## Состав и выполненные действия
- Пересняты прежние сценарии USER/recipient/ADMIN, WebDAV, отчёты, exports, recovery, help/status и мобильные экраны.
- Добавлены report trash/restore/history с неизменным JSON, settings draft/validate/apply с readback, OIDC/LDAP/SAML формы и реальный локальный SAML callback/JIT USER.
- Применены название «Север», логотипы/favicon, Ocean palette и duotone icons; отдельная сессия видит публичный бренд/light/dark/mobile. Выполнен reset исходного оформления.
- Новых подписок/production ресурсов нет; TTS получил только демонстрационные тексты.

## Озвучка
ElevenLabs Liza, `KzqxCy7zSSePwgb1Cz0Q`, `eleven_multilingual_v2`; speed=1, stability=0.5, similarity=0.75, style=0, speaker boost=true. Всего 6915 символов: 4529 в 18 неизменённых фрагментах, 2386 в восьми новых. Веб-квота 4870→2484. Последний синтез завершился перед предложением подписки; файл получен из History, повторного синтеза не было. Все 26 файлов совпали с текстом/hash, новые MP3 полностью декодированы. API-квота не проверялась.

## Итоговая поставка
- 71 PNG / 26 глав / 104 VTT cues; длительность 476.24 s (7:56).
- MP4 15971458 bytes, SHA-256 `c04f6620d8f2e099e6bb184309157dcd7ea5fff7a65a14c3e01d6bc49f1a7bac`.
- H.264 1440×1080 / 25 fps, AAC mono 48 kHz; главы, вшитые русские captions, faststart. VTT timing приблизительный по длине текста/длительности, не provider word alignment.
- [MP4](../tour/filex-tour.mp4), [скрин-тур](../screen-tour.md), [галерея](../tour/index.html), [manifest](../tour/manifest.json), [checksums](../tour/SHA256SUMS).

## Проверки и диагностика
Все 71 PNG просмотрены на 12 contact sheets, крупные планы OIDC/палитры — отдельно. Проверены девять видеофреймов: начало, завершённая выдача, report restore, settings, SSO, бренд и конец; заголовки/captions не перекрывают UI. Desktop/mobile галерея просмотрена. Финальная browser QA (Chrome 155.0.8059.79, 2026-10-14T09:35:23.988Z): 77 checksums, 71 screens, 26 chapters, desktop/mobile axe, playback/seek, navigation и семь unsafe paths PASS; JS errors отсутствуют. Плеер прочитал длительность 476.261333 s (разница с монтажной шкалой — AAC padding).

Первый encode дал истинный пик +1,5 dBFS около 398,586 s в главе 23; декодированные float samples подтвердили превышение (это не только показание плеера). В renderer sample headroom limiter изменён с −3 на −6 dB. Пересборка использует те же оплаченные MP3. Финальный full decode с `-xerror` PASS: integrated loudness −17,9 LUFS, LRA 2,7 LU, true peak −0,5 dBFS; превышения 0 dBFS нет. Это измерение финального AAC, не обещание результата будущих голосов: после новой записи нужен новый замер.

Первая попытка capture ожидала terminal header после отдельного повторного входа. В действующем UI terminal появляется сразу после ZIP; повторный вход возвращает generic отказ. Сценарий исправлен под обе реальные стадии без изменения runtime/assertion ослабления. Quiet hours ждут PUT=204 и статус до GET. Два последующих полных capture успешны; итоговый добавляет крупные планы OIDC/палитры.

## Ограничения и продолжение
Windows WebDAV **ожидает стенда**. SAML signer одноразовый учебный, OIDC/LDAP — только формы; реальные tenants Keycloak/Entra/Google/AD, SMTP/S3, устройства и независимая приёмка не квалифицированы. 5 partial MUST, 14 pending решений и production guard сохраняются. Исторические 13 local/PR/main gates UX не переносятся на media/tooling diff.

Сырые логи/кеши: `.cache/tour/refresh-20261014/`, `.cache/tour/qa/`, `.cache/tour/raw/`, `.cache/tour/elevenlabs-web-Liza-20261014/`. Они ignored и содержат синтетические sessions; не публиковать их автоматически. Повторение — [обслуживание тура](../tour-maintenance.md). При следующем UI изменении актуализировать reviewed plan, переснять изменённые сценарии и заново связать output с source/hash.

## Финальный check
`python3 scripts/acceptance.py run check` PASS, sourceSha256 `f9738f63b5a290f90554821fc1d6f9ed19f90dac562d1db850c9a6e71df72bc2`, sourceUnchanged=true, exitCode=0. Включены Go vet/staticcheck/race, TS/ESLint/Prettier, contracts/types, Python negatives (13 tour tests), repository policy, docs и Go/Next builds. Обход 95 Markdown: все 32 requirements связаны, broken local links отсутствуют. Это gate check; новые PG/full-browser/scanner/load/DR gates в этой media-only задаче не запускались. Внешний CI для этих локальных изменений ещё не выполнялся.

## Подготовка commit/push/merge
По команде владельца повторно выполнены `check`, `source`, `compatibility`: PASS на staged source `f74a78d67d771cb7ee1663f0ed951c8a3a61a8e883426be94dbdb1fcd6879122`, sourceUnchanged=true. Старый fingerprint учитывал ещё отслеживаемые удалённые PNG как deleted; после staging этих удалений их paths исключены из набора. Media bytes/hash остаются прежними. Source scan проверил историю и подготовленные файлы; secrets/HIGH/CRITICAL не найдены. Полная регрессия кандидата обязательна в PR CI перед squash merge, затем отдельно проверяется main CI.
