"use client";
import { useEffect, useEffectEvent, useState } from "react";
import type { Api, Schema, Response } from "../lib/api-types";
import LicensePanel from "./LicensePanel";
import BrandingPanel from "./BrandingPanel";
import IdentityProvidersPanel from "./IdentityProvidersPanel";
import OperationsPanel from "./OperationsPanel";
import AdminExportsPanel from "./AdminExportsPanel";
import ApplicationSettings from "./ApplicationSettings";
import Icon from "./Icon";
import { Field, Reauthentication, SectionNav } from "./Controls";

function forensicDownload(path: string, values: Record<string, string>) {
  const form = document.createElement("form");
  form.method = "POST";
  form.action = `/api/v1${path}`;
  form.target = "_blank";
  for (const [name, value] of Object.entries({
    ...values,
    _csrf: localStorage.getItem("filex.csrf") || "",
  })) {
    const input = document.createElement("input");
    input.type = "hidden";
    input.name = name;
    input.value = value;
    form.append(input);
  }
  document.body.append(form);
  form.submit();
  form.remove();
}
export default function AdminPanel({
  api,
  t,
  act,
  busy,
}: {
  api: Api;
  t: (ru: string, en: string) => string;
  act: (f: () => Promise<void>) => Promise<void>;
  busy: boolean;
}) {
  const [adminPreview, setAdminPreview] = useState(false);
  const [storage, setStorage] = useState<Schema<"StorageProvider">[]>([]);
  const [maintenance, setMaintenance] = useState<
    Response<"/api/v1/admin/maintenance">
  >([]);
  const [kind, setKind] = useState("local");
  const [section, setSection] = useState("operations");
  const [smtpTLS, setSmtpTLS] = useState("starttls");
  const [smtpPort, setSmtpPort] = useState(587);
  const [migrationSource, setMigrationSource] = useState("");
  const [migrationTarget, setMigrationTarget] = useState("");
  const payloadProviders = storage.filter(
    (provider) => provider.purpose === "payload",
  );
  const source = migrationSource || payloadProviders[0]?.id || "";
  const target =
    migrationTarget && migrationTarget !== source
      ? migrationTarget
      : payloadProviders.find(
          (provider) =>
            provider.id !== source && provider.enabled && !provider.draining,
        )?.id || "";
  async function reload() {
    setStorage(await api("/admin/storage"));
    setMaintenance(await api("/admin/maintenance"));
  }
  const initialLoad = useEffectEvent(() => act(reload));
  useEffect(() => {
    void initialLoad();
  }, []);
  return (
    <>
      <h1 className="page-title">{t("Администрирование", "Administration")}</h1>
      <p className="page-description">
        {t(
          "Управляйте оформлением, доступом и инфраструктурой из одного кабинета.",
          "Manage appearance, access and infrastructure in one place.",
        )}
      </p>
      <Reauthentication
        api={api}
        t={t}
        busy={busy}
        onSuccess={() => void act(reload)}
      />
      <SectionNav
        label={t("Разделы администрирования", "Administration sections")}
        value={section}
        onChange={setSection}
        items={[
          {
            id: "operations",
            label: t("Эксплуатация", "Operations"),
            icon: "status",
          },
          {
            id: "storage",
            label: t("Хранение файлов", "File storage"),
            icon: "server",
          },
          {
            id: "access",
            label: t("Корпоративный вход", "Corporate sign-in"),
            icon: "key",
          },
          {
            id: "license",
            label: t("Лицензия", "License"),
            icon: "key",
          },
          {
            id: "appearance",
            label: t("Оформление", "Appearance"),
            icon: "paint",
          },
          {
            id: "settings",
            label: t("Параметры системы", "System settings"),
            icon: "settings",
          },
          { id: "mail", label: t("Почта", "Email"), icon: "mail" },
          {
            id: "security",
            label: t("Безопасность и аудит", "Security and audit"),
            icon: "admin",
          },
        ]}
      />
      <button
        className="secondary"
        disabled={busy}
        onClick={() => void act(reload)}
      >
        {t("Обновить", "Refresh")}
      </button>
      <div hidden={section !== "license"}>
        <LicensePanel api={api} t={t} />
      </div>
      <div hidden={section !== "appearance"}>
        <BrandingPanel api={api} t={t} busy={busy} />
      </div>
      <div hidden={section !== "access"}>
        <IdentityProvidersPanel api={api} t={t} busy={busy} />
      </div>
      <div hidden={section !== "operations"}>
        <OperationsPanel
          api={api}
          t={t}
          act={act}
          busy={busy}
          providers={storage}
        />
      </div>
      <div hidden={section !== "security"}>
        <AdminExportsPanel api={api} t={t} act={act} busy={busy} />
      </div>
      <div hidden={section !== "storage"}>
        <section className="panel">
          <h2>{t("Хранилища", "Storage providers")}</h2>
          {storage.map((s) => (
            <div className="list-row" key={s.id}>
              <div className="break">
                <strong>
                  <Icon name="server" />
                  {s.kind.toUpperCase()} ·{" "}
                  {s.purpose === "payload"
                    ? t("Файлы", "Files")
                    : s.purpose === "artifacts"
                      ? t("Отчёты и экспорты", "Reports and exports")
                      : t("Резервные копии", "Backups")}
                </strong>
                <p>
                  {!s.enabled
                    ? t("Отключено", "Disabled")
                    : s.draining
                      ? t("Без новых загрузок", "Draining")
                      : t("Принимает загрузки", "Accepting uploads")}
                </p>
                <p>
                  {s.id} · {t("приоритет", "priority")} {s.priority}
                </p>
              </div>
              <button
                className="secondary"
                disabled={busy}
                onClick={() =>
                  void act(async () => {
                    await api(`/admin/storage/${s.id}`, "PATCH", {
                      enabled: s.enabled,
                      draining: !s.draining,
                      priority: s.priority,
                    });
                    await reload();
                  })
                }
              >
                {s.draining
                  ? t("Принимать новые загрузки", "Accept new uploads")
                  : t("Вывести из новых загрузок", "Drain")}
              </button>
            </div>
          ))}
          <details className="advanced-options">
            <summary>
              <Icon name="plus" />
              {t("Добавить хранилище", "Add storage provider")}
            </summary>
            <form
              onSubmit={(e) => {
                e.preventDefault();
                const f = new FormData(e.currentTarget);
                void act(async () => {
                  await api("/admin/storage", "POST", {
                    kind,
                    purpose: f.get("purpose"),
                    root: f.get("root") || "",
                    priority: Number(f.get("priority")),
                    s3:
                      kind === "s3"
                        ? {
                            endpoint: f.get("endpoint"),
                            region: f.get("region"),
                            bucket: f.get("bucket"),
                            accessKey: f.get("accessKey"),
                            secretKey: f.get("secretKey"),
                          }
                        : undefined,
                  });
                  await reload();
                });
              }}
            >
              <div className="form-grid">
                <Field title={t("Тип", "Type")}>
                  <select
                    value={kind}
                    onChange={(e) => setKind(e.target.value)}
                  >
                    <option value="local">Local</option>
                    <option value="s3">S3</option>
                  </select>
                </Field>
                <Field title={t("Назначение", "Purpose")}>
                  <select name="purpose">
                    <option value="payload">
                      {t("Передаваемые файлы", "Transferred files")}
                    </option>
                    <option value="artifacts">
                      {t("Отчёты и экспорты", "Reports and exports")}
                    </option>
                    <option value="backups">
                      {t("Резервные копии", "Backups")}
                    </option>
                  </select>
                </Field>
                <Field title={t("Приоритет", "Priority")}>
                  <input name="priority" type="number" defaultValue={0} />
                </Field>
                {kind === "local" ? (
                  <Field
                    title={t(
                      "Абсолютный каталог на сервере",
                      "Absolute directory on server",
                    )}
                  >
                    <input name="root" placeholder="/data/payload" required />
                  </Field>
                ) : (
                  <>
                    {[
                      ["endpoint", "HTTPS endpoint"],
                      ["region", t("Регион", "Region")],
                      ["bucket", "Bucket"],
                      ["accessKey", "Access key"],
                      ["secretKey", "Secret key"],
                    ].map(([name, label]) => (
                      <Field title={label} key={name}>
                        <input
                          name={name}
                          type={name === "secretKey" ? "password" : "text"}
                          required={name !== "endpoint"}
                          autoComplete="off"
                        />
                      </Field>
                    ))}
                  </>
                )}
              </div>
              <button disabled={busy}>
                {t("Проверить и добавить", "Check and add")}
              </button>
            </form>
          </details>
        </section>
        <section className="panel">
          <h2>{t("Перенос хранилища", "Storage migration")}</h2>
          {!target && (
            <p className="notice compact">
              {t(
                "Добавьте второе активное хранилище файлов, чтобы подготовить перенос.",
                "Add a second active payload storage provider to prepare a migration.",
              )}
            </p>
          )}
          <form
            className="toolbar"
            onSubmit={(e) => {
              e.preventDefault();
              const f = new FormData(e.currentTarget);
              void act(async () => {
                await api("/admin/storage/migrations", "POST", {
                  sourceId: f.get("source"),
                  targetId: f.get("target"),
                  dryRun: false,
                });
                await reload();
              });
            }}
          >
            {["source", "target"].map((name) => (
              <Field
                key={name}
                title={
                  name === "source"
                    ? t("Источник", "Source")
                    : t("Назначение", "Target")
                }
              >
                <select
                  name={name}
                  value={name === "source" ? source : target}
                  onChange={(e) =>
                    name === "source"
                      ? setMigrationSource(e.target.value)
                      : setMigrationTarget(e.target.value)
                  }
                  required
                >
                  <option value="" disabled>
                    {t("Выберите хранилище", "Choose storage")}
                  </option>
                  {payloadProviders
                    .filter(
                      (provider) =>
                        name === "source" ||
                        (provider.id !== source &&
                          provider.enabled &&
                          !provider.draining),
                    )
                    .map((provider) => (
                      <option key={provider.id} value={provider.id}>
                        {provider.kind.toUpperCase()} · {provider.id}
                      </option>
                    ))}
                </select>
              </Field>
            ))}
            <button disabled={busy || !source || !target || source === target}>
              <Icon name="server" />
              {t("Подготовить перенос", "Prepare migration")}
            </button>
          </form>
          {maintenance.map((j) => (
            <div className="list-row" key={j.id}>
              <div>
                <strong>
                  {j.kind} · {stateNameLocal(j.state, t)}
                </strong>
                <p>
                  {j.progress} · {j.id}
                </p>
              </div>
              {["paused", "running"].includes(j.state) && (
                <div className="toolbar">
                  {[
                    j.state === "paused" ? "running" : "paused",
                    "cancelled",
                  ].map((state) => (
                    <button
                      className="secondary"
                      key={state}
                      disabled={busy}
                      onClick={() =>
                        void act(async () => {
                          await api(
                            j.kind === "storage"
                              ? `/admin/storage/migrations/${j.id}`
                              : `/admin/keys/rotations/${j.id}`,
                            "PATCH",
                            { state },
                          );
                          await reload();
                        })
                      }
                    >
                      {stateNameLocal(state, t)}
                    </button>
                  ))}
                </div>
              )}
            </div>
          ))}
        </section>
      </div>
      <div hidden={section !== "security"}>
        <section className="panel">
          <h2>
            {t(
              "Массовые действия администратора",
              "Administrator bulk actions",
            )}
          </h2>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              const f = new FormData(e.currentTarget);
              void act(async () => {
                await api("/bulk", "POST", {
                  ids: String(f.get("ids")).split(/\s+/).filter(Boolean),
                  action: f.get("action"),
                  adminScope: true,
                  reason: f.get("reason"),
                  dryRun: true,
                });
                setAdminPreview(true);
              });
            }}
          >
            <Field
              title={t(
                "Идентификаторы через пробел или с новой строки",
                "Identifiers separated by whitespace",
              )}
            >
              <textarea name="ids" required rows={4} />
            </Field>
            <Field
              title={t("Причина массового действия", "Bulk action reason")}
            >
              <input name="reason" required minLength={5} maxLength={2000} />
            </Field>
            <Field title={t("Массовое действие", "Bulk action")}>
              <select name="action">
                <option value="revoke">
                  {t("Отозвать передачи", "Revoke transfers")}
                </option>
                <option value="cancel_upload">
                  {t("Удалить файлы черновиков", "Delete draft files")}
                </option>
              </select>
            </Field>
            <button disabled={busy}>
              {t("Подготовить предпросмотр", "Prepare preview")}
            </button>
            {adminPreview && (
              <p role="status">
                {t(
                  "Выборка сохранена. Откройте «Задачи» для проверки количества, запуска и отмены.",
                  "Selection saved. Open Jobs to review the count, run or cancel.",
                )}
              </p>
            )}
          </form>
        </section>
        <section className="panel">
          <h2>{t("Ротация ключей", "Key rotation")}</h2>
          <form
            className="toolbar"
            onSubmit={(e) => {
              e.preventDefault();
              const value = new FormData(e.currentTarget).get("class");
              void act(async () => {
                await api("/admin/keys/rotate", "POST", { class: value });
                await reload();
              });
            }}
          >
            <Field title={t("Класс ключей", "Key class")}>
              <select name="class">
                {[
                  "payload",
                  "artifacts",
                  "configuration",
                  "signing",
                  "backups",
                ].map((v) => (
                  <option key={v}>{v}</option>
                ))}
              </select>
            </Field>
            <button disabled={busy}>
              {t("Создать новую версию", "Create new version")}
            </button>
          </form>
        </section>
        <section className="panel">
          <h2>
            {t("Удержание и расследование", "Retention and investigation")}
          </h2>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              const f = new FormData(e.currentTarget);
              void act(async () => {
                await api(
                  `/admin/transfers/${f.get("transfer")}/hold`,
                  "POST",
                  {
                    fileId: f.get("file") || "",
                    frozen: f.get("hold") === "true",
                    reason: f.get("reason"),
                  },
                );
              });
            }}
          >
            <div className="form-grid">
              <Field title="Transfer ID">
                <input name="transfer" required />
              </Field>
              <Field title={t("File ID (необязательно)", "File ID (optional)")}>
                <input name="file" />
              </Field>
              <Field title={t("Причина", "Reason")}>
                <input name="reason" minLength={5} required />
              </Field>
              <Field title={t("Действие", "Action")}>
                <select name="hold">
                  <option value="true">{t("Удерживать", "Hold")}</option>
                  <option value="false">
                    {t("Снять удержание", "Release hold")}
                  </option>
                </select>
              </Field>
            </div>
            <button disabled={busy}>{t("Применить", "Apply")}</button>
          </form>
        </section>
        <section className="panel">
          <h2>{t("Экспертная проверка файлов", "Forensic file operations")}</h2>
          <p>
            {t(
              "ZIP требует удержания всей передачи. Выдача не расходует права получателя. Для повреждённого файла можно сохранить отдельный зашифрованный chunk.",
              "Hold the entire transfer before downloading its forensic ZIP. Forensic access does not consume recipient rights. A corrupted file allows a raw encrypted chunk download.",
            )}
          </p>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              const form = new FormData(e.currentTarget);
              const action = String(form.get("action")),
                id = String(form.get("id")),
                reason = String(form.get("reason"));
              void act(async () => {
                if (action === "zip")
                  forensicDownload(`/admin/transfers/${id}/forensic-zip`, {
                    reason,
                  });
                else if (action === "plain" || action === "raw")
                  forensicDownload(`/admin/files/${id}/forensic`, {
                    reason,
                    rawCiphertext: String(action === "raw"),
                    objectKey: String(form.get("object") || ""),
                  });
                else
                  await api(`/admin/files/${id}/${action}`, "POST", { reason });
              });
            }}
          >
            <div className="form-grid">
              <Field
                title={t(
                  "Идентификатор файла / передачи для ZIP",
                  "File ID / transfer ID for ZIP",
                )}
              >
                <input name="id" required />
              </Field>
              <Field title={t("Причина операции", "Operation reason")}>
                <input name="reason" required minLength={5} maxLength={2000} />
              </Field>
              <Field title={t("Операция проверки", "Investigation operation")}>
                <select name="action">
                  <option value="plain">
                    {t("Скачать файл", "Download file")}
                  </option>
                  <option value="zip">
                    {t("Скачать ZIP передачи", "Download transfer ZIP")}
                  </option>
                  <option value="raw">
                    {t(
                      "Скачать повреждённый ciphertext",
                      "Download damaged ciphertext",
                    )}
                  </option>
                  <option value="check">
                    {t(
                      "Проверить целостность / восстановить ready",
                      "Check integrity / restore ready",
                    )}
                  </option>
                  <option value="force-delete">
                    {t("Удалить файл принудительно", "Force delete file")}
                  </option>
                </select>
              </Field>
              <Field
                title={t(
                  "Object key для ciphertext",
                  "Object key for ciphertext",
                )}
              >
                <input name="object" />
              </Field>
            </div>
            <p>
              {t(
                "Принудительное удаление необратимо. Сначала снимите удержание передачи; операция явно снимает удержание выбранного файла.",
                "Force deletion is irreversible. Release the transfer hold first; the operation explicitly releases the selected file hold.",
              )}
            </p>
            <button disabled={busy}>
              {t("Выполнить выбранную операцию", "Run selected operation")}
            </button>
          </form>
        </section>
      </div>
      <div hidden={section !== "mail"}>
        <section className="panel">
          <h2>
            <Icon name="mail" />
            SMTP
          </h2>
          <p className="muted">
            {t(
              "Укажите почтовый сервер для приглашений и уведомлений. Выбор TLS подставляет стандартный порт; при необходимости измените его.",
              "Set up the mail server for invitations and notifications. Choosing TLS fills in the standard port; you can change it.",
            )}
          </p>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              const f = new FormData(e.currentTarget);
              void act(async () => {
                await api("/admin/smtp", "PUT", {
                  host: f.get("host"),
                  port: Number(f.get("port")),
                  tls: f.get("tls"),
                  username: f.get("username"),
                  password: f.get("password"),
                  from: f.get("from"),
                });
              });
            }}
          >
            <div className="form-grid">
              {[
                ["host", t("Сервер", "Host"), "text"],
                ["from", t("Адрес отправителя", "From address"), "email"],
                ["username", t("Логин", "Username"), "text"],
                ["password", t("Пароль", "Password"), "password"],
              ].map(([name, label, type]) => (
                <Field title={label} key={name}>
                  <input
                    name={name}
                    type={type}
                    defaultValue={name === "port" ? 587 : undefined}
                    autoComplete="off"
                    required={!["username", "password"].includes(name)}
                  />
                </Field>
              ))}
              <Field title={t("Порт", "Port")}>
                <input
                  name="port"
                  type="number"
                  min={1}
                  max={65535}
                  required
                  value={smtpPort}
                  onChange={(e) => setSmtpPort(Number(e.target.value))}
                />
              </Field>
              <Field title="TLS">
                <select
                  name="tls"
                  value={smtpTLS}
                  onChange={(e) => {
                    setSmtpTLS(e.target.value);
                    setSmtpPort(e.target.value === "smtps" ? 465 : 587);
                  }}
                >
                  <option value="starttls">STARTTLS</option>
                  <option value="smtps">SMTPS</option>
                </select>
              </Field>
            </div>
            <button disabled={busy}>{t("Сохранить SMTP", "Save SMTP")}</button>
          </form>
        </section>
      </div>
      <div hidden={section !== "settings"}>
        <ApplicationSettings api={api} t={t} busy={busy} />
      </div>
      <div hidden={section !== "security"}>
        <section className="panel">
          <h2>{t("Журнал безопасности", "Security audit")}</h2>
          <button
            disabled={busy}
            onClick={() =>
              void act(async () => {
                await api("/admin/audit/verify", "POST");
              })
            }
          >
            {t("Проверить целостность", "Verify integrity")}
          </button>
        </section>
      </div>
    </>
  );
}
function stateNameLocal(state: string, t: (ru: string, en: string) => string) {
  return (
    (
      {
        running: t("Запустить", "Run"),
        paused: t("Приостановить", "Pause"),
        cancelled: t("Отменить", "Cancel"),
        completed: t("Завершено", "Completed"),
      } as Record<string, string>
    )[state] || state
  );
}
