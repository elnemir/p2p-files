"use client";
import { useState } from "react";
import type { Api, Schema } from "../lib/api-types";
export default function LicensePanel({
  api,
  t,
}: {
  api: Api;
  t: (ru: string, en: string) => string;
}) {
  const [status, setStatus] = useState<Schema<"LicenseStatus"> | null>(null);
  const [envelope, setEnvelope] = useState("");
  const [preview, setPreview] = useState<Schema<"LicenseClaims"> | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  async function perform(fn: () => Promise<void>) {
    setBusy(true);
    setError("");
    try {
      await fn();
    } catch (e) {
      const code = e instanceof Error ? e.message : String(e);
      setError(
        code === "invalid_license"
          ? t(
              "Лицензия не прошла проверку подписи, срока или кода установки. Текущая лицензия сохранена.",
              "The signature, dates or installation binding are invalid. The current license is preserved.",
            )
          : code === "conflict"
            ? t(
                "Версия изменилась. Проверьте состояние и загрузите актуальное продление.",
                "The revision changed. Check status and load the current renewal.",
              )
            : code,
      );
    } finally {
      setBusy(false);
    }
  }
  const stateName = (value: string) =>
    ({
      active: t("Действует", "Active"),
      missing: t("Не установлена", "Not installed"),
      invalid: t("Не прошла проверку", "Verification failed"),
      expired: t("Срок истёк", "Expired"),
      not_yet_valid: t("Срок ещё не начался", "Not yet valid"),
      clock_suspect: t("Проверьте время сервера", "Check server time"),
      update_not_entitled: t(
        "Лицензия не разрешает эту версию",
        "This release is not licensed",
      ),
    })[value] ?? value;
  const date = (value: string) =>
    new Intl.DateTimeFormat(t("ru-RU", "en-GB"), {
      dateStyle: "medium",
      timeStyle: "short",
      timeZone: "UTC",
    }).format(new Date(value)) + " UTC";
  const featureName = (value: string) =>
    ({
      webdav: "WebDAV",
      federated_auth: t("Корпоративный вход", "Corporate sign-in"),
      white_label: t("Собственный брендинг", "Custom branding"),
    })[value] ?? value;
  return (
    <section className="license-panel">
      <h2>{t("Лицензия FileX", "FileX license")}</h2>
      <p>
        {t(
          "Проверка выполняется офлайн. Для просмотра и замены подтвердите пароль администратора.",
          "Verification is offline. Reauthenticate as administrator to inspect or replace the license.",
        )}
      </p>
      <button
        disabled={busy}
        onClick={() =>
          void perform(async () =>
            setStatus(await api<Schema<"LicenseStatus">>("/admin/license")),
          )
        }
      >
        {t("Проверить состояние", "Check status")}
      </button>
      {status && (
        <>
          <dl>
            <dt>{t("Код установки", "Installation ID")}</dt>
            <dd style={{ overflowWrap: "anywhere" }}>
              {status.installationId}
            </dd>
            <dt>{t("Состояние", "State")}</dt>
            <dd>{stateName(status.state)}</dd>
            <dt>{t("Включённые пользователи", "Enabled users")}</dt>
            <dd>
              {status.enabledUsers} /{" "}
              {status.claims
                ? (status.claims.limits.maxEnabledUsers ??
                  t("Без лимита", "Unlimited"))
                : "—"}
            </dd>
            <dt>{t("Лимит передачи, байт", "Transfer limit, bytes")}</dt>
            <dd>
              {status.claims
                ? (status.claims.limits.maxTransferBytes ??
                  t("Без лимита", "Unlimited"))
                : "—"}
            </dd>
            <dt>{t("Дополнительные функции", "Optional features")}</dt>
            <dd>
              {status.claims?.features.map(featureName).join(", ") || "—"}
            </dd>
            <dt>{t("Обновления до", "Updates until")}</dt>
            <dd>{status.claims ? date(status.claims.updatesUntil) : "—"}</dd>
            <dt>{t("Срок действия", "Expiry")}</dt>
            <dd>
              {status.claims
                ? status.claims.expiresAt
                  ? date(status.claims.expiresAt)
                  : t("Бессрочная", "Perpetual")
                : "—"}
            </dd>
          </dl>
          <button
            disabled={busy}
            onClick={() => {
              const blob = new Blob(
                [
                  JSON.stringify(
                    {
                      schemaVersion: 1,
                      product: "filex",
                      installationId: status.installationId,
                      requestId: crypto.randomUUID(),
                    },
                    null,
                    2,
                  ),
                ],
                { type: "application/json" },
              );
              const url = URL.createObjectURL(blob);
              const a = document.createElement("a");
              a.href = url;
              a.download = "filex-installation.json";
              a.click();
              URL.revokeObjectURL(url);
            }}
          >
            {t("Скачать запрос лицензии", "Download license request")}
          </button>
          {status.overLimit && (
            <p role="status">
              {t(
                "Лимит пользователей превышен. Новые учётные записи запрещены; существующие не удаляются.",
                "User limit exceeded. New accounts are blocked; existing accounts are retained.",
              )}
            </p>
          )}
          {status.pending && (
            <p>
              {t("Продление ожидает начала срока:", "Renewal starts at:")}{" "}
              {date(status.pending.notBefore)}
            </p>
          )}
        </>
      )}
      <label>
        {t("Подписанный файл лицензии", "Signed license file")}
        <input
          type="file"
          accept=".jws,.txt"
          disabled={busy}
          onChange={async (e) => {
            setPreview(null);
            const f = e.target.files?.[0];
            if (!f) return;
            if (f.size > 16384) {
              setError(t("Файл слишком большой", "File too large"));
              return;
            }
            setEnvelope((await f.text()).trim());
          }}
        />
      </label>
      <label>
        {t("Содержимое лицензии", "License content")}
        <textarea
          rows={5}
          value={envelope}
          maxLength={16384}
          onChange={(e) => {
            setEnvelope(e.target.value);
            setPreview(null);
          }}
        />
      </label>
      <button
        disabled={busy || !envelope || !status}
        onClick={() =>
          void perform(async () =>
            setPreview(
              await api<Schema<"LicenseClaims">>(
                "/admin/license/preview",
                "POST",
                { envelope },
              ),
            ),
          )
        }
      >
        {t("Проверить перед применением", "Verify before applying")}
      </button>
      {preview && (
        <>
          <pre style={{ whiteSpace: "pre-wrap", overflowWrap: "anywhere" }}>
            {JSON.stringify(preview, null, 2)}
          </pre>
          <p>
            {t(
              "Применение заменит права установки. После истечения новые загрузки и публикации будут закрыты; прежняя выдача сохраняет свой TTL.",
              "Applying replaces installation rights. Expiry blocks new uploads and publication; existing deliveries retain their TTL.",
            )}
          </p>
          <button
            disabled={busy || !status}
            onClick={() =>
              void perform(async () => {
                setStatus(
                  await api<Schema<"LicenseStatus">>("/admin/license", "PUT", {
                    envelope,
                    expectedEpoch: status?.epoch,
                  }),
                );
                setPreview(null);
                setEnvelope("");
              })
            }
          >
            {t("Применить лицензию", "Apply license")}
          </button>
        </>
      )}
      {error && <p role="alert">{error}</p>}
    </section>
  );
}
