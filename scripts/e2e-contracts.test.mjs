import assert from "node:assert/strict";
import { EventEmitter } from "node:events";
import test from "node:test";
import { contractObserver } from "./e2e/contracts.ts";

function request(body = { status: "operational" }) {
  return {
    url: () => "http://localhost/api/v1/status",
    response: async () => ({
      url: () => "http://localhost/api/v1/status",
      headers: () => ({ "content-type": "application/json" }),
      request: () => ({ method: () => "GET" }),
      status: () => 200,
      json: async () => body,
    }),
  };
}

test("observer detaches before waiting for in-flight responses", async () => {
  const observer = await contractObserver();
  const page = new EventEmitter();
  observer.observe(page);
  let release;
  const held = new Promise((resolve) => {
    release = resolve;
  });
  const pending = request();
  const response = await pending.response();
  pending.response = () => held;
  page.emit("requestfinished", pending);
  const finished = observer.finish();
  release(response);
  assert.equal(page.listenerCount("requestfinished"), 0);
  page.emit("requestfinished", request({ status: "invalid" }));
  assert.deepEqual(await finished, { failures: [], count: 1 });
});

test("observer retains asynchronous capture failures without unhandled rejections", async () => {
  const observer = await contractObserver();
  const page = new EventEmitter();
  observer.observe(page);
  const broken = request();
  broken.response = async () => {
    throw new Error("synthetic context closed");
  };
  page.emit("requestfinished", broken);
  await new Promise((resolve) => setImmediate(resolve));
  const result = await observer.finish();
  assert.equal(result.count, 0);
  assert.equal(result.failures.length, 1);
  assert.match(result.failures[0], /capture failed/);
});

test("observer still rejects invalid JSON and contract bodies", async () => {
  const observer = await contractObserver();
  const page = new EventEmitter();
  observer.observe(page);
  page.emit("requestfinished", request({ status: "invalid" }));
  const broken = request();
  const response = await broken.response();
  response.json = async () => {
    throw new Error("malformed JSON");
  };
  broken.response = async () => response;
  page.emit("requestfinished", broken);
  const result = await observer.finish();
  assert.equal(result.count, 1);
  assert.equal(result.failures.length, 2);
});

test("observer ignores non-API resources before asking the browser for their response", async () => {
  const observer = await contractObserver();
  const page = new EventEmitter();
  observer.observe(page);
  let reads = 0;
  for (const path of [
    "/_next/static/chunk.js",
    "/api/v1/artifacts/example/content",
  ]) {
    page.emit("requestfinished", {
      url: () => `http://localhost${path}`,
      response: async () => {
        reads++;
        return null;
      },
    });
  }
  assert.deepEqual(await observer.finish(), { failures: [], count: 0 });
  assert.equal(reads, 0);
});
