"use client";
import { useBranding } from "./BrandingProvider";
import type { Branding } from "../lib/branding";
const paths = {
  back: "M20 12H4 M10 6l-6 6 6 6",
  transfers: "M3 7h7l2 3h9v11H3V7z M3 7V4h7l2 3h7v3",
  reports: "M6 3h9l4 4v14H6z M14 3v5h5 M9 12h7 M9 16h7",
  jobs: "M8 4h8v4H8z M8 6H5v15h14V6h-3 M8 12h8 M8 16h5",
  notifications: "M6 16V10a6 6 0 0 1 12 0v6l2 2H4z M10 21h4",
  settings: "M4 6h16 M4 12h16 M4 18h16 M8 3v6 M16 9v6 M10 15v6",
  admin: "M12 3l8 3v6c0 5-8 9-8 9s-8-4-8-9V6z M8 12l3 3 5-6",
  help: "M9 8a3 3 0 1 1 4 3c-1 1-1 1-1 3 M12 17v1 M12 2a10 10 0 1 0 0 20 10 10 0 0 0 0-20",
  status: "M3 12h4l3-7 4 14 3-7h4",
  upload: "M12 16V3 M7 8l5-5 5 5 M4 15v6h16v-6",
  download: "M12 3v13 M7 11l5 5 5-5 M4 17v4h16v-4",
  close: "M6 6l12 12 M6 18L18 6",
  file: "M6 3h8l5 5v13H6z M14 3v6h5",
  paint:
    "M12 3a9 9 0 1 0 0 18h2a2 2 0 0 0 0-4h-1a2 2 0 0 1 0-4h5a3 3 0 0 0 3-3c0-4-4-7-9-7 M7 8h.1 M11 6h.1 M16 7h.1 M6 13h.1",
  check: "M4 12l5 5L20 6",
} as const;
export type IconName = keyof typeof paths;
export default function Icon({
  name,
  style,
}: {
  name: IconName;
  style?: Branding["iconStyle"];
}) {
  const { value: b } = useBranding();
  const variant = style || b.iconStyle;
  return (
    <svg
      className={`app-icon icon-${variant}`}
      data-icon-style={variant}
      viewBox="0 0 24 24"
      aria-hidden="true"
      focusable="false"
      fill="none"
      stroke="currentColor"
      strokeWidth={variant === "outline" ? 1.6 : 2}
      strokeLinecap={variant === "outline" ? "square" : "round"}
      strokeLinejoin={variant === "outline" ? "miter" : "round"}
    >
      {variant === "duotone" && (
        <rect
          x="1"
          y="1"
          width="22"
          height="22"
          rx="7"
          fill="currentColor"
          opacity="0.16"
          stroke="none"
        />
      )}
      <path d={paths[name]} />
    </svg>
  );
}
