# Dependabot: совместимость и сопровождение зависимостей

## Текущее изменение: Node 26, 2026-10-13

По явной команде владельца [PR #22](https://github.com/elnemir/p2p-files/pull/22) содержит согласованное обновление. Первоначальная замена двух Docker FROM давала EBADENGINE: root требовал 24.x, а образ исполнял 26.11.1. Теперь `.nvmrc`, engine, CI, builder/runtime и типы согласованы на 26; lockfile обновлён npm. [ADR-012](adr/012-node26-runtime.md) содержит решение до реализации, timeline policy, review архитектуры/угроз/crypto и rollback.

Node 26.11.1 на дату review — Current, LTS с 2026-10-28, EOL 2029-04-30 по [upstream schedule](https://github.com/nodejs/Release/blob/main/schedule.json). Это разрешённый владельцем переход до LTS, а не переименование канала или обход gates. `channel` хранит факт на `reviewedAt`; `ltsFrom` позволяет проверить его. Полные новые результаты публикуются в [session](session.md); история Node 24 ниже не заменяет их. Обновлены только `@types/node` до 26.6.5 и его transitive `undici-types` до 8.9.0; wire contracts/миграции не меняются. Browser gate выявил конфликт idle operations worker с reopen; минимальное исправление и PG RED/GREEN описаны в [recovery](operations-recovery.md). Первичный browser FAIL сохранён; все 13 gates повторены и PASS на source `bb488d14c1db0717c6aa93835f4c84e25a7b9314a5567b7700ee019b16ef8aac`: 53 integration, 35 browser и 57 toolchain tests.

## Контекст первоначального review 2026-10-12
FileX использует npm для Next.js и инженерных инструментов, Docker runtime, Go modules и GitHub Actions. Цель DEV-02 / NFR-003–004 / OPS-005 — поддерживаемые компоненты и реальные проверки, а не максимальный номер каждой версии. Проверка начата 2026-10-12 от main `d8b4e4f0fc1db06c0f7d61e3b963c9c75b31c7bc`.

## Историческая проблема 2026-10-12
На дату первоначального review все пять оставшихся PR — version updates; активных Dependabot alerts на момент проверки нет. Это не отменяет ошибок предложенных комбинаций:

| PR | Наблюдение | Решение и статус |
|---|---|---|
| [#11](https://github.com/elnemir/p2p-files/pull/11) ESLint 10 | npm ci ERESOLVE: JSX-a11y 6.10.2 поддерживает peer только до ESLint 9 | Кандидат: ESLint 10.12.0 + JSX-a11y-x 0.2.0; сохранены 31 правило, их severity и явно заданные параметры; каждый пример нарушения/исправления проходит |
| [#9](https://github.com/elnemir/p2p-files/pull/9) @eslint/js 10 | Config 10 при engine 9; CI зелёный без проверки целостности группы | Кандидат: @eslint/js 10.0.1 вместе с ESLint 10; full peer-tree и negative guard |
| [#10](https://github.com/elnemir/p2p-files/pull/10) Node types 26 | Типы обещают API более нового major, чем runtime 24 | Кандидат: types 24.19.2; major сверяется с runtime/CI |
| [#5](https://github.com/elnemir/p2p-files/pull/5) Node 25 | EOL по upstream, scanner HIGH/CRITICAL gate падает | Сохранить актуальный Node 24 LTS digest; не принимать EOL runtime |
| [#3](https://github.com/elnemir/p2p-files/pull/3) Redis 8 | Старые OpenSSL pins ломают сборку; BSD сменяется выбором новых лицензий | Redis 7.2 Extended продолжает получать security fixes, текущий digest актуален; переход 8 не требуется для устранения текущей CVE |

## Решение и поток проверки
Manifest + lockfile + runtime policy → clean npm ci → peer tree и согласованность major → lint negative/parity tests → build/browser/runtime/DR → dependency/source/image scans → exact-commit CI → merge. Инструменты реализованы; все 13 local gates выполнены, далее обязательны exact-commit GitHub checks. Порог HIGH/CRITICAL, CodeQL, digest pins и обязательные gates сохраняются.

TypeScript CLI приложения 7.0.2 и JavaScript API парсера — разные установки. Текущее дерево использует для parser TypeScript 5.9.3, удовлетворяющий заявленному диапазону `>=4.8.4 <6.1.0`. Один только app manifest не доказывает несовместимость parser. Проверять нужно resolved dependencies в контексте каждого инструмента.

## Источники и ограничения
- [Node 26.11.1 release](https://nodejs.org/en/blog/release/v26.11.1), [upstream schedule](https://github.com/nodejs/Release/blob/main/schedule.json): Current на 2026-10-13, LTS с 2026-10-28, EOL 2029-04-30.
- [Node releases](https://nodejs.org/en/about/previous-releases): Node 24 LTS; Node 25 EOL. Предпочтение LTS не означает вечный запрет нечётных major: upstream меняет модель начиная с 27.
- [Redis Open Source version management](https://redis.io/docs/latest/operate/oss_and_stack/install/version-mgmt/): 7.2 Extended, EOL 2029-12-01; не путать с Redis Enterprise Software.
- [Redis security policy](https://github.com/redis/redis/security/policy), [licenses](https://redis.io/legal/licenses/): 7.2 BSD-3-Clause; Redis 8 требует отдельного выбора владельца/reviewer.
- [ESLint Community plugin](https://github.com/eslint-community/eslint-plugin-jsx-a11y-x), [TypeScript parser support](https://typescript-eslint.io/users/dependency-versions/).

Закрытие version-update PR не равно исправлению vulnerability alert. Отказ от EOL runtime или необязательного major не сопровождается scanner suppression. Наличие green CI не означает production acceptance, юридическое разрешение поставки или результаты будущих scans.

## Реализованные проверки и примеры

`npm ci` читает [.npmrc](../.npmrc): `strict-peer-deps=true`, `engine-strict=true`. Web builder копирует тот же файл. Root engine `26.x` требует `nvm install && nvm use` перед установкой. Квалификация выполняется на Node 26.11.1; проверки на другом major не эквивалентны.

`npm run check:dependencies` проверяет [политику](../.github/dependency-policy.json), `.nvmrc`, engine, оба Node FROM/digest, Redis series/digest, major Node types и ESLint/config. Затем `npm ls --all` проверяет фактический установленный граф, включая optional peers. При наступлении срока security support gate падает; новый срок нельзя продлевать без upstream evidence. Политика фиксирует Node 26 Current на 2026-10-13, LTS с 2026-10-28, support до 2029-04-30; Redis 7.2 Extended до 2029-12-01. Даты проходят календарную проверку; future review, ложный LTS, неизвестный канал и LTS после EOL отвергаются. Наступление LTS сохраняет действительность прежнего Current review, наступление EOL всегда останавливает gate.

Зелёный PR #9 воспроизведён в изолированной папке: `npm ci --ignore-scripts` exit 0, затем `npm ls --all` exit 1 из-за `eslint@9.39.5`, не удовлетворяющего optional peer `@eslint/js@10.0.1`. Поэтому проверки установки/сборки недостаточно. Логи: ignored `.cache/dependabot-review/pr9-npm-ci.log` и `pr9-npm-ls.log`.

`npm run test:toolchain` исполняет 57 тестов: 24 для политики/ошибочных PR и сроков, 31 behavioral a11y fixture (нарушение → error, исправление → отсутствие этой ошибки), сохранность 31 настроенного правила и совместная проверка TypeScript/React Hooks. Fixtures используют настоящий project ESLint config для TSX. Новый preset совпадает с прежним по включённым правилам/явным параметрам; ESLint 10 дополнительно раскрывает defaults в calculated config. Тесты не объявляют полную эквивалентность всех возможных JSX программ и не заменяют browser axe/assistive-technology review.

Root `typescript@5.9.3` теперь явный dev dependency: он поддерживается и parser `>=4.8.4 <6.1.0`, и OpenAPI generator `^5.x`. App `typescript@7.0.2` остаётся отдельной установкой для `tsc`. Guard проверяет resolved версии через контекст parser/generator/app, не только manifest. API schema/type generation не изменены.

Dependabot объединяет ESLint, @eslint/*, typescript-eslint/@typescript-eslint/* и eslint-plugin-* в `lint-toolchain`, включая major updates. Version/security alerts и scanners не отключены, `ignore` не добавлен. Root CI `repository-policy` и `make check` исполняют проверки графа и toolchain tests.

## Почему Redis 8 и Node 25 не перенесены

Node 25 завершил поддержку 2026-06-01. Предложенный image добавляет четыре HIGH findings: CVE-2026-14456 и CVE-2026-45447 в libcrypto3/libssl3 3.5.6-r0; fixed versions 3.5.8-r0 и 3.5.7-r0. Данные взяты из [CI 38167197809](https://github.com/elnemir/p2p-files/actions/runs/38167197809), artifact `supply-chain-evidence-755285ab145a05443885806d2e66d64dad971619`. Эти CVE не скрываются исключениями: кандидат 2026-10-12 оставался на поддерживаемом Node 24.21.0 и был проверен заново целиком. Node 26 проходит отдельную квалификацию 2026-10-13.

Redis runtime — актуальный 7.2.16 из неизменённого digest. Security patches доступны в поддерживаемой BSD ветке. PR #3 является предложением нового major, не доказательством уязвимости 7.2. Локальный Redis 8 review уже устранил старые OpenSSL pins (см. [лицензии](third-party-review.md)), однако функциональная потребность и разрешение на новую лицензию отсутствуют. Решение — сопровождать 7.2 с датой окончания поддержки в CI; переход 8 остаётся отдельным изменением, а не незаметным принятием условий. Текущий image scan обязателен и при сохранении версии.

## Эксплуатационное влияние и продолжение

Изменяются dev/build tooling и Node types; бизнес-API, migrations, crypto и payload/recovery инварианты не меняются. Обычный путь обновления: синхронно обновить manifest/lockfile/config, выполнить `npm ci`, `npm run check:dependencies`, `npm run test:toolchain`, затем все обязательные [gates](testing.md). Не использовать `--force`, `--legacy-peer-deps`, ручное редактирование lockfile или отключение rules для прохождения установки. Новые версии plugin требуют тех же behavioral и browser checks. Отдельный proprietary/third-party release review остаётся pending.

## Квалификация кандидата 2026-10-12

Source SHA-256: `bbb995c65af5cd61b18ece6224dd5a1692490b4fa9fc8b318d25a1fe7585bb1d`. Все 13 local gates PASS, каждый record имеет тот же source и `sourceUnchanged=true`: check, integration, dependencies, source, licenses, browser, images, fuzz, ingress, upgrade, performance, compatibility, verifier. Node 24.21.0 на macOS ARM64; Compose/browser/scans используют Linux containers.

- 45 toolchain tests PASS; clean npm ci и полный peer tree PASS; без peer override flags.
- Browser: 5 profiles × 5 scenarios = 25 expected, 0 unexpected/flaky/skipped, сохранённые visual baselines.
- npm audit: 0 vulnerabilities. Govulncheck: 0 affected symbols/imported packages. Trivy source: 0 HIGH/CRITICAL; прежний GO-2026-5932 UNKNOWN относится к неимпортируемому openpgp, остаётся видимым в отчёте и не исключён.
- Пять runtime images: 0 HIGH/CRITICAL. Web, Redis, PostgreSQL: 0 findings; другие severities и полный SBOM сохранены в raw reports. Runtime digests не выдаются за security proof без scan.
- Local inventory: 434 npm entries / 24 Go libraries / 16 owner-review entries. Классификация не закрывает юридическую приёмку.
- Performance — стандартный compact profile; целевой VPS, multi-GB release profile и внешние решения не подтверждаются этим прогоном.

Логи/records: ignored `.cache/dependabot-review/local-gates/` и `local-evidence/`. CI проверяет тот же source отдельно на Linux и на конкретном Git commit; актуальные ссылки и итог — [session](session.md). Исторические failed Dependabot runs остаются доступными и не переименованы в успешные.

Интеграционный [PR #19](https://github.com/elnemir/p2p-files/pull/19) содержит общий diff, окончательный merge/CI статус и ссылки на скачанные/проверенные exact-commit evidence. Исходные #9/#11 переносятся совместно, #10 заменяется согласованным обновлением, #3/#5 отклоняются по причинам выше; они не выдаются за merged.
