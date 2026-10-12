import { createServer, type ServerResponse } from "node:http";
import { test, expect } from "@playwright/test";
import { contractObserver } from "./contracts";

test("contract capture survives navigation while a real HTTP response is in flight", async ({ page }) => {
  let releaseResponse!: ServerResponse;
  let markRequested!: () => void;
  const requested = new Promise<void>((resolve) => { markRequested = resolve; });
  let requests = 0;
  // A private HTTP fixture controls the exact race; it does not mock the application API.
  const server = createServer((request, response) => {
    if (request.url === "/api/v1/status") {
      requests++;
      releaseResponse = response;
      markRequested();
      return;
    }
    response.writeHead(200, { "Content-Type": "text/html" });
    response.end("<!doctype html><html><body>Contract observer fixture</body></html>");
  });
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  const address = server.address();
  if (!address || typeof address === "string") throw new Error("Missing fixture address");
  const origin = `http://127.0.0.1:${address.port}`;
  const observer = await contractObserver();
  await observer.observe(page);
  try {
    await page.goto(origin);
    await page.evaluate(() => { void fetch("/api/v1/status").catch(() => undefined); });
    await requested;
    await page.goto(origin + "/after-navigation");
    // Only finish the upstream response after the old document is gone.
    releaseResponse.writeHead(200, { "Content-Type": "application/json" });
    releaseResponse.end(JSON.stringify({ status: "operational" }));
    expect(await observer.finish()).toEqual({ failures: [], count: 1 });
    expect(requests).toBe(1);
  } finally {
    server.closeAllConnections();
    await new Promise<void>((resolve, reject) => server.close((error) => error ? reject(error) : resolve()));
  }
});
