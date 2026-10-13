# SAML/OIDC/LDAP: локальная квалификация 2026-10-13

## Контекст и граница
Ветка `codex/federated-auth` от main `73bf030102e17a2759ee48640a27cf3e4b378d53`. Измерения относятся к незакоммиченному кандидату с source SHA-256 `70ada47d3ef41da8300ef649282149af202abb915aebfbdf9b0699cc0a3c80fb`. Commit/push/GitHub CI и production deployment не входят в этот результат. Реальные tenant/DC и тестовые аккаунты не предоставлены. [Руководство](../federated-auth.md), [ADR-014](../adr/014-federated-auth.md).

## Gates
Все 13 `.cache/evidence/gates/<gate>.json`: passed=true, sourceUnchanged=true, exitCode=0, одинаковый source hash. Последовательный batch: `.cache/identity/qualification.log`; raw logs: `.cache/identity/gate-<gate>.log`. Markdown не входит в source fingerprint согласно acceptance.py; после обновления wiki выполняются docs-check и diff-check.

| Gate | Результат | Секунды |
|---|---|---|
| check | PASS | 24.1 |
| integration | PASS | 177.6 |
| dependencies | PASS | 5.4 |
| source | PASS | 2.4 |
| licenses | PASS | 3.3 |
| images | PASS | 23.3 |
| fuzz | PASS | 88.3 |
| browser | PASS | 396.9 |
| ingress | PASS | 14.3 |
| upgrade | PASS | 20.9 |
| performance | PASS | 32.7 |
| compatibility | PASS | 0.1 |
| verifier | PASS | 5.1 |

## Подтверждено
- 61 PostgreSQL/race integration case, 0 skips/FAIL, включая rclone. Пять SSO случаев проверяют JIT USER, отсутствие email auto-link, неизменный subject, local disabled/role/session authority, 16 конкурентных complete, expiry, revoked actor, запрет регистрации, CAS, secret redaction, provider revoke и audit rollback.
- Подписанные OIDC/SAML fixtures и TLS/StartTLS LDAP BER server: nonce/issuer/audience/azp/expiry/auth_time/hd, подписи/постоянный subject/request ID, malformed XML/duplicate assertions/legacy hash, escaped LDAP filter, пустой/ошибочный bind, неоднозначный search, CA trust и bounded response. Это реальные протокольные обмены с локальными fixtures, не проверка внешнего tenant.
- Recovery backup/restore сохраняет provider secrets и инвалидирует pending flows; configuration rotation переносит provider secret на новый ключ; crypto reconciliation понимает новые envelopes.
- 45 browser cases: Chromium, Firefox, WebKit, mobile Chromium, mobile WebKit, по 9. Во всех results.json expected=9, skipped=0, unexpected=0, flaky=0. Полный SAML SP→IdP→cross-site POST→ACS→same-origin completion; JIT/reauth/replay/provider revoke и ADMIN config. API FileX настоящий; перехватывается только отдельный тестовый IdP. Настройки, axe и 390px layout с 80-символьным непрерывным именем проверены. Финальный Chromium screenshot просмотрен визуально; перенос корректен. Visual baselines не менялись.
- 122 OpenAPI operations, AsyncAPI/schema/generated types и traceability согласованы. Schema 19; controlled upgrade 16→19 проверяет точный ledger/checksums, wrong-KEK/unknown-migration/downgrade guards, несовместимость прежнего backup и explicit reopen.
- Пять fuzz campaigns PASS. HTTPS ingress PASS с TLS1.2/CA/HSTS/cookies/Host/proxy/CIDR/WebDAV guards, без обращения к public ACME. Performance: 32 MiB / 2000 rows smoke, synthetic TLS S3 и DR; не multi-GB release или target-host load.
- npm audit 0; govulncheck 0 symbol/imported-package findings. Source и пять runtime images HIGH/CRITICAL=0; Gitleaks history/worktree PASS. Прежний GO-2026-5932 UNKNOWN остаётся видимым в source/API/Traefik inventory, без scanner exceptions. FileX reachability результат не является анализом стороннего Traefik binary. [Лицензионный inventory](../third-party-review.md) не заменяет owner acceptance.
- Verifier cross-build для пяти targets PASS; это не запуск на каждой реальной ОС и не опубликованный release.

## Найдено и исправлено
1. Новый AAD сначала нарушал существующую схему cryptobox: исправлены классы identity_provider/identity_flow без изменения primitive; PG tests подтвердили decrypt/restore/rotation.
2. Completion transaction читала ключ через отдельное pool connection. Чтение перенесено в ту же transaction; 16 конкурирующих complete дают ровно одну session без pool starvation.
3. Accessible name select включал option text: добавлены явные aria-label; browser lookup/axe PASS.
4. Migration regression сохранял прежний count 18 и конфликтующие номера test-only SQL. Обновлены count/prefix/fixtures для 19; реальные DDL rollback/drain/checksum/downgrade guards сохранены.
5. 80-символьное имя провайдера расширяло mobile ADMIN и следующий operations экран. RED воспроизведён в mobile WebKit, добавлен scoped wrapping; финальные 45 cases PASS. Assertions, retry policy и timeout для обхода ошибок не ослаблялись.
6. Успешное сохранение собственного SSO provider отзывает ADMIN session: UI теперь отличает успешную запись от ошибки последующего reload.

Первичные ошибки и targeted logs сохранены в `.cache/identity/`: `failed-migration-fixture.log`, `browser-long-name-red.log`, `long-name-red-traces/`, `filex-identity-*.log`. Промежуточные прогоны не объявляются финальным source-bound evidence.

## Не подтверждено и продолжение
Подключение конкретных Keycloak, Entra ID, Google Workspace, AD/OpenLDAP требует настоящих endpoints, metadata/CA и тестовых аккаунтов; независимый security review остаётся внешней приёмкой. SAML profile ограничен SP-initiated, unsigned AuthnRequest и незашифрованными подписанными response/assertions поверх TLS; SLO/SCIM/group mapping не добавлены. OIDC без свежего auth_time не даёт privileged reauth. Existing production blockers, owner license/risk decisions и Windows native WebDAV **ожидает стенда** сохраняются. Существующие фото/видеотуры не переснимались в этой задаче и не считаются SSO evidence.
