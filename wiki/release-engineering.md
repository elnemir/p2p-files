# Проверки кандидата в релиз — фаза 7

## Контекст и проблема
FileX уже имеет локальные функциональные проверки. Они не обнаруживали уязвимые инструменты внутри базовых Docker-образов, не сравнивали браузеры и не давали проверяемую инвентаризацию лицензий. Цель этой фазы — воспроизводимые проверки кандидата, с сохранением отдельной инфраструктурной и независимой приёмки.

## Реализация и поток данных
Исходники и lockfiles → staticcheck/ESLint/TypeScript/formatting → сгенерированные из OpenAPI типы → compatibility comparison → unit/race/PG/fuzz → браузеры и fault/load/DR → container/SBOM/license/secret evidence → release review. Генератор типов: `npm run generate:types`; CI требует точного совпадения результата. Существующие контракты v1 сравниваются с commit f68fc08; изменение действующего schema/operation запрещено консервативным comparator без пересмотра baseline или major API.

Scanners Trivy 0.75.0 и Gitleaks 8.30.1 загружаются из официальных releases, архив проверяется по закреплённому SHA-256 из `scripts/security-tools.json`. Исходники наружу не отправляются. Данные vulnerability database обновляются перед сканированием. Secret scan охватывает всю Git-историю и текущие неигнорируемые файлы, отчёты полностью redacted. Единственное разрешённое совпадение — конкретный публичный FXP1 test vector, с ограничением пути и точного значения.

## Найденные проблемы и изменения
- Общие `any`-модели UI заменены типами OpenAPI; ESLint проверяет hooks и JSX accessibility, warnings также завершают gate ошибкой. Чтение отсутствующего файла при массовом переносе теперь возвращает conflict. Подтверждение сверки привязано к ID отчёта, а не сбрасывается effect после рендера.
- Skip link переводит focus в main; добавлены keyboard/error/320px/theme checks и визуальные baseline tests.
- Source dependency scan отдельно от container scan: первоначально web содержал high findings в npm tooling; PostgreSQL — в старом Go runtime `gosu`; Redis — OpenSSL; Traefik — Go/x/net. Web runtime оставляет node и standalone приложение. Инфраструктурные Dockerfiles собирают исправленные варианты; окончательное состояние подтверждается только повторным scan.
- Production generator принимает digest всех пяти runtime images явно. Теги upstream больше не подставляются автоматически вместо проверенного артефакта. Генератор ничего не развёртывает; production startup guard сохранён.

## Примеры и evidence
`make check` проверяет код/контракты/сборку. `python3 scripts/supply_chain.py source`, `licenses`, `images` сохраняют JSON и CycloneDX в `.cache/evidence/supply-chain/`. Source и images возвращают ошибку при HIGH/CRITICAL; findings не исключаются по наличию исправления. License inventory фиксирует все npm lockfile packages, включая optional платформы и dev dependencies, runtime Go libraries и bundled font.

`make e2e` запускает pinned Linux Playwright 1.64.0 container (официальный digest закреплён в harness), host runner подключается по loopback WebSocket, только loopback fixture traffic доступен через exposeNetwork. В контейнер монтируются read-only Node modules, Docker socket отсутствует, root filesystem read-only. Используется отдельная одноразовая БД для Chromium, Firefox, WebKit, mobile Chromium и mobile WebKit. WebKit не означает запуск настоящего Safari/iOS. Pixel/iPhone profiles — эмуляция, не реальные устройства. Visual baselines разделяются по OS/engine; CI и локальный harness используют одни Linux browsers/fonts, изменение принимается только после просмотра изображений. `FILEX_UPDATE_VISUALS=1` — явная команда записи baseline, обычный CI их не обновляет.

`python3 scripts/test-performance.py` по умолчанию использует 32 MiB и 2000 синтетических записей. Полный локальный профиль: `FILEX_PERF_MIB=2050 FILEX_PERF_ROWS=20000 python3 scripts/test-performance.py`. Скрипт принимает только размер своих данных, не URL чужой системы. Он создаёт отдельный Compose project, TLS S3 wire fixture, временные ключи и синтетические данные; finally удаляет только собственные volumes. Фиксируются железо Docker VM, размеры, задержки, время этапов, CPU/RAM/IO samples и OOM. Числа описывают fixture, не SLA.

## Ограничения и дальнейшая приёмка
Инвентаризация нашла LGPL libvips в optional Next.js dependencies и MPL в dev-only axe. Это не одобрение copyleft: требуется отдельное решение владельца о лицензиях и обязательствах поставки; [конкретный review inventory](third-party-review.md). OS/container licenses также находятся в SBOM и должны рассматриваться для фактического набора образов. Go module-only advisory openpgp остаётся в source report; reachability проверяется отдельно govulncheck.

Реальные DNS/ACME/firewall/host, S3/SMTP provider, независимое хранение KEK/checkpoints, целевой load/DR, реальные Safari/mobile и assistive technology, независимый security review и принятие R-01–06 нельзя подтвердить локальной автоматизацией. Release gate обязан оставить их открытыми. Production deploy возможен только по отдельной команде владельца. Исторические результаты — [evidence](history/development-evidence-2026-10-11.md), текущий diff — [session](session.md).

## Решения владельца и внешний backlog
Владелец 2026-10-11 подтвердил: «Окружение ещё не подготовлено; зафиксировать внешние блокеры». Это отражено в `contracts/release-acceptance.json`. Pending decisions не считаются отказом от MUST и не переводятся автоматически в accepted.

| Блокер | Что требуется для снятия |
|---|---|
| VPS/domain/ACME | Целевой IPv4/DNS, проверенный certificate issuance/renewal, host firewall, SSH и ресурсные ограничения. |
| Provider acceptance | Тестовые реальные SMTP/S3, сверка TLS, delivery/retry и storage inventory/conditional writes на выбранном аккаунте. |
| Backup/escrow/checkpoints | Отдельное место хранения, независимый KEK access и checkpoint anchors; проверенный доступ после потери основного VPS. |
| Target load/DR | Повторный bounded профиль на целевом железе и восстановление из внешней копии; измеренные RPO/RTO без заранее придуманных SLA. |
| Real browsers/manual WCAG | Реальные Safari/mobile и проверка assistive technology, keyboard/focus/zoom/error flows человеком. |
| Independent security | Независимые architecture/threat/crypto review, triage SBOM/advisories и письменные findings. |
| Licenses и R-01–06 | Решение владельца по фактическим third-party inventories и каждому остаточному риску. |

`python3 scripts/acceptance.py run <gate>` фиксирует source SHA-256, commit, время, профиль, exit code и неизменность исходников за время проверки. Markdown с результатами исключён из hash, чтобы запись evidence не меняла исполняемый кандидат; links/traceability отдельно проверяются `make check`. `make release-check` требует все gates на текущем hash не старше 24 часов, полный browser matrix, ≥2048 MiB/≥20000 rows, 32 verified MUST, чистый Git и все 14 внешних решений с reviewer/evidence. Это проверка полноты пакета, не цифровая подпись эксперта.

CI разделён на functional и supply-chain jobs, сохраняет evidence 14 дней. Compact CI load не удовлетворяет multi-GB release gate; нужна отдельная запись полного профиля. Evidence из разных jobs необходимо собрать в один review directory без перезаписи одноимённых gate records. SHA/checksums CLI verifier создаются для Linux amd64/arm64, macOS amd64/arm64, Windows amd64; review bundle имеет статус `unpublished_review_bundle`. Наличие bundle не создаёт release/tag и не снимает production guard.

ESLint обновлён до 10.12.0 совместно с @eslint/js 10.0.1 и JSX-a11y-x 0.2.0 от ESLint Community. Сохранены 31 прежнее a11y правило/явные параметры, добавлены behavioral fixtures и full installed peer-tree gate. Неподдерживаемый ESLint 9 и его peer blocker удалены из lockfile; обход peer/engine checks запрещён. Подробности и квалификация конкретного кандидата — [dependency remediation](dependency-remediation.md) и [session](session.md).

## Продолжение после локальных gates

Безусловный APP_ENV guard требует отдельного DEV-01 design/implementation, а не изменения pending decisions ради запуска. Долговременный полный release package — DEV-04; текущий verifier bundle его не заменяет. При переносе документации изменились scripts/CLI/AGENTS и source hash; старые 13 PASS остаются историческими. [Backlog](implementation-plan.md), [пошаговая внешняя приёмка](environment-acceptance.md), [команды и границы проверок](testing.md).
