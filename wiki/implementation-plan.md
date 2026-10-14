# План продолжения разработки

## Контекст и правила использования

План продолжает [roadmap](roadmap.md) после технической реализации проверок фазы 7. Завершены документационные шаги DOC-01 и DOC-02; код будущих задач здесь не объявляется выполненным. Приоритет P0 блокирует выпуск, P1 отвечает за сопровождение. Статус ready означает готовность начать локальную работу, external-blocked — отсутствие необходимого внешнего входа. Исполнители указаны как роли, персональные назначения и даты ещё не согласованы.

## Backlog и зависимости

| ID | Приоритет / статус | Роль | Результат | Зависимости |
|---|---|---|---|---|
| DOC-01 | P0 / completed локально | Инженер | Вся документация в wiki, актуальные связи с кодом, воспроизводимые инструкции и этот план | [Сессия переноса](sessions/2026-10-11-documentation-migration.md) |
| DOC-02 | P1 / completed локально | Инженер | 48 PNG, видеотур 5:56 / 20 глав (native WebDAV), озвучка Liza через Chrome, субтитры, галерея и повторяемый capture/QA | [Тур](screen-tour.md), [обслуживание](tour-maintenance.md), запись отдельного стенда |
| REPO-01 | P1 / completed | Владелец + инженер | GitHub/main/Actions/security, templates, dependency updates и воспроизводимый audit | [Настройки и приёмка](repository-governance.md); фактическое применение/CI в session |
| DEV-01 | P0 / ready для design | Backend + security reviewer + владелец | Reviewed механизм тестового/production допуска, затем реализация и негативные тесты | Сохранить существующий запрет; применение допуска зависит от EXT и REL |
| DEV-02 | P1 / baseline qualified locally; CI в PR #19 | Инженер сопровождения | Поддерживаемые lint/Actions/toolchain/image pins с воспроизводимыми scans/tests | Совместимые peer dependencies; текущий baseline |
| DEV-03 | P0 / ready для пакета review | Release engineer + владелец/reviewer | Состав фактической поставки, license notices/решения, текст proprietary license после решения владельца | Точные image digests/SBOM; EXT-07 закрывает решение |
| DEV-04 | P0 / ready для design и tooling | Release engineer + оператор | Долговременное сохранение и восстановление evidence для exact candidate | Хранилище выбирает оператор; независимость от 14-дневных CI artifacts |
| DEV-05 | P1 / completed локально | Frontend + QA | Terminal/revoke/expiry очищают controls; временная сеть и незавершённая выдача сохраняют resume | [UX и доказательства](history/product-ux-evidence-2026-10-13.md), 60 browser / 62 PG cases PASS |
| DEV-06 | P1 / completed локально; Windows pending | Backend + frontend + QA | Загрузка через WebDAV в черновик, отдельные credentials, общий encrypted pipeline, UI и protocol/client tests | Выбор владельца 2026-10-12; [ADR-011](adr/011-native-webdav.md), [руководство](webdav-upload.md), [session](session.md) |
| EXT-01 | P0 / external-blocked | Оператор | VPS/IPv4/DNS/ACME/host firewall/backups acceptance | Карточка окружения; approved путь запуска DEV-01 |
| EXT-02 | P0 / external-blocked | Инженер + оператор | Настоящие SMTP/S3: TLS, доставка, retry, object semantics | Аккаунты/разрешённые тестовые адреса, EXT-01 |
| EXT-03 | P0 / external-blocked | Оператор + security reviewer | Независимые backups/payload strategy/KEK/checkpoints доступны при потере VPS | Отдельное хранилище и доступ; EXT-01 |
| EXT-04 | P0 / external-blocked | QA + оператор | Target load/DR и измерения RPO/RTO/ёмкости | EXT-01–03, согласованный профиль |
| EXT-05 | P0 / нужны устройства/reviewer | QA accessibility | Реальные Safari/mobile, keyboard/AT/zoom/error сценарии и исправления | Согласованная device/AT matrix и кандидат |
| EXT-06 | P0 / нужен независимый reviewer | Независимый security reviewer | Findings по коду/crypto/host/supply-chain, retest исправлений | Код доступен сейчас; финально EXT-01–04 и DEV-01 |
| EXT-07 | P0 / нужны решения владельца | Владелец + reviewer | Third-party licenses и каждый R-01–06 accepted с evidence либо исправление/отказ выпуска | DEV-03, EXT-04–06 |
| REL-01 | P0 / blocked зависимостями | Release engineer + владелец | Exact-candidate release package, 32 verified MUST, readiness PASS | Все обязательные DEV/EXT, свежие gates, отдельное разрешение deploy |

## DEV-01: устранить неопределённость production допуска

**Проблема:** `internal/config/config.go` допускает только development/test; генератор пишет APP_ENV=production, а `make release-check` не меняет код. Простое удаление проверки до приёмки или использование development на production не закрывает OPS-005.

**Работа:** описать ADR с последовательностью изолированной проверки на целевом окружении, независимой приёмки, финальной квалификации изменённого кандидата и явного допуска. Определить доверенную сторону, формат/границы доказательства, поведение при missing/stale/mismatched evidence; выбрать механизм после review. Не считать JSON с произвольным reviewer цифровым подтверждением. Затем изменить config/bootstrap/generator/tests в отдельном срезе.

**Готовность:** согласован review; default deny сохранён без допуска; неверный origin/proxy/config/кандидат отклоняется; normal development сохраняется; integration/ingress/upgrade/security проходят; финальный изменённый source hash получает повторные gates. Ни код, ни скрипт не выполняют автоматический deploy. Requirements: SEC-005, OPS-005, NFR-003–004.

## DEV-02: сопровождение toolchain и образов

**Исходная проблема:** ESLint 9.39.5 был unsupported; JSX-a11y 6.10.2 ограничивал peer compatibility. Actions обновляются с сохранением SHA pins; runtime образов и scanners требует повторных scans при advisories. В объединении [PR #16](https://github.com/elnemir/p2p-files/pull/16) 2026-10-11 включены пять Actions updates, Alpine 3.24, Go 1.27.2 и React types 19.3.0; [таблица PR, blockers и статус](branch-consolidation.md). В PR #16 Node 25 EOL, types Node 26 и ESLint 10 не включались; Redis 8 требует отдельного лицензионного решения. DEV-02 не закрывается переносом восьми patches. Продолжение 2026-10-12: [совместимый ESLint 10, a11y migration, Node types и dependency guards](dependency-remediation.md) реализованы; все 13 local gates PASS; exact-commit CI и merge фиксируются в [PR #19](https://github.com/elnemir/p2p-files/pull/19). Продолжение 2026-10-13 по явной команде владельца: [Node 26 / ADR-012](adr/012-node26-runtime.md), согласованный runtime/types/CI, fresh gates и PR #22 поверх merged WebDAV. Сопровождение будущих обновлений остаётся регулярной инженерной работой.

**Работа:** проверить совместимую комбинацию по первичным release/peer данным на момент исполнения; обновлять небольшими срезами, не использовать force/legacy-peer-deps. Для Actions проверить permissions, full Git history, runtime и artifact retention. Для пяти images сохранить pins, non-root/read-only/network policy и scanner fail-closed.

**Готовность:** npm ci без обхода peers; lint/typecheck/contracts/build и browser baseline review; source/dependency/image/license scans; оба CI jobs exact commit успешны. Смена основного стека не входит в задачу. Requirements: NFR-003–004, OPS-005.

## DEV-03: состав поставки и лицензии

**Проблема:** inventory не равен одобрению. В корне нет LICENSE; proprietary политика зафиксирована, конкретный текст/правообладатель не предоставлены. Список optional npm не равен составу каждого image.

**Работа:** связать точные digests и verifier platforms с SBOM и реальными runtime files; отдельно отметить dev/optional/fonts/OS packages. Подготовить notices и вопросы владельцу, варианты исполнения обязательств/замены зависимостей для review. Не сочинять юридическое разрешение и не переводить third-party-licenses в accepted самостоятельно.

**Готовность:** каждый поставляемый компонент сопоставлен с источником/версией/лицензией; владелец принял решение и предоставил основание/текст; выбранные изменения проходят scans и функциональные gates. При отказе — замена/исключение компонента с проверкой, а не исключение MUST. Requirements: NFR-004.

## DEV-04: долговременный пакет evidence

**Проблема:** CI artifacts удаляются через 14 дней, `.cache` локальна. Текущий review bundle содержит binaries/checksums/build metadata/release notes, но не является независимо хранимым полным пакетом приёмки.

**Работа:** определить версионируемую структуру пакета commit/source hash → 13 gate records/logs → фактические image IDs/digests/SBOM → полный load profile → human review references → verifier checksums. Объединять два CI jobs без перезаписи конфликтующих records. Не включать secrets, реальные payload и session traces. Доступ/retention/место задаёт оператор; механизм подписи, если потребуется, отдельно reviewed, а не расширяет Ed25519 payload contract молча.

**Готовность:** пакет воспроизводимо собирается, checksum проверяется после скачивания, missing/conflicting/stale candidate evidence отвергается; восстановление пакета проверено вне исходной рабочей папки. External evidence остаётся проверкой reviewer/owner, не синтетическим accepted JSON. Requirements: NFR-003–004, SEC-002.

## DEV-05: завершение выдачи на странице получателя

**Контекст и проблема:** при записи тура 2026-10-11 после полного ZIP передача переходила в `completed`; `GET /recipient/files` возвращал 401. Прежний UI показывал общий отказ, но сохранял старый список `ready` и кнопки; сервер повторную выдачу уже запрещал.

**Решение:** recipient API guard переводит локальное представление в общее недоступное состояние при 401/403/404/410. Файлы/download URL очищаются, поздние ответы прежней session generation отбрасываются. Временная сеть/5xx сохраняет доступ и resume; явный новый token открывает новую ссылку независимо от прежней cookie. [Поток, примеры, ограничения](product-ux.md).

**Приёмка:** completed, реальный revoke, natural expiry, сетевая ошибка списка, отмена автоматического content GET с дальнейшим resume и удержанный старый ответ проверены на пяти browser profiles. Настоящие interrupted-stream/Range/atomic ZIP invariants проверены PG/race. Общая матрица 60/60 browser и 62/62 integration, все 13 локальных gates PASS на [source из evidence](history/product-ux-evidence-2026-10-13.md). Причина denial не раскрывается; UI не обещает сохранность файла на диске получателя. Requirements: FR-001, FR-006–007. Внешняя AT/device приёмка остаётся EXT-05.

## EXT и REL: условия закрытия

Пошаговые действия, входы, evidence и mapping всех 14 decision IDs находятся в [environment acceptance](environment-acceptance.md). REL-01 повторяет [release checklist](release-checklist.md) на окончательном неизменном кандидате: 13 gates моложе 24 h, полный browser и ≥2048 MiB/≥20000 rows, 32 verified MUST, внешние решения с reviewer/evidence, clean Git. Выпуск/tag и production deployment не выполняются этой документационной задачей.

## Размер шага и передача работы

Каждый DEV/EXT разбивается на проверяемые изменения по [workflow](development-workflow.md): проблема → решение → контракт/data flow → негативный тест → evidence → wiki/session. Не закрывать карточку по одному design или пустому stub, если в результате нужна реализация. Для DOC-01 финальные проверки и статус — [сессия переноса](sessions/2026-10-11-documentation-migration.md); текущая работа — [session](session.md).

## DEV-06: native WebDAV и следующий внешний шаг
Реализованы owner draft read/Range, conditional atomic overwrite, durable exclusive locks, properties/timestamps и hidden encrypted staging/cleanup. macOS mount_webdav и Linux GVfs проверяются настоящими backend; Windows **ожидает стенда**, скрипт и приёмка готовы в [руководстве](webdav-upload.md). Дальше: запуск PowerShell на согласованном HTTPS стенде, server-created read, реальные copy/replace/rename/delete и отзыв/hold/publish. Результат связать с Windows build и exact server source. Отсутствующий стенд не закрывается HTTP fixtures. Office/KDE/davfs2 и production deployment не включаются автоматически.

## Настраиваемый брендинг — 2026-10-13

Вертикальный срез: [ADR-013](adr/013-white-label.md), [инструкция и data flow](white-label.md), migration 0018, API/SSR/ADMIN preview, письма/PDF и backup restore. Автоматическая приёмка — negative/concurrency/fault + пять browser profiles; фактический source-bound статус в [session](session.md). Ручная приёмка фирменных изображений и реальных устройств остаётся отдельной от автоматизированной проверки. Внешние release blockers не снимаются этим срезом.

## Внешняя аутентификация — 2026-10-13
[ADR-014](adr/014-federated-auth.md) и [руководство](federated-auth.md): migration 0019, external identity authority, API/UI, регистрация/link/reauth, protocol/PG/browser тесты и backup/key lifecycle. Статус квалификации — [session](session.md). Следующий внешний шаг: реальные Keycloak/Entra/Workspace/AD/OpenLDAP tenants, claim mappings, CA и сценарии ротации/отзыва; внешняя приёмка не подменяется fixtures.

## UX USER/ADMIN: реализовано локально
[ADR-015](adr/015-product-ux.md) и [руководство](product-ux.md) описывают навигацию, иконки, формы Settings v1, TTL controls, reauth, приватность при смене сессии и четыре новых снимка. Старые DTO сохранены, `/editor` добавляет контекст формы, API/worker/CLI содержат стандартную IANA database. Перед публикацией ветки выполняются обязательные gates; production guard и внешние release задачи остаются прежними.
