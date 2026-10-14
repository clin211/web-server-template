import { expect, test, type Page } from "@playwright/test";

const session = {
  user: {
    userID: "user-1",
    username: "ada",
    nickname: "Ada Lovelace",
    email: "ada@example.test",
    phone: "13800138000",
    postCount: "0",
    createdAt: "1",
    updatedAt: "1",
    status: 0,
    gender: 0,
    avatar: "",
    description: "",
    lastLoginAt: "1",
  },
};

async function mockCurrentSession(page: Page) {
  await page.route("**/api/auth/session", (route) =>
    route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(session) }),
  );
}

test("redirects an unauthenticated console request to login", async ({ page }) => {
  await page.goto("/");
  await expect(page).toHaveURL(/\/login(?:\?|$)/);
  await expect(page.getByRole("heading", { name: "账号登录" })).toBeVisible();
});

test("renders the public login and registration forms", async ({ page }) => {
  await page.goto("/login");
  await expect(page).toHaveTitle("登录 | Gin Console");
  await expect(page.getByLabel("账号", { exact: true })).toBeVisible();
  await expect(page.getByLabel("密码")).toBeVisible();
  await expect(page.getByRole("button", { name: "登录" })).toBeVisible();

  await page.getByRole("link", { name: "创建账号" }).click();
  await expect(page).toHaveURL(/\/register$/);
  await expect(page.getByRole("heading", { name: "创建账号" })).toBeVisible();
  await expect(page.getByLabel("用户名")).toBeVisible();
});

test("surfaces a session check failure on the login route", async ({ page }) => {
  await page.route("**/api/auth/session", (route) =>
    route.fulfill({
      status: 502,
      contentType: "application/json",
      headers: { "x-request-id": "login-session-request-123" },
      body: JSON.stringify({ error: { code: "upstream", message: "请求暂时无法完成" } }),
    }),
  );

  await page.goto("/login");
  await expect(page.getByRole("heading", { name: "页面暂时无法加载" })).toBeVisible();
  await expect(page.getByText("login-session-request-123")).toBeVisible();
});

test("supports keyboard access through the login form", async ({ page }) => {
  await page.goto("/login");
  await page.getByLabel("密码").focus();
  await page.keyboard.press("Shift+Tab");
  await expect(page.getByLabel("账号", { exact: true })).toBeFocused();
  await page.keyboard.press("Tab");
  await expect(page.getByLabel("密码")).toBeFocused();
  await page.keyboard.press("Tab");
  await expect(page.getByRole("button", { name: "显示输入内容" })).toBeFocused();
  await page.keyboard.press("Tab");
  await expect(page.getByRole("button", { name: "登录" })).toBeFocused();
});

test("shows the request ID when a protected route cannot load", async ({ page }) => {
  await page.route("**/api/auth/session", (route) =>
    route.fulfill({
      status: 502,
      contentType: "application/json",
      headers: { "x-request-id": "e2e-request-123" },
      body: JSON.stringify({ error: { code: "upstream", message: "请求暂时无法完成" } }),
    }),
  );

  await page.goto("/");
  await expect(page.getByRole("heading", { name: "页面暂时无法加载" })).toBeVisible();
  await expect(page.getByText("e2e-request-123")).toBeVisible();
});

test("renders the authenticated account workflow", async ({ page }) => {
  await mockCurrentSession(page);
  await page.goto("/");
  await expect(page.getByTestId("console-shell")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Ada Lovelace" })).toBeVisible();

  await page.getByRole("link", { name: "个人资料", exact: true }).first().click();
  await expect(page).toHaveURL(/\/settings\/profile$/);
  await expect(page.getByLabel("用户名")).toHaveValue("ada");

  await page.goto("/settings/security");
  await expect(page.getByRole("heading", { name: "安全设置" })).toBeVisible();
  await expect(page.getByLabel("当前密码")).toBeVisible();

  await page.goto("/settings/account");
  await expect(page.getByRole("heading", { name: "账号管理" })).toBeVisible();
  await expect(page.getByRole("button", { name: "永久注销账号" })).toBeDisabled();
});

for (const viewport of [
  { width: 375, height: 812 },
  { width: 768, height: 1024 },
  { width: 1280, height: 800 },
  { width: 1440, height: 900 },
]) {
  test(`keeps login content within ${viewport.width}px viewport`, async ({ page }) => {
    await page.setViewportSize(viewport);
    await page.route("**/api/auth/session", (route) =>
      route.fulfill({ status: 401, contentType: "application/json", body: "{}" }),
    );
    await page.goto("/login");
    await expect(page.getByRole("heading", { name: "账号登录" })).toBeVisible();
    await expect(page.getByText("受保护的企业工作区，仅限授权成员访问")).toBeVisible();
    const dimensions = await page.evaluate(() => ({
      scrollWidth: document.documentElement.scrollWidth,
      clientWidth: document.documentElement.clientWidth,
    }));
    expect(dimensions.scrollWidth).toBeLessThanOrEqual(dimensions.clientWidth);

    const brandHeading = page.getByRole("heading", { name: "Gin Enterprise Console" });
    if (viewport.width >= 1024) await expect(brandHeading).toBeVisible();
    else await expect(brandHeading).toBeHidden();
  });

  test(`keeps console content within ${viewport.width}px viewport`, async ({ page }) => {
    await page.setViewportSize(viewport);
    await mockCurrentSession(page);
    await page.goto("/");
    await expect(page.getByTestId("console-shell")).toBeVisible();
    const dimensions = await page.evaluate(() => ({
      scrollWidth: document.documentElement.scrollWidth,
      clientWidth: document.documentElement.clientWidth,
    }));
    expect(dimensions.scrollWidth).toBeLessThanOrEqual(dimensions.clientWidth);
  });
}

test.describe("reduced motion", () => {
  test.use({ reducedMotion: "reduce" });

  test("keeps the login page stable", async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 812 });
    await page.goto("/login");
    await expect(page.getByRole("heading", { name: "账号登录" })).toBeVisible();
    await expect(page.getByRole("button", { name: "登录" })).toBeVisible();
  });
});
