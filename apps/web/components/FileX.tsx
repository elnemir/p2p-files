"use client";
import Brand from "./Brand";
import Icon, { type IconName } from "./Icon";
import { useBranding } from "./BrandingProvider";
import BrandingPanel from "./BrandingPanel";
import IdentityLogin, { IdentityCompletion } from "./IdentityLogin";
import IdentityProvidersPanel from "./IdentityProvidersPanel";
import OperationsPanel from "./OperationsPanel";
import AdminExportsPanel from "./AdminExportsPanel";
import WebDAVPanel from "./WebDAVPanel";
import {
  useEffect,
  useCallback,
  useEffectEvent,
  useRef,
  useState,
  type FormEvent,
  type ReactNode,
} from "react";
import type { Schema, Response } from "../lib/api-types";
type User = {
  id: string;
  email: string;
  role: string;
  locale: string;
  sessionId: string;
};
type Mode = "owner" | "receive" | "help" | "status" | "verify";
async function api<T = unknown>(
  path: string,
  method = "GET",
  value?: unknown,
  signal?: AbortSignal,
): Promise<T> {
  const headers: Record<string, string> = {};
  if (path.startsWith("/uploads/")) headers["Tus-Resumable"] = "1.0.0";
  if (value !== undefined) headers["Content-Type"] = "application/json";
  if (typeof window !== "undefined")
    headers["X-CSRF-Token"] = localStorage.getItem("filex.csrf") || "";
  const response = await fetch(`/api/v1${path}`, {
    method,
    headers,
    body: value === undefined ? undefined : JSON.stringify(value),
    credentials: "same-origin",
    cache: "no-store",
    signal,
  });
  if (!response.ok) {
    const data = await response.json().catch(() => ({}));
    throw new Error(data.code || `http_${response.status}`);
  }
  return response.status === 204 ? (undefined as T) : await response.json();
}
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
function saveDownload(url: string) {
  const a = document.createElement("a");
  a.href = url;
  a.download = "";
  document.body.append(a);
  a.click();
  a.remove();
}
function size(n: number) {
  return n < 1024
    ? `${n} B`
    : n < 1048576
      ? `${(n / 1024).toFixed(1)} KB`
      : `${(n / 1048576).toFixed(1)} MB`;
}
function Field({ title, children }: { title: string; children: ReactNode }) {
  return (
    <label className="field">
      <span>{title}</span>
      {children}
    </label>
  );
}
interface Entry {
  isFile: boolean;
  name: string;
  file(
    success: (file: File) => void,
    error: (error: DOMException) => void,
  ): void;
  createReader(): {
    readEntries(
      success: (entries: Entry[]) => void,
      error: (error: DOMException) => void,
    ): void;
  };
}
async function dropped(
  data: DataTransfer,
): Promise<{ file: File; path: string }[]> {
  const output: { file: File; path: string }[] = [];
  async function walk(entry: Entry, prefix: string) {
    if (output.length > 10000) throw new Error("too_many_files");
    if (entry.isFile) {
      const file = await new Promise<File>((resolve, reject) =>
        entry.file(resolve, reject),
      );
      output.push({ file, path: prefix + file.name });
    } else {
      const reader = entry.createReader();
      for (;;) {
        const entries = await new Promise<Entry[]>((resolve, reject) =>
          reader.readEntries(resolve, reject),
        );
        if (!entries.length) break;
        for (const child of entries)
          await walk(child, prefix + entry.name + "/");
      }
    }
  }
  const entries = Array.from(data.items)
    .map((item) => item.webkitGetAsEntry() as unknown as Entry | null)
    .filter((entry): entry is Entry => entry !== null);
  if (entries.length) for (const entry of entries) await walk(entry, "");
  else
    for (const file of Array.from(data.files))
      output.push({ file, path: file.name });
  return output;
}
export default function FileX({ mode = "owner" }: { mode?: Mode }) {
  const [locale, setLocale] = useState("ru");
  const { value: branding } = useBranding();
  const [theme, setTheme] = useState("inherit");
  const [user, setUser] = useState<User | null>(null);
  const [loaded, setLoaded] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false);
  const [tab, setTab] = useState("transfers");
  const [transfers, setTransfers] = useState<Schema<"SearchResult">["items"]>(
    [],
  );
  const [nextCursor, setNextCursor] = useState("");
  const [query, setQuery] = useState("");
  const [stateFilter, setStateFilter] = useState("");
  const [views, setViews] = useState<Response<"/api/v1/views">>([]);
  const [selectedId, setSelectedId] = useState("");
  const [detail, setDetail] = useState<Schema<"Transfer"> | null>(null);
  const [chosen, setChosen] = useState<string[]>([]);
  const [chosenTransfers, setChosenTransfers] = useState<string[]>([]);
  const [fileSearch, setFileSearch] = useState("");
  const [sort, setSort] = useState("path");
  const [tree, setTree] = useState(false);
  const [recipients, setRecipients] = useState("");
  const [common, setCommon] = useState(true);
  const [pin, setPin] = useState("");
  const [ttl, setTTL] = useState(24);
  const [review, setReview] = useState(false);
  const [movePath, setMovePath] = useState("");
  const [folder, setFolder] = useState("");
  const [folders, setFolders] = useState<
    Response<"/api/v1/transfers/{id}/folders">
  >([]);
  const [title, setTitle] = useState("");
  const [journal, setJournal] = useState<Schema<"JournalEntry">[]>([]);
  const [progress, setProgress] = useState<{
    path: string;
    done: number;
    total: number;
  } | null>(null);
  const uploadAbort = useRef<AbortController | null>(null);
  const [reports, setReports] = useState<Schema<"Report">[]>([]);
  const [reportDetail, setReportDetail] =
    useState<Schema<"ReportDetail"> | null>(null);
  const [exports, setExports] = useState<Schema<"Export">[]>([]);
  const [bulk, setBulk] = useState<Schema<"Bulk">[]>([]);
  const [notifications, setNotifications] = useState<Schema<"OwnerEvent">[]>(
    [],
  );
  const [deliveries, setDeliveries] = useState<Schema<"Delivery">[]>([]);
  const lastEvent = useRef(0);
  const [preferences, setPreferences] = useState<
    Schema<"NotificationPreferences">
  >({
    locale: "ru",
    timezone: "UTC",
    digest: "immediate",
    quietStart: null as number | null,
    quietEnd: null as number | null,
    browserEnabled: false,
  });
  const [token, setToken] = useState("");
  const [recipientReady, setRecipientReady] = useState(false);
  const [recipientFiles, setRecipientFiles] = useState<
    Response<"/api/v1/recipient/files">
  >([]);
  const [downloadURL, setDownloadURL] = useState("");
  const [banner, setBanner] = useState({ ru: "", en: "" });
  const [status, setStatus] = useState("checking");
  const [verificationId, setVerificationId] = useState("");
  const [verification, setVerification] =
    useState<Response<"/api/v1/verify/reports/{id}"> | null>(null);
  const [localHash, setLocalHash] = useState("");
  const t = useCallback(
    (ru: string, en: string) => (locale === "en" ? en : ru),
    [locale],
  );
  const stateName = (s: string) =>
    ({
      draft: t("Черновик", "Draft"),
      active: t("Опубликовано", "Published"),
      ready: t("Готово", "Ready"),
      uploading: t("Загрузка", "Uploading"),
      finalizing: t("Проверка", "Checking"),
      leased: t("Скачивается", "Downloading"),
      deleting: t("Удаляется", "Deleting"),
      deleted: t("Удалено", "Deleted"),
      cancelled: t("Отменено", "Cancelled"),
      completed: t("Завершено", "Completed"),
      expired: t("Срок истёк", "Expired"),
      revoked: t("Отозвано", "Revoked"),
      integrity_failed: t("Ошибка целостности", "Integrity failure"),
      storage_missing: t("Файл недоступен", "File unavailable"),
      rejected: t("Отклонено политикой", "Rejected by policy"),
      pending: t("Ожидает", "Pending"),
      queued: t("В очереди", "Queued"),
      paused: t("Приостановлено", "Paused"),
      running: t("Выполняется", "Running"),
      rendering: t("Формируется", "Rendering"),
      materialized: t("Снимок готов", "Snapshot ready"),
      snapshotting: t("Создаётся снимок", "Creating snapshot"),
      failed: t("Ошибка", "Failed"),
      dead: t("Требует повтора", "Retry required"),
      delivered: t("Доставлено", "Delivered"),
      trashed: t("В корзине", "In trash"),
      purged: t("Удалено навсегда", "Purged"),
    })[s] || s;
  const date = (s: string) =>
    new Intl.DateTimeFormat(locale, {
      dateStyle: "medium",
      timeStyle: "short",
      timeZone: preferences.timezone || "UTC",
    }).format(new Date(s));
  const message = (code: string) =>
    ({
      maintenance: t(
        "Сервис находится на обслуживании. Администратор проверяет восстановление.",
        "The service is under maintenance. An administrator is reviewing recovery.",
      ),
      restart_required: t(
        "Требуется перезапуск API и worker после восстановления. Затем войдите снова.",
        "Restart the API and worker after recovery, then sign in again.",
      ),
      unauthenticated: t(
        "Войдите снова или проверьте ссылку и пароль.",
        "Sign in again or check your link and password.",
      ),
      forbidden: t(
        "Нет доступа. Для действий администратора подтвердите пароль в настройках.",
        "Access denied. Reauthenticate in Settings for administrator actions.",
      ),
      conflict: t(
        "Состояние изменилось. Обновите список и повторите действие.",
        "State changed. Refresh the list and try again.",
      ),
      invalid_input: t(
        "Проверьте заполненные поля, формат пути и длину пароля.",
        "Check the fields, file path and password length.",
      ),
      rate_limited: t(
        "Слишком много запросов. Повторите через минуту.",
        "Too many requests. Try again in one minute.",
      ),
      not_found: t(
        "Объект уже недоступен.",
        "This item is no longer available.",
      ),
    })[code] ||
    t(
      "Не удалось выполнить действие. Данные сохранены; повторите попытку.",
      "The action failed. Your data is retained; please retry.",
    );
  const broadcast = () => {
    const channel = new BroadcastChannel("filex");
    channel.postMessage("refresh");
    channel.close();
  };
  async function act(fn: () => Promise<void>) {
    setError("");
    setNotice("");
    setBusy(true);
    try {
      await fn();
      broadcast();
    } catch (e) {
      setError(
        e instanceof DOMException && e.name === "AbortError"
          ? t(
              "Загрузка приостановлена. Выберите тот же файл для продолжения.",
              "Upload paused. Select the same file to resume.",
            )
          : message(e instanceof Error ? e.message : ""),
      );
    } finally {
      setBusy(false);
    }
  }
  async function refresh() {
    if (!user) return;
    if (tab === "transfers") {
      if (selectedId) {
        const data = await api<Schema<"Transfer">>(`/transfers/${selectedId}`);
        setDetail(data);
        setFolders(await api(`/transfers/${selectedId}/folders`));
        setJournal(await api(`/transfers/${selectedId}/journal`));
      } else {
        const data = await api<Schema<"SearchResult">>(
          "/transfers/search",
          "POST",
          {
            query,
            state: stateFilter,
          },
        );
        setTransfers(data.items);
        setNextCursor(data.nextCursor);
        setViews(await api("/views"));
      }
    }
    if (tab === "reports") setReports(await api("/reports"));
    if (tab === "jobs") {
      setExports(await api("/exports"));
      setBulk(await api("/bulk"));
    }
    if (tab === "notifications") {
      setNotifications(await api("/notifications"));
      setDeliveries(await api("/deliveries"));
    }
  }
  const refreshRef = useRef(refresh);
  refreshRef.current = refresh;
  useEffect(() => {
    const check = () =>
      void api<{ ru: string; en: string }>("/banner")
        .then(setBanner)
        .catch(() => undefined);
    check();
    const timer = setInterval(check, 60000);
    return () => clearInterval(timer);
  }, []);
  useEffect(() => {
    const stored = localStorage.getItem("filex.locale") || "ru";
    setLocale(stored);
    setTheme(localStorage.getItem("filex.theme") || "inherit");
    if (mode === "owner") {
      api<User>("/auth/me")
        .then(async (u) => {
          setUser(u);
          try {
            const prefs = await api<Schema<"NotificationPreferences">>(
              "/notification-preferences",
            );
            setPreferences(prefs);
            setLocale(prefs.locale);
          } catch (e) {
            if (
              u.role === "admin" &&
              e instanceof Error &&
              e.message === "maintenance"
            )
              setTab("admin");
            else throw e;
          }
        })
        .catch(() => setUser(null))
        .finally(() => setLoaded(true));
    } else {
      setLoaded(true);
      if (mode === "receive") {
        setToken(
          new URLSearchParams(location.hash.slice(1)).get("token") || "",
        );
        history.replaceState(null, "", location.pathname);
        api<Response<"/api/v1/recipient/files">>("/recipient/files")
          .then((files) => {
            setRecipientFiles(files);
            setRecipientReady(true);
          })
          .catch(() => undefined);
      }
      if (mode === "verify")
        setVerificationId(
          new URLSearchParams(location.hash.slice(1)).get("report") || "",
        );
    }
  }, [mode]);
  useEffect(() => {
    document.documentElement.lang = locale;
    document.documentElement.dataset.theme =
      theme === "inherit" ? branding.defaultTheme : theme;
    document.title = `${branding.productName} — ${locale === "en" ? branding.taglineEn : branding.taglineRu}`;
    localStorage.setItem("filex.locale", locale);
    if (theme === "inherit") localStorage.removeItem("filex.theme");
    else localStorage.setItem("filex.theme", theme);
  }, [locale, theme, branding]);
  useEffect(() => {
    if (!user) return;
    void refreshRef.current().catch(() => undefined);
    const timer = setInterval(
      () => void refreshRef.current().catch(() => undefined),
      5000,
    );
    const channel = new BroadcastChannel("filex");
    channel.onmessage = () => void refreshRef.current().catch(() => undefined);
    return () => {
      clearInterval(timer);
      channel.close();
    };
  }, [user, selectedId, tab]);
  useEffect(() => {
    if (mode !== "status") return;
    const check = () =>
      api<Schema<"Status">>("/status")
        .then((data) => setStatus(data.status))
        .catch(() => setStatus("unavailable"));
    void check();
    const timer = setInterval(check, 15000);
    return () => clearInterval(timer);
  }, [mode]);
  useEffect(() => {
    if (!user) return;
    let active = true;
    const receive = (event: Schema<"OwnerEvent">) => {
      if (Number(event.id) <= lastEvent.current) return;
      lastEvent.current = Number(event.id);
      if (
        preferences.browserEnabled &&
        "Notification" in window &&
        Notification.permission === "granted"
      )
        new Notification(branding.productName, {
          icon: branding.favicon || undefined,
          body: t("Изменилось состояние передачи", "Transfer status changed"),
          tag: String(event.id),
        });
      broadcast();
    };
    const socket = new WebSocket(
      `${location.protocol === "https:" ? "wss:" : "ws:"}//${location.host}/api/v1/events?after=${lastEvent.current}`,
    );
    socket.onmessage = (event) => {
      if (active) receive(JSON.parse(event.data));
    };
    const poll = setInterval(() => {
      void api<Schema<"OwnerEvent">[]>(
        `/notifications?after=${lastEvent.current}`,
      )
        .then((items) => {
          if (active) items.forEach(receive);
        })
        .catch(() => undefined);
    }, 15000);
    return () => {
      active = false;
      socket.close();
      clearInterval(poll);
    };
  }, [
    user,
    preferences.browserEnabled,
    t,
    branding.productName,
    branding.favicon,
  ]);
  async function login(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const data = new FormData(e.currentTarget);
    await act(async () => {
      const result = await api<Schema<"Session">>("/auth/login", "POST", {
        email: data.get("email"),
        password: data.get("password"),
      });
      localStorage.setItem("filex.csrf", result.csrfToken);
      const u = await api<User>("/auth/me");
      setUser(u);
      try {
        setPreferences(
          await api<Schema<"NotificationPreferences">>(
            "/notification-preferences",
          ),
        );
      } catch (e) {
        if (
          u.role === "admin" &&
          e instanceof Error &&
          e.message === "maintenance"
        )
          setTab("admin");
        else throw e;
      }
    });
  }
  async function openTransfer(id: string) {
    const data = await api<Schema<"Transfer">>(`/transfers/${id}`);
    setDetail(data);
    setTitle(data.title);
    setSelectedId(id);
    setChosen([]);
    setReview(false);
    setFileSearch("");
  }
  async function upload(input: { file: File; path: string }[]) {
    if (!selectedId || !user) return;
    const controller = new AbortController();
    uploadAbort.current = controller;
    try {
      for (const { file, path: relative } of input) {
        const path =
          (folder ? folder.replace(/\/+$/, "") + "/" : "") +
          relative.normalize("NFC");
        const key = `filex.upload.${user.id}.${selectedId}.${path}.${file.size}.${file.lastModified}`;
        let url = localStorage.getItem(key) || "";
        let offset = 0;
        const headers = {
          "X-CSRF-Token": localStorage.getItem("filex.csrf") || "",
          "Tus-Resumable": "1.0.0",
        };
        if (url) {
          const head = await fetch(url, {
            method: "HEAD",
            headers,
            signal: controller.signal,
            cache: "no-store",
          });
          if (head.ok) offset = Number(head.headers.get("Upload-Offset"));
          else {
            localStorage.removeItem(key);
            url = "";
          }
        }
        if (!url) {
          const response = await fetch(
            `/api/v1/transfers/${selectedId}/uploads`,
            {
              method: "POST",
              headers: {
                ...headers,
                "Upload-Length": String(file.size),
                "Upload-Metadata": `filename ${btoa(String.fromCharCode(...new TextEncoder().encode(path)))}`,
              },
              signal: controller.signal,
            },
          );
          if (!response.ok) throw new Error((await response.json()).code);
          url = response.headers.get("Location")!;
          localStorage.setItem(key, url);
        }
        let attempts = 0;
        while (offset < file.size) {
          setProgress({ path, done: offset, total: file.size });
          try {
            const response = await fetch(url, {
              method: "PATCH",
              headers: {
                ...headers,
                "Content-Type": "application/offset+octet-stream",
                "Upload-Offset": String(offset),
              },
              body: file.slice(offset, offset + 4 * 1024 * 1024),
              signal: controller.signal,
            });
            if (!response.ok)
              throw new Error((await response.json()).code || "upload_failed");
            offset = Number(response.headers.get("Upload-Offset"));
            attempts = 0;
          } catch (e) {
            if (controller.signal.aborted || ++attempts > 4) throw e;
            await new Promise((resolve) => setTimeout(resolve, attempts * 500));
            const head = await fetch(url, {
              method: "HEAD",
              headers,
              signal: controller.signal,
              cache: "no-store",
            });
            if (!head.ok) throw e;
            offset = Number(head.headers.get("Upload-Offset"));
          }
        }
        setProgress({ path, done: file.size, total: file.size });
        await api(
          url.replace("/api/v1", "") + "/finalize",
          "POST",
          undefined,
          controller.signal,
        );
        localStorage.removeItem(key);
        await refreshRef.current();
      }
    } finally {
      uploadAbort.current = null;
      setProgress(null);
    }
  }
  async function downloadArtifact(id: string) {
    const credential = await api<{ token: string; artifactId: string }>(
      `/artifacts/${id}/credentials`,
      "POST",
    );
    const claimed = await api<{ url: string }>(
      `/artifacts/${id}/claim`,
      "POST",
      credential,
    );
    saveDownload(claimed.url);
  }
  async function bulkPreview(ids: string[], action: string) {
    await api("/bulk", "POST", { ids, action, dryRun: true });
    setTab("jobs");
  }
  const visibleFiles = (detail?.files || [])
    .filter((f) => f.path.toLowerCase().includes(fileSearch.toLowerCase()))
    .sort((a, b) =>
      sort === "size" ? b.size - a.size : a.path.localeCompare(b.path),
    );
  const toggle = (id: string) =>
    setChosen(
      chosen.includes(id) ? chosen.filter((v) => v !== id) : [...chosen, id],
    );
  const button = (label: string, fn: () => Promise<void>, secondary = true) => (
    <button
      type="button"
      className={secondary ? "secondary" : ""}
      disabled={busy}
      onClick={() => void act(fn)}
    >
      {label}
    </button>
  );
  return (
    <>
      <header className="site-header">
        <Brand />
        <nav aria-label={t("Навигация", "Navigation")}>
          <a href="/help">
            <Icon name="help" />
            {t("Помощь", "Help")}
          </a>
          <a href="/status">
            <Icon name="status" />
            {t("Статус", "Status")}
          </a>
          <label>
            <span className="sr-only">{t("Язык", "Language")}</span>
            <select
              aria-label={t("Язык", "Language")}
              disabled={!loaded}
              value={locale}
              onChange={(e) => setLocale(e.target.value)}
            >
              <option value="ru">RU</option>
              <option value="en">EN</option>
            </select>
          </label>
          <label>
            <span className="sr-only">{t("Тема", "Theme")}</span>
            <select
              aria-label={t("Тема", "Theme")}
              disabled={!loaded}
              value={theme}
              onChange={(e) => setTheme(e.target.value)}
            >
              <option value="inherit">{t("По умолчанию", "Default")}</option>
              <option value="system">{t("Системная", "System")}</option>
              <option value="light">{t("Светлая", "Light")}</option>
              <option value="dark">{t("Тёмная", "Dark")}</option>
            </select>
          </label>
          {user &&
            button(t("Выйти", "Sign out"), async () => {
              await api("/auth/logout", "POST");
              localStorage.removeItem("filex.csrf");
              location.reload();
            })}
        </nav>
      </header>
      <main id="main" className="workspace" tabIndex={-1}>
        {(locale === "en" ? banner.en : banner.ru) && (
          <aside
            className="notice compact"
            aria-label={t("Объявление сервиса", "Service announcement")}
          >
            {locale === "en" ? banner.en : banner.ru}
          </aside>
        )}
        <div aria-live="polite" role="status">
          {notice && <p className="notice compact">{notice}</p>}
        </div>
        {error && (
          <div role="alert" className="error">
            {error}
            <button
              aria-label={t("Закрыть сообщение", "Dismiss message")}
              onClick={() => setError("")}
            >
              <Icon name="close" />
            </button>
          </div>
        )}
        {mode === "owner" && !loaded && (
          <p>{t("Проверяем сеанс…", "Checking session…")}</p>
        )}
        {mode === "owner" && <IdentityCompletion api={api} t={t} />}
        {mode === "owner" && loaded && !user && (
          <div className="welcome">
            <section>
              <p className="eyebrow">
                {t(branding.taglineRu, branding.taglineEn)}
              </p>
              <h1>
                {t("Передать.", "Send.")}
                <br />
                <span>{t("И отпустить.", "And let go.")}</span>
              </h1>
              <p className="lead">
                {t(
                  "Одноразовая передача файлов с контролем срока, доступа и удаления.",
                  "One-time file transfer with control over access, expiry and deletion.",
                )}
              </p>
            </section>
            <form className="panel login" onSubmit={login}>
              <h2>{t("Войти в ", "Sign in to ") + branding.productName}</h2>
              <p>
                {t(
                  "Учётную запись создаёт администратор вашей организации.",
                  "Your organization administrator creates your account.",
                )}
              </p>
              <Field title="Email">
                <input
                  name="email"
                  type="email"
                  autoComplete="username"
                  required
                />
              </Field>
              <Field title={t("Пароль", "Password")}>
                <input
                  name="password"
                  type="password"
                  autoComplete="current-password"
                  required
                />
              </Field>
              <button disabled={busy}>
                {busy ? t("Входим…", "Signing in…") : t("Войти", "Sign in")}
              </button>
            </form>
            <IdentityLogin api={api} t={t} />
          </div>
        )}
        {mode === "owner" && user && (
          <>
            <div className="account">
              <p>{user.email}</p>
              <span className="badge">
                {user.role === "admin"
                  ? t("Администратор", "Administrator")
                  : t("Личный кабинет", "Workspace")}
              </span>
            </div>
            <nav
              className="tabs"
              aria-label={t("Разделы кабинета", "Workspace sections")}
            >
              {[
                ["transfers", t("Передачи", "Transfers")],
                ["reports", t("Отчёты", "Reports")],
                ["jobs", t("Задачи", "Jobs")],
                ["notifications", t("Уведомления", "Notifications")],
                ["settings", t("Настройки", "Settings")],
                ...(user.role === "admin"
                  ? [["admin", t("Администрирование", "Administration")]]
                  : []),
              ].map(([value, label]) => (
                <button
                  key={value}
                  className={tab === value ? "active" : "secondary"}
                  onClick={() => {
                    setTab(value);
                    setReportDetail(null);
                  }}
                >
                  <Icon name={value as IconName} />
                  {label}
                </button>
              ))}
            </nav>
            {tab === "transfers" && !selectedId && (
              <>
                <div className="section-title">
                  <div>
                    <p className="eyebrow">
                      {t("Рабочее пространство", "Workspace")}
                    </p>
                    <h1 className="page-title">{t("Передачи", "Transfers")}</h1>
                  </div>
                  {button(
                    t("Новая передача", "New transfer"),
                    async () => {
                      const d = await api<Schema<"Draft">>(
                        "/transfers",
                        "POST",
                      );
                      await openTransfer(d.id);
                    },
                    false,
                  )}
                </div>
                <form
                  className="toolbar"
                  onSubmit={(e) => {
                    e.preventDefault();
                    void act(refresh);
                  }}
                >
                  <Field title={t("Поиск по названию", "Search titles")}>
                    <input
                      value={query}
                      onChange={(e) => setQuery(e.target.value)}
                      maxLength={500}
                    />
                  </Field>
                  <Field title={t("Состояние", "State")}>
                    <select
                      value={stateFilter}
                      onChange={(e) => setStateFilter(e.target.value)}
                    >
                      <option value="">{t("Все", "All")}</option>
                      {[
                        "draft",
                        "active",
                        "completed",
                        "expired",
                        "revoked",
                      ].map((s) => (
                        <option key={s} value={s}>
                          {stateName(s)}
                        </option>
                      ))}
                    </select>
                  </Field>
                  <button disabled={busy}>{t("Найти", "Search")}</button>
                  {button(t("Сохранить вид", "Save view"), async () => {
                    const name =
                      query ||
                      stateName(stateFilter) ||
                      t("Все передачи", "All transfers");
                    await api("/views", "POST", {
                      name,
                      filter: { query, state: stateFilter },
                    });
                    setViews(await api("/views"));
                  })}
                </form>
                {views.length > 0 && (
                  <div className="toolbar">
                    <span>{t("Сохранённые виды:", "Saved views:")}</span>
                    {views.map((v) => (
                      <span className="view-chip" key={v.id}>
                        <button
                          className="secondary"
                          onClick={() =>
                            void act(async () => {
                              setQuery(v.filter.query || "");
                              setStateFilter(v.filter.state || "");
                              const data = await api<Schema<"SearchResult">>(
                                "/transfers/search",
                                "POST",
                                { viewId: v.id },
                              );
                              setTransfers(data.items);
                              setNextCursor(data.nextCursor);
                              history.replaceState(null, "", `/#view=${v.id}`);
                            })
                          }
                        >
                          {v.name}
                        </button>
                        <button
                          className="secondary"
                          aria-label={
                            t("Удалить вид ", "Delete view ") + v.name
                          }
                          onClick={() =>
                            void act(async () => {
                              await api(`/views/${v.id}`, "DELETE");
                              setViews(await api("/views"));
                            })
                          }
                        >
                          <Icon name="close" />
                        </button>
                      </span>
                    ))}
                  </div>
                )}
                {chosenTransfers.length > 0 && (
                  <div className="selection">
                    {chosenTransfers.length} {t("выбрано", "selected")}
                    {button(
                      t("Предпросмотр отзыва", "Preview revocation"),
                      () => bulkPreview(chosenTransfers, "revoke"),
                    )}
                  </div>
                )}
                <div className="rows">
                  {transfers.length === 0 ? (
                    <div className="empty">
                      <h2>
                        {t(
                          "Здесь появятся ваши передачи",
                          "Your transfers will appear here",
                        )}
                      </h2>
                      <p>
                        {t(
                          "Создайте черновик и добавьте файлы или папку.",
                          "Create a draft and add files or a folder.",
                        )}
                      </p>
                    </div>
                  ) : (
                    transfers.map((d) => (
                      <div className="list-row" key={d.id}>
                        <input
                          type="checkbox"
                          aria-label={
                            t("Выбрать передачу ", "Select transfer ") +
                            (d.title || d.id)
                          }
                          checked={chosenTransfers.includes(d.id)}
                          onChange={() =>
                            setChosenTransfers(
                              chosenTransfers.includes(d.id)
                                ? chosenTransfers.filter((x) => x !== d.id)
                                : [...chosenTransfers, d.id],
                            )
                          }
                        />
                        <button
                          className="row-link"
                          onClick={() => void act(() => openTransfer(d.id))}
                        >
                          <strong>
                            {d.title || t("Без названия", "Untitled")}
                          </strong>
                          <span>{date(d.createdAt)}</span>
                        </button>
                        <span className="badge">{stateName(d.state)}</span>
                      </div>
                    ))
                  )}
                </div>
                {nextCursor &&
                  button(t("Показать ещё", "Show more"), async () => {
                    const data = await api<Schema<"SearchResult">>(
                      "/transfers/search",
                      "POST",
                      {
                        query,
                        state: stateFilter,
                        after: nextCursor,
                      },
                    );
                    setTransfers([...transfers, ...data.items]);
                    setNextCursor(data.nextCursor);
                  })}
              </>
            )}
            {tab === "transfers" && selectedId && detail && (
              <>
                <button
                  className="text-button"
                  onClick={() => {
                    setSelectedId("");
                    setDetail(null);
                    setChosen([]);
                  }}
                >
                  <Icon name="back" />
                  {t("Все передачи", "All transfers")}
                </button>
                <div className="section-title">
                  <h1 className="page-title">
                    {detail.title || t("Новая передача", "New transfer")}
                  </h1>
                  <span className="badge">{stateName(detail.state)}</span>
                </div>
                <form
                  className="toolbar"
                  onSubmit={(e) => {
                    e.preventDefault();
                    void act(async () => {
                      await api(`/transfers/${selectedId}`, "PATCH", {
                        title,
                        message: "",
                      });
                      await refresh();
                    });
                  }}
                >
                  <Field title={t("Название", "Title")}>
                    <input
                      value={title}
                      onChange={(e) => setTitle(e.target.value)}
                      maxLength={300}
                    />
                  </Field>
                  <button className="secondary" disabled={busy}>
                    {t("Сохранить название", "Save title")}
                  </button>
                </form>
                {detail.state === "draft" && (
                  <>
                    <WebDAVPanel
                      key={selectedId}
                      id={selectedId}
                      t={t}
                      act={act}
                      busy={busy}
                      request={api}
                      refresh={refresh}
                    />
                    <div
                      className="dropzone"
                      onDragOver={(e) => e.preventDefault()}
                      onDrop={(e) => {
                        e.preventDefault();
                        const data = e.dataTransfer;
                        void act(async () => upload(await dropped(data)));
                      }}
                    >
                      <h2>
                        {t(
                          "Перетащите файлы или папку",
                          "Drop files or a folder",
                        )}
                      </h2>
                      <p>
                        {t(
                          "При обрыве выберите те же файлы — загрузка продолжится.",
                          "After an interruption, select the same files to resume.",
                        )}
                      </p>
                      <div className="toolbar">
                        <label className="file-picker">
                          {t("Выбрать файлы", "Choose files")}
                          <input
                            type="file"
                            multiple
                            disabled={busy}
                            onChange={(e) => {
                              const files = Array.from(e.target.files || []);
                              e.target.value = "";
                              void act(() =>
                                upload(
                                  files.map((file) => ({
                                    file,
                                    path: file.webkitRelativePath || file.name,
                                  })),
                                ),
                              );
                            }}
                          />
                        </label>
                        <label className="file-picker secondary">
                          {t("Выбрать папку", "Choose folder")}
                          <input
                            type="file"
                            multiple
                            {...{ webkitdirectory: "" }}
                            disabled={busy}
                            onChange={(e) => {
                              const files = Array.from(e.target.files || []);
                              e.target.value = "";
                              void act(() =>
                                upload(
                                  files.map((file) => ({
                                    file,
                                    path: file.webkitRelativePath || file.name,
                                  })),
                                ),
                              );
                            }}
                          />
                        </label>
                      </div>
                      <Field
                        title={t(
                          "Папка назначения (необязательно)",
                          "Destination folder (optional)",
                        )}
                      >
                        <input
                          value={folder}
                          onChange={(e) => setFolder(e.target.value)}
                          placeholder="documents/2026"
                        />
                      </Field>
                      {folder &&
                        button(
                          t(
                            "Создать виртуальную папку",
                            "Create virtual folder",
                          ),
                          async () => {
                            await api(
                              `/transfers/${selectedId}/folders`,
                              "POST",
                              { path: folder },
                            );
                            await refresh();
                          },
                        )}
                    </div>
                    {progress && (
                      <div className="panel compact" role="status">
                        <p className="break">
                          {progress.path} · {size(progress.done)} /{" "}
                          {size(progress.total)}
                        </p>
                        <progress
                          value={progress.done}
                          max={Math.max(1, progress.total)}
                          aria-label={t("Прогресс загрузки", "Upload progress")}
                        />
                        <button
                          className="secondary"
                          onClick={() => uploadAbort.current?.abort()}
                        >
                          {t("Приостановить", "Pause")}
                        </button>
                      </div>
                    )}
                  </>
                )}
                <div className="toolbar">
                  <Field title={t("Поиск файлов", "Search files")}>
                    <input
                      value={fileSearch}
                      onChange={(e) => setFileSearch(e.target.value)}
                    />
                  </Field>
                  <Field title={t("Сортировка", "Sort")}>
                    <select
                      value={sort}
                      onChange={(e) => setSort(e.target.value)}
                    >
                      <option value="path">{t("По пути", "By path")}</option>
                      <option value="size">{t("По размеру", "By size")}</option>
                    </select>
                  </Field>
                  <label className="check">
                    <input
                      type="checkbox"
                      checked={tree}
                      onChange={(e) => setTree(e.target.checked)}
                    />
                    {t("Дерево папок", "Folder tree")}
                  </label>
                </div>
                {chosen.length > 0 && detail.state === "draft" && (
                  <div className="selection">
                    <span>
                      {chosen.length} {t("выбрано", "selected")}
                    </span>
                    <Field title={t("Переместить в папку", "Move to folder")}>
                      <input
                        value={movePath}
                        onChange={(e) => setMovePath(e.target.value)}
                      />
                    </Field>
                    {button(t("Переместить", "Move"), async () => {
                      await api(`/transfers/${selectedId}/files`, "PATCH", {
                        moves: chosen.map((id) => {
                          const file = detail.files.find((f) => f.id === id);
                          if (!file) throw new Error("conflict");
                          return {
                            id,
                            path:
                              (movePath
                                ? movePath.replace(/\/+$/, "") + "/"
                                : "") + file.path.split("/").at(-1),
                          };
                        }),
                      });
                      setChosen([]);
                      await refresh();
                    })}
                    {button(
                      t("Предпросмотр удаления", "Preview deletion"),
                      () => bulkPreview(chosen, "cancel_upload"),
                    )}
                  </div>
                )}
                <div className="rows">
                  <div className="list-row table-head">
                    <input
                      type="checkbox"
                      aria-label={t("Выбрать все файлы", "Select all files")}
                      checked={
                        visibleFiles.length > 0 &&
                        visibleFiles.every((f) => chosen.includes(f.id))
                      }
                      onChange={(e) =>
                        setChosen(
                          e.target.checked ? visibleFiles.map((f) => f.id) : [],
                        )
                      }
                    />
                    <strong>{t("Файл", "File")}</strong>
                    <span>{t("Размер / состояние", "Size / state")}</span>
                  </div>
                  {tree &&
                    folders.map((f) => (
                      <div className="folder-row" key={f.id}>
                        <Icon name="transfers" />
                        {f.path}/
                        {detail.state === "draft" && (
                          <div className="toolbar">
                            <form
                              className="toolbar"
                              onSubmit={(e) => {
                                e.preventDefault();
                                const path = new FormData(e.currentTarget).get(
                                  "path",
                                );
                                void act(async () => {
                                  await api(
                                    `/transfers/${selectedId}/folders/${f.id}`,
                                    "PATCH",
                                    { path },
                                  );
                                  await refresh();
                                });
                              }}
                            >
                              <input
                                aria-label={
                                  t("Новый путь папки ", "New folder path ") +
                                  f.path
                                }
                                name="path"
                                defaultValue={f.path}
                              />
                              <button className="secondary" disabled={busy}>
                                {t("Переместить папку", "Move folder")}
                              </button>
                            </form>
                            {button(
                              t(
                                "Удалить папку и файлы",
                                "Delete folder and files",
                              ),
                              async () => {
                                await api(
                                  `/transfers/${selectedId}/folders/${f.id}`,
                                  "DELETE",
                                );
                                await refresh();
                              },
                            )}
                          </div>
                        )}
                      </div>
                    ))}
                  {visibleFiles.map((f) => (
                    <div className="file-row" key={f.id}>
                      <input
                        type="checkbox"
                        aria-label={t("Выбрать ", "Select ") + f.path}
                        checked={chosen.includes(f.id)}
                        onChange={() => toggle(f.id)}
                      />
                      <div className="file-name">
                        <strong>
                          {tree && f.path.includes("/") ? (
                            <>
                              <small>
                                {f.path.substring(0, f.path.lastIndexOf("/"))}/
                              </small>
                              {f.path.split("/").at(-1)}
                            </>
                          ) : (
                            f.path
                          )}
                        </strong>
                        {f.sha256 && (
                          <details>
                            <summary>SHA-256</summary>
                            <code>{f.sha256}</code>
                          </details>
                        )}
                      </div>
                      <div>
                        <span>{size(f.size)}</span>
                        <br />
                        <span className="muted">{stateName(f.state)}</span>
                      </div>
                      {detail.state === "draft" && (
                        <div className="row-actions">
                          <form
                            onSubmit={(e) => {
                              e.preventDefault();
                              const value = new FormData(e.currentTarget).get(
                                "path",
                              );
                              void act(async () => {
                                await api(`/files/${f.id}`, "PATCH", {
                                  path: value,
                                });
                                await refresh();
                              });
                            }}
                          >
                            <input
                              name="path"
                              defaultValue={f.path}
                              aria-label={
                                t("Новый путь ", "New path ") + f.path
                              }
                            />
                            <button className="secondary" disabled={busy}>
                              {t("Переименовать", "Rename")}
                            </button>
                          </form>
                          {button(t("Удалить", "Delete"), async () => {
                            await api(`/uploads/${f.id}`, "DELETE");
                            await refresh();
                          })}
                        </div>
                      )}
                    </div>
                  ))}
                  {visibleFiles.length === 0 && (
                    <p className="empty">{t("Нет файлов", "No files")}</p>
                  )}
                </div>
                {detail.state === "draft" && (
                  <section className="panel">
                    <h2>{t("Доступ к передаче", "Transfer access")}</h2>
                    <div className="form-grid">
                      <Field
                        title={t(
                          "Email получателей, по одному на строку",
                          "Recipient emails, one per line",
                        )}
                      >
                        <textarea
                          value={recipients}
                          onChange={(e) => setRecipients(e.target.value)}
                          rows={4}
                        />
                      </Field>
                      <div>
                        <label className="check">
                          <input
                            type="checkbox"
                            checked={common}
                            onChange={(e) => setCommon(e.target.checked)}
                          />
                          {t("Создать общую ссылку", "Create shared link")}
                        </label>
                        <Field
                          title={t(
                            "PIN или пароль для ссылок (необязательно)",
                            "Link PIN or password (optional)",
                          )}
                        >
                          <input
                            type="password"
                            value={pin}
                            onChange={(e) => setPin(e.target.value)}
                            autoComplete="new-password"
                          />
                          <small>
                            {t(
                              "6–12 цифр либо пароль от 12 символов. После публикации можно менять отдельно.",
                              "6–12 digits or a password of at least 12 characters. Each link can be changed separately after publication.",
                            )}
                          </small>
                        </Field>
                        <Field
                          title={t(
                            "Срок доступа, часов",
                            "Access duration, hours",
                          )}
                        >
                          <input
                            type="number"
                            min={1}
                            max={168}
                            value={ttl}
                            onChange={(e) => setTTL(Number(e.target.value))}
                          />
                        </Field>
                      </div>
                    </div>
                    <button
                      disabled={
                        busy ||
                        !detail.files.length ||
                        detail.files.some((f) => f.state !== "ready")
                      }
                      onClick={() => setReview(true)}
                    >
                      {t(
                        "Проверить перед публикацией",
                        "Review before publishing",
                      )}
                    </button>
                    {review && (
                      <div
                        className="review"
                        role="region"
                        aria-label={t(
                          "Проверка публикации",
                          "Publication review",
                        )}
                      >
                        <h3>{t("Готово к публикации", "Ready to publish")}</h3>
                        <p>
                          {detail.files.length} {t("файлов", "files")} ·{" "}
                          {size(detail.files.reduce((n, f) => n + f.size, 0))} ·{" "}
                          {ttl} {t("ч. доступа", "hours of access")}
                        </p>
                        <p>
                          {t(
                            "После публикации состав и пути файлов станут неизменяемыми. Успешное скачивание удалит файл. Публикация не означает доставку письма.",
                            "Publication locks the file set and paths. Successful download deletes the file. Publication does not confirm email delivery.",
                          )}
                        </p>
                        {button(
                          t("Опубликовать передачу", "Publish transfer"),
                          async () => {
                            await api(
                              `/transfers/${selectedId}/publish`,
                              "POST",
                              {
                                expectedRevision: detail.revision,
                                ttlSeconds: ttl * 3600,
                                commonLink: common,
                                commonPassword: common ? pin : "",
                                recipients: recipients
                                  .split(/\n/)
                                  .map((email) => email.trim())
                                  .filter(Boolean)
                                  .map((line) => {
                                    const [email, password] = line.split(
                                      ";",
                                      2,
                                    );
                                    return {
                                      email: email.trim(),
                                      password:
                                        password === undefined
                                          ? pin
                                          : password.trim(),
                                    };
                                  }),
                              },
                            );
                            setPin("");
                            setReview(false);
                            await refresh();
                          },
                          false,
                        )}
                      </div>
                    )}
                  </section>
                )}
                {detail.links?.length > 0 && (
                  <section className="panel">
                    <div className="section-title">
                      <h2>{t("Ссылки получателей", "Recipient links")}</h2>
                      {button(
                        t("Скачать все QR", "Download QR bundle"),
                        async () => {
                          const qr = await api<{ artifactId: string }>(
                            `/transfers/${selectedId}/qr-bundle`,
                            "POST",
                          );
                          await downloadArtifact(qr.artifactId);
                        },
                      )}
                    </div>
                    {detail.links.map((link) => (
                      <div className="link-row" key={link.id}>
                        <div>
                          <strong>
                            {link.email || t("Общая ссылка", "Shared link")}
                          </strong>
                          <input
                            readOnly
                            value={link.url}
                            aria-label={t(
                              "Ссылка получателя",
                              "Recipient link",
                            )}
                          />
                          <div className="toolbar">
                            {button(t("Копировать", "Copy"), async () => {
                              await navigator.clipboard.writeText(link.url);
                              setNotice(t("Ссылка скопирована", "Link copied"));
                            })}
                            <a
                              href={`/api/v1/transfers/${selectedId}/links/${link.id}/qr?format=png`}
                              download
                            >
                              QR PNG
                            </a>
                            <a
                              href={`/api/v1/transfers/${selectedId}/links/${link.id}/qr?format=svg`}
                              download
                            >
                              QR SVG
                            </a>
                            {button(
                              t("Отозвать ссылку", "Revoke link"),
                              async () => {
                                await api(
                                  `/transfers/${selectedId}/links/${link.id}`,
                                  "DELETE",
                                );
                                await refresh();
                              },
                            )}
                          </div>
                          <form
                            className="toolbar"
                            onSubmit={(e) => {
                              e.preventDefault();
                              const password = new FormData(
                                e.currentTarget,
                              ).get("password");
                              e.currentTarget.reset();
                              void act(async () => {
                                await api(
                                  `/transfers/${selectedId}/links/${link.id}/rotate`,
                                  "POST",
                                  { password },
                                );
                                await refresh();
                              });
                            }}
                          >
                            <input
                              name="password"
                              type="password"
                              autoComplete="new-password"
                              aria-label={t(
                                "Новый PIN или пароль ссылки",
                                "New link PIN or password",
                              )}
                            />
                            <button className="secondary" disabled={busy}>
                              {t(
                                "Сменить ссылку и пароль",
                                "Rotate link and password",
                              )}
                            </button>
                          </form>
                        </div>
                      </div>
                    ))}
                  </section>
                )}
                {detail.state !== "draft" && (
                  <div className="toolbar">
                    {detail.state === "active" &&
                      button(
                        t("Отозвать всю передачу", "Revoke transfer"),
                        async () => {
                          await api(`/transfers/${selectedId}/revoke`, "POST");
                          await refresh();
                        },
                      )}
                    {button(t("Создать отчёт", "Create report"), async () => {
                      await api(`/transfers/${selectedId}/reports`, "POST", {
                        kind: "user",
                        locale,
                        timezone: preferences.timezone,
                      });
                      setTab("reports");
                    })}
                    {user.role === "admin" &&
                      button(
                        t("Отчёт администратора", "Administrator report"),
                        async () => {
                          await api(
                            `/transfers/${selectedId}/reports`,
                            "POST",
                            {
                              kind: "admin",
                              locale,
                              timezone: preferences.timezone,
                            },
                          );
                          setTab("reports");
                        },
                      )}
                  </div>
                )}
                {journal.length > 0 && (
                  <section className="panel">
                    <h2>{t("Журнал скачиваний", "Download journal")}</h2>
                    <p>
                      {t(
                        "Общие результаты без идентификации получателей.",
                        "Aggregate results without identifying recipients.",
                      )}
                    </p>
                    {journal.map((j, i) => (
                      <p key={i}>
                        {date(j.occurredAt)} · {j.fileCount}{" "}
                        {t("файлов", "files")} · {size(j.totalBytes)} ·{" "}
                        {stateName(j.status)}
                      </p>
                    ))}
                  </section>
                )}
              </>
            )}
            {tab === "reports" && (
              <>
                <div className="section-title">
                  <h1 className="page-title">{t("Отчёты", "Reports")}</h1>
                  {button(t("Обновить", "Refresh"), refresh)}
                </div>
                <p>
                  {t(
                    "Каждая ревизия фиксирует состояние на момент создания. Отчёт создаётся из карточки передачи.",
                    "Each revision records the state at creation time. Create reports from a transfer.",
                  )}
                </p>
                <div className="rows">
                  {reports.map((rep) => (
                    <div className="list-row" key={rep.id}>
                      <button
                        className="row-link"
                        onClick={() =>
                          void act(async () =>
                            setReportDetail(await api(`/reports/${rep.id}`)),
                          )
                        }
                      >
                        <strong>
                          {rep.kind === "admin"
                            ? t("Административный", "Administrative")
                            : t("Пользовательский", "User")}{" "}
                          · #{rep.revision}
                        </strong>
                        <span>{date(rep.createdAt)}</span>
                      </button>
                      <span>
                        {stateName(rep.state)} · {stateName(rep.renderState)}
                      </span>
                      {rep.state === "active" &&
                        button(t("В корзину", "Move to trash"), async () => {
                          await api(`/reports/${rep.id}/state`, "POST", {
                            state: "trashed",
                          });
                          await refresh();
                        })}
                      {rep.state === "trashed" && (
                        <>
                          {button(t("Восстановить", "Restore"), async () => {
                            await api(`/reports/${rep.id}/state`, "POST", {
                              state: "active",
                            });
                            await refresh();
                          })}
                          <details>
                            <summary>
                              {t("Удалить навсегда", "Purge permanently")}
                            </summary>
                            <p>
                              {t(
                                "Содержимое и файлы будут удалены; история и контрольная сумма сохранятся.",
                                "Content and files will be deleted; history and the checksum will remain.",
                              )}
                            </p>
                            {button(
                              t(
                                "Удалить содержимое навсегда",
                                "Permanently delete content",
                              ),
                              async () => {
                                await api(`/reports/${rep.id}/state`, "POST", {
                                  state: "purged",
                                });
                                setReportDetail(null);
                                await refresh();
                              },
                            )}
                          </details>
                        </>
                      )}
                      {rep.renderState === "failed" &&
                        rep.state === "active" &&
                        button(
                          t("Повторить формирование", "Retry rendering"),
                          async () => {
                            await api(`/reports/${rep.id}/retry`, "POST");
                            await refresh();
                          },
                        )}
                      {rep.kind === "user" && (
                        <a href={`/verify#report=${rep.id}`}>
                          {t("Проверить", "Verify")}
                        </a>
                      )}
                    </div>
                  ))}
                  {reports.length === 0 && (
                    <p className="empty">
                      {t("Отчётов пока нет", "No reports yet")}
                    </p>
                  )}
                </div>
                {reportDetail && (
                  <section className="panel">
                    <h2>{t("Ревизия отчёта", "Report revision")}</h2>
                    <p className="break">
                      <strong>SHA-256:</strong> {reportDetail.canonicalHash}
                    </p>
                    <div className="toolbar">
                      {Object.entries(reportDetail.artifacts || {}).map(
                        ([format, id]) => (
                          <button
                            key={format}
                            disabled={busy || reportDetail.state !== "active"}
                            onClick={() =>
                              void act(() => downloadArtifact(String(id)))
                            }
                          >
                            {format.toUpperCase()}
                          </button>
                        ),
                      )}
                      {button(t("Обновить отчёт", "Refresh report"), async () =>
                        setReportDetail(
                          await api(`/reports/${reportDetail.id}`),
                        ),
                      )}
                      {reportDetail.state === "active" &&
                        reportDetail.renderState === "ready" &&
                        Object.keys(reportDetail.artifacts || {}).length < 3 &&
                        button(
                          t("Заново сформировать файлы", "Regenerate files"),
                          async () => {
                            await api(
                              `/reports/${reportDetail.id}/retry`,
                              "POST",
                            );
                            await refresh();
                          },
                        )}
                    </div>
                    <p>{stateName(reportDetail.renderState)}</p>
                    {reportDetail.history?.map((event, i) => (
                      <p key={i}>
                        {date(event.createdAt)} · {stateName(event.action)}
                      </p>
                    ))}
                    <details>
                      <summary>{t("Снимок данных", "Data snapshot")}</summary>
                      <pre>
                        {JSON.stringify(reportDetail.snapshot, null, 2)}
                      </pre>
                    </details>
                  </section>
                )}
              </>
            )}
            {tab === "jobs" && (
              <>
                <h1 className="page-title">
                  {t("Фоновые задачи", "Background jobs")}
                </h1>
                <section className="panel">
                  <h2>{t("Экспорт передач", "Export transfers")}</h2>
                  <p>
                    {t(
                      "Экспорт использует значения на момент начала задачи. Сжатие выбирается автоматически для больших наборов.",
                      "Exports use values from the start of the job. Large datasets are compressed automatically.",
                    )}
                  </p>
                  <div className="toolbar">
                    {["json", "csv"].map((format) => (
                      <button
                        key={format}
                        disabled={busy}
                        onClick={() =>
                          void act(async () => {
                            await api("/exports", "POST", {
                              kind: "user",
                              format,
                              compression: "auto",
                            });
                            await refresh();
                          })
                        }
                      >
                        + {format.toUpperCase()}
                      </button>
                    ))}
                    {user.role === "admin" &&
                      button(
                        t("Экспорт администратора", "Administrator export"),
                        async () => {
                          await api("/exports", "POST", {
                            kind: "admin",
                            format: "json",
                            compression: "zip",
                          });
                          await refresh();
                        },
                      )}
                  </div>
                  {exports.map((job) => (
                    <div className="list-row" key={job.id}>
                      <div>
                        <strong>
                          {job.format.toUpperCase()} · {date(job.createdAt)}
                        </strong>
                        <p>
                          {stateName(job.state)} · {job.processed} /{" "}
                          {job.rowCount}
                        </p>
                        {job.snapshotAt && (
                          <small>
                            {t("Снимок:", "Snapshot:")} {date(job.snapshotAt)}
                          </small>
                        )}
                      </div>
                      <div className="toolbar">
                        {job.artifactId &&
                          button(t("Скачать", "Download"), async () => {
                            if (job.artifactId)
                              await downloadArtifact(job.artifactId);
                          })}
                        {[
                          "queued",
                          "snapshotting",
                          "materialized",
                          "rendering",
                        ].includes(job.state) &&
                          button(t("Отменить", "Cancel"), async () => {
                            await api(`/exports/${job.id}/cancel`, "POST");
                            await refresh();
                          })}
                        {job.state === "failed" &&
                          button(t("Повторить", "Retry"), async () => {
                            await api(`/exports/${job.id}/retry`, "POST");
                            await refresh();
                          })}
                      </div>
                    </div>
                  ))}
                </section>
                <section className="panel">
                  <h2>{t("Массовые операции", "Bulk actions")}</h2>
                  <p>
                    {t(
                      "Предпросмотр фиксирует выбранные объекты на сервере. Изменения начнутся после запуска.",
                      "Preview freezes the selection on the server. Changes begin after you start the job.",
                    )}
                  </p>
                  {bulk.map((job) => (
                    <div className="list-row" key={job.id}>
                      <div>
                        <strong>
                          {job.action === "revoke"
                            ? t("Отзыв передач", "Revoke transfers")
                            : t(
                                "Удаление файлов черновика",
                                "Delete draft files",
                              )}
                        </strong>
                        <p>
                          {stateName(job.state)} · {job.done}/{job.total} ·{" "}
                          {t("конфликтов:", "conflicts:")} {job.conflicts}
                        </p>
                      </div>
                      {job.state === "paused" &&
                        button(
                          t(
                            "Запустить для выбранных объектов",
                            "Run for selected items",
                          ),
                          async () => {
                            await api(`/bulk/${job.id}/run`, "POST");
                            await refresh();
                          },
                          false,
                        )}
                      {["paused", "queued", "running"].includes(job.state) &&
                        button(t("Отменить", "Cancel"), async () => {
                          await api(`/bulk/${job.id}/cancel`, "POST");
                          await refresh();
                        })}
                    </div>
                  ))}
                </section>
              </>
            )}
            {tab === "notifications" && (
              <>
                <h1 className="page-title">
                  {t("Уведомления", "Notifications")}
                </h1>
                <section className="panel">
                  <h2>{t("События", "Activity")}</h2>
                  {notifications.map((n) => (
                    <div className="list-row" key={n.id}>
                      <div>
                        <strong>
                          {(
                            {
                              "transfer.published": t(
                                "Передача опубликована",
                                "Transfer published",
                              ),
                              "download.completed": t(
                                "Скачивание завершено",
                                "Download completed",
                              ),
                              "transfer.expired": t(
                                "Срок передачи истёк",
                                "Transfer expired",
                              ),
                              "transfer.expiring": t(
                                "Срок скоро истечёт",
                                "Transfer expires soon",
                              ),
                              "transfer.revoked": t(
                                "Передача отозвана",
                                "Transfer revoked",
                              ),
                            } as Record<string, string>
                          )[n.kind] ||
                            t("Изменение передачи", "Transfer updated")}
                        </strong>
                        <p>{date(n.createdAt)}</p>
                      </div>
                      {!n.readAt &&
                        button(t("Прочитано", "Mark read"), async () => {
                          await api(`/notifications/${n.id}/read`, "POST");
                          await refresh();
                        })}
                      {n.transferId &&
                        button(
                          t("Открыть передачу", "Open transfer"),
                          async () => {
                            setTab("transfers");
                            if (n.transferId) await openTransfer(n.transferId);
                          },
                        )}
                    </div>
                  ))}
                </section>
                <section className="panel">
                  <h2>
                    {t(
                      "Доставка писем и webhook",
                      "Email and webhook delivery",
                    )}
                  </h2>
                  {deliveries.map((d) => (
                    <div className="list-row" key={d.id}>
                      <div>
                        <strong>
                          {d.kind === "smtp" ? "Email" : "Webhook"}
                        </strong>
                        <p>
                          {stateName(d.state)} · {t("попыток:", "attempts:")}{" "}
                          {d.attempts}
                        </p>
                        {d.lastCode === "smtp_unconfigured" && (
                          <small>
                            {t(
                              "Администратору нужно настроить SMTP.",
                              "An administrator needs to configure SMTP.",
                            )}
                          </small>
                        )}
                      </div>
                      {["dead", "delivered"].includes(d.state) &&
                        button(t("Отправить повторно", "Resend"), async () => {
                          await api(`/deliveries/${d.id}/retry`, "POST");
                          await refresh();
                        })}
                    </div>
                  ))}
                </section>
              </>
            )}
            {tab === "settings" && (
              <>
                <h1 className="page-title">{t("Настройки", "Settings")}</h1>
                <section className="panel">
                  <h2>
                    {t(
                      "Уведомления и форматирование",
                      "Notifications and formatting",
                    )}
                  </h2>
                  <form
                    onSubmit={(e) => {
                      e.preventDefault();
                      void act(async () => {
                        await api<Schema<"NotificationPreferences">>(
                          "/notification-preferences",
                          "PUT",
                          {
                            ...preferences,
                            locale,
                          },
                        );
                        setNotice(t("Настройки сохранены", "Settings saved"));
                      });
                    }}
                  >
                    <div className="form-grid">
                      <Field title={t("Часовой пояс IANA", "IANA time zone")}>
                        <input
                          value={preferences.timezone}
                          onChange={(e) =>
                            setPreferences({
                              ...preferences,
                              timezone: e.target.value,
                            })
                          }
                          placeholder="Europe/Moscow"
                        />
                      </Field>
                      <Field title={t("Письма о событиях", "Activity emails")}>
                        <select
                          value={preferences.digest}
                          onChange={(e) =>
                            setPreferences({
                              ...preferences,
                              digest: e.target
                                .value as Schema<"NotificationPreferences">["digest"],
                            })
                          }
                        >
                          {[
                            ["immediate", t("Сразу", "Immediately")],
                            ["hourly", t("Раз в час", "Hourly")],
                            ["daily", t("Раз в день", "Daily")],
                            ["weekly", t("Раз в неделю", "Weekly")],
                            ["off", t("Выключены", "Off")],
                          ].map(([v, l]) => (
                            <option key={v} value={v}>
                              {l}
                            </option>
                          ))}
                        </select>
                      </Field>
                      <Field
                        title={t(
                          "Тихие часы: с (0–23)",
                          "Quiet hours: from (0–23)",
                        )}
                      >
                        <input
                          type="number"
                          min={0}
                          max={23}
                          value={preferences.quietStart ?? ""}
                          onChange={(e) =>
                            setPreferences({
                              ...preferences,
                              quietStart:
                                e.target.value === ""
                                  ? null
                                  : Number(e.target.value),
                            })
                          }
                        />
                      </Field>
                      <Field
                        title={t(
                          "Тихие часы: до (0–23)",
                          "Quiet hours: until (0–23)",
                        )}
                      >
                        <input
                          type="number"
                          min={0}
                          max={23}
                          value={preferences.quietEnd ?? ""}
                          onChange={(e) =>
                            setPreferences({
                              ...preferences,
                              quietEnd:
                                e.target.value === ""
                                  ? null
                                  : Number(e.target.value),
                            })
                          }
                        />
                      </Field>
                    </div>
                    <p>
                      {t(
                        "Тихие часы не задерживают письма со ссылками получателям.",
                        "Quiet hours do not delay recipient invitation emails.",
                      )}
                    </p>
                    <label className="check">
                      <input
                        type="checkbox"
                        checked={preferences.browserEnabled}
                        onChange={(e) => {
                          const checked = e.target.checked;
                          if (checked && "Notification" in window)
                            void Notification.requestPermission().then(
                              (permission) =>
                                setPreferences({
                                  ...preferences,
                                  browserEnabled: permission === "granted",
                                }),
                            );
                          else
                            setPreferences({
                              ...preferences,
                              browserEnabled: false,
                            });
                        }}
                      />
                      {t("Уведомления браузера", "Browser notifications")}
                    </label>
                    <button disabled={busy}>{t("Сохранить", "Save")}</button>
                  </form>
                </section>
                <AccountSettings t={t} act={act} busy={busy} />
                <section className="panel">
                  <h2>Webhook</h2>
                  <form
                    onSubmit={(e) => {
                      e.preventDefault();
                      const f = new FormData(e.currentTarget);
                      void act(async () => {
                        const v = await api<{ id: string; secret: string }>(
                          "/webhooks",
                          "POST",
                          {
                            url: f.get("url"),
                            secret: "",
                          },
                        );
                        setNotice(
                          t(
                            "Секрет показывается один раз. Сохраните: ",
                            "Secret is shown once. Save it: ",
                          ) +
                            v.secret +
                            " · ID: " +
                            v.id,
                        );
                      });
                    }}
                  >
                    <Field title="HTTPS URL">
                      <input
                        name="url"
                        type="url"
                        required
                        placeholder="https://example.com/events"
                      />
                    </Field>
                    <button disabled={busy}>
                      {t("Создать webhook", "Create webhook")}
                    </button>
                  </form>
                  <form
                    className="toolbar"
                    onSubmit={(e) => {
                      e.preventDefault();
                      const id = new FormData(e.currentTarget).get("id");
                      void act(async () => {
                        await api(`/webhooks/${id}`, "DELETE");
                        setNotice(t("Webhook выключен", "Webhook disabled"));
                      });
                    }}
                  >
                    <Field title="Webhook ID">
                      <input name="id" required />
                    </Field>
                    <button className="secondary" disabled={busy}>
                      {t("Выключить", "Disable")}
                    </button>
                  </form>
                </section>
              </>
            )}
            {tab === "admin" && user.role === "admin" && (
              <Admin t={t} act={act} busy={busy} />
            )}
          </>
        )}
        {mode === "receive" && (
          <div className="recipient">
            <p className="eyebrow">
              {t("Приватная передача", "Private transfer")}
            </p>
            <h1 className="page-title">
              {t("Файлы для вас", "Files for you")}
            </h1>
            <p className="lead">
              {t(
                "Файл удаляется после полного скачивания. Просмотр списка не расходует доступ.",
                "A file is deleted after a complete download. Viewing this list does not consume access.",
              )}
            </p>
            {!recipientReady ? (
              <form
                className="panel"
                onSubmit={(e) => {
                  e.preventDefault();
                  const password = new FormData(e.currentTarget).get(
                    "password",
                  );
                  void act(async () => {
                    let value = token;
                    if (token.includes("#"))
                      value =
                        new URLSearchParams(token.split("#")[1]).get("token") ||
                        "";
                    await api("/recipient/session", "POST", {
                      token: value,
                      password,
                    });
                    setToken("");
                    setRecipientFiles(await api("/recipient/files"));
                    setRecipientReady(true);
                  });
                }}
              >
                <Field
                  title={t("Ключ или ссылка доступа", "Access key or link")}
                >
                  <input
                    type="password"
                    value={token}
                    onChange={(e) => setToken(e.target.value)}
                    required
                    autoComplete="off"
                  />
                </Field>
                <Field
                  title={t(
                    "PIN или пароль (если задан)",
                    "PIN or password (if required)",
                  )}
                >
                  <input name="password" type="password" autoComplete="off" />
                </Field>
                <button disabled={busy}>
                  {t("Открыть файлы", "Open files")}
                </button>
              </form>
            ) : (
              <>
                <div className="toolbar">
                  {button(
                    t("Скачать всё ZIP", "Download all ZIP"),
                    async () => {
                      const d = await api<{ url: string }>(
                        "/recipient/downloads",
                        "POST",
                        {
                          all: true,
                        },
                      );
                      setDownloadURL(d.url);
                      saveDownload(d.url);
                    },
                    false,
                  )}
                  {button(t("Обновить список", "Refresh files"), async () =>
                    setRecipientFiles(await api("/recipient/files")),
                  )}
                </div>
                <div className="rows">
                  {recipientFiles.map((f) => (
                    <div className="list-row" key={f.ref}>
                      <div className="file-name">
                        <strong>{f.path}</strong>
                        <p>
                          {size(f.size)} · {stateName(f.state)}
                        </p>
                      </div>
                      <button
                        disabled={busy || f.state !== "ready"}
                        onClick={() =>
                          void act(async () => {
                            const d = await api<{ url: string }>(
                              "/recipient/downloads",
                              "POST",
                              { fileRef: f.ref },
                            );
                            setDownloadURL(d.url);
                            saveDownload(d.url);
                          })
                        }
                      >
                        {t("Скачать", "Download")}
                      </button>
                    </div>
                  ))}
                </div>
                {downloadURL && (
                  <p>
                    <a href={downloadURL} download>
                      {t(
                        "Продолжить начатое скачивание",
                        "Resume the current download",
                      )}
                    </a>
                  </p>
                )}
                <p className="muted">
                  {t(
                    "При обрыве используйте продолжение загрузки в браузере в течение 5 минут. Завершение означает передачу всех байтов сервером; хранение на вашем диске проверяйте самостоятельно.",
                    "If interrupted, use your browser’s resume feature within 5 minutes. Completion means the server sent all bytes; check the saved file on your device.",
                  )}
                </p>
              </>
            )}
          </div>
        )}
        {mode === "help" && (
          <article className="help">
            <p className="eyebrow">{branding.productName}</p>
            <h1 className="page-title">{t("Помощь", "Help")}</h1>
            <h2>{t("Как передать файлы", "Sending files")}</h2>
            <ol>
              <li>
                {t(
                  "Войдите и создайте новую передачу.",
                  "Sign in and create a new transfer.",
                )}
              </li>
              <li>
                {t(
                  "Добавьте файлы или папку. При обрыве выберите те же файлы для продолжения.",
                  "Add files or a folder. Select the same files to resume after interruption.",
                )}
              </li>
              <li>
                {t(
                  "Проверьте состав, пути, получателей и срок. Опубликуйте передачу.",
                  "Review files, paths, recipients and expiry. Publish the transfer.",
                )}
              </li>
              <li>
                {t(
                  "Передайте ссылки получателям. PIN рекомендуется сообщить отдельно.",
                  "Share the links with recipients. Send the PIN separately.",
                )}
              </li>
            </ol>
            <h2>{t("Одноразовое скачивание", "One-time downloads")}</h2>
            <p>
              {t(
                "Просмотр ссылки не скачивает файлы. Получатель нажимает «Скачать» или «Скачать всё ZIP». После полного скачивания файл закрывается для всех получателей и удаляется. ZIP завершается атомарно: ошибка не расходует файлы.",
                "Opening a link does not download files. The recipient chooses Download or Download all ZIP. A completed file becomes unavailable to every recipient and is deleted. ZIP completion is atomic: failure does not consume files.",
              )}
            </p>
            <h2>{t("Проверка и отчёты", "Verification and reports")}</h2>
            <p>
              {t(
                "ZIP содержит SHA256SUMS.txt, Ed25519 подпись и публичный ключ. Для проверки используйте filex-verify и доверенный ключ, полученный отдельно. Публичная проверка отчёта показывает только контрольную сумму, ревизию и состояние. Для сверки загрузите JSON отчёта.",
                "ZIP includes SHA256SUMS.txt, an Ed25519 signature and a public key. Use filex-verify with a trusted key obtained separately. Public report verification exposes only the hash, revision and state. Upload the report JSON to compare its hash.",
              )}
            </p>
            <h2>{t("Конфиденциальность", "Privacy")}</h2>
            <p>
              {t(
                "Журнал владельца показывает время, количество и объём, без связи с конкретным получателем. Административные операции фиксируются в защищённом журнале.",
                "The owner’s journal shows time, count and volume without recipient correlation. Administrative actions are audited.",
              )}
            </p>
            <h2>{t("Если что-то пошло не так", "Troubleshooting")}</h2>
            <p>
              {t(
                "Не закрывайте страницу до завершения проверки загруженных файлов. Ошибку доставки письма можно повторить в «Уведомлениях». Экспорт и массовые операции видны в «Задачах». Если доступ закончился, обратитесь к отправителю за новой передачей.",
                "Wait for uploaded files to finish validation. Retry email failures in Notifications. Exports and bulk actions appear in Jobs. If access has expired, ask the sender for a new transfer.",
              )}
            </p>
          </article>
        )}
        {mode === "status" && (
          <section className="status-card">
            <p className="eyebrow">{branding.productName}</p>
            <h1 className="page-title">
              {t("Состояние сервиса", "Service status")}
            </h1>
            <div role="status">
              <span
                className={`status-dot ${status === "operational" ? "ok" : ""}`}
              />
              {status === "operational"
                ? t("Сервис работает", "Operational")
                : status === "checking"
                  ? t("Проверяем…", "Checking…")
                  : t("Сервис временно недоступен", "Temporarily unavailable")}
            </div>
            <p>
              {t(
                "Текущий общий статус. Обновляется каждые 15 секунд.",
                "Current aggregate status. Updated every 15 seconds.",
              )}
            </p>
          </section>
        )}
        {mode === "verify" && (
          <section className="panel verify">
            <h1 className="page-title">
              {t("Проверить отчёт", "Verify a report")}
            </h1>
            <form
              onSubmit={(e) => {
                e.preventDefault();
                void act(async () =>
                  setVerification(
                    await api(`/verify/reports/${verificationId}`),
                  ),
                );
              }}
            >
              <Field title={t("Идентификатор отчёта", "Report identifier")}>
                <input
                  value={verificationId}
                  disabled={!loaded}
                  onChange={(e) => setVerificationId(e.target.value)}
                  required
                />
              </Field>
              <button disabled={!loaded || busy || !verificationId.trim()}>
                {t("Проверить", "Verify")}
              </button>
            </form>
            {verification && (
              <>
                <p>
                  {t("Ревизия", "Revision")} {verification.revision} ·{" "}
                  {stateName(verification.state)}
                </p>
                <p className="break">SHA-256: {verification.canonicalHash}</p>
                <Field
                  title={t(
                    "JSON отчёта для сравнения",
                    "Report JSON to compare",
                  )}
                >
                  <input
                    type="file"
                    accept="application/json,.json"
                    onChange={(e) => {
                      const file = e.target.files?.[0];
                      if (file)
                        void act(async () => {
                          if (file.size > 16 * 1024 * 1024)
                            throw new Error("invalid_input");
                          const bytes = await file.arrayBuffer();
                          const hash = await crypto.subtle.digest(
                            "SHA-256",
                            bytes,
                          );
                          setLocalHash(
                            Array.from(new Uint8Array(hash))
                              .map((x) => x.toString(16).padStart(2, "0"))
                              .join(""),
                          );
                        });
                    }}
                  />
                </Field>
                {localHash && (
                  <p role="status">
                    {localHash === verification.canonicalHash
                      ? t("Контрольная сумма совпадает", "Checksum matches")
                      : t(
                          "Контрольная сумма НЕ совпадает",
                          "Checksum does NOT match",
                        )}
                  </p>
                )}
              </>
            )}
          </section>
        )}
      </main>
      <footer>
        <span>{branding.productName}</span>
        <span>
          {branding.footerText || t(branding.taglineRu, branding.taglineEn)}
        </span>
      </footer>
    </>
  );
}
function AccountSettings({
  t,
  act,
  busy,
}: {
  t: (ru: string, en: string) => string;
  act: (f: () => Promise<void>) => Promise<void>;
  busy: boolean;
}) {
  const actOnMount = useEffectEvent(act);
  const [tokens, setTokens] = useState<Response<"/api/v1/auth/tokens">>([]);
  const [sessions, setSessions] = useState<Response<"/api/v1/auth/sessions">>(
    [],
  );
  const [history, setHistory] = useState<Response<"/api/v1/auth/history">>([]);
  const [token, setToken] = useState("");
  useEffect(() => {
    void actOnMount(async () => {
      setSessions(await api("/auth/sessions"));
      setHistory(await api("/auth/history"));
      setTokens(await api("/auth/tokens"));
    });
  }, []);
  return (
    <>
      <section className="panel">
        <h2>{t("Подтверждение личности", "Reauthentication")}</h2>
        <form
          className="toolbar"
          onSubmit={(e) => {
            e.preventDefault();
            const password = new FormData(e.currentTarget).get("password");
            e.currentTarget.reset();
            void act(() => api("/auth/reauth", "POST", { password }));
          }}
        >
          <Field title={t("Текущий пароль", "Current password")}>
            <input
              type="password"
              name="password"
              autoComplete="current-password"
              required
            />
          </Field>
          <button disabled={busy}>
            {t("Подтвердить на 5 минут", "Confirm for 5 minutes")}
          </button>
        </form>
      </section>
      <IdentityLogin api={api} t={t} account />
      <section className="panel">
        <h2>{t("Изменить пароль", "Change password")}</h2>
        <p>
          {t(
            "Изменение пароля доступно аккаунтам с локальным паролем. Для внешнего аккаунта используйте повторный вход через провайдер выше.",
            "Password changes require an existing local password. External accounts can reauthenticate using the provider above.",
          )}
        </p>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            const d = new FormData(e.currentTarget);
            void act(async () => {
              await api("/auth/password", "PATCH", {
                currentPassword: d.get("current"),
                newPassword: d.get("next"),
              });
              localStorage.removeItem("filex.csrf");
              location.reload();
            });
          }}
        >
          <div className="form-grid">
            <Field title={t("Текущий пароль", "Current password")}>
              <input
                name="current"
                type="password"
                required
                autoComplete="current-password"
              />
            </Field>
            <Field
              title={t(
                "Новый пароль, от 12 символов",
                "New password, at least 12 characters",
              )}
            >
              <input
                name="next"
                type="password"
                required
                minLength={12}
                autoComplete="new-password"
              />
            </Field>
          </div>
          <p>
            {t(
              "После смены пароля все сеансы и API-токены будут отозваны. Последние 5 паролей нельзя использовать повторно.",
              "Changing your password revokes all sessions and API tokens. The last 5 passwords cannot be reused.",
            )}
          </p>
          <button disabled={busy}>
            {t("Изменить пароль", "Change password")}
          </button>
        </form>
      </section>
      <section className="panel">
        <h2>{t("Сеансы", "Sessions")}</h2>
        {sessions.map((s) => (
          <div className="list-row" key={s.id}>
            <div className="break">
              {s.id}
              <br />
              <small>{s.createdAt}</small>
            </div>
            {!s.revokedAt && (
              <button
                className="secondary"
                disabled={busy}
                onClick={() =>
                  void act(async () => {
                    await api(`/auth/sessions/${s.id}`, "DELETE");
                    setSessions(await api("/auth/sessions"));
                  })
                }
              >
                {t("Завершить", "Revoke")}
              </button>
            )}
          </div>
        ))}
        <details>
          <summary>{t("История входов", "Login history")}</summary>
          {history.map((h, i) => (
            <p key={i}>
              {h.occurredAt} ·{" "}
              {h.success
                ? t("Успешно", "Successful")
                : t("Неуспешно", "Failed")}
            </p>
          ))}
        </details>
      </section>
      <section className="panel">
        <h2>{t("API-токен", "API token")}</h2>
        {tokens.map((value) => (
          <div className="list-row" key={value.id}>
            <div className="break">
              <strong>{value.scopes.join(", ")}</strong>
              <p>
                {value.id} · {value.expiresAt}
              </p>
            </div>
            {!value.revokedAt && (
              <button
                className="secondary"
                disabled={busy}
                onClick={() =>
                  void act(async () => {
                    await api(`/auth/tokens/${value.id}`, "DELETE");
                    setTokens(await api("/auth/tokens"));
                  })
                }
              >
                {t("Отозвать", "Revoke")}
              </button>
            )}
          </div>
        ))}
        <form
          onSubmit={(e) => {
            e.preventDefault();
            const f = new FormData(e.currentTarget);
            void act(async () => {
              const result = await api<
                Response<"/api/v1/auth/tokens", "post", 201>
              >("/auth/tokens", "POST", {
                scopes: f.getAll("scope"),
                ttlSeconds: Number(f.get("ttl")) * 86400,
              });
              setToken(result.token);
              setTokens(await api("/auth/tokens"));
            });
          }}
        >
          <div className="toolbar">
            {[
              "transfers:read",
              "transfers:write",
              "reports:read",
              "reports:write",
              "exports:write",
              "notifications:read",
            ].map((scope) => (
              <label className="check" key={scope}>
                <input name="scope" type="checkbox" value={scope} />
                {scope}
              </label>
            ))}
          </div>
          <Field title={t("Срок, дней (1–30)", "Lifetime, days (1–30)")}>
            <input name="ttl" type="number" min={1} max={30} defaultValue={7} />
          </Field>
          <button disabled={busy}>{t("Создать токен", "Create token")}</button>
          {token && (
            <Field
              title={t(
                "Сохраните токен: он показывается один раз",
                "Save this token: it is shown once",
              )}
            >
              <input readOnly value={token} />
            </Field>
          )}
        </form>
      </section>
    </>
  );
}
function Admin({
  t,
  act,
  busy,
}: {
  t: (ru: string, en: string) => string;
  act: (f: () => Promise<void>) => Promise<void>;
  busy: boolean;
}) {
  const [adminPreview, setAdminPreview] = useState(false);
  const [storage, setStorage] = useState<Schema<"StorageProvider">[]>([]);
  const [maintenance, setMaintenance] = useState<
    Response<"/api/v1/admin/maintenance">
  >([]);
  const [settings, setSettings] = useState<Response<"/api/v1/admin/settings">>(
    [],
  );
  const [kind, setKind] = useState("local");
  const [settingsJSON, setSettingsJSON] = useState("");
  const [settingsID, setSettingsID] = useState(0);
  async function reload() {
    setStorage(await api("/admin/storage"));
    setMaintenance(await api("/admin/maintenance"));
    setSettings(await api("/admin/settings"));
  }
  const initialLoad = useEffectEvent(() => act(reload));
  useEffect(() => {
    void initialLoad();
  }, []);
  return (
    <>
      <h1 className="page-title">{t("Администрирование", "Administration")}</h1>
      <p>
        {t(
          "Действия требуют подтверждения пароля за последние 5 минут. Подтвердить пароль можно в настройках.",
          "Actions require password confirmation within the last 5 minutes. Reauthenticate in Settings.",
        )}
      </p>
      <button
        className="secondary"
        disabled={busy}
        onClick={() => void act(reload)}
      >
        {t("Обновить", "Refresh")}
      </button>
      <BrandingPanel api={api} t={t} busy={busy} />
      <IdentityProvidersPanel api={api} t={t} busy={busy} />
      <OperationsPanel
        api={api}
        t={t}
        act={act}
        busy={busy}
        providers={storage}
      />
      <AdminExportsPanel api={api} t={t} act={act} busy={busy} />
      <section className="panel">
        <h2>{t("Хранилища", "Storage providers")}</h2>
        {storage.map((s) => (
          <div className="list-row" key={s.id}>
            <div className="break">
              <strong>
                {s.kind} · {s.purpose}
              </strong>
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
              <select value={kind} onChange={(e) => setKind(e.target.value)}>
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
      </section>
      <section className="panel">
        <h2>{t("Перенос хранилища", "Storage migration")}</h2>
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
              <select name={name}>
                {storage
                  .filter((s) => s.purpose === "payload")
                  .map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.kind} · {s.id}
                    </option>
                  ))}
              </select>
            </Field>
          ))}
          <button disabled={busy}>
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
                {[j.state === "paused" ? "running" : "paused", "cancelled"].map(
                  (state) => (
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
                  ),
                )}
              </div>
            )}
          </div>
        ))}
      </section>
      <section className="panel">
        <h2>
          {t("Массовые действия администратора", "Administrator bulk actions")}
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
          <Field title={t("Причина массового действия", "Bulk action reason")}>
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
        <h2>{t("Удержание и расследование", "Retention and investigation")}</h2>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            const f = new FormData(e.currentTarget);
            void act(async () => {
              await api(`/admin/transfers/${f.get("transfer")}/hold`, "POST", {
                fileId: f.get("file") || "",
                frozen: f.get("hold") === "true",
                reason: f.get("reason"),
              });
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
      <section className="panel">
        <h2>SMTP</h2>
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
              ["port", t("Порт", "Port"), "number"],
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
            <Field title="TLS">
              <select name="tls">
                <option value="starttls">STARTTLS</option>
                <option value="smtps">SMTPS</option>
              </select>
            </Field>
          </div>
          <button disabled={busy}>{t("Сохранить SMTP", "Save SMTP")}</button>
        </form>
      </section>
      <section className="panel">
        <h2>{t("Настройки приложения", "Application settings")}</h2>
        <p>
          {t(
            "Создайте черновик, проверьте его и примените к текущей ревизии. Секреты SMTP и S3 задаются отдельно. Изменение адреса сервиса или сетевой политики закроет доступ и потребует перезапуска API/worker, сверки данных и явного открытия доступа.",
            "Create a draft, validate it and apply it to the current revision. SMTP and S3 secrets are configured separately. Changing the service origin or network policy closes access and requires an API/worker restart, reconciliation and explicit reopening.",
          )}
        </p>
        <Field title="Settings JSON">
          <textarea
            rows={12}
            value={settingsJSON}
            onChange={(e) => setSettingsJSON(e.target.value)}
          />
        </Field>
        <div className="toolbar">
          <button
            className="secondary"
            onClick={() => {
              const active = settings.find((s) => s.active);
              setSettingsJSON(JSON.stringify(active?.value || {}, null, 2));
            }}
          >
            {t("Загрузить текущие", "Load current")}
          </button>
          <button
            disabled={busy}
            onClick={() =>
              void act(async () => {
                const v = await api<{ revision: number }>(
                  "/admin/settings",
                  "POST",
                  JSON.parse(settingsJSON),
                );
                setSettingsID(v.revision);
                await reload();
              })
            }
          >
            {t("Сохранить черновик", "Save draft")}
          </button>
          {settingsID > 0 && (
            <button
              disabled={busy}
              onClick={() =>
                void act(async () => {
                  await api(`/admin/settings/${settingsID}/validate`, "POST");
                  await reload();
                })
              }
            >
              {t("Проверить черновик", "Validate draft")} #{settingsID}
            </button>
          )}
        </div>
        {settings.map((s) => (
          <div className="list-row" key={s.revision}>
            <span>
              #{s.revision} ·{" "}
              {s.active
                ? t("Активна", "Active")
                : s.validated
                  ? t("Проверена", "Validated")
                  : t("Черновик", "Draft")}
            </span>
            {s.validated && !s.active && (
              <button
                className="secondary"
                disabled={busy}
                onClick={() =>
                  void act(async () => {
                    await api(`/admin/settings/${s.revision}/apply`, "POST", {
                      expectedRevision: settings.find((v) => v.active)
                        ?.revision,
                    });
                    await reload();
                  })
                }
              >
                {t("Применить / откатить", "Apply / roll back")}
              </button>
            )}
          </div>
        ))}
      </section>
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
