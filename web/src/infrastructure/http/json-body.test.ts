import { describe, expect, it } from "vitest";
import { readJsonBody } from "./json-body";

describe("JSON body reader", () => {
  it("reads a bounded JSON request", async () => {
    const request = new Request("https://console.test/api/write", {
      method: "POST",
      headers: { "content-type": "application/json; charset=utf-8" },
      body: JSON.stringify({ value: "ok" }),
    });
    await expect(readJsonBody(request, 1024)).resolves.toEqual({ value: "ok" });
  });

  it.each([
    ["wrong content type", "text/plain", "{}", 1024, 415],
    ["oversized body", "application/json", JSON.stringify({ value: "too long" }), 5, 413],
    ["invalid JSON", "application/json", "not-json", 1024, 400],
  ])("rejects %s", async (_label, contentType, body, maxBytes, status) => {
    const request = new Request("https://console.test/api/write", {
      method: "POST",
      headers: { "content-type": contentType },
      body,
    });
    await expect(readJsonBody(request, maxBytes)).rejects.toMatchObject({ status });
  });

  it("rejects a declared oversized body before reading it", async () => {
    const request = new Request("https://console.test/api/write", {
      method: "POST",
      headers: { "content-type": "application/json", "content-length": "4096" },
      body: "{}",
    });

    await expect(readJsonBody(request, 1024)).rejects.toMatchObject({ status: 413 });
  });
});
