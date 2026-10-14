# Обновление фото- и видеотура

## Контекст и текущая поставка
Тур знакомит с реальным FileX без подготовки окружения. [Галерея](tour/index.html), [скрин-тур](screen-tour.md) и [единый MP4](tour/filex-tour.mp4) находятся в репозитории. Для передачи другому человеку достаточно MP4: русская озвучка, заголовки, субтитры и главы уже внутри.

Съёмка **2026-10-14: 71 PNG, 26 глав, 104 VTT cues, 476,24 s (7:56)**. Видео — H.264 1440×1080 / 25 fps, AAC mono 48 kHz, faststart. Точные bytes и SHA-256 — в [evidence](history/tour-refresh-evidence-2026-10-14.md) и [SHA256SUMS](tour/SHA256SUMS); таймкоды, тексты и граница источника — в [manifest](tour/manifest.json).

Прежние 48 кадров / 20 глав не содержали white labeling, корпоративный вход и полный UX из PR #26. Теперь показаны также настройки draft/validate/apply, формы OIDC/SAML/LDAP, локальный SAML-вход и создание USER, применение бренда, история/восстановление отчёта, terminal recipient и мобильные формы. Исторические записи остаются в Git и [архиве сессии](sessions/2026-10-12-tour-merged.md).

## Что выполнено, а что только показано
На отдельном Compose реально выполнены upload/publish/ZIP/QR/PDF/JSON/CSV, WebDAV PUT, отчёт trash/restore с проверкой неизменного JSON, settings draft/validate/apply с проверкой активной ревизии, branding apply/reset, backup/restore/restart/reconcile/reopen и public verification. После ZIP показаны terminal экран и отдельный отказ повторного входа по прежней ссылке.

SAML callback и подпись настоящие; одноразовый Go signer использует синтетическую identity. Видеостраница «Локальный учебный IdP» явно обозначает fixture. Это не UI Keycloak/Entra/Google. OIDC/LDAP показаны как несохранённые конфигурационные примеры. SMTP/S3/webhooks/provider migration/rotation/forensic — формы без внешних операций. Новая брендированная почта/PDF описана в [white labeling](white-label.md), но её внешняя доставка в этом видео не выполнялась.

WebDAV-глава показывает инструкции Finder/Windows/GNOME, отдельный credential, настоящий PUT и готовый файл в списке. Реальные macOS/GVfs backend проверялись [отдельно](webdav-upload.md); Windows **ожидает стенда**. Настройки Windows в кадре не являются запуском Windows-клиента. Mobile — эмуляция viewport Chromium, не физический телефон.

## Входы и поток сборки
`capture.py` → случайный локальный Compose → реальные действия `capture.mjs` → PNG/raw videos/capture.json в ignored `.cache/tour/raw` → сверка с `plan.json` → проверенный narration cache → `render.py` → `wiki/tour` и `wiki/screen-tour.md` → визуальная, audio и browser QA.

Capture создаёт синтетические аккаунты/документы, собственные случайные credentials/KEK и pinned Playwright Linux container. Operator `.env` не читается. Занятый чужой порт 8090 отклоняется; finally удаляет только свой project/volumes/browser и останавливает signer. Пароли masked, URL из кадров относятся к удалённому стенду. PNG не ретушируются. Новый capture не перезаписывает опубликованные PNG до рендера.

`plan.json` — отдельно проверяемый порядок глав и screenshot IDs. При расширении сценария обновить его осознанно, не строить ожидаемый набор из результата прогона. `validate.py` требует passed=true, точное число/порядок глав и кадров, непустой текст, корректные временные границы, локальные PNG paths и совпадение capturePlanSHA256. Неполную запись нельзя «починить» ручным passed=true. При вставке главы озвучка связывается с точным текстом, а не прежним номером.

## Съёмка и граница исходников
Нужны Docker, `npm ci` на Node 26 из `.nvmrc`, Go из go.mod для SAML signer, Python и свободный локальный 8090. Команда не развёртывает production:

```sh
python3 scripts/tour/capture.py
```

На macOS можно запускать через `caffeinate -i`, чтобы компьютер не засыпал. Успешный capture содержит 26 глав, 71 кадр и 16 assertions. Для каждой главы сохраняются начало/конец raw video и точный текст. Ошибка останавливает сценарий; пустые картинки и искусственный success запрещены.

Текущий runtime: baseline `28a6b4a0aac2ed23c7699ec35e468f59255949b7`, applicationSourceSHA256 `76574e49e37a5fdfd1bbd7e935c9e22691222070a912f1260ca058e3d53bcc42`. Scope: sorted unique `git ls-files --cached --others --exclude-standard` по apps/cmd/internal/migrations/deploy/go.mod/go.sum/package*.json, hash `path + NUL + bytes + NUL`. Он включает untracked runtime files, но не tour tooling и документацию. Repository acceptance fingerprint имеет другой scope; исторические 13 gates не являются проверкой нового media/tooling diff.

## Озвучка Liza и безопасный кеш
Голос ElevenLabs Liza, ID `KzqxCy7zSSePwgb1Cz0Q`, модель `eleven_multilingual_v2`; speed 1, stability 0.5, similarity 0.75, style 0, speaker boost true. Текст — 6915 символов. Переиспользованы 18 неизменённых глав (4529 символов) по exact text + SHA-256. В Chrome синтезированы восемь новых фрагментов (2386 символов): 4870 → 2484 credits. Последний MP3 получен из History после предложения подписки, без повторной генерации и оплаты плана.

Новый приватный cache — `.cache/tour/elevenlabs-web-Liza-20261014`; прежний `.cache/tour/elevenlabs-web-Liza-native` сохранён. API keys, cookies, raw journals и отдельные MP3 не коммитятся. TTS получает только текст демонстрации, не файлы пользователей/скриншоты/пароли. Успешный web synthesis не доказывает наличие API-квоты; исторический API payment_required в этой задаче не перепроверялся.

`progress.json` содержит provider=ElevenLabs, channel=Chrome web UI, voiceId/voiceName/modelId/voiceSettings и ровно одну запись для каждой главы: number, text, textSha256, status=downloaded, file=NN.mp3, audioSha256. `load_web_narration` отвергает missing/extra/duplicate chapter, изменённый текст/хеш, неполный status, чужой путь и symlink. При вставке главы переносить запись по совпадению текста, не старому номеру. Создавать новый cache, не переписывать прежний журнал.

После неопределённого результата синтеза сначала проверить History и Downloads. Не повторять оплачиваемый запрос автоматически. Скачанный файл должен полностью декодироваться, а его checksum — совпасть с журналом. `narrate.py` сохраняет отдельный API-вариант с preflight, лимитом символов и with-timestamps. `--narration-dir` принимает API manifest с character alignment; `--web-narration-dir` — журнал UI MP3. Без внешнего cache остаётся локальная Milena, но текущая поставка целиком Liza.

## Рендер и публикация артефактов
Нужен FFmpeg с libx264/libass и читаемый Arial. После полного capture и audio validation:

```sh
python3 scripts/tour/render.py --ffmpeg /absolute/path/to/ffmpeg \
  --web-narration-dir .cache/tour/elevenlabs-web-Liza-20261014
```

Renderer сокращает ожидания и подстраивает скорость записанного UI под речь. Он не рисует интерфейс. Заголовки/субтитры находятся за границами UI. Web MP3 не содержит проверенного provider alignment: cue timing оценивается по длине текста и длительности речи, точная синхронизация каждого слова не заявляется. Loudness нормализуется по главам; limiter сохраняет запас для AAC. Проверять надо декодированный итоговый файл, а не только настройки фильтра.

После успешного рендера глав копируются оригинальные PNG и удаляются устаревшие картинки вне manifest. Source шаблоны находятся в scripts/tour; число кадров, глав и дата в HTML берутся из manifest. Для изменения только оболочки:

```sh
python3 scripts/tour/render.py --gallery-only
```

Команда обновляет HTML/Markdown/checksums, сохраняя видео и PNG. Generated files вручную не редактировать. Галерея открывается локально без сборки; для распространения галереи передавать весь `wiki/tour`. Manifest text вставляется через textContent, `<` в JSON экранируется; внешние/data/javascript/traversal screenshot URLs отвергаются.

## Проверка и продолжение
```sh
python3 -m unittest discover -s scripts -p 'test_tour_*.py'
FILEX_TOUR_BROWSER_CHANNEL=chrome node scripts/tour/check.mjs
ffmpeg -v error -i wiki/tour/filex-tour.mp4 -f null -
ffmpeg -hide_banner -i wiki/tour/filex-tour.mp4 -af ebur128=peak=true -f null -
make check
```

Browser QA сверяет reviewed plan, 77 checksum entries, PNG dimensions, 71 карточку, 26 глав, поиск/empty/filter/lightbox/keyboard/focus, desktop/mobile reflow, axe, playback/seek и семь опасных URL. Unit pack проверяет неполную/устаревшую съёмку и ошибки narration cache. Все PNG надо визуально просмотреть; для длинных форм — также оригиналы. Для видео проверить начало, новые разделы, конец, читаемость captions и пиковый уровень после AAC. Новая версия после любой правки media получает новую проверку/hash.

Результаты этого обновления, включая выявленный audio peak и его исправление, — [evidence](history/tour-refresh-evidence-2026-10-14.md). Скрипты/медиа изменены, runtime/API/env/migrations — нет. Следующая работа: переснимать затронутые сценарии после UI изменений, отдельно выполнить Windows и внешнюю IdP/device/release приёмку. Этот тур не закрывает 5 partial MUST, 14 external decisions или production deployment.
