"use client";
import { useEffect, useState } from "react";
type Claims = {
  schemaVersion: 1;
  product: "filex";
  issuer: "filex-vendor";
  licenseId: string;
  entitlementId: string;
  revision: number;
  customerId: string;
  installationId: string;
  kind: "subscription" | "perpetual";
  issuedAt: string;
  notBefore: string;
  expiresAt: string | null;
  updatesUntil: string;
  supportUntil: string;
  limits: { maxEnabledUsers: number | null; maxTransferBytes: number | null };
  features: string[];
};
type Session = {
  subject: string;
  role: string;
  csrfToken: string;
  authenticatedAt: string;
};
type Issuance = {
  id: string;
  state: string;
  claims: Claims;
  sha256: string;
  createdAt: string;
  keyId: string;
};
type Request = { idempotencyKey: string; claims: Claims };
async function api<T>(
  path: string,
  session: Session | null,
  body?: unknown,
): Promise<T> {
  const r = await fetch(`/issuer/api${path}`, {
    method: body === undefined ? "GET" : "POST",
    cache: "no-store",
    headers: {
      "Content-Type": "application/json",
      "X-CSRF-Token": session?.csrfToken ?? "",
    },
    ...(body === undefined ? {} : { body: JSON.stringify(body) }),
  });
  if (!r.ok) {
    if (r.status === 401 || r.status === 403)
      throw new Error(
        "Войдите с MFA или подтвердите вход заново. Для выпуска нужен свежий вход и роль издателя.",
      );
    throw new Error(
      r.status === 409
        ? "Конфликт версии или повторного запроса. Обновите историю выдачи."
        : `Операция не выполнена (${r.status}).`,
    );
  }
  return r.json() as Promise<T>;
}
export default function Page() {
  const [session, setSession] = useState<Session | null>(null);
  const [history, setHistory] = useState<Issuance[]>([]);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState("");
  const [customer, setCustomer] = useState("");
  const [installation, setInstallation] = useState("");
  const [entitlement, setEntitlement] = useState("");
  const [revision, setRevision] = useState(1);
  const [kind, setKind] = useState<"subscription" | "perpetual">(
    "subscription",
  );
  const [users, setUsers] = useState("50");
  const [bytes, setBytes] = useState("10737418240");
  const [end, setEnd] = useState("");
  const [start, setStart] = useState("");
  const [updates, setUpdates] = useState("");
  const [support, setSupport] = useState("");
  const [features, setFeatures] = useState<string[]>([]);
  const [pending, setPending] = useState<Request | null>(null);
  useEffect(() => {
    let live = true;
    api<Session>("/session", null)
      .then((s) => {
        if (live) setSession(s);
        return api<Issuance[]>("/issuances", s);
      })
      .then((v) => {
        if (live) setHistory(v);
      })
      .catch((e) => {
        if (live) setError(String(e.message));
      });
    return () => {
      live = false;
    };
  }, []);
  async function run(fn: () => Promise<void>) {
    setBusy(true);
    setError("");
    try {
      await fn();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }
  function number(v: string) {
    if (v === "") return null;
    const n = Number(v);
    if (!Number.isSafeInteger(n) || n < 1)
      throw new Error(
        "Лимит должен быть положительным целым числом; пустое поле означает отсутствие лимита.",
      );
    return n;
  }
  function date(v: string) {
    if (!v) throw new Error("Заполните даты срока и обновлений.");
    return new Date(v + "T23:59:59Z").toISOString();
  }
  function prepare() {
    const now = new Date().toISOString();
    const notBefore = start ? new Date(start + "Z").toISOString() : now;
    const expiresAt = kind === "subscription" ? date(end) : null;
    if (expiresAt && expiresAt <= notBefore)
      throw new Error("Окончание подписки должно быть позже начала.");
    setPending({
      idempotencyKey: crypto.randomUUID(),
      claims: {
        schemaVersion: 1,
        product: "filex",
        issuer: "filex-vendor",
        licenseId: crypto.randomUUID(),
        entitlementId: entitlement || crypto.randomUUID(),
        revision,
        customerId: customer,
        installationId: installation,
        kind,
        issuedAt: now,
        notBefore,
        expiresAt,
        updatesUntil: date(updates),
        supportUntil: date(support),
        limits: {
          maxEnabledUsers: number(users),
          maxTransferBytes: number(bytes),
        },
        features,
      },
    });
  }
  function renew(row: Issuance) {
    setCustomer(row.claims.customerId);
    setInstallation(row.claims.installationId);
    setEntitlement(row.claims.entitlementId);
    setRevision(row.claims.revision + 1);
    setKind(row.claims.kind);
    setUsers(row.claims.limits.maxEnabledUsers?.toString() ?? "");
    setBytes(row.claims.limits.maxTransferBytes?.toString() ?? "");
    setFeatures(row.claims.features);
    setStart("");
    setEnd(row.claims.expiresAt?.slice(0, 10) ?? "");
    setUpdates(row.claims.updatesUntil.slice(0, 10));
    setSupport(row.claims.supportUntil.slice(0, 10));
    setPending(null);
    window.scrollTo({ top: 0, behavior: "smooth" });
  }
  return (
    <main>
      <header>
        <div>
          <small>FILEX · КОНТУР ИЗДАТЕЛЯ</small>
          <h1>Лицензии продукта</h1>
          <p className="muted">Выдача прав для автономных установок</p>
        </div>
        <div>
          <a className="button" href="/issuer/auth/login">
            {session ? "Подтвердить вход с MFA" : "Войти с MFA"}
          </a>
          {session && (
            <p>
              {session.subject} · {session.role}
            </p>
          )}
        </div>
      </header>
      <a href="/issuer/setup">Первичная настройка</a>
      {error && <p role="alert">{error}</p>}
      {notice && <p role="status">{notice}</p>}
      {session && (
        <>
          <section>
            <h2>Новая лицензия или продление</h2>
            <p className="muted">
              Подпись создаётся изолированным сервисом. Перед выдачей проверьте
              клиента, установку и точные ограничения.
            </p>
            <form
              onChange={() => setPending(null)}
              onSubmit={(e) => {
                e.preventDefault();
                void run(async () => prepare());
              }}
            >
              <div className="grid">
                <label>
                  Клиент / номер договора
                  <input
                    required
                    value={customer}
                    pattern="[A-Za-z0-9][A-Za-z0-9._-]{0,127}"
                    onChange={(e) => setCustomer(e.target.value)}
                  />
                </label>
                <label>
                  Код установки
                  <input
                    required
                    value={installation}
                    onChange={(e) => setInstallation(e.target.value)}
                  />
                </label>
                <label>
                  Импорт запроса установки
                  <input
                    type="file"
                    accept=".json"
                    onChange={(e) => {
                      const f = e.target.files?.[0];
                      if (!f) return;
                      void run(async () => {
                        if (f.size > 4096)
                          throw new Error("Запрос слишком большой");
                        const v = JSON.parse(await f.text());
                        if (
                          v.schemaVersion !== 1 ||
                          v.product !== "filex" ||
                          typeof v.installationId !== "string"
                        )
                          throw new Error("Неверный запрос установки");
                        setInstallation(v.installationId);
                        setPending(null);
                      });
                    }}
                  />
                </label>
                <label>
                  Модель
                  <select
                    value={kind}
                    onChange={(e) => setKind(e.target.value as typeof kind)}
                  >
                    <option value="subscription">Подписка</option>
                    <option value="perpetual">Бессрочная</option>
                  </select>
                </label>
                <label>
                  Цепочка лицензии (пусто — новая)
                  <input
                    value={entitlement}
                    onChange={(e) => setEntitlement(e.target.value)}
                  />
                </label>
                <label>
                  Версия выдачи
                  <input
                    type="number"
                    min={1}
                    step={1}
                    required
                    value={revision}
                    onChange={(e) => setRevision(Number(e.target.value))}
                  />
                </label>
                <label>
                  Включённые пользователи
                  <input
                    type="number"
                    min={1}
                    step={1}
                    value={users}
                    onChange={(e) => setUsers(e.target.value)}
                  />
                </label>
                <label>
                  Размер одной передачи, байты
                  <input
                    type="number"
                    min={1}
                    step={1}
                    value={bytes}
                    onChange={(e) => setBytes(e.target.value)}
                  />
                </label>
                <label>
                  Начало действия (UTC, пусто — сразу)
                  <input
                    type="datetime-local"
                    value={start}
                    onChange={(e) => setStart(e.target.value)}
                  />
                </label>
                {kind === "subscription" && (
                  <label>
                    Подписка до, включительно (UTC)
                    <input
                      required
                      type="date"
                      value={end}
                      onChange={(e) => setEnd(e.target.value)}
                    />
                  </label>
                )}
                <label>
                  Обновления до (UTC)
                  <input
                    required
                    type="date"
                    value={updates}
                    onChange={(e) => setUpdates(e.target.value)}
                  />
                </label>
                <label>
                  Поддержка до (UTC)
                  <input
                    required
                    type="date"
                    value={support}
                    onChange={(e) => setSupport(e.target.value)}
                  />
                </label>
              </div>
              <fieldset>
                <legend>Дополнительные функции</legend>
                {[
                  ["webdav", "WebDAV"],
                  ["federated_auth", "Корпоративный вход"],
                  ["white_label", "Собственный брендинг"],
                ].map(([id, label]) => (
                  <label key={id}>
                    <input
                      type="checkbox"
                      checked={features.includes(id)}
                      onChange={(e) =>
                        setFeatures(
                          e.target.checked
                            ? [...features, id]
                            : features.filter((v) => v !== id),
                        )
                      }
                    />
                    {label}
                  </label>
                ))}
              </fieldset>
              <p className="muted">
                Пустой лимит означает отсутствие ограничения. Получатели без
                учётной записи не занимают пользовательские места.
              </p>
              <button disabled={busy || session.role === "viewer"}>
                Подготовить к выпуску
              </button>
            </form>
            {pending && (
              <>
                <h3>Проверка перед подписью</h3>
                <pre>{JSON.stringify(pending.claims, null, 2)}</pre>
                <p>
                  После подтверждения эти права и версия выдачи неизменяемы.
                  Исправления оформляются новой лицензией.
                </p>
                <button
                  disabled={busy}
                  onClick={() =>
                    void run(async () => {
                      await api<Issuance>("/issuances", session, pending);
                      setNotice(
                        "Выдача принята. Обновите историю после обработки signer, затем скачайте файл.",
                      );
                      setHistory(await api<Issuance[]>("/issuances", session));
                      setPending(null);
                    })
                  }
                >
                  Подтвердить и передать на подпись
                </button>
              </>
            )}
          </section>
          <section>
            <header>
              <h2>История выдачи</h2>
              <button
                disabled={busy}
                onClick={() =>
                  void run(async () =>
                    setHistory(await api<Issuance[]>("/issuances", session)),
                  )
                }
              >
                Обновить
              </button>
            </header>
            <p className="muted">
              Показаны последние 200 выдач. Более ранние записи сохраняются в
              реестре. Выдача файла не подтверждает его установку заказчиком.
            </p>
            <div className="scroll">
              <table>
                <thead>
                  <tr>
                    <th>Клиент / установка</th>
                    <th>Модель / версия</th>
                    <th>Состояние</th>
                    <th>Действия</th>
                  </tr>
                </thead>
                <tbody>
                  {history.map((row) => (
                    <tr key={row.id}>
                      <td>
                        {row.claims.customerId}
                        <br />
                        <small>{row.claims.installationId}</small>
                      </td>
                      <td>
                        {row.claims.kind} · {row.claims.revision}
                      </td>
                      <td>
                        {row.state}
                        <br />
                        <small>{row.keyId}</small>
                      </td>
                      <td>
                        {row.state === "issued" && (
                          <a href={`/issuer/api/issuances/${row.id}`}>
                            Скачать .jws
                          </a>
                        )}{" "}
                        <button
                          disabled={busy || session.role === "viewer"}
                          onClick={() => renew(row)}
                        >
                          Продлить
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
          <button
            disabled={busy}
            onClick={() =>
              void run(async () => {
                await api("/logout", session, {});
                setSession(null);
                setHistory([]);
                setPending(null);
              })
            }
          >
            Выйти
          </button>
        </>
      )}
    </main>
  );
}
