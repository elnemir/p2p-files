"use client";

import { useEffect, useEffectEvent, useState } from "react";
import type { Api, Response, Schema } from "../lib/api-types";
import Icon from "./Icon";
import { Field, useConfirmation, type Translate } from "./Controls";

type Settings = Schema<"Settings">;
type Revisions = Response<"/api/v1/admin/settings">;

function networkKey(value: Settings["networkPolicy"]) {
  return JSON.stringify({
    requireProxy: !!value?.requireProxy,
    requireHttps: !!value?.requireHttps,
    trustedProxyCidrs: value?.trustedProxyCidrs || [],
    allowedClientCidrs: value?.allowedClientCidrs || [],
  });
}
function LifetimeSetting({
  value,
  onChange,
  t,
  session = false,
}: {
  value: number;
  onChange: (value: number) => void;
  t: Translate;
  session?: boolean;
}) {
  const [custom, setCustom] = useState(false);
  const options = session
    ? [
        [900, t("15 минут", "15 minutes")],
        [3600, t("1 час", "1 hour")],
        [43200, t("12 часов", "12 hours")],
        [86400, t("1 день", "1 day")],
        [604800, t("7 дней", "7 days")],
      ]
    : [
        [3600, t("1 час", "1 hour")],
        [86400, t("1 день", "1 day")],
        [259200, t("3 дня", "3 days")],
        [604800, t("7 дней", "7 days")],
        [2592000, t("30 дней", "30 days")],
        [31536000, t("365 дней", "365 days")],
      ];
  const exact = custom || !options.some(([seconds]) => seconds === value);
  return (
    <div>
      <Field
        title={
          session
            ? t("Длительность сеанса", "Session lifetime")
            : t("Максимальный срок передачи", "Maximum transfer lifetime")
        }
      >
        <select
          value={exact ? "custom" : String(value)}
          onChange={(event) => {
            setCustom(event.target.value === "custom");
            if (event.target.value !== "custom")
              onChange(Number(event.target.value));
          }}
        >
          {options.map(([seconds, label]) => (
            <option key={seconds} value={seconds}>
              {label}
            </option>
          ))}
          <option value="custom">
            {t("Другой срок…", "Custom duration…")}
          </option>
        </select>
      </Field>
      {exact && (
        <Field
          title={
            session
              ? t("Срок сеанса, секунд", "Session lifetime, seconds")
              : t(
                  "Максимальный срок передачи, секунд",
                  "Maximum transfer lifetime, seconds",
                )
          }
        >
          <input
            type="number"
            min={session ? 300 : 1}
            max={session ? 604800 : 31536000}
            required
            value={value || ""}
            onChange={(event) => onChange(Number(event.target.value))}
          />
        </Field>
      )}
    </div>
  );
}
function readSettings(raw: string): Settings {
  const value = JSON.parse(raw);
  // The server validates the full strict schema; this guard keeps the form safe for arbitrary expert JSON.
  if (
    !value ||
    Array.isArray(value) ||
    value.version !== 1 ||
    !["maxFiles", "maxPathBytes", "transferMaxTtlSeconds"].every(
      (key) => typeof value[key] === "number",
    ) ||
    ["publicOrigin", "bannerRu", "bannerEn", "fileTypeMode"].some(
      (key) => value[key] !== undefined && typeof value[key] !== "string",
    ) ||
    (value.sessionTtlSeconds !== undefined &&
      typeof value.sessionTtlSeconds !== "number") ||
    (value.fileTypes !== undefined &&
      (!Array.isArray(value.fileTypes) ||
        value.fileTypes.some((item: unknown) => typeof item !== "string")))
  )
    throw new Error("invalid_input");
  if (value.networkPolicy !== undefined) {
    const n = value.networkPolicy;
    if (
      !n ||
      typeof n !== "object" ||
      Array.isArray(n) ||
      ["requireProxy", "requireHttps"].some(
        (key) => n[key] !== undefined && typeof n[key] !== "boolean",
      ) ||
      ["trustedProxyCidrs", "allowedClientCidrs"].some(
        (key) =>
          n[key] !== undefined &&
          (!Array.isArray(n[key]) ||
            n[key].some((item: unknown) => typeof item !== "string")),
      )
    )
      throw new Error("invalid_input");
  }
  return value;
}

export default function ApplicationSettings({
  api,
  t,
  busy,
}: {
  api: Api;
  t: Translate;
  busy: boolean;
}) {
  const [revisions, setRevisions] = useState<Revisions>([]);
  const [value, setValue] = useState<Settings | null>(null);
  const [base, setBase] = useState<number | null>(null);
  const [raw, setRaw] = useState("");
  const [expert, setExpert] = useState(false);
  const [draft, setDraft] = useState<number | null>(null);
  const [working, setWorking] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const { confirm, confirmation } = useConfirmation(t);
  const disabled = busy || working;
  const active = revisions.find((item) => item.active);
  function loadCurrent(items: Revisions) {
    const current = items.find((item) => item.active);
    setRevisions(items);
    if (current) {
      setValue(current.value);
      setBase(current.revision);
      setRaw(JSON.stringify(current.value, null, 2));
      setDraft(null);
    }
  }
  const loadError = useEffectEvent(() =>
    setError(
      t(
        "Не удалось загрузить настройки. Подтвердите личность и повторите.",
        "Could not load settings. Confirm your identity and retry.",
      ),
    ),
  );
  useEffect(() => {
    let live = true;
    void api<Revisions>("/admin/settings")
      .then((items) => {
        if (live) loadCurrent(items);
      })
      .catch(() => {
        if (live) loadError();
      });
    return () => {
      live = false;
    };
  }, [api]);
  function update(next: Settings) {
    setValue(next);
    setRaw(JSON.stringify(next, null, 2));
    setDraft(null);
    setMessage("");
  }
  async function run(fn: () => Promise<void>) {
    setError("");
    setMessage("");
    setWorking(true);
    try {
      await fn();
    } catch (e) {
      setError(
        e instanceof Error && e.message === "conflict"
          ? t(
              "Настройки уже изменены другой сессией. Ваш ввод сохранён. Загрузите текущую версию и повторите изменение.",
              "Another session changed the settings. Your input is retained. Load the current version and repeat your change.",
            )
          : t(
              "Не удалось выполнить действие. Проверьте поля, ограничения и подтверждение личности. Ввод сохранён.",
              "Action failed. Check the fields, limits and identity confirmation. Your input is retained.",
            ),
      );
    } finally {
      setWorking(false);
    }
  }
  async function apply(revision: number) {
    if (base === null) return;
    const candidate = revisions.find(
      (item) => item.revision === revision,
    )?.value;
    const networkChanged =
      candidate &&
      (candidate.publicOrigin !== active?.value.publicOrigin ||
        networkKey(candidate.networkPolicy) !==
          networkKey(active?.value.networkPolicy));
    if (
      !(await confirm({
        title: t("Применить настройки?", "Apply settings?"),
        action: t("Применить", "Apply"),
        description: networkChanged
          ? t(
              "Изменяется адрес или сетевая политика. Доступ будет закрыт. Потребуются перезапуск API/worker, сверка и явное открытие доступа.",
              "The origin or network policy changes. Access will close. Restart the API/worker, reconcile and explicitly reopen access.",
            )
          : t(
              "Проверенная ревизия станет активной для всей установки. Существующие опубликованные передачи сохраняют свой срок доступа.",
              "The validated revision becomes active for the installation. Published transfers retain their existing expiry.",
            ),
      }))
    )
      return;
    try {
      await api(`/admin/settings/${revision}/apply`, "POST", {
        expectedRevision: base,
      });
    } catch (error) {
      const current = await api<Revisions>("/admin/settings");
      if (current.find((item) => item.active)?.revision !== base)
        throw new Error("conflict", { cause: error });
      throw error;
    }
    setDraft(null);
    // Applying network policy can intentionally close the data plane; never report the mutation as failed on refresh.
    let refreshFailed = false;
    try {
      loadCurrent(await api<Revisions>("/admin/settings"));
    } catch {
      refreshFailed = true;
    }
    if (refreshFailed) {
      setMessage(
        t(
          "Настройки применены, но свежий статус недоступен. Обновите страницу; при изменении сети выполните восстановление доступа в «Эксплуатация».",
          "Settings applied, but the latest status is unavailable. Reload the page; after a network change, complete access recovery in Operations.",
        ),
      );
      return;
    }
    setMessage(
      networkChanged
        ? t(
            "Настройки применены. Откройте «Эксплуатация» и выполните перезапуск, сверку и открытие доступа.",
            "Settings applied. Open Operations and complete restart, reconciliation and reopening.",
          )
        : t("Настройки применены.", "Settings applied."),
    );
  }
  function number(
    key:
      | "maxFiles"
      | "maxPathBytes"
      | "sessionTtlSeconds"
      | "transferMaxTtlSeconds",
    title: string,
    min: number,
    max: number,
  ) {
    return (
      <Field title={title}>
        <input
          type="number"
          min={min}
          max={max}
          required
          value={value?.[key] ?? ""}
          onChange={(event) =>
            value && update({ ...value, [key]: Number(event.target.value) })
          }
        />
      </Field>
    );
  }
  return (
    <section
      className="panel"
      aria-label={t("Настройки приложения", "Application settings")}
    >
      {confirmation}
      <div className="section-title">
        <div>
          <h2>
            <Icon name="settings" />
            {t("Настройки приложения", "Application settings")}
          </h2>
          <p className="muted">
            {t(
              "Заполните поля → проверьте черновик → примените изменения.",
              "Fill in the form → validate a draft → apply the changes.",
            )}
          </p>
        </div>
        {base !== null && (
          <span className="badge">
            {t("Ревизия", "Revision")} #{base}
          </span>
        )}
      </div>
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
      {message && (
        <p className="notice compact" role="status">
          {message}
        </p>
      )}
      <div className="toolbar">
        <button
          type="button"
          className="secondary"
          disabled={disabled}
          onClick={() =>
            void run(async () => {
              if (
                value &&
                !(await confirm({
                  title: t(
                    "Загрузить текущие настройки?",
                    "Load current settings?",
                  ),
                  description: t(
                    "Несохранённые изменения формы будут заменены активной ревизией.",
                    "Unsaved form changes will be replaced by the active revision.",
                  ),
                  action: t("Загрузить", "Load"),
                }))
              )
                return;
              loadCurrent(await api<Revisions>("/admin/settings"));
            })
          }
        >
          <Icon name="refresh" />
          {t("Загрузить текущие", "Load current")}
        </button>
        <button
          type="button"
          className="secondary"
          disabled={disabled || !value}
          aria-pressed={expert}
          onClick={() => {
            try {
              if (expert) setValue(readSettings(raw));
              setExpert(!expert);
              setError("");
            } catch {
              setError(
                t(
                  "Исправьте JSON перед переходом к форме. Ввод сохранён.",
                  "Fix the JSON before switching to the form. Your input is retained.",
                ),
              );
            }
          }}
        >
          {expert
            ? t("Обычная форма", "Form editor")
            : t("Экспертный JSON", "Expert JSON")}
        </button>
      </div>
      {value && (
        <form
          onSubmit={(event) => {
            event.preventDefault();
            void run(async () => {
              const lines = (items: string[] | undefined) =>
                (items || []).map((item) => item.trim()).filter(Boolean);
              const data = expert
                ? readSettings(raw)
                : {
                    ...value,
                    fileTypes: lines(value.fileTypes),
                    networkPolicy: {
                      ...value.networkPolicy,
                      trustedProxyCidrs: lines(
                        value.networkPolicy?.trustedProxyCidrs,
                      ),
                      allowedClientCidrs: lines(
                        value.networkPolicy?.allowedClientCidrs,
                      ),
                    },
                  };
              const created = await api<{ revision: number }>(
                "/admin/settings",
                "POST",
                data,
              );
              setDraft(null);
              await api(`/admin/settings/${created.revision}/validate`, "POST");
              setRevisions(await api<Revisions>("/admin/settings"));
              setDraft(created.revision);
              setMessage(
                t(
                  "Черновик проверен. Изменения ещё не применены.",
                  "Draft validated. Changes have not been applied yet.",
                ),
              );
            });
          }}
        >
          <fieldset disabled={disabled} className="plain-fieldset">
            {expert ? (
              <Field title="Settings JSON">
                <textarea
                  rows={16}
                  value={raw}
                  spellCheck={false}
                  onChange={(event) => {
                    setRaw(event.target.value);
                    setDraft(null);
                  }}
                />
              </Field>
            ) : (
              <>
                <h3>{t("Передачи и сеансы", "Transfers and sessions")}</h3>
                <div className="form-grid">
                  <LifetimeSetting
                    value={value.transferMaxTtlSeconds}
                    onChange={(seconds) =>
                      update({ ...value, transferMaxTtlSeconds: seconds })
                    }
                    t={t}
                  />
                  <LifetimeSetting
                    value={value.sessionTtlSeconds ?? 43200}
                    onChange={(seconds) =>
                      update({ ...value, sessionTtlSeconds: seconds })
                    }
                    t={t}
                    session
                  />
                  {number(
                    "maxFiles",
                    t(
                      "Максимум файлов в передаче",
                      "Maximum files per transfer",
                    ),
                    1,
                    100000,
                  )}
                  {number(
                    "maxPathBytes",
                    t("Максимум байт в пути файла", "Maximum file path bytes"),
                    1,
                    4096,
                  )}
                </div>
                <h3>{t("Разрешённые типы файлов", "File type policy")}</h3>
                <Field title={t("Режим проверки типов", "File type mode")}>
                  <select
                    value={value.fileTypeMode || "disabled"}
                    onChange={(event) =>
                      update({
                        ...value,
                        fileTypeMode: event.target
                          .value as Settings["fileTypeMode"],
                      })
                    }
                  >
                    <option value="disabled">
                      {t("Без ограничений по типу", "No type restrictions")}
                    </option>
                    <option value="allow">
                      {t(
                        "Разрешать только перечисленные",
                        "Allow listed types only",
                      )}
                    </option>
                    <option value="deny">
                      {t("Запрещать перечисленные", "Deny listed types")}
                    </option>
                  </select>
                </Field>
                {value.fileTypeMode && value.fileTypeMode !== "disabled" && (
                  <Field
                    title={t(
                      "MIME-типы, по одному на строку",
                      "MIME types, one per line",
                    )}
                  >
                    <textarea
                      rows={4}
                      placeholder={"application/pdf\nimage/png"}
                      value={(value.fileTypes || []).join("\n")}
                      onChange={(event) =>
                        update({
                          ...value,
                          fileTypes: event.target.value.split("\n"),
                        })
                      }
                    />
                  </Field>
                )}
                <h3>
                  {t("Объявление для пользователей", "User announcement")}
                </h3>
                <p className="muted">
                  {t(
                    "Пустое поле убирает объявление для этого языка. Максимум 2000 байт на язык.",
                    "An empty field hides the announcement for that language. Maximum 2000 bytes per language.",
                  )}
                </p>
                <div className="form-grid">
                  {(["bannerRu", "bannerEn"] as const).map((key) => (
                    <Field
                      key={key}
                      title={
                        key === "bannerRu"
                          ? t("Объявление на русском", "Russian announcement")
                          : t(
                              "Объявление на английском",
                              "English announcement",
                            )
                      }
                    >
                      <textarea
                        rows={3}
                        maxLength={2000}
                        value={value[key] || ""}
                        onChange={(event) =>
                          update({ ...value, [key]: event.target.value })
                        }
                      />
                    </Field>
                  ))}
                </div>
                <details className="advanced-options">
                  <summary>
                    <Icon name="globe" />
                    {t(
                      "Адрес сервиса и сетевая политика",
                      "Service origin and network policy",
                    )}
                  </summary>
                  <p className="notice compact">
                    {t(
                      "Изменение этих параметров закроет доступ и потребует перезапуска, сверки и открытия через «Эксплуатация».",
                      "Changing these settings closes access and requires restart, reconciliation and reopening in Operations.",
                    )}
                  </p>
                  <Field
                    title={t(
                      "Публичный адрес сервиса",
                      "Public service origin",
                    )}
                  >
                    <input
                      type="url"
                      required
                      value={value.publicOrigin || ""}
                      placeholder="https://files.example.com"
                      onChange={(event) =>
                        update({ ...value, publicOrigin: event.target.value })
                      }
                    />
                  </Field>
                  {(["requireProxy", "requireHttps"] as const).map((key) => (
                    <label className="check" key={key}>
                      <input
                        type="checkbox"
                        checked={!!value.networkPolicy?.[key]}
                        onChange={(event) =>
                          update({
                            ...value,
                            networkPolicy: {
                              ...value.networkPolicy,
                              [key]: event.target.checked,
                            },
                          })
                        }
                      />
                      {key === "requireProxy"
                        ? t(
                            "Принимать запросы только от доверенного прокси",
                            "Require a trusted proxy",
                          )
                        : t("Требовать HTTPS", "Require HTTPS")}
                    </label>
                  ))}
                  {(["trustedProxyCidrs", "allowedClientCidrs"] as const).map(
                    (key) => (
                      <Field
                        key={key}
                        title={
                          key === "trustedProxyCidrs"
                            ? t(
                                "Доверенные IPv4 прокси /32, по одному на строку",
                                "Trusted IPv4 proxies /32, one per line",
                              )
                            : t(
                                "Разрешённые IPv4 сети клиентов (пусто — любые)",
                                "Allowed IPv4 client networks (empty allows any)",
                              )
                        }
                      >
                        <textarea
                          rows={3}
                          value={(value.networkPolicy?.[key] || []).join("\n")}
                          onChange={(event) =>
                            update({
                              ...value,
                              networkPolicy: {
                                ...value.networkPolicy,
                                [key]: event.target.value.split("\n"),
                              },
                            })
                          }
                        />
                      </Field>
                    ),
                  )}
                </details>
              </>
            )}
            <div className="toolbar">
              <button>
                <Icon name="check" />
                {t("Сохранить и проверить", "Save and validate")}
              </button>
              {draft !== null && (
                <button
                  type="button"
                  onClick={() => void run(() => apply(draft))}
                >
                  <Icon name="save" />
                  {t("Применить настройки", "Apply settings")} #{draft}
                </button>
              )}
            </div>
          </fieldset>
        </form>
      )}
      <details className="advanced-options">
        <summary>
          {t("История ревизий и откат", "Revision history and rollback")}
        </summary>
        {revisions.map((item) => (
          <div className="list-row" key={item.revision}>
            <span>
              #{item.revision} ·{" "}
              {item.active
                ? t("Активна", "Active")
                : item.validated
                  ? t("Проверена", "Validated")
                  : t("Черновик", "Draft")}
            </span>
            {item.validated && !item.active && (
              <button
                className="secondary"
                disabled={disabled}
                onClick={() => void run(() => apply(item.revision))}
              >
                {t("Применить ревизию", "Apply revision")} #{item.revision}
              </button>
            )}
          </div>
        ))}
      </details>
    </section>
  );
}
