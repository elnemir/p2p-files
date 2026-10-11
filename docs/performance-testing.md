# Performance и recovery acceptance

## Контекст
Профиль проверяет ограниченное потребление ресурсов и целостность данных FileX при нагрузке и сбоях. Локальные числа не являются SLA, целевыми RPO/RTO или результатом теста реального S3.

## Воспроизведение
`make performance` создаёт отдельный Compose project с 32 MiB payload и 2000 синтетических записей. Полный профиль: `FILEX_PERF_MIB=2050 FILEX_PERF_ROWS=20000 python3 scripts/acceptance.py run performance`. Допустимые пределы: 8…8192 MiB и 100…40000 записей. Внешний URL не принимается, случайные credentials используются только в disposable project; finally удаляет его containers/volumes.

## Поток данных и проверки
1. Один файл загружается кусками по 4 MiB через tus; HEAD подтверждает возможность resume, finalize вычисляет серверный SHA-256.
2. Local → synthetic TLS S3 migration → payload key rotation сохраняют данные. TLS certificate проверяется тестовой CA; fixture проверяет envelope SigV4, но не криптографическую подпись AWS. Это wire compatibility, не acceptance реального провайдера.
3. Redis останавливается. Восемь concurrent acquire одного файла дают ровно один 201; Range первого MiB сверяется с исходным хэшем.
4. PostgreSQL содержит большой набор synthetic transfers. Snapshot запись блокируется отдельной транзакцией; после claim worker получает настоящий SIGKILL. Скрипт освобождает только свой blocker, искусственно истекает lease этой job и запускает worker. Gzip CSV проверяется по числу строк, VACUUM ANALYZE завершается без оставленного idle snapshot. Время этого шага не включает обычное трёхминутное ожидание lease и не является crash RTO.
5. Сто `/status` запросов с concurrency 8 дают p50/p95/max.
6. Encrypted backup/verify → поздняя запись → restore с обязательным pre-backup → restart → новая сессия → reconciliation → явный reopen. Поздняя запись отсутствует, прежняя сессия отозвана, до reopen data plane закрыт.
7. ZIP читается потоком и сверяется по размеру/SHA-256; повторная выдача lease запрещена, audit проходит. OOMKilled завершает проверку ошибкой.

## Evidence и ресурсы
`.cache/evidence/performance/results-<MiB>mib-<rows>rows.json` содержит CPU/RAM/kernel/storage Docker VM, размеры, concurrency, время этапов и throughput, latency и timestamped `docker stats` CPU/RAM/IO samples. Sampled peaks не являются непрерывным измерением пика. `results.json` — последний прогон; частичный отчёт при ошибке не содержит полного набора assertions и не считается PASS. Gate record содержит exit code и hash исходников.

Полный профиль требует свободного места для нескольких encrypted generations и скачанного ZIP. На CI выполняется компактный профиль, отдельный release gate требует ≥2048 MiB и ≥20000 записей. Точные измеренные результаты публикуются в [development](../wiki/development.md) и [session](../wiki/session.md).

## Внешние ограничения
Владелец 2026-10-11 подтвердил, что окружение не подготовлено. Повторение профиля на целевом VPS, реальном S3/SMTP, DR с отдельного хранилища и внешними KEK/checkpoints остаётся заблокированным этой подготовкой. Нагрузка на живую инфраструктуру требует конкретного целевого профиля; локальный harness такую нагрузку не создаёт.
