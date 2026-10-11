# Production hardening

> Нормативный scope следующих фаз 6–7. Этот документ не является отчётом о выполнении; текущее evidence и ограничения — [development](../wiki/development.md), порядок работы — [roadmap](../wiki/roadmap.md).

Traefik ACME HTTPS, TLS >=1.2 (1.3 preferred), HSTS после корректной настройки, strict forwarded headers. Docker networks edge/backend; DB/Redis не публиковать. Non-root, no-new-privileges, минимальные capabilities, read-only root FS где возможно, никаких Docker socket mounts. Защита SSH/firewall/host backups/keys; swap-off — рекомендация, не обязательный runtime invariant. Resource limits, header/body/idle limits, NAT-friendly per-IP connection limits, no plaintext SMTP/HTTP S3 production, CA trust store без skip verify. Secrets never in logs, images, repo or shell history.
