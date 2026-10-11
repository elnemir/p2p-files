# Third-party license review — требуется решение владельца

## Контекст и источник
Инвентаризация для кандидата фазы 7 берётся из package-lock.json (все платформы/dev/optional), go-licenses runtime report и bundled font LICENSE. Это машинная классификация по метаданным, не юридическое заключение и не разрешение поставки. Code proprietary сохраняется.

## Записи вне базового permissive набора
На 2026-10-11 выявлены следующие записи; optional платформы могут отсутствовать в конкретном runtime image. Их наличие в lockfile не означает, что все перечисленные бинарники поставляются одному пользователю.

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
