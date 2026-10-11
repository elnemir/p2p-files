# Фаза 7.1: контролируемое обновление и простой

## Контекст
FileX — single-VPS Compose приложение. API/worker автоматически применяют embedded migrations; бизнес-запросы удерживают shared runtime locks, recovery использует durable gate. Эта страница описывает безопасную границу замены версии, [ADR-008](adr/008-controlled-upgrade.md). Production запуск остаётся запрещён до release acceptance; локальный drill не является разрешением deployment.

## Проблема и изменение
Прежний startup проверял checksum известных SQL, но мог игнорировать неизвестные строки в migration ledger. Новый `database.CheckUpgrade` и startup используют общий план: ledger должен быть непрерывным префиксом SQL текущего binary. Unknown migration, checksum mismatch или пропуск внутри префикса запрещают запуск. Это проверка совместимости схемы, а не гарантия совместимости любых версий приложения.

Новый host `upgrade prepare` не применяет SQL, не скачивает images и не управляет Docker. Он использует прежние operations runner lock, audit и generation вместо отдельной системы deployment. HTTP/OpenAPI/AsyncAPI и JSON Settings не изменились; новых env или SQL migrations нет.

## Поток подготовки и блокировок
1. JSON `{reason}` проходит strict stdin decode. Причина — 5…2000 Unicode-символов. CLI проверяет схему без bootstrap/migrations: prepare требует ноль pending migrations и запускается binary, соответствующим установленной схеме.
2. Service получает единственный operations runner lock; stale generation и queued/running operations отклоняются. В transaction блокируется singleton ops_control, gate закрывается, reconciliation инвалидируется, generation меняется, `upgrade.preparing` добавляется в tamper-evident ops audit.
3. Старые HTTP/worker contexts, включая ADMIN, отменяются очередным guard (200 ms). Prepare ждёт exclusive runtime lock: сигнал cancellation сам по себе не означает завершение внешнего storage effect. После drain повторно проверяются generation/gate/pending jobs и записывается `upgrade.prepared`.
4. Успех возвращает `{generation,mode,drained:true,requiresRestart:true}`. Timeout/conflict не открывает gate. CLI bound 30 s, PostgreSQL lock timeout 3 s; это предел одной попытки, не SLA. При неудаче проверить состояние и завершить мешающую операцию, затем повторить matching CLI.
5. Pending migrations сериализуются migration advisory lock → exclusive runtime transaction lock → ops_control row lock. Для существующего control plane gate проверяется closed до и после захвата. Весь SQL и ledger коммитятся атомарно. При обновлении существующей БД новая generation и invalidated reconciliation фиксируются вместе с DDL; failed transaction сохраняет прежнюю закрытую state. Fresh install использует начальный gate из migrations.
6. Bootstrap берёт shared runtime lock для KEK/settings только после migrations. Старые binaries до этой механики не становятся безопасными автоматически; все процессы старой версии необходимо остановить.

## Порядок обновления оператором
Команды ниже используют доверенную host/container границу. Здесь нет команды production deploy: конкретный deployment project, secrets и image digests проверяются отдельно.

1. Подготовить candidate immutable image и сохранить прежний digest/config. На candidate выполнить `filex-cli upgrade check --json`, подключив существующий bootstrap env. Это read-only проверка; она не создаёт даже schema_migrations в пустой БД.
2. Выделить окно простоя; остановить worker, чтобы scheduler не поставил новую recovery job. Завершить queued/running operations штатным CLI. На установленной версии вызвать `filex-cli upgrade prepare --json` со stdin:

   ```json
   {"reason":"Плановое обновление приложения с проверкой данных"}
   ```

3. После confirmed drain остановить все API/worker/web instances, включая restart supervisor. PostgreSQL/storage остаются доступны. Prepare не удерживает постоянный deployment lock: новый процесс может стартовать с новой generation, поэтому полный stop обязателен.
4. Одноразовым CLI **старого image** выполнить `backup create` с `{providerId,reason}`, затем `backup verify` с полученным `{backupId}`. Сохранить descriptor/backup, старый binary/digest и KEK вне изменяемого окружения. DB archive не содержит payload/artifact bytes. Автоматического pre-migration backup нет; отказ backup/verify останавливает процедуру до замены приложения.
5. Ещё раз выполнить candidate preflight и запустить candidate с теми же volumes/bootstrap secrets. Startup применяет pending SQL под locks; API/worker остаются за закрытым gate. `/api/v1/status` и internal readiness не ready, ADMIN control plane доступен для проверки. Прежние web sessions не отзываются самим upgrade; restore имеет отдельные invalidation semantics.
6. Выполнить `recovery reconcile`, прочитать report из результата job, проверить findings/capabilities. `recovery reopen` требует report ID/hash, reason и `acknowledgeCapabilities:true`. Только после него data plane возобновляется.
7. Проверить status, owner данные, recipient read/Range, одноразовое завершение и audit. Зафиксировать измеренный простой, exact images/schema и evidence.

Для первой установки этого CLI на опубликованную версию `50de642` candidate prepare допустим только потому, что обе версии имеют одну и ту же схему из 16 migrations. Для будущего candidate с pending migrations нужен prepare установленной версии. Для исторической схемы до 0014 (без recovery gate) сначала полностью остановить старые процессы; candidate startup создаст gate и оставит его закрытым до reconciliation.

## Ошибка и rollback
Unknown/checksum/hole либо DDL error не разрешают частичный startup. DDL transaction откатывается целиком; gate после подготовки остаётся закрытым. Неверный KEK также блокирует запуск, нельзя заменять секрет произвольным новым ключом или стирать sentinel/ledger для обхода.

Rollback image не выполняется автоматически. Запустить read-only preflight **предыдущего binary** на текущей БД: при совместимой схеме можно вернуть прежний image в рамках согласованного окна, затем выполнить reconciliation/reopen. При неизвестной новой migration требуется matching binary/schema и [recovery runbook](operations-recovery.md); down migrations отсутствуют. Старый backup с другим schema fingerprint не импортируется поверх новой схемы. Нужны отдельное matching-schema окружение, обязательный verified pre-restore backup и независимая сохранность payload.

## Воспроизведение и evidence
`make upgrade-drill` собирает baseline из закреплённого commit `50de642f2bf9e2c58e5e3ae0e92f92903cfa634e` через `git archive` и candidate из текущих файлов. Нужны Docker, Git history с baseline и Python с tar `filter='data'`; CI checkout получает полную history. Это не загрузка исполняемых release artifacts из случайного latest.

Скрипт создаёт только собственный `filex-upgrade-*` Compose project, random loopback port и случайные тестовые credentials/KEK, не читает operator .env. Через Traefik создаётся файл и partial Range; затем prepare/drain, stop, verified backup, отказ unknown migration и неверного KEK, запуск другого image, reconciliation/explicit reopen, проверка retained session, payload SHA и одного завершения. При истёкшей reservation файл приобретается заново явным acquire; повторная выдача completed запрещена. Synthetic migration row добавляется/удаляется только внутри fixture для fault injection и не является способом починки настоящей БД.

Status sampler выполняется каждые 200 ms (запрос до 5 s). `maintenanceWindowSeconds` — время от запуска prepare до успешного status после reopen, включая backup и injected failures; `observedUnavailableSeconds` — интервал между первым/последним неуспешным sample, не точное время переключения. Результат `.cache/evidence/phase7-upgrade.json` содержит SHA baseline/HEAD, dirty flag candidate, реальные Docker image IDs, schema count и флаги проверок, без credentials/recipient links. Project/volumes удаляются в finally; images/build cache остаются.

Compose drill — обновление между разными образами с **одной схемой 16**. Реальное SQL обновление prefix 15→16, concurrent startup, failed multi-DDL rollback, lock timeout и older-binary rejection проверяются отдельными PG/race tests. Полные фактические результаты — [session](session.md), [development](development.md). Они не обещают production downtime/RPO/RTO и не заменяют load/DR/public ACME/independent security/browser/WCAG acceptance.
