import { describe, expect, it } from "vitest";
import { readCookie, serializeCookie } from "./cookies";

describe("cookie helpers", () => {
  it.each([
    ["console-csrf=abc; console-access=token", "console-csrf", "abc"],
    ["console-access=token; console-csrf=a%20b", "console-csrf", "a b"],
  ])("reads %s", (header, name, expected) => {
    expect(
      readCookie(new Request("http://console.test", { headers: { cookie: header } }), name),
    ).toBe(expected);
  });

  it("ignores malformed percent encoding", () => {
    expect(
      readCookie(
        new Request("http://console.test", {
          headers: { cookie: "console-access=%E0%A4%A" },
        }),
        "console-access",
      ),
    ).toBeNull();
  });

  it("rejects an empty cookie value", () => {
    expect(
      readCookie(
        new Request("http://console.test", { headers: { cookie: "console-access=" } }),
        "console-access",
      ),
    ).toBeNull();
  });

  it("serializes secure HttpOnly cookies", () => {
    expect(
      serializeCookie("__Host-console-access", "token", {
        httpOnly: true,
        secure: true,
        sameSite: "Lax",
        maxAge: 120,
      }),
    ).toBe("__Host-console-access=token; Max-Age=120; HttpOnly; Secure; SameSite=Lax; Path=/");
  });

  it("omits optional attributes when they are not requested", () => {
    expect(serializeCookie("console-preference", "compact view", { path: "/settings" })).toBe(
      "console-preference=compact%20view; Path=/settings",
    );
  });
});
