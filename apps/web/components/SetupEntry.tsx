"use client";
import { useEffect, useState } from "react";
export default function SetupEntry() {
  const [pending, setPending] = useState(false);
  useEffect(() => {
    let current = true;
    void fetch("/api/v1/setup", { cache: "no-store" })
      .then(async (r) => {
        if (r.ok) {
          const value = await r.json();
          if (current) setPending(value.state === "pending");
        }
      })
      .catch(() => {
        /* The dedicated setup page exposes connectivity errors. */
      });
    return () => {
      current = false;
    };
  }, []);
  return pending ? (
    <aside style={{ padding: 16, textAlign: "center" }}>
      <a href="/setup">Завершить первоначальную настройку FileX</a>
    </aside>
  ) : null;
}
