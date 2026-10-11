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
