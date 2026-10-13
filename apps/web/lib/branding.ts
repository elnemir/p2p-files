import defaults from "../../../internal/branding/defaults.json";
import type { CSSProperties } from "react";
import type { Schema } from "./api-types";
export type Branding = Schema<"Branding">;
export type BrandRevision = Schema<"BrandRevision">;
export type Palette = Schema<"BrandPalette">;
export const DEFAULT_BRANDING = defaults as Branding;
export const DEFAULT_FAVICON =
  "data:image/svg+xml," +
  encodeURIComponent(
    '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32"><rect width="32" height="32" rx="8" fill="#196342"/><path d="M8 10h7l3 3h6v11H8z" fill="white"/></svg>',
  );
const properties: Record<keyof Palette, string> = {
  background: "bg",
  foreground: "fg",
  muted: "muted",
  surface: "card",
  border: "line",
  accent: "accent",
  onAccent: "on-accent",
  danger: "danger",
  dangerSurface: "danger-surface",
};
export function paletteStyle(p: Palette): CSSProperties {
  return Object.fromEntries(
    Object.entries(properties).map(([key, variable]) => [
      `--${variable}`,
      /^#[\da-f]{6}$/i.test(p[key as keyof Palette])
        ? p[key as keyof Palette]
        : DEFAULT_BRANDING.light[key as keyof Palette],
    ]),
  ) as CSSProperties;
}
export function brandCSS(b: Branding): string {
  const declarations = (p: Palette) =>
    Object.entries(paletteStyle(p))
      .map(([k, v]) => `${k}:${v}`)
      .join(";");
  return `:root[data-theme="light"],:root[data-theme="system"],:root:not([data-theme]){${declarations(b.light)}}:root[data-theme="dark"]{color-scheme:dark;${declarations(b.dark)}}:root[data-theme="light"]{color-scheme:light}@media(prefers-color-scheme:dark){:root[data-theme="system"]{${declarations(b.dark)}}}`;
}
export function contrast(a: string, b: string): number {
  if (!/^#[\da-f]{6}$/i.test(a) || !/^#[\da-f]{6}$/i.test(b)) return 0;
  const luminance = (s: string) =>
    [1, 3, 5]
      .map((i) => parseInt(s.slice(i, i + 2), 16) / 255)
      .map((v) => (v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4))
      .reduce((sum, v, i) => sum + v * [0.2126, 0.7152, 0.0722][i], 0);
  const x = luminance(a),
    y = luminance(b);
  return (Math.max(x, y) + 0.05) / (Math.min(x, y) + 0.05);
}
export function paletteValid(p: Palette): boolean {
  return (
    Object.values(p).every((v) => /^#[\da-f]{6}$/i.test(v)) &&
    [p.background, p.surface].every(
      (bg) =>
        contrast(p.foreground, bg) >= 4.5 &&
        contrast(p.muted, bg) >= 4.5 &&
        contrast(p.accent, bg) >= 4.5 &&
        contrast(p.border, bg) >= 3,
    ) &&
    contrast(p.accent, p.onAccent) >= 4.5 &&
    contrast(p.danger, p.dangerSurface) >= 4.5
  );
}
