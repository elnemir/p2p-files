"use client";

import { useEffect, useState } from "react";
import type { Schema } from "../lib/api-types";

type Config = Schema<"IdentityConfig">;
type Provider = Schema<"IdentityProvider">;
const defaults: Config = {
  trustEmail: false,
  hostedDomain: "",
  name: "",
  protocol: "oidc",
  enabled: false,
  allowRegistration: false,
  allowedEmailDomains: [],
  privateCidrs: [],
  caPem: "",
  issuer: "",
  clientId: "",
  extraOrigins: [],
  metadataXml: "",
  emailAttribute: "email",
  ldapUrl: "",
  baseDn: "",
  bindDn: "",
  loginAttribute: "uid",
  subjectAttribute: "entryUUID",
};
type Props = {
  api: <T>(path: string, method?: string, data?: unknown) => Promise<T>;
  t: (ru: string, en: string) => string;
  busy: boolean;
};
export default function IdentityProvidersPanel({ api, t, busy }: Props) {
  const [items, setItems] = useState<Provider[]>([]);
  const [selected, setSelected] = useState<Provider | null>(null);
  const [config, setConfig] = useState<Config>({ ...defaults });
  const [secret, setSecret] = useState("");
  const [replaceSecret, setReplaceSecret] = useState(false);
  const [working, setWorking] = useState(false);
  const [message, setMessage] = useState("");
  const disabled = busy || working;
  async function reload() {
    setItems(await api<Provider[]>("/admin/identity-providers"));
  }
  useEffect(() => {
    void api<Provider[]>("/admin/identity-providers")
      .then(setItems)
      .catch(() =>
        setMessage(
          t(
            "Подтвердите аккаунт и обновите список провайдеров.",
            "Reauthenticate and reload the provider list.",
          ),
        ),
      );
  }, [api, t]);
  function edit(p: Provider | null) {
    setSelected(p);
    setConfig(p ? { ...p.config } : { ...defaults });
    setSecret("");
    setReplaceSecret(false);
    setMessage("");
  }
  function textField(
    key: keyof Config,
    label: string,
    multiline = false,
    immutable = false,
  ) {
    const value = config[key];
    if (typeof value !== "string") return null;
    return (
      <label key={key}>
        {label}
        {multiline ? (
          <textarea
            rows={key === "metadataXml" ? 8 : 4}
            value={value}
            disabled={disabled || (immutable && !!selected)}
            onChange={(e) => setConfig({ ...config, [key]: e.target.value })}
          />
        ) : (
          <input
            value={value}
            disabled={disabled || (immutable && !!selected)}
            onChange={(e) => setConfig({ ...config, [key]: e.target.value })}
          />
        )}
      </label>
    );
  }
  function listField(
    key: "allowedEmailDomains" | "privateCidrs" | "extraOrigins",
    label: string,
  ) {
    return (
      <label>
        {label}
        <textarea
          rows={2}
          value={config[key].join("\n")}
          disabled={disabled}
          onChange={(e) =>
            setConfig({ ...config, [key]: e.target.value.split("\n") })
          }
        />
      </label>
    );
  }
  return (
    <section
      className="panel identity-providers"
      aria-label={t("Провайдеры авторизации", "Identity providers")}
    >
      <h2>{t("SAML / OIDC / LDAP", "SAML / OIDC / LDAP")}</h2>
      <p>
        {t(
          "Первый вход может создавать только USER. Роли остаются локальными; совпадение email не связывает аккаунты. Локальный вход администратора сохраняется.",
          "First sign-in can create USER accounts only. Roles remain local; matching emails never link accounts automatically. Local administrator login remains available.",
        )}
      </p>
      <div className="actions">
        <button
          type="button"
          className="secondary"
          disabled={disabled}
          onClick={() => edit(null)}
        >
          {t("Новый провайдер", "New provider")}
        </button>
        <button
          type="button"
          className="secondary"
          disabled={disabled}
          onClick={() =>
            void reload().catch(() =>
              setMessage(
                t("Не удалось обновить список.", "Could not reload providers."),
              ),
            )
          }
        >
          {t("Обновить провайдеров", "Reload providers")}
        </button>
      </div>
      <ul>
        {items.map((p) => (
          <li key={p.id}>
            <button
              type="button"
              className="secondary"
              disabled={disabled}
              onClick={() => edit(p)}
            >
              {p.config.name} · {p.config.protocol.toUpperCase()} ·{" "}
              {p.config.enabled
                ? t("включён", "enabled")
                : t("отключён", "disabled")}
            </button>
          </li>
        ))}
      </ul>
      <form
        onSubmit={(event) => {
          event.preventDefault();
          setWorking(true);
          setMessage("");
          const value = {
            ...config,
            allowedEmailDomains: config.allowedEmailDomains
              .map((v) => v.trim().toLowerCase())
              .filter(Boolean),
            privateCidrs: config.privateCidrs
              .map((v) => v.trim())
              .filter(Boolean),
            extraOrigins: config.extraOrigins
              .map((v) => v.trim())
              .filter(Boolean),
          };
          const input = {
            expectedRevision: selected?.revision ?? 0,
            config: value,
            ...(!selected || replaceSecret ? { secret } : {}),
          };
          void api<Provider>(
            `/admin/identity-providers${selected ? `/${selected.id}` : ""}`,
            selected ? "PUT" : "POST",
            input,
          )
            .then(async (p) => {
              edit(p);
              try {
                await reload();
              } catch {
                setMessage(
                  t(
                    "Провайдер сохранён. Список недоступен: войдите заново, если ваша сессия была отозвана.",
                    "Provider saved. The list is unavailable: sign in again if your session was revoked.",
                  ),
                );
                return;
              }
              setMessage(
                t(
                  "Провайдер сохранён. Прежние внешние сессии отозваны.",
                  "Provider saved. Previous provider sessions have been revoked.",
                ),
              );
            })
            .catch(() =>
              setMessage(
                t(
                  "Не удалось сохранить. Проверьте поля, свежую повторную авторизацию и ревизию. При конфликте обновите список и откройте провайдера заново.",
                  "Could not save. Check fields, fresh reauthentication and revision. On conflict, reload the list and reopen the provider.",
                ),
              ),
            )
            .finally(() => setWorking(false));
        }}
      >
        <h3>
          {selected
            ? t("Изменить провайдера", "Edit provider")
            : t("Создать провайдера", "Create provider")}
        </h3>
        {textField("name", t("Название кнопки входа", "Sign-in button label"))}
        <label>
          {t("Протокол", "Protocol")}
          <select
            aria-label={t("Протокол", "Protocol")}
            value={config.protocol}
            disabled={disabled || !!selected}
            onChange={(e) =>
              setConfig({
                ...defaults,
                protocol: e.target.value as Config["protocol"],
                emailAttribute: e.target.value === "oidc" ? "email" : "mail",
                name: config.name,
              })
            }
          >
            <option value="oidc">OpenID Connect</option>
            <option value="saml">SAML 2.0</option>
            <option value="ldap">LDAP</option>
          </select>
        </label>
        <label>
          <input
            type="checkbox"
            checked={config.enabled}
            disabled={disabled}
            onChange={(e) =>
              setConfig({ ...config, enabled: e.target.checked })
            }
          />
          {t("Разрешить вход", "Enable sign-in")}
        </label>
        <label>
          <input
            type="checkbox"
            checked={config.allowRegistration}
            disabled={disabled}
            onChange={(e) =>
              setConfig({ ...config, allowRegistration: e.target.checked })
            }
          />
          {t(
            "Создавать USER при первом входе",
            "Register USER on first sign-in",
          )}
        </label>
        {listField(
          "allowedEmailDomains",
          t(
            "Разрешённые email-домены, по одному на строку (пусто — любые)",
            "Allowed email domains, one per line (empty allows any)",
          ),
        )}
        {config.protocol === "oidc" && (
          <>
            {textField("issuer", "Issuer URL (https://)", false, true)}
            {textField("clientId", "Client ID", false, true)}
            {textField("emailAttribute", "Email claim", false, true)}
            {textField(
              "hostedDomain",
              t(
                "Google Workspace: обязательный hd domain",
                "Google Workspace: required hd domain",
              ),
            )}
            <label>
              <input
                type="checkbox"
                checked={config.trustEmail}
                disabled={disabled}
                onChange={(e) =>
                  setConfig({ ...config, trustEmail: e.target.checked })
                }
              />
              {t(
                "Доверять email из корпоративного каталога без email_verified (требует списка email-доменов)",
                "Trust directory email without email_verified (requires allowed email domains)",
              )}
            </label>
            {listField(
              "extraOrigins",
              t(
                "Дополнительные доверенные HTTPS origins для endpoints",
                "Additional trusted HTTPS endpoint origins",
              ),
            )}
          </>
        )}
        {config.protocol === "saml" && (
          <>
            {textField(
              "metadataXml",
              t(
                "Metadata XML провайдера (с сертификатом подписи)",
                "Identity provider metadata XML (with signing certificate)",
              ),
              true,
            )}
            {textField(
              "emailAttribute",
              t("Имя SAML attribute с email", "SAML email attribute name"),
              false,
              true,
            )}
            <p>
              {t(
                "Нужен persistent NameID, подписанная response/assertion и SP-initiated вход. SLO и encrypted assertions не поддерживаются.",
                "Requires persistent NameID, signed response/assertion and SP-initiated sign-in. SLO and encrypted assertions are not supported.",
              )}
            </p>
          </>
        )}
        {config.protocol === "ldap" && (
          <>
            {textField(
              "ldapUrl",
              "LDAP URL (ldaps:// / ldap:// + StartTLS)",
              false,
              true,
            )}
            {textField("baseDn", "Base DN", false, true)}
            {textField("bindDn", "Service bind DN")}
            {textField(
              "loginAttribute",
              t(
                "Атрибут логина (uid / sAMAccountName)",
                "Login attribute (uid / sAMAccountName)",
              ),
            )}
            {textField(
              "emailAttribute",
              t("Атрибут email", "Email attribute"),
              false,
              true,
            )}
            <label>
              {t("Постоянный идентификатор", "Stable identifier")}
              <select
                aria-label={t("Постоянный идентификатор", "Stable identifier")}
                value={config.subjectAttribute}
                disabled={disabled || !!selected}
                onChange={(e) =>
                  setConfig({ ...config, subjectAttribute: e.target.value })
                }
              >
                <option>entryUUID</option>
                <option>objectGUID</option>
              </select>
            </label>
          </>
        )}
        {config.protocol !== "saml" && (
          <>
            {selected && (
              <label>
                <input
                  type="checkbox"
                  checked={replaceSecret}
                  disabled={disabled}
                  onChange={(e) => setReplaceSecret(e.target.checked)}
                />
                {t("Заменить сохранённый секрет", "Replace stored secret")} (
                {selected.hasSecret ? t("есть", "present") : t("нет", "absent")}
                )
              </label>
            )}
            <label>
              {config.protocol === "ldap"
                ? t("Пароль service bind", "Service bind password")
                : "Client secret"}
              <input
                type="password"
                autoComplete="new-password"
                value={secret}
                disabled={disabled || (!!selected && !replaceSecret)}
                onChange={(e) => setSecret(e.target.value)}
                maxLength={4096}
              />
            </label>
            {textField(
              "caPem",
              t(
                "Дополнительная CA, PEM (пусто — системное доверие)",
                "Additional CA PEM (empty uses system trust)",
              ),
              true,
            )}
            {listField(
              "privateCidrs",
              t(
                "Разрешённые внутренние CIDR для подключения",
                "Allowed private CIDRs for connections",
              ),
            )}
          </>
        )}
        {selected && (
          <p>
            {t("Callback / metadata: ", "Callback / metadata: ")}
            <code>
              {typeof window !== "undefined" ? location.origin : ""}
              /api/v1/auth/providers/{selected.id}/
              {config.protocol === "saml"
                ? "metadata"
                : config.protocol === "oidc"
                  ? "callback"
                  : "ldap"}
            </code>
          </p>
        )}
        <p>
          {t(
            "Сохранение отзывает сессии этого провайдера. Внешние logout/SCIM/group mapping не выполняются. Секреты обратно не возвращаются.",
            "Saving revokes this provider’s sessions. External logout, SCIM and group mapping are not performed. Secrets are never returned.",
          )}
        </p>
        <button disabled={disabled}>
          {t("Сохранить провайдера", "Save provider")}
        </button>
      </form>
      {message && <p role="status">{message}</p>}
    </section>
  );
}
