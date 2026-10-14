# Аудит актуальности wiki

## Контекст и проблема
Срез 2026-10-12: владелец запросил штатные WebDAV клиенты, актуальную базу знаний и обновлённые туры. Upload-only тексты противоречили новым GET/LOCK/overwrite, а обзорные документы всё ещё называли 16 migrations, 9 ADR и 107 HTTP операций. Исторический [аудит переноса](history/documentation-audit-2026-10-11.md) сохранён отдельно.

## Сверка baseline → текущая реализация
| Статус | Результат и основание |
|---|---|
| Реализовано | Owner draft GET/Range, atomic PUT replacement, unknown-length encrypted rewrap, PG locks/If, persisted properties/timestamps, stages/cleanup; migration 0017, ADR-011 |
| Изменено | Полностью переписан WebDAV runbook; синхронизированы architecture/core/file delivery/data model/states/security/threat/design review, config/API/operations/recovery/upgrade, testing/map/backlog/current state/index |
| Добавлено | Скрипты фактической проверки macOS/GVfs и Windows PowerShell; 6 native server tests, If fuzz, guards REST/tus/bulk, explicit resource budgets; 4 кадра и озвученная глава WebDAV |
| Не подтверждено | Windows ожидает стенда; другие файловые менеджеры/Office, публичный TLS и external release acceptance не заявлены проверенными |
| Требует доработки | DEV-01/03/04/05 и EXT/REL из плана остаются открыты; 5 partial MUST и 14 внешних решений не закрываются этим срезом |

## Карта сверки документации
| Группа | Действие |
|---|---|
| requirements/decisions/ADR | 32 MUST сохранены; новое прямое требование описано как дополнение. ADR-010 помечен историческим, ADR-011 текущий; прежние ADR и бизнес-инварианты сохранены. |
| architecture/core/design/security/data/state/crypto | Описаны owner-versus-recipient boundary, hidden swap/lease/freeze, parent locks и unchanged FXP1; добавлены размер/CPU/storage trade-offs. |
| development/config/API/CLI/operations/ingress/upgrade | Новых env/портов нет; migration 16→17 требует baseline prepare и backup. Тестовые команды используют disposable ресурсы; production guard сохранён. CLI интерфейс не изменён. |
| testing/contracts/matrix/release | 108 REST/tus операций плюс native wire profile, 17 migrations, 11 ADR. Traceability генерируется; source-bound gates не переносятся с прежнего diff. External criteria/risk/license registry не переутверждаются. |
| overview/backlog/repository/docs audit | Текущее состояние отделено от historical evidence; задачи Windows и release имеют явные входы и критерии. Branch/dependency/governance страницы остаются историей своих GitHub проверок, не новым заявлением CI. |
| screen-tour/tour-maintenance | Все 48 кадров пересняты, 20 глав/79 cues, единый MP4 Liza; screenshots без ретуши, synthetic data. Видео отдельно от проверки native backend. |
| history/sessions | Исторические hashes/counts не переписаны; добавлены links к текущему состоянию. |

## Проверка и продолжение
`check_docs.py` обходит все Markdown, локальные ссылки и достижимость страниц; traceability проверяет 32 IDs и существующие implementation/test paths. `build_contracts.py --check` и type generation сверяют wire/schema/types. Тур имеет собственные hashes, просмотр всех кадров, полный decode и browser QA. Окончательные числа/gates/source hashes — [session](session.md); [план](implementation-plan.md) и [внешняя приёмка](environment-acceptance.md) задают продолжение. Code review внутри этой задачи не является независимым security approval.

## Дополнение: white labeling, 2026-10-13

Предыдущая таблица фиксирует аудит native WebDAV и его исторические числа. Срез white labeling: 111 API operations, 18 migrations, ADR-013; [инструкция](white-label.md), architecture/configuration/repository map, controlled upgrade/recovery, testing, requirements/threat/states/review, changelog/matrix/session синхронизированы. Добавлен FILEX_BRANDING_API_URL только для SSR web. Технические названия протоколов и исторические документы/медиа не переименовываются. Текущие результаты квалификации не выводятся из предыдущего PASS — см. [session](session.md).

## Дополнение SAML/OIDC/LDAP, 2026-10-13

Текущий контракт: 122 API operations, 19 migrations, ADR-014. Добавлены [инструкция и границы совместимости](federated-auth.md); обновлены architecture/configuration/data model/recovery/controlled upgrade, threat/state/design review, testing/traceability/changelog, repository map и current state. Browser pack теперь содержит 45 cases. Исторические delivery/evidence и туры сохраняют прежнюю границу проверки; фактическая квалификация новой реализации фиксируется отдельно в [session](session.md).

## Дополнение UX USER/ADMIN, 2026-10-13

[ADR-015](adr/015-product-ux.md), [ревью и инструкция](product-ux.md): 123 API operations, прежние 19 migrations, 62 PG integration cases и 60 browser cases в текущем наборе. Отдельный owner editor DTO сохраняет строгий старый контракт. Навигация white labeling, SSO и operations актуализирована; configuration/testing/threat/states/traceability и changelog согласованы. API/worker/CLI включают IANA database Go. Предыдущие числа выше остаются историческими, фактические результаты нового набора фиксируются в [session](session.md). На момент этого шага нарративный тур сохранял дату 2026-10-12; UX снимки имели собственную границу проверки.

## Дополнение: полный тур и актуальная навигация, 2026-10-14

| Статус | Результат и граница |
|---|---|
| Реализовано | Пересняты все 71 PNG и 26 глав (7:56), 104 cues, единый MP4 ElevenLabs Liza. Настройки/apply, report trash/restore, SAML JIT USER, бренд/apply/reset и recovery выполнены на одноразовом Compose. |
| Изменено | Обзор README/index/current-state/MEMORY и план отражают main 28a6b4a/PR #26. Product UX/white label/SSO ссылаются на новую запись; старые снимки review остаются датированными. |
| Добавлено | Reviewed capture plan, проверка полноты до рендера, приватный staging PNG, новые негативные tests, актуальные инструкции сборки/озвучки/QA и [evidence](history/tour-refresh-evidence-2026-10-14.md). |
| Не подтверждено | Реальные IdP tenants, Windows WebDAV, физические устройства, внешняя SMTP/S3/escrow/DR приёмка. Их статус не меняется из-за демонстрации форм. |
| Требует доработки | DEV-01/03/04, EXT/REL и 5 partial MUST / 14 pending decisions остаются открыты. DEV-05 уже доставлен с UX; историческая строка среза 2026-10-12 выше не описывает нынешний backlog. |

Runtime/API/env/migrations не изменены. Architecture/security/contracts документы проверены на применимость, их правила не переписываются ради медиа. `screen-tour.md`, HTML, manifest, captions и checksums создаются генератором; historical counts/hashes в history/sessions сохраняются. Документационный checker проверяет весь Markdown-корпус и достижимость архивов; текущая граница QA — [session](session.md).
