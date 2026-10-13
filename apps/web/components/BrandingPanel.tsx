"use client";
import { useState, type FormEvent } from "react";
import {
  DEFAULT_BRANDING,
  paletteStyle,
  paletteValid,
  type Branding,
  type BrandRevision,
  type Palette,
} from "../lib/branding";
import { useBranding } from "./BrandingProvider";
import Icon from "./Icon";
type API = <T = unknown>(
  path: string,
  method?: string,
  value?: unknown,
) => Promise<T>;
const colorNames: Record<keyof Palette, [string, string]> = {
  background: ["Фон", "Background"],
  foreground: ["Основной текст", "Text"],
  muted: ["Вторичный текст", "Secondary text"],
  surface: ["Карточки", "Surface"],
  border: ["Границы", "Borders"],
  accent: ["Акцент", "Accent"],
  onAccent: ["Текст на акценте", "Text on accent"],
  danger: ["Текст ошибки", "Error text"],
  dangerSurface: ["Фон ошибки", "Error surface"],
};
export default function BrandingPanel({
  api,
  t,
  busy,
}: {
  api: API;
  t: (ru: string, en: string) => string;
  busy: boolean;
}) {
  const current = useBranding();
  const [base, setBase] = useState(current.revision);
  const [draft, setDraft] = useState<Branding>(() =>
    structuredClone(current.value),
  );
  const [saving, setSaving] = useState(false);
  const [pendingAssets, setPendingAssets] = useState(0);
  const [error, setError] = useState("");
  const [saved, setSaved] = useState(false);
  const [conflict, setConflict] = useState(false);
  const dirty = JSON.stringify(draft) !== JSON.stringify(current.value);
  const validName =
    draft.productName.trim() === draft.productName &&
    draft.productName.length > 0 &&
    [...draft.productName].length <= 80 &&
    !/[\p{Cc}\p{Cf}]/u.test(draft.productName);
  const valid =
    validName && paletteValid(draft.light) && paletteValid(draft.dark);
  const change = <K extends keyof Branding>(key: K, value: Branding[K]) => {
    setDraft((d) => ({ ...d, [key]: value }));
    setSaved(false);
  };
  async function reload() {
    setError("");
    setSaving(true);
    try {
      const next = await api<BrandRevision>("/admin/branding");
      setDraft(structuredClone(next.value));
      setBase(next.revision);
      current.publish(next);
      setConflict(false);
      setSaved(false);
    } catch {
      setError(
        t(
          "Не удалось загрузить оформление. Подтвердите пароль в настройках и повторите.",
          "Could not load branding. Reauthenticate in Settings and try again.",
        ),
      );
    } finally {
      setSaving(false);
    }
  }
  async function save(event: FormEvent) {
    event.preventDefault();
    if (!valid || saving || busy || pendingAssets > 0) return;
    setSaving(true);
    setError("");
    setSaved(false);
    try {
      const next = await api<BrandRevision>("/admin/branding", "PUT", {
        expectedRevision: base,
        value: draft,
      });
      setDraft(structuredClone(next.value));
      setBase(next.revision);
      current.publish(next);
      setConflict(false);
      setSaved(true);
    } catch (e) {
      if (e instanceof Error && e.message === "conflict") {
        setConflict(true);
        setError(
          t(
            "Оформление уже изменено в другой сессии. Ваш черновик сохранён на экране. Загрузите актуальную версию перед следующими изменениями.",
            "Branding changed in another session. Your draft remains on screen. Load the current version before editing again.",
          ),
        );
      } else
        setError(
          t(
            "Не удалось подтвердить сохранение. Проверьте цвета и изображения, подтвердите пароль в настройках и загрузите актуальную версию перед повтором.",
            "Could not confirm saving. Check colors and images, reauthenticate in Settings and load the current version before retrying.",
          ),
        );
    } finally {
      setSaving(false);
    }
  }
  async function upload(
    key: "logoLight" | "logoDark" | "favicon",
    file?: File,
  ) {
    if (!file) return;
    setError("");
    if (
      file.size > 128 * 1024 ||
      !["image/png", "image/jpeg"].includes(file.type)
    ) {
      setError(
        t(
          "Нужен PNG или JPEG размером до 128 КиБ.",
          "Choose a PNG or JPEG up to 128 KiB.",
        ),
      );
      return;
    }
    setPendingAssets((n) => n + 1);
    try {
      const data = await new Promise<string>((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => resolve(String(reader.result));
        reader.onerror = () => reject(new Error("read_failed"));
        reader.readAsDataURL(file);
      });
      const img = await new Promise<HTMLImageElement>((resolve, reject) => {
        const image = new Image();
        image.onload = () => resolve(image);
        image.onerror = () => reject(new Error("image_failed"));
        image.src = data;
      });
      const maxW = key === "favicon" ? 256 : 1024,
        maxH = key === "favicon" ? 256 : 512;
      if (img.naturalWidth > maxW || img.naturalHeight > maxH)
        throw new Error("dimensions");
      change(key, data);
    } catch {
      setError(
        t(
          "Изображение повреждено или превышает размеры: логотип 1024×512, favicon 256×256.",
          "Invalid image or dimensions exceeded: logo 1024×512, favicon 256×256.",
        ),
      );
    } finally {
      setPendingAssets((n) => n - 1);
    }
  }
  function preset(kind: string) {
    const light = { ...DEFAULT_BRANDING.light },
      dark = { ...DEFAULT_BRANDING.dark };
    if (kind === "ocean") {
      light.accent = "#2455a4";
      dark.accent = "#accbff";
      dark.onAccent = "#12223c";
    }
    if (kind === "plum") {
      light.accent = "#743b8a";
      dark.accent = "#dfb8f4";
      dark.onAccent = "#2b1433";
    }
    setDraft((d) => ({ ...d, light, dark }));
    setSaved(false);
  }
  return (
    <section
      className="panel branding-panel"
      aria-label={t("Брендинг и оформление", "Branding and appearance")}
    >
      <div className="section-title">
        <div>
          <p className="eyebrow">White label</p>
          <h2>
            <Icon name="paint" />
            {t("Брендинг и оформление", "Branding and appearance")}
          </h2>
        </div>
        <span className="badge">
          {t("Ревизия", "Revision")} {base}
        </span>
      </div>
      <p className="muted">
        {t(
          "Общее оформление входа, кабинета, страниц получателя, помощи и статуса. Название используется также в уведомлениях, письмах и новых PDF. Предпросмотр не меняет оформление для пользователей.",
          "Shared branding for sign-in, workspace, recipient, help and status pages. The name also appears in notifications, emails and new PDFs. Preview does not change the live appearance.",
        )}
      </p>
      {current.unavailable && (
        <p className="error" role="status">
          {t(
            "Не удалось обновить оформление с сервера. На экране последняя загруженная версия.",
            "Could not refresh branding from the server. Showing the last loaded revision.",
          )}
        </p>
      )}
      <form onSubmit={save}>
        <fieldset disabled={busy || saving || pendingAssets > 0}>
          <legend>{t("Идентичность продукта", "Product identity")}</legend>
          <div className="form-grid">
            <label className="field">
              <span>{t("Название продукта", "Product name")}</span>
              <input
                value={draft.productName}
                maxLength={80}
                required
                onChange={(e) => change("productName", e.target.value)}
              />
            </label>
            <label className="field">
              <span>{t("Подпись внизу страницы", "Footer text")}</span>
              <input
                value={draft.footerText}
                maxLength={160}
                onChange={(e) => change("footerText", e.target.value)}
              />
            </label>
            <label className="field">
              <span>{t("Описание на русском", "Russian tagline")}</span>
              <input
                value={draft.taglineRu}
                maxLength={160}
                onChange={(e) => change("taglineRu", e.target.value)}
              />
            </label>
            <label className="field">
              <span>{t("Описание на английском", "English tagline")}</span>
              <input
                value={draft.taglineEn}
                maxLength={160}
                onChange={(e) => change("taglineEn", e.target.value)}
              />
            </label>
            <label className="field">
              <span>{t("Тема по умолчанию", "Default theme")}</span>
              <select
                aria-label={t("Тема по умолчанию", "Default theme")}
                value={draft.defaultTheme}
                onChange={(e) =>
                  change(
                    "defaultTheme",
                    e.target.value as Branding["defaultTheme"],
                  )
                }
              >
                <option value="system">{t("Системная", "System")}</option>
                <option value="light">{t("Светлая", "Light")}</option>
                <option value="dark">{t("Тёмная", "Dark")}</option>
              </select>
            </label>
            <label className="field">
              <span>{t("Стиль иконок", "Icon style")}</span>
              <select
                aria-label={t("Стиль иконок", "Icon style")}
                value={draft.iconStyle}
                onChange={(e) =>
                  change("iconStyle", e.target.value as Branding["iconStyle"])
                }
              >
                <option value="outline">{t("Контурные", "Outline")}</option>
                <option value="rounded">{t("Скруглённые", "Rounded")}</option>
                <option value="duotone">{t("Двухтоновые", "Duotone")}</option>
              </select>
            </label>
          </div>
          <p className="muted">
            {t(
              "Пользователь может выбрать светлую, тёмную или системную тему для своего браузера.",
              "Users may override the default with light, dark or system mode in their browser.",
            )}
          </p>
        </fieldset>
        <fieldset disabled={busy || saving || pendingAssets > 0}>
          <legend>
            {t("Логотипы и значок вкладки", "Logos and tab icon")}
          </legend>
          <p>
            {t(
              "PNG/JPEG до 128 КиБ: логотип до 1024×512, favicon до 256×256. Изображения публикуются вместе с брендом. Для тёмной темы без отдельного логотипа используется светлый.",
              "PNG/JPEG up to 128 KiB: logo up to 1024×512, favicon up to 256×256. Images are public branding assets. Dark mode falls back to the light logo.",
            )}
          </p>
          <div className="brand-assets">
            {(
              [
                ["logoLight", t("Светлый логотип", "Light logo")],
                ["logoDark", t("Тёмный логотип", "Dark logo")],
                ["favicon", "Favicon"],
              ] as const
            ).map(([key, label]) => (
              <div key={key} className="brand-asset">
                <label className="field">
                  <span>{label}</span>
                  <input
                    type="file"
                    accept="image/png,image/jpeg"
                    onChange={(e) => {
                      void upload(key, e.target.files?.[0]);
                      e.target.value = "";
                    }}
                  />
                </label>
                {draft[key] && (
                  <>
                    <img
                      src={draft[key]}
                      alt={label}
                      className="asset-thumbnail"
                    />
                    <button
                      type="button"
                      className="secondary"
                      onClick={() => change(key, "")}
                    >
                      {t("Удалить", "Remove")} · {label}
                    </button>
                  </>
                )}
              </div>
            ))}
          </div>
        </fieldset>
        <fieldset disabled={busy || saving || pendingAssets > 0}>
          <legend>{t("Цветовые палитры", "Color palettes")}</legend>
          <div className="actions">
            {[
              ["forest", t("Лес", "Forest")],
              ["ocean", t("Океан", "Ocean")],
              ["plum", t("Слива", "Plum")],
            ].map(([id, label]) => (
              <button
                type="button"
                className="secondary"
                key={id}
                onClick={() => preset(id)}
              >
                {label}
              </button>
            ))}
          </div>
          <p className="muted">
            {t(
              "Контраст текста и акцента — от 4,5:1; границ — от 3:1. Плохая читаемость блокирует применение.",
              "Text and accent contrast must be at least 4.5:1; borders at least 3:1. Insufficient contrast prevents publishing.",
            )}
          </p>
          <div className="brand-palettes">
            {(["light", "dark"] as const).map((mode) => (
              <fieldset key={mode}>
                <legend>
                  {mode === "light"
                    ? t("Светлая палитра", "Light palette")
                    : t("Тёмная палитра", "Dark palette")}
                </legend>
                {Object.entries(colorNames).map(([key, names]) => {
                  const name = key as keyof Palette,
                    label = `${mode === "light" ? t("Светлая", "Light") : t("Тёмная", "Dark")} · ${t(...names)}`;
                  return (
                    <div className="color-field" key={key}>
                      <label htmlFor={`brand-${mode}-${key}`}>
                        {t(...names)}
                      </label>
                      <input
                        type="color"
                        aria-label={
                          label + " · " + t("Выбор цвета", "Color picker")
                        }
                        value={
                          /^#[\da-f]{6}$/i.test(draft[mode][name])
                            ? draft[mode][name]
                            : "#000000"
                        }
                        onChange={(e) =>
                          change(mode, {
                            ...draft[mode],
                            [name]: e.target.value,
                          })
                        }
                      />
                      <input
                        id={`brand-${mode}-${key}`}
                        aria-label={label}
                        value={draft[mode][name]}
                        maxLength={7}
                        spellCheck={false}
                        onChange={(e) =>
                          change(mode, {
                            ...draft[mode],
                            [name]: e.target.value,
                          })
                        }
                      />
                    </div>
                  );
                })}
                {!paletteValid(draft[mode]) && (
                  <p className="error" role="alert">
                    {t(
                      "Проверьте HEX-коды и контраст этой палитры.",
                      "Check this palette’s HEX values and contrast.",
                    )}
                  </p>
                )}
              </fieldset>
            ))}
          </div>
        </fieldset>
        <h3>{t("Предпросмотр", "Preview")}</h3>
        <div className="brand-previews">
          {(["light", "dark"] as const).map((mode) => (
            <section
              className="brand-preview"
              style={paletteStyle(draft[mode])}
              key={mode}
              aria-label={
                mode === "light"
                  ? t("Светлый предпросмотр", "Light preview")
                  : t("Тёмный предпросмотр", "Dark preview")
              }
            >
              <p className="muted">
                {mode === "light"
                  ? t("Светлая тема", "Light mode")
                  : t("Тёмная тема", "Dark mode")}
              </p>
              {(mode === "dark"
                ? draft.logoDark || draft.logoLight
                : draft.logoLight || draft.logoDark) && (
                <img
                  className="preview-logo"
                  src={
                    mode === "dark"
                      ? draft.logoDark || draft.logoLight
                      : draft.logoLight || draft.logoDark
                  }
                  alt={draft.productName}
                />
              )}
              <h3>
                {draft.productName || t("Название продукта", "Product name")}
              </h3>
              <p>{t(draft.taglineRu, draft.taglineEn)}</p>
              <div className="preview-card">
                <p>
                  <Icon name="file" style={draft.iconStyle} />{" "}
                  {t("Документы проекта", "Project documents")}
                </p>
                <p className="muted">
                  {t("3 файла · Доступ по ссылке", "3 files · Link access")}
                </p>
                <span className="preview-button">
                  <Icon name="download" style={draft.iconStyle} />
                  {t("Скачать файлы", "Download files")}
                </span>
              </div>
              <div className="preview-icons">
                {(
                  [
                    "transfers",
                    "reports",
                    "jobs",
                    "notifications",
                    "settings",
                    "admin",
                  ] as const
                ).map((name) => (
                  <Icon key={name} name={name} style={draft.iconStyle} />
                ))}
              </div>
              <p className="error">
                {t("Пример сообщения об ошибке", "Sample error message")}
              </p>
              <small>{draft.footerText}</small>
            </section>
          ))}
        </div>
        {!validName && (
          <p className="error" role="alert">
            {t(
              "Укажите название до 80 символов без управляющих знаков и пробелов по краям.",
              "Enter a name of up to 80 characters without control characters or surrounding spaces.",
            )}
          </p>
        )}
        {error && (
          <p className="error" role="alert">
            {error}
          </p>
        )}
        {saved && (
          <p className="notice compact" role="status">
            {t(
              "Брендинг применён. Новые страницы и документы используют это оформление.",
              "Branding applied. New pages and documents use this appearance.",
            )}
          </p>
        )}
        <div className="actions brand-save">
          <button
            disabled={
              busy ||
              saving ||
              pendingAssets > 0 ||
              !valid ||
              !dirty ||
              conflict
            }
          >
            {saving
              ? t("Сохраняем…", "Saving…")
              : t("Применить брендинг", "Apply branding")}
          </button>
          <button
            type="button"
            className="secondary"
            disabled={busy || saving || pendingAssets > 0}
            onClick={() => {
              setDraft(structuredClone(DEFAULT_BRANDING));
              setSaved(false);
            }}
          >
            {t("Исходное оформление", "Reset to defaults")}
          </button>
          <button
            type="button"
            className="secondary"
            disabled={busy || saving || pendingAssets > 0}
            onClick={() => void reload()}
          >
            {conflict
              ? t("Загрузить актуальную версию", "Load current version")
              : t("Отменить изменения", "Discard changes")}
          </button>
        </div>
        <p className="muted">
          {t(
            "Сброс сначала меняет предпросмотр. Для публикации нажмите «Применить брендинг». Уже созданные PDF сохранят прежнее оформление.",
            "Reset updates the preview first. Click Apply branding to publish. Existing PDFs retain their original appearance.",
          )}
        </p>
      </form>
    </section>
  );
}
