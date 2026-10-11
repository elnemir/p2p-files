"use client";
import { useState } from "react";
import type { Api, Response } from "../lib/api-types";
type Props = {
  api: Api;
  t: (ru: string, en: string) => string;
  act: (fn: () => Promise<void>) => Promise<void>;
  busy: boolean;
};
export default function AdminExportsPanel({ api, t, act, busy }: Props) {
  const [owner, setOwner] = useState("");
  const [rows, setRows] = useState<Response<"/api/v1/admin/exports">>([]);
  const [reason, setReason] = useState("");
  const [ttl, setTTL] = useState(86400);
  const valid = reason.trim().length >= 5 && reason.length <= 2000;
  async function load() {
    setRows(
      await api(
        `/admin/exports${owner ? `?ownerId=${encodeURIComponent(owner)}` : ""}`,
      ),
    );
  }
  return (
    <section
      className="panel"
      aria-label={t("Экспорты пользователей", "User exports")}
    >
      <h2>{t("Экспорты пользователей", "User exports")}</h2>
      <p>
        {t(
          "Последние 100 заданий. Просмотр содержимого и изменение срока хранения требуют причины и записываются в аудит. Владелец не получает уведомление.",
          "Latest 100 jobs. Reading content and changing retention require a reason and are audited. The owner is not notified.",
        )}
      </p>
      <form
        className="toolbar"
        onSubmit={(e) => {
          e.preventDefault();
          void act(load);
        }}
      >
        <label className="field">
          <span>
            {t("UUID владельца (необязательно)", "Owner UUID (optional)")}
          </span>
          <input value={owner} onChange={(e) => setOwner(e.target.value)} />
        </label>
        <button disabled={busy}>
          {t("Загрузить экспорты", "Load exports")}
        </button>
      </form>
      <div className="form-grid">
        <label className="field">
          <span>{t("Причина доступа к экспорту", "Export access reason")}</span>
          <input
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            maxLength={2000}
          />
        </label>
        <label className="field">
          <span>
            {t("Новый срок хранения, секунд", "New retention, seconds")}
          </span>
          <input
            type="number"
            min={60}
            max={604800}
            value={ttl}
            onChange={(e) => setTTL(Number(e.target.value))}
          />
        </label>
      </div>
      {rows.map((row) => (
        <div className="list-row" key={row.id}>
          <div className="break">
            <strong>
              {row.format.toUpperCase()} · {row.state}
            </strong>
            <p>
              {row.id}
              <br />
              {t("Владелец", "Owner")}: {row.ownerId}
            </p>
            {row.expiresAt && (
              <p>
                {t("Хранится до", "Retained until")}:{" "}
                {new Date(row.expiresAt).toLocaleString()}
              </p>
            )}
          </div>
          <div className="toolbar">
            <button
              disabled={
                busy || !valid || row.state !== "completed" || !row.artifactId
              }
              onClick={() =>
                void act(async () => {
                  const credential = await api<{
                    artifactId: string;
                    token: string;
                  }>(`/admin/exports/${row.id}/credentials`, "POST", {
                    reason,
                  });
                  const claimed = await api<{ url: string }>(
                    `/artifacts/${credential.artifactId}/claim`,
                    "POST",
                    { token: credential.token },
                  );
                  const link = document.createElement("a");
                  link.href = claimed.url;
                  link.download = "";
                  link.click();
                })
              }
            >
              {t("Скачать с записью в аудит", "Download with audit record")}
            </button>
            <button
              className="secondary"
              disabled={
                busy ||
                !valid ||
                !Number.isInteger(ttl) ||
                ttl < 60 ||
                ttl > 604800 ||
                row.state !== "completed" ||
                !row.artifactId
              }
              onClick={() =>
                void act(async () => {
                  await api(`/admin/exports/${row.id}/ttl`, "PATCH", {
                    reason,
                    ttlSeconds: ttl,
                  });
                  await load();
                })
              }
            >
              {t("Изменить срок хранения", "Change retention")}
            </button>
          </div>
        </div>
      ))}
    </section>
  );
}
