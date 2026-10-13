# ADR-014: внешняя аутентификация и регистрация

## Контекст и проблема
Владелец запросил SAML/OIDC/LDAP. FileX имеет локальные users, web_sessions, auth_epoch и ADMIN reauth. Требуется вход через корпоративный источник и создание аккаунта при первом входе без изменения локальных ролей/защищённых инвариантов. Это дополнение SEC-001/003/005, OPS-001/003, NFR-001/003.

## Решение и поток данных
ADMIN со свежим reauth → typed provider config + encrypted secret → revision CAS/audit в PG. Публичный список содержит только ID, название, protocol и признак регистрации. Провайдер отключён по умолчанию, регистрация включается явно. Идентификатор учётной записи — provider UUID + неизменный subject; email не является ключом привязки. Изменение issuer/LDAP базы/attribute mapping после создания запрещено: для другого authority создаётся новый provider.

OIDC: Authorization Code + PKCE S256, state, nonce, issuer/audience/signature/expiry/azp/sub/email policy/auth_time. SAML: SP-initiated Redirect request / POST signed response, pinned metadata, issuer/audience/destination/request ID/expiry/signature; unsolicited/artifact flow запрещён. LDAP: TLS (LDAPS или StartTLS), service bind → escaped search одного объекта → непустой password user bind; stable entryUUID/objectGUID и единственный mail. Никаких ролей из groups/claims.

SSO start POST с exact Origin → PG flow TTL 5 минут и HttpOnly SameSite=Lax browser nonce → внешняя проверка → pending verified identity → фиксированный возврат на корень → same-origin POST complete с nonce → атомарные consume/JIT/link/session/audit. SAML POST не требует ослаблять SameSite основной сессии: завершение выполняется браузером после возврата. Raw tokens/password/assertions не попадают в URL приложения, logs или persistent storage; flow verifier/pending identity шифруются configuration DEK.

Первый вход создаёт USER без локального password только при JIT и отсутствии коллизии email. Коллизия отклоняется общим отказом. Уже вошедший пользователь может явно связать внешний аккаунт после свежей reauth. Обычный OIDC login без свежего auth_time не даёт fresh reauth; Google использует select_account и auth_time essential, Entra допускает явное trustEmail только с domain allowlist. Reauth через провайдера требует ту же identity, прежнюю действующую session/epoch и свежую проверку IdP; замена пользователя запрещена. Обычный logout локальный; завершение upstream IdP session не обещается.

## State machine
Provider: disabled ↔ enabled; update revision инвалидирует ожидающие flows и выданные через provider sessions. Flow: pending → verified → consumed; invalid/expired/revision-changed → denied. Повторный callback/complete не создаёт вторую session. Restore инвалидирует все flows и sessions. Local disabled/auth_epoch/role остаются authoritative.

## Threat model
Login CSRF: exact Origin start/complete, 256-bit state + отдельная browser nonce. Replay/mix-up: provider/revision/origin/purpose/request ID и atomic consume. Account takeover: no email auto-link, verified OIDC email либо explicit directory trust + domain allowlist, explicit trusted SAML/LDAP directory, no ADMIN claims. SSRF: HTTPS, bounded responses/timeouts, no redirects, DNS address checks at dial; private address доступен только в явно заданном ADMIN CIDR, loopback/link-local/metadata запрещены. LDAP filter injection: EscapeFilter, DN берётся только из единственного search result; empty bind запрещён. XML signature wrapping проверяется protocol library плюс обязательные subject/audience/confirmation и запрет SHA-1; unsigned/IdP-initiated responses отвергаются. Secrets не возвращаются, raw error от IdP не публикуется.

## Architecture consistency и crypto review
PG — authority, без Redis correctness и нового сервиса. Configuration DEK/keyring.Seal с отдельными AAD для provider/flow, durable nonce budget и active-key guard; криптографические primitives не добавляются. Подписи OIDC/XML выполняются библиотеками, не собственной реализацией. Recipient выдача, report hashes, backup semantics и ADMIN/USER privacy не меняются. AsyncAPI не меняется: authentication события не публикуются в owner channels. API и схемы создаются до реализации.

## Ограничения и приёмка
Нет SCIM, group-to-role mapping, автоматического directory sync и IdP logout. Отзыв upstream account влияет на следующий вход/reauth; существующие локальные sessions живут до TTL/локального отзыва/отключения провайдера. SAML использует подписанные незашифрованные assertions поверх TLS; encrypted assertions и request signing не входят в этот профиль. Предварительная проверка metadata и TLS нужна до enable. Локальный ADMIN остаётся путём восстановления.

Нужны negative/concurrency/fault проверки signature/issuer/audience/nonce/state/replay/email collision/disabled/revision/recovery, LDAP TLS/bind/filter, bounded outbound, secret redaction; UI login/link/reauth/provider lifecycle и browser roundtrip. Фактические результаты — [session](../session.md), а не этот design.
