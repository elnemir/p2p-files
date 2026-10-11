# Фаза 7.1: HTTPS ingress и сетевая политика

## Контекст и статус
Фаза 6 опубликована как `02c53d9`; [CI](https://github.com/elnemir/p2p-files/actions/runs/38131827674) прошёл. Фаза 7 начинается с ingress, границы доверия к proxy и воспроизводимой локальной проверки. Здесь описаны реализация и порядок подготовки конфигурации; это не разрешение production deploy. `APP_ENV=production` пока отклоняется binary до завершения release acceptance.

## Проблема и решение
За development Traefik прежний rate limit видел общий IP proxy. Доверие присланному клиентом X-Forwarded-For создало бы обход rate limit и CIDR ограничений. [ADR-007](adr/007-ingress-and-network-policy.md) фиксирует один доверенный hop: Traefik удаляет недоверенные forwarded headers на входе, API принимает их только от заранее указанных IPv4 `/32` peers.

`internal/config/network.go` проверяет policy в typed settings; `internal/networkpolicy` разрешает effective IP; `internal/httpapi/network.go` применяет allowlist/proxy/HTTPS/Host guards до login/доступа к данным. Login throttle использует этот IP. Proxy chains, повторные headers, IP с port/zone, malformed values, non-HTTPS/чужой Host при requireHttps отклоняются. Нет fallback к присланному клиентом адресу.

## Вход → настройки → gate → runtime
Поле `networkPolicy` необязательно для прежних settings v1. Без него поведение development остаётся direct peer, без доверия forwarded headers. Все CIDR должны быть canonical IPv4; trustedProxyCidrs — до 16 точных `/32`, allowedClientCidrs — до 128 сетей. Пустой allowlist разрешает все IP, но не снимает остальные guards.

```json
{
  "publicOrigin": "https://files.example.com",
  "networkPolicy": {
    "trustedProxyCidrs": ["172.30.90.2/32"],
    "allowedClientCidrs": ["203.0.113.0/24"],
    "requireProxy": true,
    "requireHttps": true
  }
}
```

Это фрагмент полного settings v1: сохраняются version, TTL, file limits и остальные поля текущей revision. Примерные адреса не являются настройкой реального VPS.

1. ADMIN создаёт/валидирует draft обычным API/UI либо host CLI читает `settings show`, изменяет полный `value` и передаёт `{expectedRevision,value}` в `settings apply` через stdin.
2. Apply проверяет expectedRevision. Изменение publicOrigin/networkPolicy в той же transaction закрывает gate, инвалидирует reconciliation, меняет generation и записывает audit. Ошибочный CAS не меняет active settings/gate.
3. Старые data-plane и привилегированные запросы отменяются при очередной проверке runtime guard (интервал 200 ms); старый процесс затем возвращает restart_required. Перезапустить все API/worker, войти ADMIN из разрешённой сети, провести reconciliation и явный reopen.
4. Ошибочный allowlist можно снять с доверенного host: `network reset` получает `{reason}` на stdin, создаёт новую settings revision и audit event. Proxy/HTTPS/origin остаются прежними. Reset также меняет generation и требует restart/reconcile/reopen; это не изменение host firewall.

`settings show` и `config ingress-check` только читают БД, не применяют migrations. Последняя команда проверяет requireProxy/requireHttps/HTTPS origin и возвращает `ingressConfigured=true` вместе с `productionReleaseApproved=false`; DNS, сертификат, доступность provider и разрешение запуска она не подтверждает. Для исправления неверного trustedProxy/origin используется полный settings apply с host, а не ослабляющий reset.

## Генерация конфигурации
`scripts/production_config.py` использует стандартную библиотеку Python. Команда создаёт новый каталог и отказывается перезаписывать существующий. Images API/web должны иметь `@sha256:…`; базовые PG/Redis/Traefik pins берутся из существующего development Compose.

```sh
python3 scripts/production_config.py \
  --domain files.example.com --email operator@example.com \
  --subnet 172.30.90.0/29 \
  --api-image registry.example.com/filex-api@sha256:<64-hex-digest> \
  --web-image registry.example.com/filex-web@sha256:<64-hex-digest> \
  --output .cache/production-config
```

Заменить placeholders реальными проверенными digests до исполнения. Subnet — свободная RFC1918 IPv4 сеть `/24`…`/29`; генератор проверяет формат, но не маршруты реального host/VPN. Traefik получает адрес network+2, API +3, web +4. Traefik обращается к literal API/web адресам proxy network: service DNS при общей egress network мог бы выбрать другой интерфейс и потерять ожидаемый peer identity. Внешний proxy/load balancer не поддерживается этим профилем.

Результат: `compose.json`, `traefik-static.yaml`, `routes.yaml`, `settings-ingress.json`, `runtime.env.example`. YAML файлы используют совместимое JSON-представление YAML; расширение `.yaml` обязательно для Traefik file provider. Генератор не подключается к DNS/ACME/Docker daemon и ничего не запускает.

- Только Traefik публикует 80/443; точный Host routing, HTTP→HTTPS redirect, TLS ≥1.2, strict SNI, HSTS 1 год без includeSubDomains/preload. Access logs выключены; application no-store/no-referrer сохраняются.
- ACME HTTP-01 использует порт 80; storage `/var/lib/traefik/acme.json` хранится в отдельном host-каталоге `acme`. Перед реальным запуском оператор готовит каталог с владельцем UID/GID 65532 и mode 0700; существующий acme.json сохраняется, mode 0600. Нельзя обнулять действующее хранилище сертификатов.
- API/web/worker/Traefik/PG/Redis работают non-root, cap_drop ALL, no-new-privileges, read-only root FS; mutable данные вынесены в volumes/tmpfs. Internal API bind — 127.0.0.1:9090 внутри API container. DB и Redis находятся в private backend, proxy network тоже internal; outbound API/worker/Traefik используют отдельную egress network. Docker socket не монтируется.
- CPU/RAM/PID/log rotation limits — инженерные стартовые bounds, не подтверждённая ёмкость/SLA. PG/Redis UID привязаны к закреплённым Alpine images и должны перепроверяться при изменении pins.
- Secret env-файл создаётся отдельно mode 0600: POSTGRES_PASSWORD — URL-safe случайный пароль (например, hex), MASTER_KEK — независимо сгенерированные 32 bytes base64. Эти значения не передаются через argv, не сохраняются в Git и не выводятся `docker compose config` в общие логи. Settings/Local/S3 provisioning выполняются явными CLI actions; production профиль не содержит DEV_* autoprovision.

## Локальная проверка
`make ingress` создаёт уникальный `filex-ingress-*` project, свободную private subnet, случайные credentials/KEK и собственную однодневную CA. Используется та же сгенерированная конфигурация с loopback/random ports, локально собранными images, development runtime и тестовым сертификатом вместо ACME. Публичный ACME не вызывается.

Проверяются реальный TLS/CA, HTTP redirect, HSTS/privacy, secure session cookie, Host routing, недоступность public /health /ready /metrics, spoofed forwarded headers, client CIDR denial, host reset и restart fencing. Unit/PG tests отдельно проверяют malformed headers/CIDR, effective IP bucket, stale revision rollback, отмену привилегированного запроса и audit reset. Скрипт в finally удаляет только собственный project/volumes; images/build cache остаются.

Фактические результаты сохраняются в [session](session.md) и [development](development.md). Наличие генератора и локальной CA не доказывает issuance/renewal ACME, host firewall, public DNS или readiness реального VPS.

## Продолжение фазы 7
Остаются controlled upgrade/downtime drill, полный browser/manual WCAG, load/DR на целевой топологии, SBOM/license/container/secret/static analysis, независимый security review и рассмотрение residual risks владельцем. Production guard снимается только после review этих gates; реальные DNS/host/deploy mutations требуют отдельного поручения. Все 32 MUST сохранены, SEC-005/OPS-005 пока partial.
