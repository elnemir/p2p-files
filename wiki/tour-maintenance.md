# Обновление фото- и видеотура

## Контекст и текущая поставка
Тур знакомит с реальным FileX без подготовки окружения. [Галерея](tour/index.html), [скрин-тур](screen-tour.md) и [единый MP4](tour/filex-tour.mp4) находятся в Git. Для передачи другому человеку достаточно одного MP4: русская озвучка, вшитые заголовки/субтитры и главы уже внутри.

Съёмка 2026-10-12: **48 PNG, 20 глав, 79 VTT cues**, видео **355,96 s (5:56)**, **9 955 451 bytes**, H.264 1440×1080 / 25 fps, AAC mono 48 kHz. SHA-256 MP4: `8457a74f51773997816ef587e6a06f2ca757ab9bf5621e11fa7bf9efc58e133a`. Полный список — [SHA256SUMS](tour/SHA256SUMS), параметры и таймкоды — [manifest](tour/manifest.json).

Глава 3 показывает инструкции Finder/Windows/GNOME, отдельный credential, настоящий PUT и готовый файл в списке. Это запись веб-интерфейса; реальные macOS/GVfs backend проверены [отдельными тестами](webdav-upload.md), Windows ожидает стенда. Настройки Windows в кадре не выдаются за запуск на Windows.

## Входы и поток сборки
`scripts/tour/capture.py` → случайный локальный Compose project → реальные Chromium действия из `capture.mjs` → PNG в `wiki/tour/screenshots`, raw videos и capture.json в ignored `.cache/tour/raw` → проверенный narration cache → `render.py` → MP4/VTT/manifest/HTML/Markdown/SHA256SUMS → визуальная и media/browser QA.

Capture использует синтетические аккаунты/документы, собственные случайные credentials/KEK и pinned Playwright Linux image. Он не читает operator `.env`, отказывается занимать чужой порт 8090 и удаляет только свой project/volumes/browser в finally. Пароли остаются masked, URL из кадров относятся к уже удалённому стенду. PNG не ретушируются. Mobile — viewport emulation, не реальные телефоны.

Сценарий реально выполняет upload/publish/ZIP/QR/PDF/JSON/CSV, backup/restore/restart/reconcile/reopen, public verification и повторный отказ recipient после completed. S3/SMTP/webhooks/provider migration/rotation/forensic показаны как формы без внешних операций. Наблюдение DEV-05 о старом recipient UI после refresh остаётся [backlog](implementation-plan.md).

## Съёмка
Нужны Docker, зависимости `npm ci` на поддерживаемом Node 24, Python, свободный локальный 8090. Никакой production:

```sh
python3 scripts/tour/capture.py
```

Успешный capture.json имеет `passed: true`, 20 chapters, 48 screens и assertions; неполный сценарий renderer отвергает. В каждой главе сохраняются начало/конец raw video и точный narration text. Ошибка остановит сценарий: не подменять её пустой картинкой или ручным success.

Manifest связывает кадры с applicationSourceSHA256: sorted unique `git ls-files --cached --others --exclude-standard` по apps/cmd/internal/migrations/deploy/go.mod/go.sum/package*.json, hash `path + NUL + bytes + NUL`. Новые untracked source files включены. Текущая съёмка: `aa0afcdedbd0d1f2dfebca3a2b84fd44598ac4a87dce6cb83f4182ab2c7acd3d`, baseline `28ab4f3`. Повторная съёмка в 13:58 UTC включает окончательный runtime/UI и нейтральную подсказку подключения без привязки к Mac/8090. Позже изменён только generated TypeScript comment с названием WebDAV operation (upload-only → owner draft); это меняет application source hash, но не код исполнения и не интерфейс. Полная регрессия использует отдельный repository source hash в [session](session.md), включающий scripts/contracts/media; эти два hash имеют разные scope и не должны сравниваться как одинаковые.

## Озвучка Liza и безопасный кеш
Голос ElevenLabs Liza, ID `KzqxCy7zSSePwgb1Cz0Q`, `eleven_multilingual_v2`; speed 1, stability 0.5, similarity 0.75, style 0, speaker boost true. Для нового тура повторно использованы 19 неизменённых глав (4647 символов) по exact text + SHA-256. Через открытую владельцем вкладку Chrome синтезирована только новая глава WebDAV: 454 символа, 5324 → 4870 credits. Всего текст текущего тура — 5101 символ. Повторная генерация после зависшего browser download не выполнялась: скачанный MP3 найден локально и проверен.

Рабочий cache: `.cache/tour/elevenlabs-web-Liza-native`, исходный неизменённый — `.cache/tour/elevenlabs-web-Liza`. Ключи API, cookies, приватные raw journals и отдельные MP3 не коммитятся. Внешнему TTS передаётся только согласованный текст демонстрации, не файлы пользователей/скриншоты/пароли. Предыдущий API payment_required не объявляется исправленным успешной web-генерацией.

В `progress.json` нужны `provider: ElevenLabs`, `channel: Chrome web UI`, voiceId/voiceName/modelId/voiceSettings и ровно один entry для каждой главы: number, text, textSha256, status=downloaded, file=NN.mp3, audioSha256. `load_web_narration` отвергает missing/extra/duplicate chapter, изменённый текст/хеш, неполный status, чужой путь и symlink. При вставке главы перенумеровать кеш по совпадению текста, не по старому номеру. Старый журнал не переписывать: создать новый cache.

API-вариант сохранён в `scripts/tour/narrate.py` (план/лимит символов, model/voice preflight, with-timestamps, validated resume cache); доступная API-квота проверяется отдельно. `--narration-dir` принимает manifest API с character alignment, `--web-narration-dir` — журнал скачанных UI MP3. Эти форматы не взаимозаменяемы. Без внешнего cache renderer может использовать локальную macOS Milena, но текущая поставка озвучена Liza.

## Рендер и обновление галереи
Нужен FFmpeg с libx264/libass, читаемый Arial и полный capture. Пример:

```sh
python3 scripts/tour/render.py --ffmpeg /absolute/path/to/ffmpeg \
  --web-narration-dir .cache/tour/elevenlabs-web-Liza-native
```

Renderer сокращает ожидания и подстраивает скорость реальных записей под звук; новых кадров интерфейса не рисует. Главы собираются в один faststart MP4, captions также сохраняются отдельным VTT. Для web MP3 timings оценены по длине текста/длительности главы: word-perfect/provider alignment не заявляется. Loudness нормализуется по главам с headroom; границы звука проверяются на итоговом файле.

После полного успешного capture renderer удаляет устаревшие нумерованные PNG, которых нет в новом manifest. Source `gallery.html/css/js` находится в scripts/tour. Если меняется только оболочка галереи:

```sh
python3 scripts/tour/render.py --gallery-only
```

Эта команда синхронизирует HTML/Markdown/checksums, не меняет PNG/MP4. Все generated файлы обновлять генератором. Галерея не требует сборки или сервера: `index.html` открывается локально. Manifest text вставляется через textContent, `<` в embedded JSON экранируется, screenshot URL ограничен `screenshots/NN-name.png`; внешние/data/javascript/traversal пути отвергаются.

## Проверка результата
```sh
FILEX_TOUR_BROWSER_CHANNEL=chrome node scripts/tour/check.mjs
ffmpeg -v error -i wiki/tour/filex-tour.mp4 -f null -
python3 scripts/check_docs.py
```

Browser QA проверяет все 54 checksum entries, PNG dimensions, 48 карточек, 20 глав, поиск/пустое состояние/filter/lightbox/keyboard/focus, desktop/mobile reflow, axe, playback/seek и 7 опасных URL. Full decode не должен содержать ошибок. Визуально просмотреть все PNG на contact sheets и ключевые оригиналы, затем видео в начале, новой главе, середине и конце. Проверять читаемость/обрезание, соответствие narration текущему сценарию, отсутствие секретов и clipping звука.

Текущий результат: все 48 кадров просмотрены, Windows status читаем, подписи/субтитры не перекрывают UI; full decode PASS, gallery/axe/playback PASS. Loudness −16,8 LUFS, LRA 2,9 LU, true peak −1,0 dBFS. Raw evidence: `.cache/tour/qa/screens-*.png`, `.cache/tour/qa/native-video-*.png`, `.cache/webdav-native-tour-{capture,render,qa,decode,volume}.log`; hash/media metadata — manifest/SHA256SUMS. История предыдущего тура — [сессия merge](sessions/2026-10-12-tour-merged.md).

Тур — демонстрация локального функционала. Он не закрывает 5 partial MUST, 14 external decisions, Windows acceptance или production deployment.
