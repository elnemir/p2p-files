# White labeling: локальная квалификация 2026-10-13

## Контекст и граница
Настраиваемый бренд установки реализован в `codex/white-label`, база main `cf4392d65a597e9d5281a666b1cdc503f6dcec7b`. Результаты ниже относятся к незакоммиченному кандидату с source SHA-256 `e100d8b6197331d7273194e3251d006453fdc806fc0eae03282a0145d7c0f289`. Commit/push, GitHub CI и production deployment в этой задаче не выполнялись. Инструкция — [white labeling](../white-label.md), решение — [ADR-013](../adr/013-white-label.md).

## Выполненные gates
Все 13 записей `.cache/evidence/gates/<gate>.json` имеют `passed=true`, `sourceUnchanged=true`, `exitCode=0` и одинаковый hash. Полный последовательный batch и raw logs — `.cache/white-label/gates-final-progress.log`, `.cache/white-label/gate-<gate>.log`. Markdown исключён из fingerprint согласно acceptance.py; после публикации результатов docs-check запускается отдельно.

| Gate | Результат | Время, s |
|---|---|---:|
| check | PASS | 33.5 |
| integration | PASS | 182.0 |
| dependencies | PASS | 5.2 |
| source | PASS | 2.6 |
| licenses | PASS | 3.4 |
| images | PASS | 7.8 |
| fuzz | PASS | 90.6 |
| browser | PASS | 434.2 |
| ingress | PASS | 14.4 |
| upgrade | PASS | 20.8 |
| performance | PASS | 32.8 |
| compatibility | PASS | 0.1 |
| verifier | PASS | 4.4 |

## Что подтверждено
- 56 PostgreSQL integration cases с race, **0 skip/FAIL**, включая rclone. USER/bearer/Origin/CSRF/reauth, текущие роль/epoch, concurrent CAS, audit rollback, immutable report/incident presentation, branding backup/restore и ошибки SMTP.
- 40 browser cases: Chromium, Firefox, WebKit, mobile Chromium, mobile WebKit — по 8, **0 skipped/unexpected/flaky**. Final run без обновления baselines. Проверены preview/apply/reset/conflict, три theme modes, default inheritance, logos/favicon/icon style, 320px/long name, axe, SSR без JavaScript, публичные страницы и conditional HTTP.
- Десять RU-light/EN-dark baselines пересозданы с явным ожиданием темы/языка и просмотрены; [редактор](../images/branding-editor.png), [320px](../images/branding-mobile.png), [публичная страница](../images/branding-public.png) просмотрены отдельно. PDF с длинным кириллическим именем и normalized raster отрендерен в PNG и визуально проверен; обрезания/перекрытия не обнаружены.
- 111 OpenAPI operations, AsyncAPI/schema/type generation sync; compatibility к прежнему baseline. 14 contract/harness tests PASS. SQL schema 18, controlled upgrade 16→18, checksum/unknown migration/wrong KEK/downgrade/old-backup guards и explicit reopen PASS.
- Пять fuzz campaigns PASS. Performance — **32 MiB / 2000 rows smoke**, включая synthetic S3/DR; это не release-size или target-host load.
- npm audit: 0 vulnerabilities. Govulncheck: 0 symbol/imported-package vulnerabilities; один module-only advisory для невызываемого кода остаётся в raw report. Source и пять container scans: HIGH/CRITICAL=0; Gitleaks worktree/history PASS. License inventory не заменяет owner acceptance.
- Verifier cross-build для пяти targets PASS; это не запуск binary на каждой реальной ОС и не публикация release.

## Найденные и устранённые дефекты
1. Длинное название расширяло header из-за inherited nowrap: добавлены перенос и мобильная компоновка. Dropdown names заданы явно; theme/locale disabled до hydration, чтобы ранний выбор не терялся.
2. В новом SMTP branch shadowed error мог теряться: исправлено и подтверждено delivery failure regression. Максимальное Unicode имя раздувало Q-encoded Subject за 998 bytes: TLS fixture RED/GREEN, B-encoding сохраняет исходную строку.
3. Старая preflight fixture удаляла 0017 при существующей 0018: guard правильно обнаруживал gap. Теперь тест выбирает последнюю ledger migration, проверяя pending и отсутствие side effects.
4. WebKit не поддерживает route.fulfill(304): conditional branding GET оставлен native без replay; browser явно проверяет ETag/304/no-body/stale-200, unit tests ограничивают исключение только этим GET. Остальные API responses продолжают проходить прежнюю проверку.
5. Длинная ADMIN страница превышала WebKit screenshot pixel limit на Retina: evidence снимается в CSS pixels, редактор — отдельным элементом. Assertions и timeout limits не ослаблялись.

Первичные FAIL сохранены: `integration-preflight-fixture-failure.log`, `integration-parallel-timeout.log`, `browser-conditional-failure.log`, `smtp-unicode-red.log` и browser traces в `.cache/white-label/`. Параллельный тяжёлый PG/browser прогон исчерпал общий 10-minute Go timeout; финальный последовательный прогон прошёл без изменения timeout/gates.

## Ограничения и продолжение
Внешняя приёмка, реальные устройства/assistive technology, независимый security/license review и production guard сохраняются. Windows native WebDAV по-прежнему **ожидает стенда**. Фото/видеотур предыдущей версии не объявляется новым branding evidence. При delivery выполнить commit/push и проверить exact-head GitHub checks отдельно; локальный PASS их не заменяет.
