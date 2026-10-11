# ADR-007: HTTPS ingress и доверие к адресу клиента

Статус: первый срез реализован и проверен локально 2026-10-11, фаза 7.1; SEC-005, OPS-005. Production release/deploy пока не разрешён.

## Контекст и проблема
Development Compose обслуживает loopback HTTP. API использует socket peer для login rate limit: за Traefik все клиенты имеют общий bucket. Доверие произвольному X-Forwarded-For позволило бы обойти rate limit и будущий CIDR allowlist. Production требует TLS, одного origin и отсутствия публичных backend/internal listeners.

## Решение и поток
1. Версионируемые PG settings получают `networkPolicy`: точные IPv4 `/32` адреса доверенных Traefik peers, необязательный список разрешённых IPv4 CIDR клиентов, `requireProxy` и `requireHttps`.
2. Без доверенного peer forwarded headers игнорируются; при requireProxy запрос отклоняется. Для доверенного peer принимаются только один X-Forwarded-For адрес и один X-Forwarded-Proto (`http`/`https`). Множественные значения/цепочки отклоняются. Поддерживается один Traefik hop, внешний proxy не предусмотрен исходными decisions.
3. RequireHttps требует requireProxy и HTTPS publicOrigin; проверяются proto и совпадение Host с configured origin. Effective client IP используется в allowlist и login rate limit, не возвращается в публичных ответах.
4. Изменение networkPolicy/publicOrigin атомарно применяет новую settings revision и закрывает maintenance gate. Одновременно меняется runtime generation: отменяются также привилегированные streams. Требуется restart API/worker; новая TCP connection к старому процессу не разрешает продолжение. ADMIN выполняет reconciliation/reopen из разрешённой сети либо через host CLI.
5. Host CLI `network reset` снимает только client allowlist, сохраняет proxy/HTTPS/origin, записывает audit reason и меняет generation и оставляет gate закрытым до restart/reconciliation/reopen. `settings apply` использует существующие draft/validate/expectedRevision semantics для provisioning/исправления policy.
6. Отдельная генерация production Compose/Traefik config создаёт HTTPS/ACME, fixed proxy address на отдельной сети, non-root/read-only services, resource bounds и private DB/internal API. Генерация ничего не запускает. Текущий binary продолжает отклонять APP_ENV=production до остальных release gates.

## Пример
`networkPolicy: {"trustedProxyCidrs":["172.30.90.2/32"],"allowedClientCidrs":["203.0.113.0/24"],"requireProxy":true,"requireHttps":true}` с `publicOrigin=https://files.example.com`. Адреса примера нельзя использовать вместо проверки выбранной сети/VPS. Пустой allowedClientCidrs разрешает любые адреса клиентов; proxy/HTTPS guards продолжают действовать.

## Проверка и ограничения
Unit tests: spoofed headers, untrusted peer, duplicate/comma chains, malformed IP/proto/Host/CIDR. PG tests: реальный middleware/login bucket, settings revision и gate, recovery reset/audit. Изолированный TLS smoke проверил pinned Traefik, собственную тестовую CA, TLS 1.2/отказ TLS 1.1, redirect/HSTS, secure cookie, Host и закрытость internal routes; public ACME issuance не симулируется как пройденная инфраструктурная приёмка.

API CIDR policy ограничивает API; статические страницы UI не являются confidential data. Public aggregate status при закрытом gate остаётся generic unavailable. Компрометация доверенного Traefik/host вне этой модели; `/32` не заменяет network isolation. RPO/RTO, public ACME, независимые secret/container/license/SBOM/security и browser/a11y evidence остаются последующими gates фазы 7.

Основания конфигурации: [Traefik entrypoints/forwarded headers](https://doc.traefik.io/traefik/reference/install-configuration/entrypoints/), [ACME](https://doc.traefik.io/traefik/reference/install-configuration/tls/certificate-resolvers/acme/), [TLS options](https://doc.traefik.io/traefik/reference/routing-configuration/http/tls/tls-options/). Проверка относится к закреплённому image digest, а не к неподтверждённому latest.
