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
  plus: "M12 5v14 M5 12h14",
  search: "M21 21l-6-6 M10 3a7 7 0 1 0 0 14 7 7 0 0 0 0-14",
  clock: "M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18 M12 7v5l3 2",
  link: "M10 13l4-4 M8 15l-1 1a4 4 0 0 1-6-6l5-5a4 4 0 0 1 6 0 M16 9l1-1a4 4 0 0 1 6 6l-5 5a4 4 0 0 1-6 0",
  copy: "M8 8h13v13H8z M16 8V3H3v13h5",
  edit: "M4 16L16 4l4 4L8 20H4z M14 6l4 4",
  trash: "M3 6h18 M8 6V3h8v3 M6 6l1 15h10l1-15 M10 10v7 M14 10v7",
  refresh:
    "M20 10a8 8 0 0 0-14-5L3 8 M3 3v5h5 M4 14a8 8 0 0 0 14 5l3-3 M21 21v-5h-5",
  lock: "M5 10h14v11H5z M8 10V6a4 4 0 0 1 8 0v4 M12 14v3",
  key: "M14 6a5 5 0 1 0 4 8L22 10l-3-3-3 3 M7 7h.1",
  mail: "M3 5h18v14H3z M3 5l9 8 9-8",
  server: "M3 3h18v7H3z M3 14h18v7H3z M7 6h.1 M7 17h.1 M12 6h5 M12 17h5",
  globe:
    "M3 12h18 M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18 M12 3c5 5 5 13 0 18-5-5-5-13 0-18",
  chevron: "M9 5l7 7-7 7",
  info: "M12 10v7 M12 6h.1 M12 2a10 10 0 1 0 0 20 10 10 0 0 0 0-20",
  warning: "M12 3L2 21h20z M12 9v5 M12 17h.1",
  logout: "M9 3H3v18h6 M8 12h13 M16 7l5 5-5 5",
  save: "M4 3h13l4 4v14H3V3z M8 3v6h8V3 M7 21v-8h10v8",
  folder: "M3 6h7l2 3h9v11H3z",
  pause: "M8 4v16 M16 4v16",
  eye: "M2 12s4-7 10-7 10 7 10 7-4 7-10 7S2 12 2 12 M12 9a3 3 0 1 0 0 6 3 3 0 0 0 0-6",
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
