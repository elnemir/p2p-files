import { cache } from "react";
import { DEFAULT_BRANDING, type BrandRevision } from "./branding";
// Only an operator-controlled internal endpoint is used; request Host is never a fetch target.
export const readBranding = cache(async (): Promise<BrandRevision> => {
  try {
    const url =
      process.env.FILEX_BRANDING_API_URL || "http://127.0.0.1:9090/branding";
    const response = await fetch(url, {
      cache: "no-store",
      redirect: "error",
      signal: AbortSignal.timeout(3000),
    });
    if (!response.ok) throw new Error("branding_unavailable");
    const value: BrandRevision = await response.json();
    if (!value.value?.productName || !value.value.light || !value.value.dark)
      throw new Error("branding_unavailable");
    return value;
  } catch {
    // A neutral fallback keeps recovery/status usable without claiming the wrong customer brand.
    return {
      revision: -1,
      value: { ...DEFAULT_BRANDING, productName: "Files" },
    };
  }
});
