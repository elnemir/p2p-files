import SwaggerParser from "@apidevtools/swagger-parser";
import Ajv from "ajv/dist/2020.js";
import addFormats from "ajv-formats";
import type { Page, Request } from "@playwright/test";

export async function contractObserver() {
  const api: any = await SwaggerParser.dereference("contracts/openapi.json");
  const normalize = (value: any): any => {
    if (Array.isArray(value)) return value.map(normalize);
    if (value && typeof value === "object")
      return Object.fromEntries(
        Object.entries(value)
          .filter(([k]) => k !== "$id" && k !== "$schema")
          .map(([k, v]) => [k, normalize(v)]),
      );
    return value;
  };
  const ajv = new Ajv({ strict: false });
  addFormats(ajv);
  const checks = new Map<string, any>();
  const failures: string[] = [];
  const pending: Promise<void>[] = [];
  const listeners = new Map<Page, (request: Request) => void>();
  let count = 0;
  let finished = false;

  async function validateResponse(request: Request) {
    const response = await request.response();
    if (!response) throw new Error("Missing finished response");
    if (!response.headers()["content-type"]?.includes("application/json"))
      return;
    // Read bytes before schema compilation so navigation cannot evict a completed body first.
    const body = await response.json();
    const url = new URL(response.url());
    const path = Object.keys(api.paths).find((path) =>
      new RegExp("^" + path.replace(/\{[^}]+\}/g, "[^/]+") + "$").test(
        url.pathname,
      ),
    );
    const method = response.request().method().toLowerCase();
    const op = path && api.paths[path][method];
    const schema =
      op?.responses[String(response.status())]?.content?.["application/json"]
        ?.schema;
    if (!schema) {
      failures.push(
        `${method} ${path || url.pathname}: undocumented JSON status ${response.status()}`,
      );
      return;
    }
    const key = method + path + response.status();
    if (!checks.has(key)) checks.set(key, ajv.compile(normalize(schema)));
    const validate = checks.get(key);
    count++;
    if (!validate(body))
      failures.push(`${method} ${path}: ${JSON.stringify(validate.errors)}`);
  }

  function observe(page: Page) {
    if (finished) throw new Error("Contract observation already finished");
    if (listeners.has(page)) return;
    const listener = (request: Request) => {
      const path = new URL(request.url()).pathname;
      if (path.endsWith("/content") || !path.startsWith("/api/v1/")) return;
      // Attach rejection handling immediately; browser events can outlive an assertion.
      pending.push(
        validateResponse(request).catch(() => {
          failures.push("API response capture failed");
        }),
      );
    };
    listeners.set(page, listener);
    page.on("requestfinished", listener);
  }

  return {
    observe,
    async finish() {
      // Close the event boundary before draining so teardown cannot enqueue new jobs.
      finished = true;
      for (const [page, listener] of listeners)
        page.off("requestfinished", listener);
      listeners.clear();
      await Promise.all(pending);
      return { failures, count };
    },
  };
}
