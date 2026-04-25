import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { StatusBadge } from "./StatusBadge";

describe("StatusBadge", () => {
  it("connected=true のときに接続中ラベルが表示される", () => {
    render(<StatusBadge connected={true} />);
    expect(screen.getByText("● 接続中")).toBeInTheDocument();
  });

  it("connected=false のときに切断ラベルが表示される", () => {
    render(<StatusBadge connected={false} />);
    expect(screen.getByText("● 切断")).toBeInTheDocument();
  });

  it("connected=true のときに緑色の背景クラスが適用される", () => {
    render(<StatusBadge connected={true} />);
    const badge = screen.getByText("● 接続中");
    expect(badge).toHaveClass("bg-ctp-green");
  });

  it("connected=false のときに赤色の背景クラスが適用される", () => {
    render(<StatusBadge connected={false} />);
    const badge = screen.getByText("● 切断");
    expect(badge).toHaveClass("bg-ctp-red");
  });
});
