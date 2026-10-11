# Session State

## Task
Создать полноценный скрин-тур и записанный видеотур по локальному FileX; включить материалы в wiki и описать воспроизведение.

Уточнение 2026-10-11: пользователь интересуется более естественной озвучкой готового тура. Сейчас используется системный голос macOS Milena. Обсуждение варианта замены не означает, что новая звуковая дорожка уже создана или провайдер выбран.

## Status
2026-10-11: DOC-02 завершён локально. Получены 44 исходных PNG, видео 5:26 с русской синтезированной озвучкой, субтитрами и 19 главами, автономная галерея и повторяемые capture/render/QA scripts. Рабочее дерево также содержит завершённый ранее перенос документации DOC-01; его [протокол](../sessions/2026-10-11-documentation-migration.md) сохранён. Базовый HEAD — `21aaa00c34ddbe7511ca950f1aacec2454989047`; commit/push в задачах DOC-01/02 не выполнялись. Production не запускался.

## Completed steps
- Прочитаны AGENTS и канонические страницы wiki, проверены существующий E2E harness, UI и Compose.
- Созданы [скрин-тур](../screen-tour.md), [галерея](../tour/index.html), [MP4](../tour/filex-tour.mp4), [VTT](../tour/captions.ru.vtt), [manifest](../tour/manifest.json), [50 checksums](../tour/SHA256SUMS) и [инструкция воспроизведения](../tour-maintenance.md).
- Записаны владелец/черновик/загрузка/папки/публикация с PIN, получатель/ZIP/отказ использованной ссылки, отчёт/публичная проверка JSON, поиск/views/выбор строк, exports/notifications/settings, desktop/mobile/RU/EN/dark, admin/backup/restore/reconcile/reopen/checkpoints/storage/SMTP/forensic/settings/help/status.
- Сценарий действительно получил ZIP/QR/PDF/JSON/CSV и checkpoint. ZIP CRC и SHA-256 трёх payload совпали; публичный UI подтвердил JSON checksum. После ZIP сервер подтвердил completed/401 и отказ повторного входа. Полный backup/restore/restart/reconcile/reopen завершился; reopen до явного подтверждения был отключён.
- Исходные PNG сняты без ретуши. Просмотрены все 44 кадра и ключевые оригиналы, а также образцы видео в начале/середине/конце. Подписи уточнены по видимому UI; субтитры разбиты без однословных коротких вспышек.
- Галерея: поиск/пустой результат/фильтры, lightbox/стрелки/Escape/возврат фокуса, desktop/mobile, axe без violations, no overflow/pageerror, настоящий playback и seek. Проверены 50 hashes и размеры PNG.
- FFmpeg декодировал полный MP4 без ошибок. Видео H.264 1440 × 1080 / 25 fps, AAC mono 48 kHz, 325,94 s, 8 616 914 bytes. Озвучка Milena: mean −17,3 dB, peak −1,2 dB. 19 глав встроены в MP4; субтитры вшиты и сохранены отдельно.
- Source-bound `make check` прошёл: Go race/vet/staticcheck, TypeScript/ESLint/Prettier, contract/negative tests, generated type sync, Go/Next build и docs. 62 Markdown / 32 requirement mappings без битых ссылок; git diff --check чистый.
- Обновлены README, index, current state, repository map, changelog и план DOC-02/DEV-05. Старый session перенесён с исправлением относительных ссылок.
- Все Docker containers/networks/volumes и 48 image tags восьми временных проектов этой задачи удалены; финальная проверка ресурсов пуста. Общий Docker cache и чужие ресурсы не удалялись.

## Next steps
1. Материалы готовы к просмотру. При изменении UI переснять по [инструкции](../tour-maintenance.md); PNG вручную не исправлять.
2. DEV-05: отдельным шагом исправить устаревшее recipient view после completed/401, сохранив privacy/resume. Карточка содержит воспроизведение и критерии, исправление не включено в задачу документационного тура.
3. Продолжить DEV-01/03, затем DEV-02/04 и EXT/REL по [плану](../implementation-plan.md). Внешнее окружение по-прежнему не подготовлено.

## Decisions
- Для возможной новой озвучки предложен короткий пробный фрагмент с русскоязычным голосом ElevenLabs; запасной вариант — Yandex SpeechKit. Сначала выбирается звучание, затем текст 19 глав из `tour/manifest.json` синтезируется отдельно по главам и повторно собирается с исходными видеозаписями, пересчитанными таймингами и субтитрами. `render.py` пока использует macOS `say`: импорт внешнего аудио потребует доработки. Проверены официальные [руководство ElevenLabs](https://elevenlabs.io/docs/eleven-creative/playground/text-to-speech) и [голоса SpeechKit](https://aistudio.yandex.ru/ru/docs/speechkit/tts/voices). Внешний синтез не запускался; доступ к сервису и выбор голоса ещё не определены.
- Продуктовые кадры сняты в pinned Chromium Linux на отдельном Compose со случайными секретами и синтетическими данными. Код приложения ради съёмки не изменялся.
- Полный видеоряд — записанные действия; темп изменён под повествование, ожидания сокращены. Озвучка синтезированная; заголовки/субтитры добавлены вне UI.
- SMTP/S3/webhook/перенос/ротация/forensic показаны как конфигурация, без внешних вызовов. Физические устройства не тестировались. Проверка галереи в локальном Chrome не объявляется новой полной browser matrix продукта.
- `.cache/tour` содержит raw recordings, скачанные synthetic artifacts, инструменты и QA; эти файлы не поставляются. FFmpeg/голос/системный шрифт не добавлены в runtime или Git.
- После записи действует только read-only preview wiki на `127.0.0.1:8765`, exec session 2325. URL галереи: `http://127.0.0.1:8765/tour/`. Открытие в Codex поставлено в очередь (`queued`), наличие видимой вкладки не подтверждалось. HTML/MP4 доступны и без этого сервера.

## Context
[Индекс](../index.md), [текущее состояние](../current-state.md), [тур](../screen-tour.md), [эксплуатация](../operations-recovery.md), [предыдущая сессия](../sessions/2026-10-11-documentation-migration.md).

Съёмка начата `2026-10-11T15:13:18.711Z`; отдельный hash исходников приложения `9dae044484d8476a01c4fdc0fcf244e75806fc1c322f2ce22ac123a8bf94d3cb`, алгоритм/область в manifest. Полный source hash финального рабочего дерева для check gate: `dc804519c046084c205e1067772c244afbf4b2e6cb723d47f7e51c0d6b8fcaef`; `passed=true`, `sourceUnchanged=true`, `exitCode=0`, duration 24,13 s. Record `.cache/evidence/gates/check.json`; предыдущий record DOC-01 сохранён в `.cache/evidence/documentation-migration/check-gate.json`.

QA галереи: Chrome `154.0.8037.98`, desktop 1440 × 1000 / mobile 390 × 844; `.cache/tour/qa/result.json`. Логи `.cache/tour/{capture,render,gallery-check,check-gate,video-decode,audio-check}.log`. SHA-256 MP4: `08eb067ab50cbd1db789dd13801277b304c14cf34b3390bf8c23e89ab50dbb92`.

## Risks / blockers
- Владелец подтвердил: «Окружение ещё не подготовлено; зафиксировать внешние блокеры». 14 внешних decisions pending и 5 MUST partial сохранены. Production guard не менялся.
- DEV-05 — UI-ограничение: после completed сервер отклоняет recipient API, но уже открытая страница может оставить прежние ready-строки рядом с ошибкой. Это не разрешение повторной серверной выдачи. Тур показывает реальный отказ повторного открытия ссылки.
- Прочие release gate records относятся к прежним source hashes и не объявляются свежими; текущая задача выполнила check и отдельный tour QA. Новый CI/commit/release/production approval не создавались.
