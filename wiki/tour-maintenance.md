# Как обновлять скрин-тур и видеотур

## Контекст и задача

[Тур](screen-tour.md) знакомит с FileX через настоящий интерфейс: от загрузки до получения файлов и восстановления системы. Он хранится рядом с канонической документацией в `wiki/tour/`. PNG — исходные снимки Playwright; MP4 — монтаж записанного Chromium video, а не анимация статичных изображений. Галерея не входит в runtime приложения.

Старые картинки быстро расходятся с продуктом. Поэтому сценарий, подписи, сборка видео и галерея находятся в репозитории. При изменении интерфейса нужно переснять затронутый маршрут и проверить подписи, а не ретушировать состояние на картинках.

## Компоненты и поток данных

1. [capture.py](../scripts/tour/capture.py) создаёт случайный проект `filex-tour-<10 hex>` с отдельными volumes. Секреты генерируются в памяти, `.env` оператора не читается. Занятый порт 8090 вызывает отказ, чтобы не подключиться к чужому стенду.
2. Compose собирает настоящие API/worker/web из рабочего дерева. CLI создаёт две учётные записи `designer@example.test` и `operator@example.test`; браузер запускается из закреплённого E2E image. Docker socket внутрь браузера не передаётся.
3. [capture.mjs](../scripts/tour/capture.mjs) выполняет UI-сценарий, скачивает артефакты и проверяет результаты. Каждый PNG попадает в `wiki/tour/screenshots/`; непрерывные видео по ролям, внутренний timeline и временные файлы — в `.cache/tour/raw/`.
4. Harness перезапускает только API/worker своего проекта при restore. После завершения, включая обычную ошибку сценария, удаляет собственные контейнеры, сети и volumes. Не выполняет global prune и не затрагивает существующие проекты.
5. [render.py](../scripts/tour/render.py) берёт записанные фрагменты, использует русскую озвучку macOS Milena или проверенные внешние audio/character timings и создаёт ASS/VTT субтитры, собирает H.264/AAC MP4 с главами. Ускорение ожиданий ограничено; короткие фрагменты замедляются под повествование. Заголовки и вшитые субтитры находятся в полях вокруг UI.
6. Шаблоны [HTML](../scripts/tour/gallery.html), [CSS](../scripts/tour/gallery.css) и [JavaScript](../scripts/tour/gallery.js) создают автономную галерею. Встроенный manifest позволяет открывать её через `file://` без fetch и без внешних шрифтов/сервисов. Отдельный [manifest](tour/manifest.json) содержит описания, временные отметки и границу проверки.

Поля URL из manifest проходят отдельную проверку перед установкой src/href: разрешён только `screenshots/NN-name.png` без query, traversal, внешнего origin, data/javascript schemes или другого формата. Имя кодируется как один URL segment. Текстовые поля вставляются через textContent; JSON внутри HTML экранирует `<`. Это закрывает предупреждения CodeQL DOM XSS, обнаруженные при первом включении сканера: три места в исходнике и три в generated copy. Негативный browser QA подставляет семь недопустимых путей в настоящий embedded manifest и проверяет отказ до появления URL в DOM. После изменения gallery.js выполняется `render.py --gallery-only`, а затем повторные checksums/browser QA; PNG и MP4 не пересоздаются.

## Подготовка

Нужны Docker/Compose, Node/npm и зависимости репозитория (`npm ci`), Python 3, свободный loopback-порт 8090 и несколько гигабайт под Docker cache. Сценарий длится несколько минут; cold build может идти дольше. Не запускать одновременно с E2E/dev на том же порту.

Для сборки озвученного видео на macOS нужны установленный голос `Milena` (`say -v '?'`) и FFmpeg с libx264, AAC, drawtext и libass. Эти инструменты используются только для документации. Runtime images продукта не меняются. Можно указать свой бинарник FFmpeg через `--ffmpeg` и шрифт с кириллицей через `--font`.

При этой съёмке использован FFmpeg 7.1 из wheel `imageio_ffmpeg-0.6.0-py3-none-macosx_11_0_arm64.whl`; SHA-256 wheel: `b1ae3173414b5fc5f538a726c4e48ea97edc0d2cdc11f103afee655c463fa742`. Wheel скачан и распакован в игнорируемый `.cache/tour-tools`, без глобальной установки. Он и исходный FFmpeg не включены в поставку приложения. Голос и системный шрифт также не копируются в репозиторий.

## Воспроизведение

Запускать из корня репозитория:

```sh
python3 scripts/tour/capture.py
python3 scripts/tour/render.py --ffmpeg /absolute/path/to/ffmpeg
make docs-check
git diff --check
```

Проверка готовой галереи и media выполняется [check.mjs](../scripts/tour/check.mjs):

```sh
node scripts/tour/check.mjs
```

Для неё нужен локальный Chromium Playwright (`npx playwright install chromium`, если он ещё не установлен). Можно использовать установленный Chrome: `FILEX_TOUR_BROWSER_CHANNEL=chrome node scripts/tour/check.mjs`; создаётся отдельный временный профиль. Проверка не подключается к приложению: открывает готовый HTML через file URL, сверяет hashes/PNG, проверяет поиск/фильтры/lightbox, axe на desktop/mobile и воспроизведение/перемотку видео. Результат и contact sheets для визуального review находятся в `.cache/tour/qa/`. Системные скрытые файлы Finder не включаются в checksums.

Скрипт записи изменяет галерею только через её assets; файлы приложения и production-конфигурация не редактируются. Ошибочный сценарий оставляет диагностический screenshot и текст ошибки в `.cache/tour/raw/`, а не в публикуемой галерее. При аварийном завершении процесса/хоста проверить оставшиеся ресурсы по точному случайному project name из capture log; не использовать общий `docker compose down`.

Для изменения оформления или текста страницы без повторного кодирования видео:

```sh
python3 scripts/tour/render.py --gallery-only
python3 -m http.server 8765 --bind 127.0.0.1 --directory wiki
```

Открыть `http://127.0.0.1:8765/tour/`. `--gallery-only` использует существующий manifest и обновляет Markdown/HTML/CSS/JS и `SHA256SUMS`. После любого изменения assets пересчитать checksums этим режимом или полной сборкой; прежний hash не переносить на изменённый файл.

## Проверки и критерии готовности

- Capture проходит полностью, без pageerror и горизонтального переполнения на сохранённых viewport. Ошибка сценария не считается частично успешным туром.
- ZIP содержит реальные демонстрационные документы; PDF/JSON/CSV/QR скачиваются; JSON проверяется публичным UI. Восстановление действительно доходит до restart/reconcile/reopen; до подтверждения reopen недоступен.
- Каждый PNG визуально проверен: текст читается, важное действие попало в кадр, нет случайного failure state и рабочих секретов.
- MP4 декодируется целиком без ошибок; есть видео/аудио и главы. Проверяются начало, середина, конец, субтитры, синхронизация повествования с главами и переход по главам в браузере.
- Галерея проверяется на desktop/mobile: поиск, фильтры, lightbox, Escape/стрелки, навигация клавиатурой и отсутствие переполнения. Все ссылки и файлы существуют.
- Wiki/session, индекс и текущий статус отражают дату и границу новой съёмки. Состояния внешней приёмки не изменяются по факту наличия тура.

## Результат съёмки 2026-10-11

Получены 44 PNG и видео длительностью 325,94 s (5:26): H.264, 1440 × 1080, 25 fps; AAC mono 48 kHz, 19 встроенных глав. Размер MP4 около 8,2 MiB; SHA-256: `08eb067ab50cbd1db789dd13801277b304c14cf34b3390bf8c23e89ab50dbb92`. Все 50 assets перечислены в [SHA256SUMS](tour/SHA256SUMS). Базовый commit, отдельный hash исходников приложения, сценарные assertions и timeline находятся в [manifest](tour/manifest.json). Этот hash не является полным release source hash.

Полный capture завершился с PASS; браузерные ошибки отсутствуют. Фактически скачаны ZIP с тремя документами (CRC и все три payload SHA-256 совпали), QR ZIP, PDF, JSON, CSV и audit checkpoint. JSON проверен публичным интерфейсом. Backup/restore/restart/reconcile/reopen завершены. Повторный доступ к completed передаче отклонён сервером.

Просмотрены все 44 кадра на contact sheets и ключевые оригиналы; проверены образцы видео в начале, середине и конце, читаемость подписей и субтитров. FFmpeg декодировал весь MP4 без ошибок; audio mean −17,3 dB, peak −1,2 dB. Галерея проверена в отдельном Chrome 154.0.8037.98: desktop 1440 × 1000, mobile 390 × 844, поиск/пустой результат/фильтры, lightbox/стрелки/Escape/возврат фокуса, axe без violations, отсутствие горизонтального переполнения, воспроизведение и переход к главе. Проверены 50 checksum и PNG dimensions. Это QA галереи; продуктовые кадры записаны отдельно в закреплённом Chromium Linux.

Доказательства текущего прогона: `.cache/tour/{capture,render,gallery-check,check-gate,video-decode,audio-check}.log`, `.cache/tour/qa/result.json`, contact sheets и кадры видео в `.cache/tour/qa/`. Эти локальные файлы не публикуются и не заменяют release evidence. Временные Compose контейнеры/сети/volumes и 48 tags восьми созданных в ходе отладки проектов удалены; общий Docker cache и чужие ресурсы не очищались.

## Озвучка ElevenLabs

### Контекст и текущее состояние

Для более естественной речи добавлен отдельный локальный [narrate.py](../scripts/tour/narrate.py). Он не является частью API/worker/web и не запускается в CI. Пользователь предоставил существующий ключ и добавил голос `xyu8HSCv1JYrhLx4m8UG` — Ariana, Energetic and Inspiring. Models/Voices API подтверждены; выбран `eleven_multilingual_v2` с поддержкой русского. Первый синтез главы 1 получил HTTP 402 `payment_required`, аудио не получено. **Пользователь отложил озвучку: текущий MP4 остаётся с Milena.** Успешный provider response, произношение и качество полного тура ещё не приняты.

Ранее подготовленная в браузере форма нового ключа не отправлялась: пользователь заменил выпуск предоставлением существующего секрета. Его срок, серверный spending limit и полный набор scopes не изменялись. Subscription API ответил `missing_permissions`; права не расширялись, оплата не выполнялась.

### Секреты и поток данных

Ключ хранится в `~/.config/filex/elevenlabs.env`: каталог 0700, файл 0600. Имена полей — `ELEVENLABS_API_KEY` и `ELEVENLABS_VOICE_ID`; значение ключа никогда не записывается в wiki/Git/manifest/logs. Генератор читает только эти два поля без shell source/eval; symlink и доступный группе/остальным файл отвергаются. Переменные окружения могут переопределить значения, `--env-file` задаёт другой приватный файл. Приложение, CI и production этот секрет не получают. Для минимального нового ключа нужны TTS Access, Voices Read, Models Access; [официальная инструкция по authentication/scopes](https://elevenlabs.io/docs/api-reference/authentication).

`capture.json chapters[].narration` → проверка номера/текста/лимита → HTTPS Models/Voices preflight → [TTS with timestamps](https://elevenlabs.io/docs/api-reference/text-to-speech/convert-with-timestamps) с текущим и соседними текстами главы → `.cache/tour/elevenlabs/NN.mp3` и приватный manifest → offline renderer → MP4/VTT/public manifest/checksums → media/browser QA. Пользовательские файлы, скриншоты и credentials FileX провайдеру не отправляются; только заранее подготовленный текст демонстрации. API использует обычный режим сервиса, zero-retention не заявляется.

Параметры: `mp3_44100_128`, stability 0.5, similarity_boost 0.75, style 0, speaker boost true, speed 1, seed `101100 + номер главы` (не гарантия идентичной генерации). В туре 19 глав / 4647 Unicode-символов; default `--max-characters 6000` ограничивает суммарный текст всех глав даже при sample. Это лимит текста, а не гарантия суммы списания или наличия квоты.

### Возобновление после решения пользователя

Текущий кэш содержит известный отказ 402. Следующие команды показаны для будущего запуска после сообщения пользователя о доступной квоте; они сами по себе не пополняют баланс:

```sh
# First sample; the explicit flag acknowledges the prior known payment rejection.
python3 scripts/tour/narrate.py --chapter 1 --retry-rejected
# After listening to the sample, reuse chapter 1 and generate remaining chapters.
python3 scripts/tour/narrate.py
# Rendering reads cached files only; it makes no provider requests.
python3 scripts/tour/render.py --narration-dir .cache/tour/elevenlabs --ffmpeg /absolute/path/to/ffmpeg
node scripts/tour/check.mjs
make docs-check
```

Для нового пустого output каталога достаточно `--chapter 1` без retry flag. `--source`, `--output`, `--voice-id`, `--model` поддерживают отдельную новую запись. Изменение текста/голоса/модели/настроек меняет plan hash и требует нового каталога; нельзя смешивать аудио разных планов или удалять кэш ради автоматического повторного списания.

### Сбои, возобновление и проверки

- Перед платным POST сохраняется `requested`. Timeout/неизвестный исход оставляет главу incomplete; автоматический повтор запрещён, даже с `--retry-rejected`. Нужно проверить фактический исход у провайдера перед любым новым запросом.
- Только известный HTTP 402 сохраняется как `rejected` и допускает явный `--retry-rejected`. Ключ/сырой provider body не печатаются в ошибках.
- Полученный ответ сохраняется до проверки alignment. Повреждённый MP3, несовпадающий текст или неверные timings останавливают сборку и не инициируют новый synthesis.
- Manifest содержит SHA-256 аудио/текста, voice/model/settings, даты и character alignment. Resume полностью готового кэша не обращается к API. `.generation.lock` запрещает параллельные генераторы одного каталога; после аварии stale lock удаляется только после проверки отсутствия живого процесса. При этом `requested` marker всё равно защищает от автоматического повтора.
- Renderer проверяет все главы до обработки, использует исходный character alignment для VTT/ASS, проверяет границы по декодированной длительности, нормализует внешнее аудио до -16 LUFS / -1.5 dBTP и записывает источник/тайминги в public manifest. FFmpeg и кириллический шрифт обязательны, macOS `say` для внешнего audio не требуется.
- `python3 -m unittest discover -s scripts -p test_tour_narration.py` проверяет resume, лимит/voice validation, конкурентный запуск, неоднозначный сбой, 402, stale/corrupt audio, invalid alignment и приватность файла. Тесты используют synthetic responses и никогда не вызывают платный API; входят в make test/test-race/check.

Проверено 2026-10-11: 8 unit/fault tests PASS; отдельный offline fixture собрал и полностью декодировал 19 глав с MP3 и character timings, сохранил 19 VTT cues. Готовая публичная галерея отдельно прошла 50 checksums / 44 screens / 19 chapters / a11y / playback. Тестовое видео находится только в ignored `.cache/consolidation/offline-render/` и не заменяет тур.

Offline fixtures доказывают работу инструментов, но не качество Ariana и не принятие реальной новой дорожки. После успешного синтеза нужно прослушать все главы, проверить переходы/субтитры и заново декодировать весь MP4; затем обновить результат съёмки, session и checksums.

## Ограничения и продолжение

Тур показывает основные пользовательские и административные маршруты, но не все 107 API операций и fault-injection варианты. Шифрование, race/fencing и security-инварианты подтверждаются отдельными [тестами](testing.md), а не изображениями.

SMTP/S3/webhook/миграция/ротация/forensic показаны формами, без реальных интеграций. Мобильные снимки используют viewport 390 × 844 в Chromium Linux. 5 partial MUST, 14 pending decisions и запрет самостоятельного production deployment сохранены. Временный PIN и ссылки видимы только на уничтоженном синтетическом стенде и не должны заменяться рабочими данными при пересъёмке.

Следующий шаг при изменении UI — обновить сценарий и переснять тур; при подготовке внешнего окружения — выполнить [план приёмки](environment-acceptance.md) независимо от тура. Опубликованная озвучка использует macOS Milena; внешний ElevenLabs путь подготовлен, его реальная приёмка отложена после 402.
