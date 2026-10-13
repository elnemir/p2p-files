"use client";

import { useEffect, useRef, useState } from "react";
import type { Schema } from "../lib/api-types";

type API = <T>(path: string, method?: string, data?: unknown) => Promise<T>;
type Props = {
  api: API;
  t: (ru: string, en: string) => string;
  account?: boolean;
};

export function IdentityCompletion({ api, t }: Props) {
  const started = useRef(false);
  const [error, setError] = useState("");
  useEffect(() => {
    const state = new URLSearchParams(location.search).get("authFlow");
    if (!state || started.current) return;
    started.current = true;
    history.replaceState(null, "", location.pathname);
    void api<Schema<"Session">>("/auth/complete", "POST", { state })
      .then((session) => {
        localStorage.setItem("filex.csrf", session.csrfToken);
        location.replace("/");
      })
      .catch(() => setError("failed"));
  }, [api]);
  return error ? (
    <p className="error" role="alert">
      {t(
        "Внешний вход не завершён. Начните вход заново. При совпадении email сначала войдите в существующий аккаунт и свяжите провайдер в настройках.",
        "External sign-in did not complete. Start again. If the email already exists, sign into the existing account and link the provider in Settings.",
      )}
    </p>
  ) : null;
}

export default function IdentityLogin({ api, t, account = false }: Props) {
  const [providers, setProviders] = useState<Schema<"IdentityPublic">[]>([]);
  const [linked, setLinked] = useState<string[]>([]);
  const [selected, setSelected] = useState("");
  const [purpose, setPurpose] = useState<"login" | "link" | "reauth">("login");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  useEffect(() => {
    void api<Schema<"IdentityPublic">[]>("/auth/providers")
      .then(setProviders)
      .catch(() => setError("load"));
    if (account)
      void api<{ providerId: string }[]>("/auth/identities")
        .then((rows) => setLinked(rows.map((r) => r.providerId)))
        .catch(() => setError("load"));
  }, [api, account]);
  async function start(id: string, protocol: string, action: typeof purpose) {
    setError("");
    setPurpose(action);
    if (protocol === "ldap") {
      setSelected(id);
      return;
    }
    setBusy(true);
    try {
      const result = await api<{ url: string }>(
        `/auth/providers/${id}/start`,
        "POST",
        { purpose: action },
      );
      location.assign(result.url);
    } catch {
      setError("auth");
      setBusy(false);
    }
  }
  if (!providers.length && !error) return null;
  return (
    <section
      className="panel identity-login"
      aria-label={t("Внешняя авторизация", "External authentication")}
    >
      <h2>
        {t(
          account ? "Корпоративные аккаунты" : "Корпоративный вход",
          account ? "Connected accounts" : "Corporate sign-in",
        )}
      </h2>
      <p>
        {account
          ? t(
              "Повторный вход подтверждает действия на 5 минут. Для новой привязки сначала подтвердите текущую учётную запись. Связывание по одному совпавшему email не выполняется.",
              "Sign in again to confirm actions for 5 minutes. Before linking, reauthenticate your current account. Matching email alone does not link accounts.",
            )
          : t(
              "Если администратор разрешил регистрацию, первый успешный вход создаст ваш личный кабинет.",
              "If registration is enabled by your administrator, your first successful sign-in creates your account.",
            )}
      </p>
      {error && (
        <p role="alert" className="error">
          {t(
            error === "load"
              ? "Не удалось загрузить провайдеров. Обновите страницу."
              : "Вход отклонён. Проверьте данные и повторное подтверждение аккаунта.",
            error === "load"
              ? "Could not load providers. Reload this page."
              : "Authentication rejected. Check credentials and current account reauthentication.",
          )}
        </p>
      )}
      <div className="actions">
        {providers.map((p) => (
          <button
            key={p.id}
            type="button"
            className="secondary"
            disabled={busy}
            onClick={() =>
              void start(
                p.id,
                p.protocol,
                account ? (linked.includes(p.id) ? "reauth" : "link") : "login",
              )
            }
          >
            {account
              ? linked.includes(p.id)
                ? t("Подтвердить через ", "Reauthenticate with ")
                : t("Связать ", "Link ")
              : t("Войти через ", "Sign in with ")}
            {p.name}
          </button>
        ))}
      </div>
      {selected && (
        <form
          onSubmit={(event) => {
            event.preventDefault();
            const data = new FormData(event.currentTarget);
            event.currentTarget.reset();
            setBusy(true);
            setError("");
            void api<Schema<"Session">>(
              `/auth/providers/${selected}/ldap`,
              "POST",
              {
                username: data.get("username"),
                password: data.get("password"),
                purpose,
              },
            )
              .then((session) => {
                localStorage.setItem("filex.csrf", session.csrfToken);
                location.replace("/");
              })
              .catch(() => setError("auth"))
              .finally(() => setBusy(false));
          }}
        >
          <h3>{providers.find((p) => p.id === selected)?.name}</h3>
          <label>
            {t("Имя пользователя каталога", "Directory username")}
            <input
              name="username"
              autoComplete="username"
              maxLength={254}
              required
            />
          </label>
          <label>
            {t("Пароль каталога", "Directory password")}
            <input
              name="password"
              type="password"
              autoComplete="current-password"
              maxLength={1024}
              required
            />
          </label>
          <button disabled={busy}>
            {t("Подтвердить вход", "Confirm sign-in")}
          </button>
          <button
            type="button"
            className="secondary"
            disabled={busy}
            onClick={() => setSelected("")}
          >
            {t("Отмена", "Cancel")}
          </button>
        </form>
      )}
    </section>
  );
}
