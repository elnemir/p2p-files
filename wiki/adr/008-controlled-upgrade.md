# ADR-008: контролируемое обновление с закрытым доступом

Статус: реализован, targeted PG/race и Compose drill проверены 2026-10-11, фаза 7.1, OPS-005/NFR-003. Полная регрессия — в session. Production deployment и residual risks этим решением не одобряются.

## Контекст и проблема
FileX работает на одном VPS. Startup применяет embedded SQL, а runtime handlers/worker используют общий recovery gate. Read-only `upgrade check` уже проверяет checksum/unknown migrations, но прежний startup не проверял неизвестные записи. Нельзя обещать безопасный downgrade или обновление без простоя на основании одного preflight.

## Решение и поток данных
1. Read-only preflight и startup используют один план: applied migrations обязаны быть точным непрерывным префиксом embedded набора. Unknown, checksum mismatch и пропуск внутри префикса отклоняются до DDL.
2. Host `upgrade prepare` принимает `{reason}` и работает только с полностью совпадающей схемой, без startup migrations. Под existing operations runner lock он отвергает pending recovery jobs, атомарно закрывает gate/инвалидирует reconciliation/меняет generation и сохраняет audit. Затем ждёт exclusive runtime lock: все старые обычные и privileged handlers должны освободить shared locks. Неудачный drain оставляет gate закрытым.
3. Оператор останавливает все API/worker/web instances, используя конфигурацию своего deployment. На старом binary создаёт и проверяет backup; сохраняет текущие image IDs/digests, bootstrap secrets и независимую стратегию payload. Автоматический pre-migration backup не добавляется.
4. Candidate startup сериализует migration plan/DDL под migration lock. При pending SQL дополнительно берётся exclusive runtime lock; для существующего recovery control plane gate обязан быть закрыт до и после захвата. DDL, ledger и новая generation/invalidated reconciliation существующей instance коммитятся одной transaction. Bootstrap key/settings snapshot берёт shared runtime lock только после migrations, исключая shared→exclusive upgrade deadlock.
5. Новый runtime остаётся закрытым до audit/crypto/storage reconciliation и явного reopen с matching report/hash/generation. Upgrade не восстанавливает старые данные, не сбрасывает сессии/usage budgets и не меняет payload.
6. Ошибка candidate не запускает автоматический downgrade/reopen. Предыдущий image допустим только при успешном preflight его binary на фактической схеме; при несовместимой схеме нужен recovery matching binary/schema по существующему runbook.

## Пример и проверки
`filex-cli upgrade check --json` → `{applied,pending,compatible}`. `filex-cli upgrade prepare --json` со stdin `{"reason":"Плановое обновление приложения"}` → закрытый gate, новая generation и подтверждённый drain. Никакие image/Docker/SSH действия команда не выполняет.

PG tests проверяют fresh/prefix upgrade, unknown/checksum/hole rejection, concurrent startup, rollback DDL, exclusive drain и closed gate. Изолированный Compose drill заменяет image закреплённого предыдущего commit на candidate, проверяет сохранность payload, deliberate startup failure, измеренный простой и explicit reopening. Измерение небольшого локального fixture не является production SLA/RTO.

## Ограничения и review
Все instances, включая автоматически перезапускаемые supervisor процессы, должны быть остановлены перед заменой версии. Сам `prepare` не блокирует создание новых процессов навсегда; он фиксирует границу завершения старого поколения. Direct SQL/DDL, Docker/root и бинарники до появления shared runtime locks остаются доверенной host-границей. Для схем до migration 0014 требуется полный stop старых процессов до candidate startup. Криптографические primitives и backup format не меняются. Новых HTTP/event contracts и SQL migration для этой механики не требуется.
