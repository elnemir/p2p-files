# Разработка: локальный запуск и диагностика

## Контекст

Инструкция поднимает собственный development stack на loopback. Она не создаёт production deployment и не требует внешних SMTP/S3 accounts. Архитектура — [здесь](architecture.md), runtime настройки — [configuration](configuration.md), тесты — [testing](testing.md). Исторические результаты вынесены в [evidence](history/development-evidence-2026-10-11.md).

## Необходимые инструменты

- Git с историей: compatibility использует `f68fc08`, upgrade drill — `50de642`. Shallow clone без этих commits не подходит для полных gates.
- Go toolchain 1.26.9 из [go.mod](../go.mod), Node.js 24 как в CI/Dockerfiles, npm с `npm ci`, Python 3.12+ (upgrade harness использует `tarfile.extractall(filter='data')`).
- Для nvm выполнить `nvm install && nvm use` из корня: [.nvmrc](../.nvmrc) содержит major 24. Editor/Git defaults и GitHub workflow — [repository governance](repository-governance.md).
- Docker daemon и Compose v2 с `--wait`, Linux containers; macOS использует Docker Desktop. OpenSSL CLI нужен TLS fixtures. `curl` нужен ручной проверке; `jq` — только приведённым API примерам.
- Свободный loopback порт 8090 и место для images, нескольких копий encrypted payload, ZIP и evidence. Full load profile существенно тяжелее compact. Конкретная ёмкость production не определена.

Команды выполняются **из корня репозитория**. Версии Node/Go закреплены в коде/CI; это описание текущей конфигурации, не обещание бессрочной актуальности версий.

## Первый запуск

```sh
npm ci
python3 scripts/init-dev.py
make up
curl --fail http://localhost:8090/api/v1/status
```

`npm ci` устанавливает workspace dependencies строго по lockfile. `init-dev.py` создаёт только локальный `.env` mode 0600 с независимыми случайными POSTGRES_PASSWORD и MASTER_KEK; существующий файл не перезаписывается. Не выводить его содержимое в общий лог. `make up` собирает пять runtime images и запускает шесть сервисов: postgres, redis, api, worker, web, traefik. API/worker используют один image. Холодная сборка Traefik может занимать несколько минут.

Startup: config → PG migrations → KEK sentinel/settings → development provisioning трёх storage purposes → HTTP/worker. Payload/artifacts/backups используют отдельные volumes. Бизнес-состояние хранится в PostgreSQL. После readiness открыть http://localhost:8090.

## Первый ADMIN

Создать вне репозитория приватный JSON-файл mode 0600 со следующей формой, заменив placeholder настоящим локальным паролем:

```json
{"email":"admin@example.test","password":"<локальный пароль длиной 12–1024 байт>","role":"admin"}
```

Передать файл через stdin, не пароль через argv:

```sh
docker compose --env-file .env -f deploy/compose.yaml exec -T api /usr/local/bin/filex-cli user create < /absolute/private-user.json
```

Путь — placeholder заранее созданного приватного файла. Удалить этот временный secret input после provisioning согласно локальной политике. Войти указанным email/паролем. ADMIN операции требуют повторного ввода пароля с ограничением 5 минут. Нет стандартного пароля, автоматической публичной регистрации или скрытого обхода auth. Дополнительного USER можно создать той же командой с role=user. Подробный [CLI](cli.md).

## Ежедневный цикл

```sh
make check
make integration-docker
make security
```

Для UI/сквозного сценария добавить `make e2e`; для соответствующих изменений — ingress/upgrade/fuzz/supply-chain/load. Полная таблица scope/gates — [testing](testing.md). Go cache и module cache Makefile помещает в `.cache`, telemetry Next отключена. Изменение Go/Next исходников требует пересборки Compose через `make up`; это не hot reload окружение.

`make e2e` сам запускает pinned Linux browser container, устанавливать браузеры на host не нужно. Пять проектов исполняются последовательно, каждый со своей БД. Перед E2E остановить свой dev stack, если он занимает 8090. Fixtures не читают операторский `.env` и удаляют только собственные временные containers/volumes.

## Остановка и сохранность данных

```sh
make down
```

Команда останавливает project из `deploy/compose.yaml`, сохраняя named volumes. Для продолжения — `make up` с тем же `.env`. Не менять MASTER_KEK у существующей БД: sentinel заблокирует запуск; штатная смена ключа описана в [key management](key-management.md). Удаление volumes стирает данные и не является обычным способом устранения ошибок. Не использовать общий `docker system prune` для диагностики проекта.

## Типовые ошибки

| Симптом | Проверить и продолжить |
|---|---|
| `.env already exists` | Файл намеренно сохранён; продолжить с ним. Не регенерировать ключ ради повторного запуска |
| 8090 занят | Посмотреть `docker compose --env-file .env -f deploy/compose.yaml ps` и владельца порта; остановить только собственный dev project |
| PG tests skipped в Go output | Обычный unit suite не имеет тестовой БД; выполнить `make integration-docker`. Skip не считается PG PASS |
| `production gates are not complete` | Production guard безусловный. Работа по DEV-01 в [плане](implementation-plan.md); обхода для настоящего production нет |
| `critical key check failed` | Сверить факт использования нужного secret без его вывода; не стирать sentinel/key ledger. Следовать recovery runbook |
| `restart_required` или закрытый gate | Сверить operations status/reason; после network/restore/upgrade перезапустить все API/worker, выполнить свежую reconciliation и reviewed reopen |
| `snapshot_limit_exceeded` | Export превышает 50000 rows/16 MiB; сузить фильтр. Не заменять snapshot live pagination |
| Ошибка certificate/S3/SMTP | Проверить endpoint/CA/режим и ADMIN config; не отключать TLS verification |
| Compatibility не видит baseline | Получить необходимую Git history; не менять baseline ради зелёного результата |

Логи own containers можно читать через `docker compose ... logs --tail 100 api worker`; не публиковать secret input/полный env, реальные links и session traces. Для инцидента сохранить time, IDs/errorCode и состояние, затем [incident response](incident-response.md).
