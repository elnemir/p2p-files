# Docker-поставка сервиса лицензирования

## Контекст и проблема
Портал издателя отделён от клиентского FileX: Go API и signer, Next.js UI, своя PostgreSQL. Ранее [Compose](../deploy/issuer-compose.yaml) требовал исходники и сборку на сервере. Новый [pull-only Compose](../deploy/issuer-registry.compose.yaml) использует готовые images; application/API/crypto semantics не меняются. Требования FR-017/018, SEC-008; design и threat model остаются [ADR-016](adr/016-offline-licensing.md) и [ADR-017](adr/017-first-run-wizards.md).

## Решение и поток поставки
[Workflow](../.github/workflows/issuer-images.yml) собирает три linux/amd64 образа из exact Git SHA: `ghcr.io/elnemir/p2p-files-postgres`, `ghcr.io/elnemir/p2p-files-issuer`, `ghcr.io/elnemir/p2p-files-issuer-web`. API, migration и signer используют один issuer image с разными entrypoint/command. После сборки **всех** образов Trivy должен подтвердить отсутствие HIGH/CRITICAL (включая unfixed); только затем выполняется push `sha-<полный commit>` и сохраняются immutable digest references в `images.env`. Частичный network failure при push требует сверки результата; весь комплект считается готовым только после успешного workflow и проверки трёх digests. Для запуска используются digests, а не изменяемый tag.

Workflow запускается при push delivery-ветки и вручную; PR не получает packages write. Token выдаётся GitHub только publish job, Docker credentials находятся во временном каталоге и удаляются trap. OCI source/revision связывают package с репозиторием. SBOM и deployment files сохраняются 14 дней как artifact; оператор должен скачать их в постоянное хранилище. Это техническая поставка, не одобрение third-party licenses и не полный release acceptance.

Architecture/security review: разделение database/identity/presentation и три PG роли сохранены; ключ доступен только signer, отсутствующий bind source даёт ошибку, не создаёт каталог. TLS upstream порты остаются loopback. Образы не содержат приватных ключей, фиктивных trust roots или runtime secrets. Криптография, state machines, OpenAPI/AsyncAPI/JSON Schemas не изменены; новых application states/events нет.

## Подготовка оператора
Нужны Linux amd64, Docker Compose v2, доступ к GHCR, домен с HTTPS reverse proxy и OIDC provider с принудительным MFA. Скачайте artifact `issuer-deployment-<commit>` и сохраните `images.env`, Compose и routes example. Для private package нужен registry login с `read:packages`; видимость package этой задачей не меняется.

1. Создайте закрытый каталог `/srv/filex-issuer`, поместите туда файлы поставки. Скопируйте `issuer.env.example` в `issuer.env`, mode 0600. Задайте три различных случайных URL-safe пароля минимум 24 символа, HTTPS origin, key ID и абсолютный путь seed. `images.env` содержит три обязательных image digest.
2. Создайте отдельный каталог secrets для uid 65532 с mode 0700. Сгенерируйте seed **на сервере**, пользуясь опубликованным image (значение ISSUER_IMAGE из images.env):

```sh
# Run in a privileged operator shell; replace the image placeholder with the digest.
install -d -m 0700 -o 65532 -g 65532 /srv/filex-issuer/secrets
docker run --rm --network none --read-only --cap-drop ALL \
  --security-opt no-new-privileges:true --user 65532:65532 \
  --mount type=bind,src=/srv/filex-issuer/secrets,dst=/secrets \
  --entrypoint filex-license-signer IMAGE_AT_SHA256 \
  keygen vendor-2026-01 /secrets/vendor-license.seed \
  > /srv/filex-issuer/vendor-trust.json
```

Seed создаётся O_EXCL/0600; повтор намеренно отклоняется. stdout — только public trust. Сохраните seed в отдельном escrow; не помещайте его в Git, image или PostgreSQL backup. Public trust нужен для будущей сборки клиентского FileX. Release metadata требует **другого** ключа, см. [licensing operations](licensing-operations.md).

3. Из этого же commit репозитория запустите `python3 scripts/setup-access.py issuer /srv/filex-issuer/bootstrap`. Сохраните постоянный ISSUER_CONFIG_KEY отдельно от backup БД; token имеет срок один час. Если подготовка затянулась, не теряйте существующий config key при обновлении capability. Секретный token вводится только в мастер, не в URL.
4. Настройте внешний HTTPS proxy по [routes example](../deploy/issuer-routes.example.yaml): `/issuer/api/` и `/issuer/auth/` → loopback 8082, `/issuer` → loopback 3002. Если proxy контейнерный, его localhost не является host localhost; нужна отдельно настроенная приватная связь. Запрещено публиковать API/UI напрямую без TLS. OIDC callback: `<ISSUER_ORIGIN>/issuer/auth/callback`.

## Порядок запуска и проверка
После подготовки host/секретов оператор выполняет из `/srv/filex-issuer`:

```sh
docker compose --env-file images.env --env-file issuer.env --env-file bootstrap/bootstrap.env -f issuer-registry.compose.yaml config --quiet
docker compose --env-file images.env --env-file issuer.env --env-file bootstrap/bootstrap.env -f issuer-registry.compose.yaml pull
docker compose --env-file images.env --env-file issuer.env --env-file bootstrap/bootstrap.env -f issuer-registry.compose.yaml up -d --wait postgres
docker compose --env-file images.env --env-file issuer.env --env-file bootstrap/bootstrap.env -f issuer-registry.compose.yaml --profile setup run --rm migrate
docker compose --env-file images.env --env-file issuer.env --env-file bootstrap/bootstrap.env -f issuer-registry.compose.yaml up -d api signer web
```

Откройте `<ISSUER_ORIGIN>/issuer/setup`, введите token, настройте реальные IdP/MFA/allowlist; выполните вход и пробную выдачу на согласованную installation identity. API использует app роль, signer — signer роль, owner применяется только migration. PG volume сохраняет issuances/config/audit, immutable issued records не редактируются. Restart policy применяется к постоянным сервисам, migration одноразовая. Не используйте `down -v` для обновления; сохраните БД, seed и config key перед операцией.

## Ограничения и последующие шаги
Эта поставка не запускает сервер, не настраивает DNS/ACME/IdP и не создаёт реальные keys. Customer FileX images исключены по уточнению владельца; APP_ENV production guard клиента сохранён. Работающее MFA, backup/restore/escrow и внешняя security/license acceptance требуют проверки оператора. Проверенные commit/digests и точные результаты публикуются в [session](session.md); старый PASS не квалифицирует новую сборку.

## Исправление доступности UI, подтверждённое Docker-проверкой
Docker 29.6.2 принял HostConfig.PortBindings для web, но на единственной internal presentation network оставил NetworkSettings.Ports[3000/tcp] пустым; Next.js при этом Ready на 0.0.0.0:3000, API с отдельной external network доступен. Поэтому presentation bridge в обоих issuer Compose сделан обычным bridge, port по-прежнему привязан только к 127.0.0.1. Web не подключается к database и не получает secret/seed; signer остаётся только на internal database. Trade-off: web теперь имеет исходящий сетевой маршрут, как обычный frontend container. Host firewall/egress policy может дополнительно ограничивать его; исходящая изоляция signer сохранена. Ни public port, ни Docker socket не добавлены.
