"use client";
import { useEffect, useState } from "react";

async function readStatus(endpoint: string) {
  const response = await fetch(endpoint, { cache: "no-store" });
  if (!response.ok)
    throw new Error(
      "Сервис настройки недоступен. Проверьте контейнер API и миграции, затем повторите проверку.",
    );
  const data = await response.json();
  if (!["pending", "complete", "disabled"].includes(data.state))
    throw new Error("Не удалось проверить состояние установки.");
  return data.state as string;
}

export default function SetupWizard({ issuer = false }: { issuer?: boolean }) {
  const endpoint = issuer ? "/issuer/api/setup" : "/api/v1/setup";
  const [state, setState] = useState("loading");
  const [step, setStep] = useState(1);
  const [token, setToken] = useState("");
  const [values, setValues] = useState<Record<string, string>>({});
  const [confirm, setConfirm] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [finished, setFinished] = useState(false);
  async function refresh() {
    const next = await readStatus(endpoint);
    setState(next);
    return next;
  }
  useEffect(() => {
    let current = true;
    void readStatus(endpoint)
      .then((next) => {
        if (current) setState(next);
      })
      .catch((e: Error) => {
        if (current) {
          setError(e.message);
          setState("error");
        }
      });
    return () => {
      current = false;
    };
  }, [endpoint]);
  const fields = issuer
    ? [
        ["issuer", "Адрес OIDC-провайдера", "url"],
        ["clientId", "Client ID", "text"],
        ["clientSecret", "Client secret", "password"],
        ["requiredAcr", "MFA ACR", "text"],
        ["subject", "Subject (sub) первого оператора", "text"],
      ]
    : [
        ["email", "Email администратора", "email"],
        ["password", "Пароль администратора", "password"],
      ];
  async function complete() {
    setBusy(true);
    setError("");
    try {
      const response = await fetch(endpoint, {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-Setup-Token": token },
        body: JSON.stringify(values),
      });
      if (!response.ok)
        throw new Error(
          response.status === 403
            ? "Токен неверен или истёк, либо адрес страницы не совпадает с настроенным origin. Проверьте bootstrap-параметры контейнера."
            : response.status === 409
              ? "Настройка уже завершена другим запросом. Можно перейти ко входу."
              : issuer
                ? "Не удалось сохранить настройки. Проверьте HTTPS-адрес провайдера, его доступность из API и заполнение полей."
                : "Не удалось создать администратора. Проверьте email и пароль (12–1024 байта, не распространённый пароль).",
        );
      setFinished(true);
      setState("complete");
      setToken("");
      setValues({});
      setConfirm("");
    } catch (e) {
      setError((e as Error).message);
      try {
        if ((await refresh()) === "complete") {
          setToken("");
          setValues({});
          setConfirm("");
        }
      } catch {
        /* Keep the original error and allow an explicit retry. */
      }
    } finally {
      setBusy(false);
    }
  }
  return (
    <main id="main" tabIndex={-1} className="setup-wizard">
      <style>{`.setup-wizard{overflow-wrap:anywhere;max-width:720px;margin:4vh auto;padding:28px;background:#fff;color:#17213b;border:1px solid #cbd5e1;border-radius:18px;font:16px/1.6 system-ui}.setup-wizard h1{font-size:28px;line-height:1.25}.setup-wizard p{color:#44516a}.setup-wizard label{display:grid;gap:6px;margin:18px 0;font-weight:600}.setup-wizard input{box-sizing:border-box;width:100%;padding:12px;border:1px solid #73809a;border-radius:8px;background:#fff;color:#17213b;font:inherit}.setup-wizard button,.setup-wizard a{display:inline-block;margin:8px 10px 0 0;padding:10px 18px;border-radius:8px;font:inherit}.setup-wizard button{background:#184ec9;color:#fff;border:0;cursor:pointer}.setup-wizard button:disabled{opacity:.6;cursor:wait}.setup-wizard a{color:#184ec9}.setup-wizard [role=alert]{padding:12px;background:#fff0ef;color:#9b2018;border-radius:8px}.setup-wizard dl{overflow-wrap:anywhere}.setup-wizard dd{margin:0 0 12px}.setup-wizard dt{font-weight:600}.setup-wizard :focus-visible{outline:3px solid #a34b00;outline-offset:3px}@media(max-width:600px){.setup-wizard{margin:12px;padding:18px}.setup-wizard h1{font-size:24px}}`}</style>
      <p>{issuer ? "Портал лицензирования" : "FileX"} · Первый запуск</p>
      <h1>
        {state === "complete"
          ? "Настройка завершена"
          : "Настроим вашу установку"}
      </h1>
      {error && <p role="alert">{error}</p>}
      {state === "loading" && (
        <p role="status">Проверяем состояние контейнера…</p>
      )}
      {state === "complete" && (
        <>
          <p>
            {finished
              ? "Настройки сохранены. Мастер закрыт для повторного запуска."
              : "Эта установка уже настроена. Повторная первоначальная настройка недоступна."}
          </p>
          <p>
            {issuer
              ? "Войдите через OIDC с MFA. Первый указанный subject получит роль security_admin. Ключ подписи обслуживается отдельным signer."
              : "Войдите с паролем администратора. В разделе «Лицензия» скачайте запрос установки, передайте издателю и примените полученный файл. До активации платные операции недоступны."}
          </p>
          <a href={issuer ? "/issuer/auth/login" : "/"}>Перейти ко входу</a>
        </>
      )}
      {(state === "disabled" || state === "error") && (
        <>
          <p>
            Мастер недоступен. Оператор сервера должен включить его временным
            bootstrap-токеном. Существующие данные и ключи сохраняются.
          </p>
          <button
            onClick={() => {
              setError("");
              void refresh().catch((e: Error) => setError(e.message));
            }}
          >
            Проверить снова
          </button>
        </>
      )}
      {state === "pending" && (
        <>
          <p aria-live="polite">
            Шаг {step} из 3 ·{" "}
            {step === 1 ? "Доступ" : step === 2 ? "Настройки" : "Проверка"}
          </p>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              setError("");
              if (step === 2 && !issuer && values.password !== confirm) {
                setError("Пароли не совпадают.");
                return;
              }
              if (step < 3) setStep(step + 1);
              else void complete();
            }}
          >
            {step === 1 && (
              <>
                <p>
                  Введите токен, подготовленный оператором контейнера. Он нужен
                  только для первого запуска и не является паролем
                  администратора.
                </p>
                <label>
                  Токен настройки
                  <input
                    type="password"
                    autoComplete="off"
                    required
                    minLength={32}
                    maxLength={256}
                    value={token}
                    onChange={(e) => setToken(e.target.value)}
                  />
                </label>
              </>
            )}
            {step === 2 && (
              <>
                {issuer && (
                  <p>
                    Создайте OIDC client с callback{" "}
                    <strong>
                      {typeof window !== "undefined"
                        ? window.location.origin
                        : "https://ваш-домен"}
                      /issuer/auth/callback
                    </strong>
                    . Укажите ACR, который провайдер выдаёт после обязательного
                    MFA, и стабильный sub, а не email.
                  </p>
                )}
                {fields.map(([key, label, type]) => (
                  <label key={key}>
                    {label}
                    <input
                      required
                      type={type}
                      autoComplete={
                        type === "password" ? "new-password" : "off"
                      }
                      minLength={key === "password" ? 12 : 1}
                      maxLength={
                        key === "password" ? 1024 : key === "email" ? 254 : 2048
                      }
                      value={values[key] || ""}
                      onChange={(e) =>
                        setValues({ ...values, [key]: e.target.value })
                      }
                    />
                  </label>
                ))}
                {!issuer && (
                  <label>
                    Повторите пароль
                    <input
                      required
                      type="password"
                      autoComplete="new-password"
                      value={confirm}
                      onChange={(e) => setConfirm(e.target.value)}
                    />
                  </label>
                )}
              </>
            )}
            {step === 3 && (
              <>
                <p>
                  Проверьте параметры. После сохранения мастер закрывается;
                  изменение bootstrap-настроек потребует доступа оператора
                  сервера.
                </p>
                <dl>
                  {fields
                    .filter(([, , type]) => type !== "password")
                    .map(([key, label]) => (
                      <div key={key}>
                        <dt>{label}</dt>
                        <dd>{values[key]}</dd>
                      </div>
                    ))}
                </dl>
                <p>
                  Секреты введены и не отображаются в проверке.
                  {issuer &&
                    " Доступность провайдера будет проверена при сохранении; MFA — при первом входе."}
                </p>
              </>
            )}
            {step > 1 && (
              <button
                type="button"
                disabled={busy}
                onClick={() => setStep(step - 1)}
              >
                Назад
              </button>
            )}
            <button disabled={busy}>
              {busy
                ? "Сохраняем…"
                : step < 3
                  ? "Продолжить"
                  : "Завершить настройку"}
            </button>
          </form>
        </>
      )}
    </main>
  );
}
