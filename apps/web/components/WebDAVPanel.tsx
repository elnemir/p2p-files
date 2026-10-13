"use client";

import { useState } from "react";
import type { Schema } from "../lib/api-types";

export default function WebDAVPanel({
  id,
  t,
  act,
  busy,
  request,
  refresh,
}: {
  id: string;
  t: (ru: string, en: string) => string;
  act: (f: () => Promise<void>) => Promise<void>;
  busy: boolean;
  request: <T>(path: string, method?: string, value?: unknown) => Promise<T>;
  refresh: () => Promise<void>;
}) {
  const [credential, setCredential] =
    useState<Schema<"WebDAVCredential"> | null>(null);
  const [visible, setVisible] = useState(false);
  const [client, setClient] = useState("macos");
  return (
    <details className="panel compact">
      <summary>{t("Загрузка через WebDAV", "Upload with WebDAV")}</summary>
      <p>
        {t(
          "Подключите файловый менеджер к этому черновику и скопируйте файлы. После загрузки обновите список и опубликуйте передачу в FileX. Доступ закроется после публикации.",
          "Connect a file manager to this draft and copy your files. Refresh the list, then publish the transfer in FileX. Publication closes WebDAV access.",
        )}
      </p>
      <p>
        {t(
          "До публикации можно читать, добавлять, заменять и переименовывать файлы. Если новая загрузка оборвётся, прежний файл сохранится. Перед публикацией закройте открытые файлы и отключите сетевую папку. Для возобновления больших загрузок используйте браузер.",
          "Before publication you can read, add, replace and rename files. An interrupted replacement preserves the old file. Close open files and disconnect the network folder before publishing. Use the browser for resumable large uploads.",
        )}
      </p>
      <label className="field">
        <span>{t("Ваш файловый менеджер", "Your file manager")}</span>
        <select
          aria-label={t("Ваш файловый менеджер", "Your file manager")}
          value={client}
          onChange={(e) => setClient(e.target.value)}
        >
          <option value="macos">macOS · Finder</option>
          <option value="windows">
            Windows · {t("Проводник", "File Explorer")}
          </option>
          <option value="linux">Linux · GNOME Files</option>
        </select>
      </label>
      <ol aria-label={t("Подключение WebDAV", "WebDAV connection")}>
        {client === "macos" && (
          <li>
            {t(
              "Finder → Переход → Подключение к серверу (⌘K). Вставьте адрес WebDAV из этой панели.",
              "Finder → Go → Connect to Server (⌘K). Paste the WebDAV address from this panel.",
            )}
          </li>
        )}
        {client === "windows" && (
          <li>
            {t(
              "Проводник → Этот компьютер → Добавить сетевое расположение. Укажите адрес WebDAV по HTTPS. Требуется служба WebClient и доверенный сертификат сервера.",
              "File Explorer → This PC → Add a network location. Enter the HTTPS WebDAV address. The WebClient service and a trusted server certificate are required.",
            )}
          </li>
        )}
        {client === "linux" && (
          <li>
            {t(
              "GNOME Files → Другие места → Подключиться к серверу. Используйте davs:// вместо https:// в адресе; для локального HTTP — dav://. Нужен пакет gvfs-backends.",
              "GNOME Files → Other Locations → Connect to Server. Replace https:// with davs://; for local HTTP use dav://. The gvfs-backends package is required.",
            )}
          </li>
        )}
        <li>
          {t(
            "Войдите как filex с паролем подключения. Скопируйте файлы в папку черновика.",
            "Sign in as filex with the connection password. Copy files into the draft folder.",
          )}
        </li>
        <li>
          {t(
            "Обновите список в FileX, проверьте готовность файлов, отключите папку и опубликуйте передачу.",
            "Refresh the list in FileX, check that the files are ready, disconnect the folder and publish the transfer.",
          )}
        </li>
      </ol>
      {client === "windows" && (
        <p>
          {t(
            "Фактическая проверка Windows ожидает тестового стенда. Системный WebClient может ограничивать размер файла; для больших файлов используйте браузер с возобновлением загрузки.",
            "Windows validation is awaiting a test machine. WebClient may limit file sizes; use the resumable browser upload for large files.",
          )}
        </p>
      )}
      {!credential && (
        <form
          className="toolbar"
          onSubmit={(event) => {
            event.preventDefault();
            const values = new FormData(event.currentTarget);
            const password = String(values.get("password") || "");
            const ttlSeconds = Number(values.get("days")) * 86400;
            event.currentTarget.reset();
            void act(async () => {
              await request("/auth/reauth", "POST", { password });
              const value = await request<Schema<"WebDAVCredential">>(
                `/transfers/${id}/webdav-credentials`,
                "POST",
                { ttlSeconds },
              );
              setCredential(value);
            });
          }}
        >
          <label className="field">
            <span>
              {t(
                "Пароль аккаунта для выдачи доступа",
                "Account password to issue access",
              )}
            </span>
            <input
              type="password"
              name="password"
              autoComplete="current-password"
              required
              disabled={busy}
            />
          </label>
          <label className="field">
            <span>{t("Срок доступа WebDAV", "WebDAV access lifetime")}</span>
            <select name="days" defaultValue="7" disabled={busy}>
              <option value="1">{t("1 день", "1 day")}</option>
              <option value="7">{t("7 дней", "7 days")}</option>
              <option value="30">{t("30 дней", "30 days")}</option>
            </select>
          </label>
          <button disabled={busy}>
            {t("Создать доступ WebDAV", "Create WebDAV access")}
          </button>
        </form>
      )}
      {credential && (
        <div>
          <p role="status">
            {t(
              "Сохраните пароль подключения: он показывается один раз. Отозвать доступ также можно в настройках API-токенов.",
              "Save the connection password: it is shown only once. You can also revoke access in API token settings.",
            )}
          </p>
          <label className="field">
            <span>{t("Адрес WebDAV", "WebDAV address")}</span>
            <input readOnly value={credential.url} />
          </label>
          <label className="field">
            <span>{t("Имя пользователя WebDAV", "WebDAV username")}</span>
            <input readOnly value={credential.username} />
          </label>
          <label className="field">
            <span>
              {t("Пароль подключения WebDAV", "WebDAV connection password")}
            </span>
            <input
              readOnly
              type={visible ? "text" : "password"}
              autoComplete="off"
              value={credential.password}
            />
          </label>
          <label className="check">
            <input
              type="checkbox"
              checked={visible}
              onChange={(e) => setVisible(e.target.checked)}
            />
            {t("Показать пароль подключения", "Show connection password")}
          </label>
          <p>
            {t("Действует до", "Expires at")}:{" "}
            <time dateTime={credential.expiresAt}>{credential.expiresAt}</time>
          </p>
          <button
            className="secondary"
            disabled={busy}
            onClick={() =>
              void act(async () => {
                await request(`/auth/tokens/${credential.id}`, "DELETE");
                setCredential(null);
                setVisible(false);
              })
            }
          >
            {t("Отозвать доступ WebDAV", "Revoke WebDAV access")}
          </button>
        </div>
      )}
      <p>
        {t(
          "Используйте адрес из этой панели, имя filex и отдельный пароль подключения. Пароль аккаунта нужен только для выдачи доступа в FileX. Для удалённого подключения требуется HTTPS.",
          "Use the address from this panel, username filex and the separate connection password. Your account password is only needed to issue access in FileX. Remote connections require HTTPS.",
        )}
      </p>
      <button
        className="secondary"
        disabled={busy}
        onClick={() => void act(refresh)}
      >
        {t("Обновить файлы после WebDAV", "Refresh files after WebDAV")}
      </button>
    </details>
  );
}
