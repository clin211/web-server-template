import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { StatusPanel } from "./StatusPanel";

describe("StatusPanel", () => {
  it("shows a request ID only when one is available", () => {
    const { rerender } = render(
      <StatusPanel title="请求失败" requestId="request-123">
        请稍后重试。
      </StatusPanel>,
    );

    expect(screen.getByText("request-123")).toBeVisible();

    rerender(<StatusPanel title="请求失败">请稍后重试。</StatusPanel>);
    expect(screen.queryByText("request-123")).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { level: 2, name: "请求失败" })).toBeVisible();

    rerender(
      <StatusPanel title="页面失败" headingLevel={1}>
        请稍后重试。
      </StatusPanel>,
    );
    expect(screen.getByRole("heading", { level: 1, name: "页面失败" })).toBeVisible();
  });
});
