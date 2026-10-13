"use client";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";
import {
  brandCSS,
  DEFAULT_BRANDING,
  DEFAULT_FAVICON,
  type BrandRevision,
} from "../lib/branding";
const BrandContext = createContext<
  BrandRevision & {
    unavailable: boolean;
    refresh: () => Promise<void>;
    publish: (value: BrandRevision) => void;
  }
>({
  unavailable: false,
  revision: 0,
  value: DEFAULT_BRANDING,
  refresh: async () => {},
  publish: () => {},
});
export function useBranding() {
  return useContext(BrandContext);
}
export default function BrandingProvider({
  initial,
  children,
}: {
  initial: BrandRevision;
  children: ReactNode;
}) {
  const [brand, setBrand] = useState(initial);
  const [unavailable, setUnavailable] = useState(initial.revision < 0);
  const etag = useRef("");
  const request = useRef(0);
  const refresh = useCallback(async () => {
    const id = ++request.current;
    const response = await fetch("/api/v1/branding", {
      cache: "no-store",
      signal: AbortSignal.timeout(5000),
      headers: etag.current ? { "If-None-Match": etag.current } : {},
    });
    if (response.status === 304) {
      setUnavailable(false);
      return;
    }
    if (!response.ok) throw new Error("branding_unavailable");
    const next: BrandRevision = await response.json();
    if (id !== request.current) return;
    etag.current = response.headers.get("ETag") || "";
    setUnavailable(false);
    setBrand(next);
  }, []);
  const publish = useCallback((next: BrandRevision) => {
    request.current++;
    etag.current = "";
    setUnavailable(false);
    setBrand(next);
    const channel = new BroadcastChannel("filex.branding");
    channel.postMessage("changed");
    channel.close();
  }, []);
  useEffect(() => {
    // Failed background refresh preserves the last authoritative presentation and retries on focus/tick.
    const update = () => {
      void refresh().catch(() => setUnavailable(true));
    };
    const channel = new BroadcastChannel("filex.branding");
    channel.onmessage = update;
    const visible = () => {
      if (document.visibilityState === "visible") update();
    };
    const generation = request;
    const timer = setInterval(visible, 60000);
    window.addEventListener("focus", update);
    document.addEventListener("visibilitychange", visible);
    if (initial.revision < 0) update();
    return () => {
      clearInterval(timer);
      channel.close();
      window.removeEventListener("focus", update);
      document.removeEventListener("visibilitychange", visible);
      generation.current++;
    };
  }, [refresh, initial.revision]);
  useEffect(() => {
    const links = document.querySelectorAll<HTMLLinkElement>(
      'link[rel="icon"],link[rel="apple-touch-icon"]',
    );
    for (const link of links)
      link.href = brand.value.favicon || DEFAULT_FAVICON;
  }, [brand.value.favicon]);
  return (
    <BrandContext.Provider value={{ ...brand, unavailable, refresh, publish }}>
      <style data-branding="true">{brandCSS(brand.value)}</style>
      {children}
    </BrandContext.Provider>
  );
}
