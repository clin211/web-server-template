import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AuthLayout } from "./AuthLayout";

describe("AuthLayout", () => {
  it("presents the console brand and authentication region", () => {
    render(
      <AuthLayout eyebrow="身份验证" title="账号登录" description="使用企业账号继续访问">
        <form aria-label="登录表单" />
      </AuthLayout>,
    );

    expect(screen.getByRole("heading", { level: 1, name: "Gin Enterprise Console" })).toBeVisible();
    expect(screen.getByRole("region", { name: "Gin Console 品牌信息" })).toBeVisible();
    expect(screen.getByRole("region", { name: "账号登录" })).toContainElement(
      screen.getByRole("form", { name: "登录表单" }),
    );
    expect(screen.getAllByText("Gin Console")).toHaveLength(2);
  });
});
