import assert from "node:assert/strict";
import test from "node:test";
import { contractObserver } from "./e2e/contracts.ts";

function fixture(body = { status: "operational" }, path = "/api/v1/status") {
  let handler;
  const events = [];
  const response = {
    url: () => `http://localhost${path}`,
    headers: () => ({ "content-type": "application/json" }),
    status: () => 200,
    json: async () => body,
    dispose: async () => events.push("dispose"),
  };
  const route = {
    request: () => ({ url: response.url, method: () => "GET" }),
    fetch: async (options) => {
      assert.deepEqual(options, { maxRetries: 0, maxRedirects: 0 });
      events.push("fetch");
      return response;
    },
    fulfill: async (options) => {
      assert.deepEqual(options, { response });
      events.push("fulfill");
    },
    abort: async () => events.push("abort"),
    fallback: async () => events.push("fallback"),
  };
  const page = {
    route: async (_, fn) => { handler = fn; },
    unroute: async (_, fn) => { assert.equal(fn, handler); handler = undefined; },
    isClosed: () => false,
  };
  return { page, route, response, events, run: () => handler(route), installed: () => !!handler };
}

test("observer forwards a real response once without redirect, retry or payload override", async () => {
  const observer = await contractObserver();
  const f = fixture();
  await observer.observe(f.page);
  await f.run();
  assert.deepEqual(await observer.finish(), { failures: [], count: 1 });
  assert.deepEqual(f.events, ["fetch", "fulfill", "dispose"]);
});

test("observer closes the capture boundary before waiting for an in-flight response", async () => {
  const observer = await contractObserver();
  const f = fixture();
  await observer.observe(f.page);
  let release;
  f.response.json = () => new Promise((resolve) => { release = resolve; });
  const running = f.run();
  await new Promise((resolve) => setImmediate(resolve));
  const finished = observer.finish();
  assert.equal(f.installed(), true);
  await f.run();
  assert.deepEqual(f.events, ["fetch", "fallback"]);
  release({ status: "operational" });
  await running;
  assert.deepEqual(await finished, { failures: [], count: 1 });
  assert.equal(f.installed(), false);
});

test("observer retains asynchronous capture failures without replaying the request", async () => {
  const observer = await contractObserver();
  const f = fixture();
  await observer.observe(f.page);
  f.route.fetch = async () => { f.events.push("fetch"); throw new Error("synthetic reset"); };
  await f.run();
  const result = await observer.finish();
  assert.equal(result.count, 0);
  assert.deepEqual(result.failures, ["API response capture failed"]);
  assert.deepEqual(f.events, ["fetch", "abort"]);
});

test("observer still rejects malformed JSON, undocumented statuses and invalid contract bodies", async () => {
  for (const mode of ["malformed", "undocumented", "invalid"]) {
    const observer = await contractObserver();
    const f = fixture({ status: "invalid" });
    await observer.observe(f.page);
    if (mode === "malformed") f.response.json = async () => { throw new Error("malformed JSON"); };
    if (mode === "undocumented") f.response.status = () => 299;
    await f.run();
    const result = await observer.finish();
    assert.equal(result.failures.length, 1);
    assert.equal(result.count, mode === "invalid" ? 1 : 0);
    assert.ok(f.events.includes("dispose"));
  }
});

test("observer leaves downloads and non-API resources on their native transport", async () => {
  for (const path of ["/_next/static/chunk.js", "/api/v1/artifacts/example/content", "/api/v1/uploads/example"]) {
    const observer = await contractObserver();
    const f = fixture(undefined, path);
    await observer.observe(f.page);
    await f.run();
    assert.deepEqual(await observer.finish(), { failures: [], count: 0 });
    assert.deepEqual(f.events, ["fallback"]);
  }
});

test("non-JSON responses retain upstream bytes and are not treated as JSON contracts", async () => {
  const observer = await contractObserver();
  const f = fixture();
  f.response.headers = () => ({ "content-type": "text/plain" });
  f.response.json = async () => { throw new Error("must not read JSON"); };
  await observer.observe(f.page);
  await f.run();
  assert.deepEqual(await observer.finish(), { failures: [], count: 0 });
  assert.deepEqual(f.events, ["fetch", "fulfill", "dispose"]);
});
