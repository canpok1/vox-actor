import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { SilentBadge } from "./SilentBadge";

describe("SilentBadge", () => {
  it("バッジボタンを常に表示する", () => {
    render(<SilentBadge reason="エンジン停止中" />);
    expect(
      screen.getByRole("button", { name: /無音モード/ }),
    ).toBeInTheDocument();
  });

  it("初期表示では reason の popover を表示しない", () => {
    render(<SilentBadge reason="エンジン停止中" />);
    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();
  });

  it("ホバーで popover を開き reason を表示する", async () => {
    const user = userEvent.setup();
    render(<SilentBadge reason="エンジン停止中" />);
    await user.hover(screen.getByRole("button", { name: /無音モード/ }));
    expect(screen.getByRole("tooltip")).toHaveTextContent("エンジン停止中");
  });

  it("reason が空文字の時は popover を開いても表示しない", async () => {
    const user = userEvent.setup();
    render(<SilentBadge reason="" />);
    await user.hover(screen.getByRole("button", { name: /無音モード/ }));
    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();
  });
});
