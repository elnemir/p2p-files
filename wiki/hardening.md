# Hardening: реализация и внешняя приёмка

## Контекст и проблема

FileX обрабатывает capabilities, ключи и частные payload. Нужна защита на уровнях app, контейнера и host; локальный TLS fixture не доказывает безопасность публичного VPS. [Threat model](threat-model.md) определяет границы; [environment acceptance](environment-acceptance.md) — внешние проверки.

## Реализованный профиль

- Generator [production_config.py](../scripts/production_config.py) требует пять проверенных immutable digests. Traefik публикует только 80/443, exact Host, HTTP redirect, TLS ≥1.2, strict SNI и HSTS после корректной конфигурации; ACME HTTP-01 storage отдельный.
- Сервисы non-root, read-only root FS, cap_drop ALL/no-new-privileges, volumes/tmpfs только для записываемых данных. Нет Docker socket. Private backend/proxy networks, отдельный egress; DB/Redis/API/internal listeners не публичные.
- API доверяет forwarding только designated IPv4 /32 proxy, отвергает chains/duplicates, использует effective IP для allowlist/login throttling. Настройка сети CAS/audit/generation/restart/reconcile, не прямое изменение env.
- No-store/no-referrer/CSP, отсутствие body/raw query/secret logging; SMTP TLS и S3 HTTPS с проверкой сертификата. Webhook public IPv4/HTTPS443/DNS pin/no redirects отдельны от доверенной ADMIN provider config.
- Пять фактических runtime images сканируются Trivy с fail-closed HIGH/CRITICAL, включая unfixed; source/history/worktree secrets, SBOM и лицензии проверяются отдельно. Pins указаны в Dockerfiles и security-tools.json, их актуальность требует сопровождения.

## Порядок оператора

Заполнить карточку окружения → подготовить private network и secrets → сгенерировать профиль offline → проверить digests/config → выполнить host/DNS/ACME/provider qualification по согласованному пути допуска → сохранить evidence. Generator сам не подключается к Docker/DNS и не запускает deployment. [Ingress runbook](production-ingress.md) содержит команду с placeholders и требования к acme storage permissions; [upgrade](controlled-upgrade.md) — порядок замены кандидата.

## Открытые host controls

SSH/firewall/operator privileges, независимые backups/KEK/checkpoints, resource limits/capacity, certificate issuance/renewal и воспроизводимый DR проверяет оператор на целевой инфраструктуре. Swap-off — рекомендация исходного проекта, не runtime invariant. Limits контейнеров — стартовые инженерные bounds, не измеренная ёмкость. Existing ACME store не обнулять; CA verification не отключать.

## Проверки и ограничения

TLS/private routes/CIDR/reset и исправленные images проверены локально на 7dce226; исторический [отчёт](history/development-evidence-2026-10-11.md) не заменяет scan нового candidate. Production startup сейчас безусловно запрещён config.Load; DEV-01 в [плане](implementation-plan.md) требует reviewed решения, а не обхода. Окружение не подготовлено, deploy отдельно не разрешён.
