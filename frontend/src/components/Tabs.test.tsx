import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { Tabs } from "./Tabs";

describe("Tabs", () => {
  it("active='stream' のときに配信タブにアクティブ表示が付く", () => {
    render(<Tabs active="stream" onChange={() => {}} />);
    const streamTab = screen.getByRole("tab", { name: "配信" });
    expect(streamTab).toHaveAttribute("aria-selected", "true");
  });

  it("active='test' のときに音声テストタブにアクティブ表示が付く", () => {
    render(<Tabs active="test" onChange={() => {}} />);
    const testTab = screen.getByRole("tab", { name: "音声テスト" });
    expect(testTab).toHaveAttribute("aria-selected", "true");
  });

  it("非アクティブタブをクリックすると onChange が対応する TabName で呼ばれる", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<Tabs active="stream" onChange={onChange} />);

    const testTab = screen.getByRole("tab", { name: "音声テスト" });
    await user.click(testTab);

    expect(onChange).toHaveBeenCalledWith("test");
    expect(onChange).toHaveBeenCalledTimes(1);
  });

  it("アクティブタブをクリックしても onChange が呼ばれる", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<Tabs active="stream" onChange={onChange} />);

    const streamTab = screen.getByRole("tab", { name: "配信" });
    await user.click(streamTab);

    expect(onChange).toHaveBeenCalledWith("stream");
  });

  it("hideTest=true のときに音声テストタブが描画されない", () => {
    render(<Tabs active="stream" onChange={() => {}} hideTest={true} />);
    expect(
      screen.queryByRole("tab", { name: "音声テスト" }),
    ).not.toBeInTheDocument();
  });

  it("hideTest=false のときに音声テストタブが描画される", () => {
    render(<Tabs active="stream" onChange={() => {}} hideTest={false} />);
    expect(screen.getByRole("tab", { name: "音声テスト" })).toBeInTheDocument();
  });

  it("hideTest を指定しない場合（デフォルト）に音声テストタブが描画される", () => {
    render(<Tabs active="stream" onChange={() => {}} />);
    expect(screen.getByRole("tab", { name: "音声テスト" })).toBeInTheDocument();
  });
});
