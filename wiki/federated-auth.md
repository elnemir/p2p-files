# Внешний вход и регистрация: OIDC, SAML и LDAP

## Контекст и проблема
FileX сохраняет локальные users/roles/sessions, но проверяет внешнюю identity через выбранный ADMIN провайдер. Это позволяет сотруднику войти корпоративным аккаунтом и, при разрешении администратора, создать личный кабинет без отдельного пароля. Стек и one-time выдача файлов не меняются. Решение: [ADR-014](adr/014-federated-auth.md). Фактические результаты и граница проверки: [session](session.md), [квалификация 2026-10-13](history/federated-auth-evidence-2026-10-13.md).

## Настройка администратора

[Фото/видеотур от 2026-10-14](screen-tour.md) показывает формы трёх протоколов и полный локальный SAML-вход с созданием USER. Учебная IdP-страница явно обозначена; одноразовый Go signer возвращает подписанное подтверждение на настоящий callback FileX. OIDC/LDAP показаны как несохранённые примеры конфигурации, без контакта с внешним каталогом. Это не приёмка Keycloak, Entra ID/AD или Google Workspace: реальные tenants проверяются по этому runbook отдельно.

1. Войти локальным ADMIN, раскрыть «Подтвердить личность» в ADMIN или открыть «Настройки → Аккаунт и безопасность» (подтверждение действует 5 минут).
2. Открыть «Администрирование → Корпоративный вход → SAML / OIDC / LDAP». Задать название кнопки, протокол и поля по таблицам ниже. Новый провайдер выключен; JIT-регистрация тоже выключена.
3. Сохранить, скопировать provider UUID и callback/metadata URL. Внешний IdP настраивается отдельно, без wildcard redirect URLs. Не изменять publicOrigin ради обхода TLS: смена origin имеет штатный restart/recovery guard.
4. Разрешить вход. Если нужны новые аккаунты, явно включить «Создавать USER при первом входе». Указать допустимые email-домены. Проверить вход в отдельном браузерном контексте, сохранив локальную ADMIN-сессию для восстановления.
5. При изменении секретов включить «Заменить сохранённый секрет». Пустое значение очищает секрет, отсутствие поля сохраняет. API возвращает только hasSecret, не сам секрет.

Любое сохранение увеличивает revision и отзывает сессии этого провайдера и незавершённые SSO-переходы. Коллизия ревизий возвращает 409: обновить список и открыть провайдера заново. Issuer/clientId, LDAP authority и stable/email attribute mapping существующего провайдера не переопределяются: другой authority требует нового provider UUID. Metadata certificate можно обновить с сохранением entityID. Старые привязки сохраняются при отключении провайдера; удаление внешних данных не является удалением пользователя.

## OIDC: общий профиль и конкретные сервисы
Используется Authorization Code + PKCE S256, confidential client secret при необходимости, scopes `openid email`. Issuer проверяется точно, включая путь. Все backchannel endpoints требуют HTTPS; redirects запрещены. Дополнительные origins разрешаются только отдельным списком (без путей). Токены проверяются библиотекой go-oidc с разрешёнными асимметричными алгоритмами; issuer/audience/azp/expiry/iat/sub/nonce не отключаются.

| Провайдер | Настройки FileX / настройки IdP |
|---|---|
| Keycloak | Issuer `https://sso.example.test/realms/company`; Client ID отдельного confidential OIDC client. Standard flow включён, exact redirect `https://files.example.test/api/v1/auth/providers/UUID/callback`, PKCE S256. Email claim `email`, email_verified=true; выдавать auth_time. Встроенные role/groups не импортируются. |
| Microsoft Entra ID | Issuer `https://login.microsoftonline.com/TENANT_UUID/v2.0`, конкретный tenant, не `common/organizations`. App registration → Web → exact callback. Client ID и значение client secret. Настроить optional claims `email` и `auth_time`; поле email должно быть корректным email. Если tenant не выдаёт email_verified, ADMIN может явно включить доверие каталогу и обязан заполнить allowed email domains. Никогда не использовать email/preferred_username как stable ID: связь остаётся по `sub`. |
| Google Workspace | Issuer `https://accounts.google.com`; Web OAuth client, client ID/secret, exact callback. Extra origins: `https://oauth2.googleapis.com`, `https://www.googleapis.com`. Email claim `email`, trustEmail выключен. Для доступа только организации задать Hosted Domain: сервер проверяет подписанный `hd`, а не только домен email. При необходимости отдельно ограничить OAuth app организацией в Google Cloud. |
| Другой OIDC IdP | Точный HTTPS issuer, client ID, callback, email claim, metadata endpoints и выдача нужных claims. Несовместимый/отсутствующий claim отклоняется; автоматической подмены issuer или проверки подписи нет. |

Обычный вход допускает valid ID token без свежего auth_time, но такой сеанс **не получает** свежую privileged reauth. Для reauth запрашиваются max_age=0 и auth_time essential; обычно prompt=login, у Google — поддерживаемый select_account. Сервер проверяет реальное auth_time, не считает account chooser/новый iat подтверждением пароля. Если IdP не выдаёт свежий auth_time, повторная авторизация отклоняется: требуется свежая аутентификация в IdP либо настроенный SAML ForceAuthn. Для локального аккаунта доступен локальный пароль. Это явная граница совместимости, не silently weakened step-up.

`trustEmail` — явно выбранное доверие корпоративному каталогу при отсутствии email_verified; с пустым списком допустимых доменов конфигурация запрещена. Это не связывает существующие аккаунты и не предоставляет ADMIN. Изменения email внешнего аккаунта не переписывают автоматически email локального пользователя.

## SAML 2.0
Провайдер получает XML metadata с сертификатом подписи **как текст** из доверенного источника администратора; произвольная серверная загрузка metadata URL не выполняется.

- FileX entityID и metadata: `https://files.example.test/api/v1/auth/providers/UUID/metadata`.
- ACS: `https://files.example.test/api/v1/auth/providers/UUID/saml`, HTTP-POST.
- IdP SSO: HTTPS HTTP-Redirect endpoint из metadata.
- Persistent NameID со стабильным уникальным значением; не transient и не изменяемый email. Email Attribute по точному имени (например `mail` либо полное claim URI у Entra).
- Требуются подписанная response/assertion с SHA-256/384/512, корректные issuer, audience, destination, recipient, request ID и сроки; ForceAuthn и свежий AuthnInstant.

В Keycloak создать SAML client с Client ID=entityID, ACS POST, persistent NameID и mapper email. В Entra Enterprise Application → SAML указать Identifier/Reply URL, настроить persistent NameID на стабильный ID объекта и email claim; скопировать federation metadata. В Google Workspace custom SAML app указать Entity ID/ACS, persistent NameID на неизменяемый идентификатор каталога и mapping email. Возможность этих настроек и фактический flow проверяются на конкретном tenant: наличие универсального adapter не является внешней приёмкой каждого сервиса.

Поддерживается только SP-initiated вход. IdP-initiated, artifact binding, SLO, request signing и encrypted assertions не поддерживаются этим профилем; metadata не рекламирует несуществующие signing/encryption keys. SAML assertions идут через HTTPS, secrets из них не сохраняются в logs. После POST браузер возвращается на FileX и завершает вход same-origin запросом: основная cookie остаётся SameSite=Strict.

## LDAP: Active Directory и OpenLDAP
| Поле | Active Directory | OpenLDAP |
|---|---|---|
| URL | `ldaps://dc.example.test:636` | `ldaps://ldap.example.test:636` либо `ldap://ldap.example.test:389` с обязательным StartTLS |
| Base DN | `DC=example,DC=test` | `ou=people,dc=example,dc=test` |
| Service bind DN | DN отдельного read-only service account | DN отдельного read-only service account |
| Login attribute | `sAMAccountName` либо `userPrincipalName` | `uid` |
| Subject attribute | `objectGUID` (16 binary bytes, нормализуется Base64url) | `entryUUID` |
| Email attribute | `mail` | `mail` |

В FileX вводится пароль **service bind**; пользовательский пароль принимается только при входе и не сохраняется. Порядок: проверенный TLS → service bind → subtree search с escaped literal login, без referrals/alias dereference и максимум двумя результатами → ровно один DN со stable ID и одним email → user bind с непустым паролем → локальная session. Пустой пароль/анонимный bind не допускаются. Списки групп и права AD не переносятся в роли FileX.

Для внутреннего каталога ADMIN явно задаёт private CIDR назначения, например `10.20.30.0/24`, и дополнительную CA PEM при внутреннем PKI. Проверяются hostname/certificate chain, TLS≥1.2. Loopback, link-local/cloud metadata, multicast/unspecified запрещены даже при попытке CIDR-разрешения. DNS проверяется при каждом dial; соединение устанавливается с проверенным IP без повторного разрешения. Указание private CIDR не меняет входящий networkPolicy FileX.

## Регистрация, связь и reauth
Внешняя identity — `(provider UUID, subject)`. При первом входе включённый JIT создаёт локальный USER без password_hash. Если email уже занят, вход отклоняется общим сообщением: автоматическое присвоение существующего аккаунта запрещено. Пользователь сначала входит существующим способом, подтверждает текущий аккаунт, затем в «Настройки → Аккаунт и безопасность → Корпоративные аккаунты» выбирает «Связать». Внешняя identity уже связанного другого пользователя не переназначается. Один provider связывается с одним subject для каждого локального аккаунта.

«Подтвердить через …» требует действующую исходную session/epoch и ту же внешнюю identity. Успех создаёт новую session и CSRF, отзывает прежнюю session; ошибка не повышает privileges. Локальные role/disabled/auth_epoch проверяются повторно в transaction. Внешние claims никогда не создают ADMIN. Logout отзывает локальную сессию, но не upstream IdP session.

## Данные, API и эксплуатация
Migration 0019: identity_providers (config/revision/encrypted secret), external_identities, identity_flows (TTL 5 минут, state/binding digests, encrypted verifier/proof), web_sessions.identity_provider_id. Current provider revision и local authority проверяются при завершении в одной PG transaction вместе с созданием аккаунта/session и audit. Одновременные callback/complete не создают повторную сессию. При audit fault нет частично созданного пользователя.

Configuration DEK используется через существующий keyring с AAD `filex/identity_provider/v1/UUID` и `filex/identity_flow/v1/STATE_HASH`. Rotation переносит provider secrets; временные flows истекают и удаляются mandatory cleanup, старые key versions сохраняются штатно. Backup catalog автоматически включает новые таблицы, crypto reconciliation проверяет envelopes. Restore инвалидирует все pending flows и web sessions. Новых env нет; внешние endpoints/CA/secrets хранятся в БД. AsyncAPI не меняется, auth события не публикуются в owner notifications.

Контракты — [OpenAPI](../contracts/openapi.json) и [provider schema](../contracts/schemas/identity-provider-v1.json). Основные endpoints: публичный `GET /auth/providers`, ADMIN `GET/POST /admin/identity-providers`, CAS `PUT /admin/identity-providers/{id}`, `POST /auth/providers/{id}/start`, OIDC callback, SAML POST/metadata, LDAP POST, `POST /auth/complete`, `GET /auth/identities`. Все пути имеют prefix `/api/v1`. Cookie mutations требуют Origin/CSRF; anonymous start/complete/LDAP требуют exact Origin; только проверяемый SAML cross-site callback освобождён от Origin. API secrets write-only; публичный список не выдаёт issuer/config/identity bindings.

## Ограничения и продолжение
Это не SCIM/directory sync, multi-tenancy или group mapping. Отключение пользователя в IdP влияет на следующий вход/reauth; локальная сессия остаётся до TTL, локального отзыва/disabled/auth_epoch либо обновления провайдера. Bearer API tokens следуют прежнему самостоятельному lifecycle и не являются IdP sessions; при компрометации локальный ADMIN отзывает аккаунт/токены. Локальный ADMIN fallback сохраняется всегда.

Настоящие tenants/DC, CA и разрешённые тестовые аккаунты ещё не предоставлены; результат локальных fixtures не выдаётся за внешнюю приёмку. До подключения проверить claims, clocks, certificate rotation, logout expectations, emergency local login и закрытие доступа disabled пользователю. Windows WebDAV и прежние production blockers остаются вне этой задачи.

## Первичные источники
[Google OIDC reference](https://developers.google.com/identity/openid-connect/reference), [Entra ID token claims](https://learn.microsoft.com/en-us/entra/identity-platform/id-token-claims-reference), [Entra optional claims](https://learn.microsoft.com/en-us/entra/identity-platform/optional-claims-reference), [go-oidc](https://github.com/coreos/go-oidc), [crewjam/saml](https://github.com/crewjam/saml), [go-ldap](https://github.com/go-ldap/ldap). Проверены при проектировании 2026-10-13; версии зависимостей закреплены в go.mod/go.sum, security evidence относится к текущему source из session.

## Уточнения проверки конфигурации и жизненного цикла

API требует все поля `IdentityConfig` согласно JSON Schema: отсутствие поля, `null` или неизвестное поле отклоняются. Для SAML metadata необходим хотя бы один signing certificate с RSA ≥2048 или EC ≥256; сертификаты являются явно заданными pin ключами IdP, а не TLS trust chain. Несколько Assertion в одном ответе отклоняются до signature parsing. Ротацию сертификатов выполняет ADMIN заменой metadata при неизменном entityID.

Сохранение настроек провайдера отзывает и собственную ADMIN-сессию, если она выдана этим провайдером. UI сообщает об успешном сохранении даже если последующее чтение списка требует нового входа. Во время recovery/maintenance внешние auth endpoints не заменяют локальный аварийный ADMIN-вход; восстановление очищает незавершённые flows, сохраняя provider secrets и постоянные связи аккаунтов.

Кнопки провайдеров переносят даже непрерывное имя длиной 80 символов внутри панели. Browser regression проверяет editor и public login на ширине 390px; название не расширяет весь ADMIN/operations экран.

## Лимиты и диагностика

В установке допускается до 32 записей провайдеров, включая выключенные; hard delete и переназначение их identity namespace не реализованы. Имя кнопки — до 80 Unicode символов, secret — до 4096 bytes, CA PEM — до 16384 bytes, SAML metadata — до 45000 bytes. Списки ограничены 100 email domains, 16 private CIDR и 8 additional origins. API отклоняет несоответствие конфигурации; при CAS conflict нужно перечитать текущую revision.

SSO flow живёт 5 минут, одновременно допускается до 10000 pending/verified flows; expired rows удаляются при старте и периодической cleanup. Внешние HTTP ответы ограничены 2 MiB и 12 секундами; TLS/dial имеют собственные deadlines. LDAP требует завершить TLS/search/bind в пределах общего 12-секундного deadline. Authentication endpoints используют существующий durable rate limit; LDAP дополнительно ограничивает provider+username. Пользователю возвращается общий отказ, без раскрытия наличия email или upstream token/error. Для диагностики сверять поля provider, доверенные origins/CA/CIDR, exact callback и claims на стороне IdP, не записывая пользовательский пароль или token в logs.
