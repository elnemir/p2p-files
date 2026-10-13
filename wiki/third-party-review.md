# Third-party license review — требуется решение владельца

## Контекст и источник
Инвентаризация для кандидата фазы 7 берётся из package-lock.json (все платформы/dev/optional), go-licenses runtime report и bundled font LICENSE. Это машинная классификация по метаданным, не юридическое заключение и не разрешение поставки. Code proprietary сохраняется.

## Записи вне базового permissive набора
Для кандидата 7dce226 на 2026-10-11 выявлены следующие записи; optional платформы могут отсутствовать в конкретном runtime image. Их наличие в lockfile не означает, что все перечисленные бинарники поставляются одному пользователю.

| Package | Version | Reported license | Scope |
|---|---|---|---|
| @axe-core/playwright | 4.13.0 | MPL-2.0 | dev |
| @img/sharp-libvips-darwin-arm64 | 1.3.4 | LGPL-3.0-or-later | runtime/optional |
| @img/sharp-libvips-darwin-x64 | 1.3.4 | LGPL-3.0-or-later | runtime/optional |
| @img/sharp-libvips-linux-arm | 1.3.4 | LGPL-3.0-or-later | runtime/optional |
| @img/sharp-libvips-linux-arm64 | 1.3.4 | LGPL-3.0-or-later | runtime/optional |
| @img/sharp-libvips-linux-ppc64 | 1.3.4 | LGPL-3.0-or-later | runtime/optional |
| @img/sharp-libvips-linux-riscv64 | 1.3.4 | LGPL-3.0-or-later | runtime/optional |
| @img/sharp-libvips-linux-s390x | 1.3.4 | LGPL-3.0-or-later | runtime/optional |
| @img/sharp-libvips-linux-x64 | 1.3.4 | LGPL-3.0-or-later | runtime/optional |
| @img/sharp-libvips-linuxmusl-arm64 | 1.3.4 | LGPL-3.0-or-later | runtime/optional |
| @img/sharp-libvips-linuxmusl-x64 | 1.3.4 | LGPL-3.0-or-later | runtime/optional |
| @img/sharp-wasm32 | 0.35.5 | Apache-2.0 AND LGPL-3.0-or-later AND MIT | runtime/optional |
| @img/sharp-win32-arm64 | 0.35.5 | Apache-2.0 AND LGPL-3.0-or-later | runtime/optional |
| @img/sharp-win32-ia32 | 0.35.5 | Apache-2.0 AND LGPL-3.0-or-later | runtime/optional |
| @img/sharp-win32-x64 | 0.35.5 | Apache-2.0 AND LGPL-3.0-or-later | runtime/optional |
| axe-core | 4.13.0 | MPL-2.0 | dev |

## Поток решения и ограничения
`python3 scripts/supply_chain.py licenses` → `.cache/evidence/supply-chain/licenses.json` и go-licenses.csv. Source и пять container SBOM создаются отдельными scans; OS/image licenses нужно оценить по точным поставляемым digests, они не исчерпываются таблицей npm. Bundled DejaVu font имеет отдельный LICENSE и SHA-256 в inventory.

Владелец/reviewer определяет допустимость поставки, необходимые notices/source obligations либо замену зависимости. Решение и evidence записываются в `contracts/release-acceptance.json` для `third-party-licenses` только после фактического подтверждения. До этого release blocked. Обновление dependency требует повторной инвентаризации, scan и функциональных gates. Полные отчёты CI хранятся 14 дней; для принятого релиза их требуется сохранить вместе с immutable artifacts дольше этого срока.

## Сопровождение поставки

Каноническая карточка подготовки — DEV-03 в [плане](implementation-plan.md). В корне репозитория нет LICENSE: proprietary политика сохранена в AGENTS, правообладателя и конкретные условия должен определить владелец. Этот документ не подменяет ни это решение, ни third-party acceptance. Inventory обновляется на окончательном candidate/digests; числовой срез выше не переносится на новые lockfiles автоматически.

## Redis 8 при объединении веток

PR #3 предлагает Redis 8.10 вместо текущего 7.2. По [официальной таблице Redis](https://redis.io/legal/licenses/) ветка 7.2 поставляется под BSD-3-Clause, а 8+ предлагает выбор RSALv2, SSPLv1 или AGPLv3. Автоматический dependency bump не является принятием условий для proprietary FileX. Поэтому предложение не включено в общий кандидат: владелец/reviewer должен выбрать допустимый вариант и необходимые обязательства либо сохранить/заменить компонент в рамках согласованного стека. Статус `third-party-licenses` остаётся pending.

Исходный patch #3 не собирался с прежними OpenSSL apk pins. Отдельный review image Redis 8.10.2 уже собран с libcrypto3/libssl3 3.5.8-r0 и прошёл Trivy с 0 findings; Dockerfile, scan и SBOM сохранены в ignored `.cache/consolidation/redis8-review/`. После лицензионного решения нужны перенос проверенных pins и полная функциональная/DR квалификация. Наличие Redis только в ephemeral роли не отменяет review состава поставки. [Учёт всех веток](branch-consolidation.md).

## Срез после исправления CI

Для source `8390f72c6ac08927e609dc9eccfb78d93ebc7fbbdae8363c47e449cf50feb210` новый локальный inventory содержит 443 npm packages, 24 Go library records и 16 owner-review entries. Скачанный Linux CI artifact [38172130775](https://github.com/elnemir/p2p-files/actions/runs/38172130775) имеет 443 / 25 / 16: в go-licenses.csv дополнительно присутствует `golang.org/x/sys/cpu` (reported BSD-3-Clause). Это наблюдаемая разница между host и CI инвентаризацией; число локальных библиотек нельзя переносить на состав всех платформ. Для DEV-03 использовать конкретные target/image IDs, SBOM и license files, а не только суммарный счётчик. Перечень owner-review не считается принятым из-за PASS машинного gate.

## Решение по сопровождению Redis и lint 2026-10-12

Для текущего исправления Dependabot выбран актуальный Redis 7.2.16 BSD-3-Clause; upstream предоставляет security fixes до 2029-12-01. Это поддерживаемая линия Redis Open Source, её нельзя путать с Enterprise Software. Срок контролируется dependency gate. Redis 8 не включён и его условия не приняты. ESLint JSX-a11y заменяется MIT-пакетом `eslint-plugin-jsx-a11y-x` 0.2.0 от ESLint Community; Свежий локальный inventory source `bbb995c65af5cd61b18ece6224dd5a1692490b4fa9fc8b318d25a1fe7585bb1d`: 434 npm entries, 24 Go libraries, прежние 16 owner-review entries. Это машинная классификация, не приёмка поставки; target Linux inventory проверяется отдельно в CI. Полный процесс/ограничения: [dependency remediation](dependency-remediation.md). Общий release license review остаётся pending.

## Native WebDAV test tools
GVfs/gio в отдельном Ubuntu Docker image используется только для квалификации Linux клиента; он не входит в пять runtime images FileX и не включён в продуктовые containers. Base image digest закреплён в [test Dockerfile](../scripts/webdav/Dockerfile), package/backend версии записываются тестом. macOS mount_webdav и Windows WebClient — клиентские компоненты тестовых ОС, не vendored binaries. Лицензионные решения для поставки продукта остаются отдельной внешней приёмкой.

Итог native WebDAV qualification 2026-10-12: source и все пять runtime images имеют 0 HIGH/CRITICAL; `GO-2026-5932` UNKNOWN остаётся видимым в source x/crypto v0.58.0, API/CLI binary locations v0.58.0 и Traefik v0.57.0. Govulncheck FileX: 0 reachable/0 imported-package findings; это не binary reachability проверка стороннего Traefik. Scanner exceptions не добавлены. Exact source и границы — [session](session.md).

## Протокольные зависимости SSO, 2026-10-13

Добавлены закреплённые в go.mod/go.sum runtime libraries: go-oidc v3.21.0 (Apache-2.0), crewjam/saml v0.5.1 (BSD-2-Clause), go-ldap v3.4.15 (MIT). XML signature parser использует актуализированные etree v1.8.1 (BSD-2-Clause) и goxmldsig v1.6.1 (Apache-2.0); OIDC — go-jose v4.1.4 (Apache-2.0, вложенный json BSD-3-Clause) и oauth2 v0.36.0 (BSD-3-Clause). Источник — текущий go-licenses inventory, версии не выводятся из старых evidence. Новые crypto primitives в FileX не добавляются.

Инвентаризация кандидата содержит 37 Go library entries и 434 npm packages; 16 записей требуют owner review согласно существующей политике. Это техническая инвентаризация, не юридическое принятие обязательств. Source/container UNKNOWN GO-2026-5932 остаётся видимым; FileX govulncheck 0 imported-package/symbol findings не является binary reachability review Traefik. Фактические hashes/gates — [session](session.md).
