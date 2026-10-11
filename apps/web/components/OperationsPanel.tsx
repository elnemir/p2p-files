"use client";

import { useEffect, useEffectEvent, useState, type ReactNode } from "react";

import type { Api, Schema, Response } from "../lib/api-types";
type Props = {
  api: Api;
  t: (ru: string, en: string) => string;
  act: (fn: () => Promise<void>) => Promise<void>;
  busy: boolean;
  providers: Schema<"StorageProvider">[];
};
function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="field">
      <span>{label}</span>
      {children}
    </label>
  );
}
function download(name: string, data: unknown) {
  const url = URL.createObjectURL(
    new Blob([JSON.stringify(data, null, 2)], { type: "application/json" }),
  );
  const link = document.createElement("a");
  link.href = url;
  link.download = name;
  link.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

export default function OperationsPanel({
  api,
  t,
  act,
  busy,
  providers,
}: Props) {
  const [data, setData] = useState<Response<"/api/v1/admin/operations"> | null>(
    null,
  );
  const [reason, setReason] = useState("");
  const [provider, setProvider] = useState("");
  const [source, setSource] = useState("");
  const [review, setReview] = useState(false);
  const [report, setReport] = useState<Schema<"Reconciliation"> | null>(null);
  const [acknowledgedReport, setAcknowledgedReport] = useState("");
  const ack = !!report && acknowledgedReport === report.id;
  const [error, setError] = useState("");
  const [checkpointKind, setCheckpointKind] = useState("regular");
  const [evidence, setEvidence] = useState("");
  const [lastAnchor, setLastAnchor] =
    useState<Schema<"SignedCheckpoint"> | null>(null);
  async function load() {
    const next =
      await api<Response<"/api/v1/admin/operations">>("/admin/operations");
    setData(next);
    setError("");
    if (next.state.reconciliationId)
      setReport(
        await api(
          `/admin/operations/reconciliations/${next.state.reconciliationId}`,
        ),
      );
    else setReport(null);
  }
  const pollLoad = useEffectEvent(load);
  useEffect(() => {
    let active = true;
    const poll = () => {
      void pollLoad().catch((e: Error) => {
        if (active) setError(e.message);
      });
    };
    poll();
    const timer = setInterval(poll, 4000);
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, []);
  const backups = (data?.jobs || []).filter(
    (job) => job.kind === "backup" && job.state === "completed",
  );
  const destinations = providers.filter(
    (p) => p.purpose === "backups" && p.enabled && !p.draining,
  );
  const destination = provider || destinations[0]?.id || "";
  const chosen = backups.find((b) => b.id === source);
  const label = (state: string) =>
    ({
      open: t("Доступ открыт", "Access open"),
      maintenance: t("Обслуживание", "Maintenance"),
      restoring: t("Восстановление", "Restoring"),
      reconcile: t(
        "Ожидает проверки и открытия",
        "Review and reopening required",
      ),
      blocked: t("Доступ заблокирован", "Access blocked"),
      queued: t("В очереди", "Queued"),
      running: t("Выполняется", "Running"),
      completed: t("Завершено", "Completed"),
      failed: t("Ошибка", "Failed"),
      acknowledged: t("Принят в работу", "Acknowledged"),
      resolved: t("Разрешён", "Resolved"),
    })[state] || state;
  const kind = (value: string) =>
    ({
      backup: t("Резервная копия", "Backup"),
      restore: t("Восстановление", "Restore"),
      reconcile: t("Сверка данных", "Reconciliation"),
      audit: t("Проверка аудита", "Audit check"),
      storage: t("Проверка файлов", "File check"),
    })[value] || value;
  const validReason = reason.trim().length >= 5 && reason.length <= 2000;
  const pending = (data?.jobs || []).some(
    (j) => j.state === "queued" || j.state === "running",
  );
  async function queue(value: string, extras: Record<string, string> = {}) {
    await api("/admin/operations/jobs", "POST", {
      kind: value,
      reason,
      ...extras,
    });
    await load();
  }

  return (
    <section
      className="panel"
      aria-label={t("Эксплуатация и восстановление", "Operations and recovery")}
    >
      <h2>{t("Эксплуатация и восстановление", "Operations and recovery")}</h2>
      {error && (
        <div className="error" role="alert">
          {error === "restart_required"
            ? t(
                "Восстановление или смена ключа завершены. Перезапустите API и worker на сервере, затем войдите снова. Доступ остаётся закрытым до проверки.",
                "Restore or key rewrap completed. Restart the API and worker on the host, then sign in again. Access remains closed until reviewed.",
              )
            : error === "unauthenticated"
              ? t(
                  "Сеанс завершён. Войдите снова для проверки восстановления.",
                  "Your session ended. Sign in again to review recovery.",
                )
              : t(
                  "Не удалось обновить эксплуатационные данные. Подтвердите пароль в настройках и повторите.",
                  "Cannot refresh operations. Reauthenticate in Settings and retry.",
                )}
          <a href="/">{t("Обновить страницу / войти", "Reload / sign in")}</a>
        </div>
      )}
      {data && (
        <p role="status">
          <strong>{label(data.state.mode)}</strong> ·{" "}
          {new Date(data.state.changedAt).toLocaleString()}
        </p>
      )}
      <p>
        {t(
          "Копия содержит данные PostgreSQL и зашифрованные настройки. Файлы пользователей и временные артефакты хранятся отдельно и в неё не входят.",
          "Backups contain PostgreSQL data and encrypted settings. User files and temporary artifacts remain in their separate storage.",
        )}
      </p>
      <Field
        label={t(
          "Причина эксплуатационного действия",
          "Operational action reason",
        )}
      >
        <input
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          minLength={5}
          maxLength={2000}
          placeholder={t(
            "Что проверяем или восстанавливаем",
            "What is being checked or recovered",
          )}
        />
      </Field>
      <div className="form-grid">
        <Field label={t("Хранилище резервных копий", "Backup destination")}>
          <select
            value={destination}
            onChange={(e) => setProvider(e.target.value)}
          >
            {!destinations.length && (
              <option value="">
                {t(
                  "Сначала добавьте BackupProvider ниже",
                  "Add a backup provider below first",
                )}
              </option>
            )}
            {destinations.map((p) => (
              <option key={p.id} value={p.id}>
                {p.kind} · {p.id}
              </option>
            ))}
          </select>
        </Field>
        <Field label={t("Копия для восстановления", "Source backup")}>
          <select
            value={source}
            onChange={(e) => {
              setSource(e.target.value);
              setReview(false);
            }}
          >
            <option value="">
              {t("Выберите проверенную копию", "Choose a verified backup")}
            </option>
            {backups.map((b) => (
              <option key={b.id} value={b.id}>
                {new Date(b.createdAt).toLocaleString()} · {b.id}
              </option>
            ))}
          </select>
        </Field>
      </div>
      <div className="toolbar">
        <button
          disabled={busy || !validReason || !destination}
          onClick={() =>
            void act(() => queue("backup", { providerId: destination }))
          }
        >
          {t("Создать резервную копию", "Create backup")}
        </button>
        <button
          className="secondary"
          disabled={busy || !chosen || !destination || !validReason || pending}
          onClick={() => setReview(true)}
        >
          {t("Подготовить восстановление", "Review restore")}
        </button>
        <button
          className="secondary"
          disabled={busy || !validReason || data?.state.mode !== "open"}
          onClick={() =>
            void act(async () => {
              await api("/admin/operations/maintenance", "POST", { reason });
              await load();
            })
          }
        >
          {t("Закрыть доступ для обслуживания", "Close access for maintenance")}
        </button>
      </div>
      {review && chosen && (
        <div
          className="notice compact"
          role="region"
          aria-label={t("Проверка восстановления", "Restore review")}
        >
          <h3>
            {t("Проверьте выбранную копию", "Review the selected backup")}
          </h3>
          <p className="break">
            {chosen.id} · {new Date(chosen.createdAt).toLocaleString()}
            <br />
            SHA-256:{" "}
            {typeof chosen.result.sha256 === "string"
              ? chosen.result.sha256
              : "—"}
          </p>
          <p>
            {t(
              "Прикладные данные БД будут заменены. Сначала сервис остановит активные операции и создаст проверенную предварительную копию. Старые сеансы будут отозваны; после перезапуска сервиса потребуются повторный вход, сверка и явное открытие доступа.",
              "Application database contents will be replaced. The service drains active operations and verifies a pre-restore backup first. Existing sessions are revoked; after restarting services, sign in, reconcile and explicitly reopen access.",
            )}
          </p>
          <p>
            {t(
              "Ссылки и API-токены могут вернуться к состоянию из копии. Отсутствующие файлы будут закрыты для выдачи; объекты без записей в БД автоматически не публикуются и не удаляются.",
              "Links and API tokens can return to their backed-up state. Missing files are quarantined; objects without database records are neither published nor deleted automatically.",
            )}
          </p>
          <button
            disabled={busy || !validReason || pending}
            onClick={() =>
              void act(async () => {
                await queue("restore", {
                  sourceId: source,
                  providerId: destination,
                });
                setReview(false);
              })
            }
          >
            {t(
              "Создать предварительную копию и восстановить",
              "Back up current data and restore",
            )}
          </button>
          <button className="secondary" onClick={() => setReview(false)}>
            {t("Отмена", "Cancel")}
          </button>
        </div>
      )}
      <h3>{t("Проверки и открытие доступа", "Checks and reopening")}</h3>
      <div className="toolbar">
        <button
          className="secondary"
          disabled={busy || !validReason}
          onClick={() => void act(() => queue("audit"))}
        >
          {t("Проверить аудит", "Check audit")}
        </button>
        <button
          className="secondary"
          disabled={busy || !validReason}
          onClick={() => void act(() => queue("storage"))}
        >
          {t("Проверить файлы", "Check files")}
        </button>
        <button
          disabled={
            busy || !validReason || data?.state.mode === "open" || pending
          }
          onClick={() => void act(() => queue("reconcile"))}
        >
          {t("Сверить восстановленные данные", "Reconcile restored data")}
        </button>
      </div>
      {report && (
        <div className="notice compact">
          <h3>
            {report.passed
              ? t("Сверка завершена", "Reconciliation completed")
              : t("Есть блокирующие проблемы", "Blocking findings remain")}
          </h3>
          <p>
            {t("Проверено файлов", "Files checked")}: {report.files};{" "}
            {t("артефактов", "artifacts")}: {report.artifacts};{" "}
            {t("объектов", "objects")}: {report.objects}.
          </p>
          <p>
            {t("Действующих ссылок", "Active links")}: {report.links};{" "}
            {t("API-токенов", "API tokens")}: {report.tokens}.
          </p>
          {report.findings.slice(0, 100).map((f, i: number) => (
            <div className="list-row" key={i}>
              <span className="break">
                {f.kind} · {f.subject}
              </span>
              <strong>{f.severity}</strong>
            </div>
          ))}
          <button
            className="secondary"
            onClick={() => download(`reconciliation-${report.id}.json`, report)}
          >
            {t(
              "Скачать полный отчёт сверки",
              "Download full reconciliation report",
            )}
          </button>
          <p className="break">SHA-256: {report.hash}</p>
          <label className="check">
            <input
              type="checkbox"
              checked={ack}
              onChange={(e) =>
                setAcknowledgedReport(e.target.checked ? report.id : "")
              }
            />
            {t(
              "Я проверил восстановленные ссылки и токены, недоступные файлы и объекты без записей в БД.",
              "I reviewed restored links and tokens, quarantined files and orphan objects.",
            )}
          </label>
          <div className="toolbar">
            <button
              disabled={
                busy || !ack || !validReason || !report.passed || pending
              }
              onClick={() =>
                void act(async () => {
                  await api("/admin/operations/reopen", "POST", {
                    reconciliationId: report.id,
                    hash: report.hash,
                    reason,
                    acknowledgeCapabilities: ack,
                  });
                  await load();
                })
              }
            >
              {t("Открыть доступ после проверки", "Reopen reviewed data plane")}
            </button>
            <button
              className="secondary"
              disabled={busy || !validReason || pending}
              onClick={() =>
                void act(async () => {
                  await api("/admin/operations/capabilities/revoke", "POST", {
                    reason,
                  });
                  await load();
                })
              }
            >
              {t(
                "Отозвать все ссылки и API-токены",
                "Revoke all links and API tokens",
              )}
            </button>
          </div>
        </div>
      )}
      <h3>{t("Задания эксплуатации", "Operational jobs")}</h3>
      {(data?.jobs || []).map((j) => (
        <div className="list-row" key={j.id}>
          <div className="break">
            <strong>
              {kind(j.kind)} · {label(j.state)}
            </strong>
            <p>
              {j.id} · {j.progress} B
            </p>
            {j.preBackupId && (
              <p>
                {t("Предварительная копия", "Pre-restore backup")}:{" "}
                {j.preBackupId}
              </p>
            )}
            {j.errorCode && (
              <p>
                {t(
                  "Операция не завершена. Доступ не открывается автоматически.",
                  "The operation did not complete. Access is not reopened automatically.",
                )}{" "}
                ({j.errorCode})
              </p>
            )}
          </div>
        </div>
      ))}
      <h3>{t("Инциденты и отчёты", "Incidents and reports")}</h3>
      {(data?.incidents || []).length === 0 && (
        <p>{t("Инцидентов нет.", "No incidents.")}</p>
      )}
      {(data?.incidents || []).map((incident) => (
        <div className="list-row" key={incident.id}>
          <div className="break">
            <strong>
              {incident.kind} · {incident.severity} · {label(incident.state)}
            </strong>
            <p>{incident.subject}</p>
            <div className="toolbar">
              <a
                href={`/api/v1/admin/operations/incidents/${incident.id}/report?format=json`}
                target="_blank"
                rel="noreferrer"
              >
                JSON
              </a>
              <a
                href={`/api/v1/admin/operations/incidents/${incident.id}/report?format=pdf`}
                target="_blank"
                rel="noreferrer"
              >
                PDF
              </a>
              {incident.state !== "resolved" &&
                ["acknowledged", "resolved"].map((state) => (
                  <button
                    key={state}
                    className="secondary"
                    disabled={busy || !validReason}
                    onClick={() =>
                      void act(async () => {
                        await api(
                          `/admin/operations/incidents/${incident.id}`,
                          "PATCH",
                          { state, reason },
                        );
                        await load();
                      })
                    }
                  >
                    {state === "acknowledged"
                      ? t("Принять в работу", "Acknowledge")
                      : t(
                          "Разрешить после проверки",
                          "Resolve after verification",
                        )}
                  </button>
                ))}
            </div>
          </div>
        </div>
      ))}
      <details>
        <summary>{t("Контрольные точки аудита", "Audit checkpoints")}</summary>
        <p>
          {t(
            "Сохраните JSON контрольной точки отдельно от этого сервера. При повреждении аудита новая граница сохраняет повреждённую историю; она не делает старую цепочку исправной.",
            "Store checkpoint JSON independently of this host. A recovery boundary preserves damaged history; it does not make the previous chain valid.",
          )}
        </p>
        <Field label={t("Тип контрольной точки", "Checkpoint type")}>
          <select
            value={checkpointKind}
            onChange={(e) => setCheckpointKind(e.target.value)}
          >
            <option value="regular">{t("Обычная", "Regular")}</option>
            <option value="recovery">
              {t(
                "После расследования повреждения",
                "After investigating corruption",
              )}
            </option>
          </select>
        </Field>
        {checkpointKind === "recovery" && (
          <>
            <Field
              label={t(
                "Hash сохранённой повреждённой истории",
                "Preserved history evidence hash",
              )}
            >
              <input
                value={evidence}
                onChange={(e) => setEvidence(e.target.value)}
              />
            </Field>
            <button
              className="secondary"
              disabled={busy}
              onClick={() =>
                void act(async () => {
                  const v = await api<{ evidenceHash: string }>(
                    "/admin/operations/audit-evidence",
                  );
                  setEvidence(v.evidenceHash);
                })
              }
            >
              {t(
                "Получить текущий hash для проверки",
                "Read current evidence hash",
              )}
            </button>
          </>
        )}
        <button
          disabled={
            busy ||
            !validReason ||
            (checkpointKind === "recovery" && evidence.length !== 64)
          }
          onClick={() =>
            void act(async () => {
              const anchor = await api<Schema<"SignedCheckpoint">>(
                "/admin/operations/checkpoints",
                "POST",
                {
                  kind: checkpointKind,
                  reason,
                  ...(checkpointKind === "recovery"
                    ? { evidenceHash: evidence }
                    : {}),
                },
              );
              setLastAnchor(anchor);
              await load();
            })
          }
        >
          {t("Зафиксировать контрольную точку", "Create checkpoint")}
        </button>
        {lastAnchor && (
          <button
            className="secondary"
            onClick={() =>
              download(`audit-${lastAnchor.checkpoint.id}.json`, lastAnchor)
            }
          >
            {t("Сохранить новую контрольную точку", "Save new checkpoint")}
          </button>
        )}
        {(data?.checkpoints || []).map((anchor) => (
          <div className="list-row" key={anchor.checkpoint.id}>
            <span className="break">
              {anchor.checkpoint.kind} · {anchor.checkpoint.createdAt}
            </span>
            <button
              className="secondary"
              onClick={() =>
                download(`audit-${anchor.checkpoint.id}.json`, anchor)
              }
            >
              {t("Скачать JSON", "Download JSON")}
            </button>
          </div>
        ))}
      </details>
      <details>
        <summary>{t("Периодические проверки", "Scheduled checks")}</summary>
        <p>
          {t(
            "Интервал в секундах: 0 отключает выбранное расписание, иначе минимум 60. Обязательная очистка файлов остаётся отдельной задачей.",
            "Interval in seconds: 0 disables this schedule, otherwise at least 60. Mandatory file cleanup runs separately.",
          )}
        </p>
        {(data?.schedules || []).map((schedule) => (
          <form
            className="toolbar"
            key={`${schedule.kind}-${schedule.intervalSeconds}`}
            onSubmit={(event) => {
              event.preventDefault();
              const input = new FormData(event.currentTarget);
              void act(async () => {
                await api("/admin/operations/schedules", "PUT", {
                  kind: schedule.kind,
                  intervalSeconds: Number(input.get("interval")),
                  ...(schedule.kind === "backup" && destination
                    ? { providerId: destination }
                    : {}),
                });
                await load();
              });
            }}
          >
            <Field label={kind(schedule.kind)}>
              <input
                name="interval"
                type="number"
                min={0}
                max={2592000}
                defaultValue={schedule.intervalSeconds}
                required
              />
            </Field>
            <button className="secondary" disabled={busy}>
              {t("Сохранить расписание", "Save schedule")}
            </button>
          </form>
        ))}
      </details>
    </section>
  );
}
