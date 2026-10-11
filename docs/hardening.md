# Production hardening

> Нормативный production scope фазы 7. Этот документ не является отчётом о выполнении; текущее evidence и ограничения — [development](../wiki/development.md), порядок работы — [roadmap](../wiki/roadmap.md).

Traefik ACME HTTPS, TLS >=1.2 (1.3 preferred), HSTS после корректной настройки, strict forwarded headers. Docker networks edge/backend; DB/Redis не публиковать. Non-root, no-new-privileges, минимальные capabilities, read-only root FS где возможно, никаких Docker socket mounts. Защита SSH/firewall/host backups/keys; swap-off — рекомендация, не обязательный runtime invariant. Resource limits, header/body/idle limits, NAT-friendly per-IP connection limits, no plaintext SMTP/HTTP S3 production, CA trust store без skip verify. Secrets never in logs, images, repo or shell history.

Первый срез фазы 7 реализует генератор отдельного HTTPS/ACME профиля и PG trusted proxy/CIDR policy с защищённым host reset: [runbook](../wiki/production-ingress.md). Реальное состояние checks — [session](../wiki/session.md). APP_ENV=production пока запрещён; конфигурация и synthetic TLS drill не являются host/ACME/release acceptance.
