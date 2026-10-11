# Performance testing

> Нормативный scope следующих фаз 6–7. Этот документ не является отчётом о выполнении; текущее evidence и ограничения — [development](../wiki/development.md), порядок работы — [roadmap](../wiki/roadmap.md).

В CI — компактные boundary fixtures и memory-bounded streaming tests. Отдельно — multi-GB upload/download/ZIP, Range resume, large export, Local/S3 migration, key rotation, concurrency, Redis loss, worker crash, PG snapshot VACUUM pressure. Записывать железо, CPU/RAM/IO, throughput/latency, peak memory, размер/число файлов, конфигурацию providers. Не придумывать SLA.
