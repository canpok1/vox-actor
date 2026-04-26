import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { TimelineEntry } from "../types/api";
import { Timeline } from "./Timeline";

const defaultProps = {
  playingClipId: null,
  showSpeakerName: false,
  showStyleName: false,
  showTimestamp: false,
};

const clipEntry = (id: number, text: string): TimelineEntry => ({
  kind: "clip",
  id,
  url: `http://example.com/${id}.mp3`,
  text,
  speakerName: "話者A",
  styleName: "ノーマル",
  timestamp: 1700000000000,
});

describe("Timeline", () => {
  it("entries が空のとき何も描画されない", () => {
    render(<Timeline {...defaultProps} entries={[]} />);
    const list = screen.getByRole("list");
    expect(list).toBeEmptyDOMElement();
  });

  it("entries の数だけリストアイテムが描画される", () => {
    const entries: TimelineEntry[] = [
      clipEntry(1, "テキスト1"),
      clipEntry(2, "テキスト2"),
      clipEntry(3, "テキスト3"),
    ];
    render(<Timeline {...defaultProps} entries={entries} />);
    const items = screen.getAllByRole("listitem");
    expect(items).toHaveLength(3);
  });

  it("entries のテキストが描画される", () => {
    const entries: TimelineEntry[] = [
      clipEntry(1, "最初のテキスト"),
      clipEntry(2, "次のテキスト"),
    ];
    render(<Timeline {...defaultProps} entries={entries} />);
    expect(screen.getByText("最初のテキスト")).toBeInTheDocument();
    expect(screen.getByText("次のテキスト")).toBeInTheDocument();
  });

  it("playingClipId が一致するエントリにのみ再生中アイコンが表示される", () => {
    const entries: TimelineEntry[] = [
      clipEntry(1, "テキスト1"),
      clipEntry(2, "テキスト2"),
    ];
    render(<Timeline {...defaultProps} entries={entries} playingClipId={1} />);
    const playIcons = screen.getAllByText("▶");
    expect(playIcons).toHaveLength(1);
  });

  it("playingClipId が null のとき再生中アイコンが表示されない", () => {
    const entries: TimelineEntry[] = [
      clipEntry(1, "テキスト1"),
      clipEntry(2, "テキスト2"),
    ];
    render(
      <Timeline {...defaultProps} entries={entries} playingClipId={null} />,
    );
    expect(screen.queryByText("▶")).not.toBeInTheDocument();
  });

  it("error エントリも描画される", () => {
    const entries: TimelineEntry[] = [
      {
        kind: "error",
        id: 1,
        category: "synthesis",
        message: "合成に失敗しました",
        timestamp: 1700000000000,
      },
    ];
    render(<Timeline {...defaultProps} entries={entries} />);
    expect(screen.getByText("合成に失敗しました")).toBeInTheDocument();
  });
});
