"use client";

import {
  Children,
  cloneElement,
  isValidElement,
  useEffect,
  useId,
  useRef,
  useState,
  type ReactNode,
} from "react";
import Icon, { type IconName } from "./Icon";

export function Field({
  title,
  children,
}: {
  title: string;
  children: ReactNode;
}) {
  const id = useId();
  const help = Children.toArray(children).some(
    (child) => isValidElement(child) && child.type === "small",
  );
  return (
    <label className="field">
      <span id={id}>{title}</span>
      {Children.map(children, (child) => {
        if (!isValidElement<Record<string, unknown>>(child)) return child;
        if (["input", "select", "textarea"].includes(String(child.type)))
          return cloneElement(child, {
            "aria-labelledby": id,
            ...(help ? { "aria-describedby": `${id}-help` } : {}),
          });
        if (child.type === "small")
          return cloneElement(child, { id: `${id}-help` });
        return child;
      })}
    </label>
  );
}

export type Translate = (ru: string, en: string) => string;
export function SectionNav({
  label,
  value,
  onChange,
  items,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  items: { id: string; label: string; icon: IconName }[];
}) {
  return (
    <nav className="section-nav" aria-label={label}>
      {items.map((item) => (
        <button
          type="button"
          key={item.id}
          className={item.id === value ? "active" : "secondary"}
          aria-current={item.id === value ? "page" : undefined}
          onClick={() => onChange(item.id)}
        >
          <Icon name={item.icon} />
          {item.label}
        </button>
      ))}
    </nav>
  );
}

export function EmptyState({
  icon,
  title,
  children,
}: {
  icon: IconName;
  title: string;
  children?: ReactNode;
}) {
  return (
    <div className="empty">
      <span className="empty-icon">
        <Icon name={icon} />
      </span>
      <h2>{title}</h2>
      {children}
    </div>
  );
}

export function durationLabel(seconds: number, t: Translate) {
  if (!Number.isFinite(seconds) || seconds < 1) return "—";
  return [
    [86400, t("д.", "d")],
    [3600, t("ч.", "h")],
    [60, t("мин.", "min")],
    [1, t("сек.", "s")],
  ]
    .flatMap(([unit, label]) => {
      const amount = Math.floor(seconds / Number(unit));
      seconds %= Number(unit);
      return amount ? [`${amount} ${label}`] : [];
    })
    .join(" ");
}

export function DurationControl({
  value,
  maxSeconds,
  onChange,
  t,
  disabled,
}: {
  value: number;
  maxSeconds: number;
  onChange: (value: number) => void;
  t: Translate;
  disabled: boolean;
}) {
  const id = useId();
  // Keep both endpoints reachable by keyboard even when policy uses an unusual duration.
  const unit = maxSeconds % 3600 === 0 ? 3600 : maxSeconds % 60 === 0 ? 60 : 1;
  const unitLabel =
    unit === 3600
      ? t("часов", "hours")
      : unit === 60
        ? t("минут", "minutes")
        : t("секунд", "seconds");
  const presets: [number, string][] = [
    [3600, t("1 час", "1 hour")],
    [86400, t("1 день", "1 day")],
    [259200, t("3 дня", "3 days")],
    [604800, t("7 дней", "7 days")],
    [2592000, t("30 дней", "30 days")],
  ];
  const available = presets.filter(([seconds]) => seconds <= maxSeconds);
  if (!available.some(([seconds]) => seconds === maxSeconds))
    available.push([
      maxSeconds,
      `${t("Максимум", "Maximum")} · ${durationLabel(maxSeconds, t)}`,
    ]);
  return (
    <fieldset className="duration-control" disabled={disabled}>
      <legend>
        <Icon name="clock" />
        {t("Срок доступа", "Access duration")}
      </legend>
      <div
        className="presets"
        aria-label={t("Быстрый выбор срока", "Duration presets")}
      >
        {available.map(([seconds, label]) => (
          <button
            type="button"
            key={seconds}
            aria-pressed={value === seconds}
            className={value === seconds ? "active" : "secondary"}
            onClick={() => onChange(seconds)}
          >
            {label}
          </button>
        ))}
      </div>
      <label htmlFor={id} className="sr-only">
        {t("Изменить срок ползунком", "Adjust duration slider")}
      </label>
      <input
        id={id}
        type="range"
        min={1}
        max={Math.floor(maxSeconds / unit)}
        step={1}
        value={Math.max(
          1,
          Math.min(
            Math.floor(maxSeconds / unit),
            Number.isFinite(value) ? value / unit : 1,
          ),
        )}
        aria-valuetext={durationLabel(value, t)}
        onChange={(e) => onChange(Number(e.target.value) * unit)}
      />
      <label className="duration-exact">
        <span>{t("Срок доступа, ", "Access duration, ") + unitLabel}</span>
        <input
          type="number"
          min={1 / unit}
          max={maxSeconds / unit}
          step="any"
          required
          value={value ? value / unit : ""}
          onChange={(e) => onChange(Math.round(Number(e.target.value) * unit))}
        />
      </label>
      <small>
        {t("Лимит установки: ", "Installation limit: ") +
          durationLabel(maxSeconds, t) +
          ". "}
        {t(
          "Срок отсчитывается с публикации. Полное скачивание закрывает доступ раньше.",
          "The duration starts at publication. A complete download ends access earlier.",
        )}
      </small>
    </fieldset>
  );
}

type Confirmation = { title: string; description: string; action: string };
export function useConfirmation(t: Translate) {
  const [pending, setPending] = useState<Confirmation | null>(null);
  const result = useRef<((ok: boolean) => void) | null>(null);
  const dialog = useRef<HTMLDialogElement>(null);
  const trigger = useRef<HTMLElement | null>(null);
  const title = useId();
  const description = useId();
  useEffect(() => {
    if (pending) dialog.current?.showModal();
    else {
      dialog.current?.close();
      if (trigger.current) {
        const element = trigger.current;
        requestAnimationFrame(() => {
          if (element.isConnected) element.focus();
        });
        trigger.current = null;
      }
    }
  }, [pending]);
  useEffect(() => () => result.current?.(false), []);
  function finish(ok: boolean) {
    result.current?.(ok);
    result.current = null;
    setPending(null);
  }
  return {
    confirm: (value: Confirmation) =>
      new Promise<boolean>((resolve) => {
        result.current?.(false);
        result.current = resolve;
        trigger.current =
          document.activeElement instanceof HTMLElement
            ? document.activeElement
            : null;
        setPending(value);
      }),
    confirmation: (
      <dialog
        ref={dialog}
        className="confirmation"
        aria-labelledby={title}
        aria-describedby={description}
        onCancel={(event) => {
          event.preventDefault();
          finish(false);
        }}
      >
        <span className="empty-icon">
          <Icon name="warning" />
        </span>
        <h2 id={title}>{pending?.title}</h2>
        <p id={description}>{pending?.description}</p>
        <div className="toolbar">
          <button
            type="button"
            className="secondary"
            onClick={() => finish(false)}
          >
            {t("Отмена", "Cancel")}
          </button>
          <button type="button" className="danger" onClick={() => finish(true)}>
            {pending?.action}
          </button>
        </div>
      </dialog>
    ),
  };
}

export function Reauthentication({
  api,
  t,
  busy,
  onSuccess,
}: {
  api: <T>(path: string, method?: string, data?: unknown) => Promise<T>;
  t: Translate;
  busy: boolean;
  onSuccess?: () => void;
}) {
  const [working, setWorking] = useState(false);
  const [result, setResult] = useState<"" | "success" | "error">("");
  return (
    <details className="reauth-panel">
      <summary>
        <Icon name="lock" />
        {t("Подтвердить личность", "Confirm identity")}
      </summary>
      <p>
        {t(
          "Для защищённых действий нужен свежий вход. Подтверждение действует 5 минут; сервер проверяет его при каждой операции.",
          "Protected actions require recent sign-in. Confirmation lasts 5 minutes and is checked by the server for every operation.",
        )}
      </p>
      <form
        className="toolbar"
        onSubmit={(event) => {
          event.preventDefault();
          const form = event.currentTarget;
          const password = new FormData(form).get("password");
          setWorking(true);
          setResult("");
          void api("/auth/reauth", "POST", { password })
            .then(() => {
              form.reset();
              setResult("success");
              onSuccess?.();
            })
            .catch(() => setResult("error"))
            .finally(() => setWorking(false));
        }}
      >
        <label className="field">
          <span>{t("Текущий пароль", "Current password")}</span>
          <input
            name="password"
            type="password"
            autoComplete="current-password"
            required
            disabled={busy || working}
          />
        </label>
        <button disabled={busy || working}>
          <Icon name="check" />
          {t("Подтвердить на 5 минут", "Confirm for 5 minutes")}
        </button>
      </form>
      {result === "success" && (
        <p role="status">
          {t(
            "Личность подтверждена. Можно продолжать.",
            "Identity confirmed. You can continue.",
          )}
        </p>
      )}
      {result === "error" && (
        <p role="alert">
          {t(
            "Не удалось подтвердить личность. Проверьте пароль или повторите вход.",
            "Could not confirm identity. Check your password or sign in again.",
          )}
        </p>
      )}
    </details>
  );
}
