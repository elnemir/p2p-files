import SwaggerParser from "@apidevtools/swagger-parser";
import Ajv from "ajv/dist/2020.js";
import addFormats from "ajv-formats";
import type { APIResponse, Page, Route } from "@playwright/test";

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
  const routes = new Map<Page, (route: Route) => Promise<void>>();
  const pattern = "**/api/v1/**";
  let count = 0;
  let finished = false;

  async function validateResponse(response: APIResponse, method: string) {
    if (!response.headers()["content-type"]?.includes("application/json"))
      return;
    // APIResponse owns its bytes independently of the document being navigated away from.
    const body = await response.json();
    const url = new URL(response.url());
    const path = Object.keys(api.paths).find((path) =>
      new RegExp("^" + path.replace(/\{[^}]+\}/g, "[^/]+") + "$").test(
        url.pathname,
      ),
    );
    const op = path && api.paths[path][method];
    const schema =
      op?.responses[String(response.status())]?.content?.["application/json"]
        ?.schema;
    if (!schema) {
      failures.push(
        `${method} ${path || "<unmatched>"}: undocumented JSON status ${response.status()}`,
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

  async function observe(page: Page) {
    if (finished) throw new Error("Contract observation already finished");
    if (routes.has(page)) return;
    const handler = async (route: Route) => {
      const request = route.request();
      const path = new URL(request.url()).pathname;
      // WebKit cannot fulfill a routed 304. Conditional branding reads keep native HTTP;
      // the branding scenario verifies ETag, 304/no-body, and stale-validator 200 explicitly.
      const conditionalBranding = path === "/api/v1/branding" && request.method() === "GET" && request.headers()["if-none-match"] !== undefined;
      if (finished || conditionalBranding || path.endsWith("/content") || /^\/api\/v1\/uploads\/[^/]+$/.test(path) || !path.startsWith("/api/v1/")) {
        await route.fallback();
        return;
      }
      let responding = false;
      const capture = async () => {
        // Forward the real request exactly once. Never replay mutations or follow redirects.
        const response = await route.fetch({ maxRetries: 0, maxRedirects: 0 });
        try {
          await validateResponse(response, request.method().toLowerCase());
          // Preserve the upstream status, headers and body, including non-JSON responses.
          responding = true;
          await route.fulfill({ response });
        } finally {
          await response.dispose();
        }
      };
      // Attach rejection handling immediately; navigation can outlive a test action.
      const job = capture().catch(async () => {
        failures.push("API response capture failed");
        if (!responding) await route.abort("failed");
      });
      pending.push(job);
      await job;
    };
    routes.set(page, handler);
    await page.route(pattern, handler);
  }

  return {
    observe,
    async finish() {
      // Close the capture boundary before draining so teardown cannot enqueue new jobs.
      finished = true;
      // Removing the route first would release intercepted requests before capture finishes.
      await Promise.all(pending);
      for (const [page, handler] of routes)
        if (!page.isClosed()) await page.unroute(pattern, handler);
      routes.clear();
      return { failures, count };
    },
  };
}
