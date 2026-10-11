# Подготовка окружения и внешняя приёмка

## Контекст и текущий блокер

Владелец 2026-10-11 ответил: **«Окружение ещё не подготовлено; зафиксировать внешние блокеры»**. Документ описывает будущую подготовку и проверку; реальные адреса/учётные записи не назначены, внешние действия не выполнялись. Все 14 решений в [release-acceptance.json](../contracts/release-acceptance.json) pending.

Локальные fixtures не подтверждают public ACME, доставку в реальный mailbox, конкретный S3, независимые keys/backups/checkpoints или настоящие устройства. Работы EXT-01–07 и зависимости — [implementation plan](implementation-plan.md).

## Карточка окружения для оператора

Заполнить перед началом внешних работ. Secrets хранить в выбранном secret store; здесь только идентификаторы и безопасные ссылки.

| Поле | Текущее значение / что требуется |
|---|---|
| Environment ID / назначение / ответственный / окно работ | Не определены; отдельно указать staging или production |
| VPS / ОС / CPU/RAM/disk / внешний IPv4 | Не подготовлено; зафиксировать host ID, ресурсы и доступ с доверенного места |
| Domain / DNS zone / ACME contact / порты | Не подготовлено; authority DNS, IPv4 A, достижимость 80/443 и проверка renewal |
| Docker/Compose / private subnet / operator access | Версии и свободная сеть; SSH/firewall policy; отсутствие публичных DB/Redis/API/internal ports |
| Candidate | Commit/source hash, пять image digests, schema fingerprint и config revision |
| SMTP | Host/port STARTTLS или SMTPS/CA, sender domain, разрешённый тестовый mailbox, ссылка на secret |
| S3 | Endpoint/region/buckets для нужных purposes, права/conditional writes/inventory, ссылка на secret |
| Backup и payload durability | Отдельный BackupProvider, независимая сохранность payload/artifacts, retention/capacity и restore access |
| KEK/checkpoints | Независимый escrow, исторические KEK, внешние anchors, доступ при полной потере VPS |
| Browser/AT matrix | Реальные OS/device/browser/assistive tool версии и reviewer |
| Load/DR | Профиль размера/числа файлов/экспорта/concurrency, допустимое окно; целевые RPO/RTO согласовать до оценки |
| Evidence store / reviewer / owner | Durable location, права доступа/retention; персональные назначения ещё не определены |

Hostname/email из примеров wiki не являются выбранным окружением. Не копировать credentials в repository, argv или публичные CI artifacts.

## Последовательность проверки

1. Согласовать карточку окружения и разрешённые действия. DEV-01 должен определить reviewed способ target qualification при действующем production guard; не использовать development как незаметный production обход.
2. Проверить host/DNS/network/ACME отдельно от app; построить immutable кандидат и просканировать реальные images. Применение generated config не означает автоматическое одобрение.
3. Выполнить provider acceptance на согласованных тестовых данных/адресах. Затем подготовить независимый backup/escrow/checkpoint доступ.
4. Выполнить target load и полное DR на отдельном целевом стенде: encrypted backup → потеря исходного окружения в пределах тестового плана → matching-schema восстановление с новым master KEK и historical recoveryKek → reconciliation/capabilities review/reopen → проверка payload SHA и одноразовой выдачи.
5. Пройти real-device/AT и независимый security/license review. Исправления меняют candidate; повторить затронутые проверки, затем все финальные gates для окончательного release hash.
6. Reviewer/owner принимает каждую фактическую запись. Обновить decisions/matrix со ссылками на evidence; `make release-check` повторить на clean candidate. Деплой production требует отдельной команды даже после принятого выпуска.

## Все 14 внешних decision IDs

| Decision ID | Задача / роль | Минимальное evidence для рассмотрения |
|---|---|---|
| public-acme-dns | EXT-01 / оператор | DNS/IPv4/host, issuance и renewal, TLS/Host/redirect checks, timestamps |
| host-firewall-backups | EXT-01 / оператор | Список доступных портов снаружи, SSH/firewall, non-root/resources, проверка host backup/restore |
| real-smtp-s3 | EXT-02 / инженер + оператор | Verified TLS, отправка на разрешённый mailbox и получение, retry/DLQ; реальный S3 Put/Get/Delete/conditional write/inventory, SHA |
| independent-key-checkpoint-escrow | EXT-03 / оператор + reviewer | Внешнее хранение anchors/KEK и успешный доступ без исходного VPS; секреты не в evidence |
| target-load-dr | EXT-04 / QA + оператор | Точный профиль/железо/resources/latency, естественный crash lease, внешнее восстановление, измеренные RPO/RTO и сверка с согласованными целями |
| real-browser-assistive-technology | EXT-05 / QA reviewer | Реальные Safari/mobile, клавиатура/focus/zoom/error, screen reader, RU/EN/themes; версии и найденные/исправленные дефекты |
| independent-security-review | EXT-06 / независимый reviewer | Scope/commit/digests, architecture/crypto/threat/host review, findings/triage/retest; нет открытых HIGH/CRITICAL |
| third-party-licenses | DEV-03 + EXT-07 / владелец/reviewer | Состав реальной поставки, SBOM/notices/fonts/OS, решение по каждому спорному компоненту |
| R-01 | EXT-07 / владелец | Restore resurrection сценарий, capability review/revoke, принятие оставшегося риска |
| R-02 | EXT-07 / владелец | Server-observed completion wording, сетевой предел повторных bytes до commit |
| R-03 | EXT-07 / владелец | Измерения export snapshot/MVCC/VACUUM и bounds, остаточный риск больших объёмов |
| R-04 | EXT-07 / владелец | Capacity/retention для бессрочных holds/audit/reports; operator procedure |
| R-05 | EXT-07 / владелец | Class DEK blast radius, escrow/rotation/recovery и оставшаяся граница доверия |
| R-06 | EXT-07 / владелец | Host CLI trust/SSH/access/audit и последствия host compromise |

Одна запись decision может ссылаться на несколько протоколов, но не на пустой «проверено». Environment.status=not_prepared описывает вход; одна его замена не закрывает decisions. Текущий readiness валидирует полноту 14 записей, а не достоверность подписи/решения человека.

## Форма протокола приёмки

```text
Acceptance ID / requirement IDs / decision ID:
Дата UTC / reviewer / оператор / полномочия на действия:
Environment ID / commit / source hash / image digests / schema/config revision:
Входные synthetic данные / профиль / предусловия:
Шаги, ожидаемые и фактические результаты:
Негативные сценарии / ошибки / corrective action / retest:
Raw evidence location / checksums / доступ и retention:
Ограничения, что не проверено:
Решение reviewer/owner и основание:
```

В JSON хранить status=accepted, reviewer и evidence только после такого реального решения. Не записывать секреты или пользовательские данные. Если проверка неуспешна — сохранить pending и завести конкретный defect, а не ослаблять gate.

## Границы load/DR и безопасности

DB archive не содержит payload/artifacts; они должны пережить loss test самостоятельно. Одно восстановление DB без контрольных SHA объектов не закрывает DR. Локальный SIGKILL harness искусственно истекает lease и не измеряет production recovery timeout. RPO/RTO заранее не обещаны. Ручная host/firewall политика и legal/license выводы принадлежат назначенному оператору/reviewer/владельцу, не генератору или scanner.
