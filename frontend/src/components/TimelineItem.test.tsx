import { render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { TimelineEntry } from "../types/api";
import { TimelineItem } from "./TimelineItem";

const clipEntry: TimelineEntry = {
  kind: "clip",
  id: 1,
  url: "http://example.com/audio.mp3",
  text: "クリップテキスト",
  speakerName: "話者A",
  styleName: "スタイル1",
  timestamp: 1700000000000,
};

const defaultProps = {
  entry: clipEntry,
  playing: false,
  showSpeakerName: false,
  showStyleName: false,
  showTimestamp: false,
};

describe("TimelineItem - clip", () => {
  it("テキストが描画される", () => {
    render(<TimelineItem {...defaultProps} />);
    expect(screen.getByText("クリップテキスト")).toBeInTheDocument();
  });

  it("playing=false のとき再生アイコンが表示されない", () => {
    render(<TimelineItem {...defaultProps} playing={false} />);
    expect(screen.queryByText("▶")).not.toBeInTheDocument();
  });

  it("playing=true のとき再生アイコンが表示される", () => {
    render(<TimelineItem {...defaultProps} playing={true} />);
    expect(screen.getByText("▶")).toBeInTheDocument();
  });

  it("playing=true のときハイライトクラスが付く", () => {
    render(<TimelineItem {...defaultProps} playing={true} />);
    const item = screen.getByRole("listitem");
    expect(item).toHaveClass("bg-ctp-overlay");
  });

  it("playing=false のときハイライトクラスが付かない", () => {
    render(<TimelineItem {...defaultProps} playing={false} />);
    const item = screen.getByRole("listitem");
    expect(item).not.toHaveClass("bg-ctp-overlay");
  });

  it("highlightPlaying=false のとき playing=true でもハイライトクラスが付かない", () => {
    render(<TimelineItem {...defaultProps} playing={true} highlightPlaying={false} />);
    const item = screen.getByRole("listitem");
    expect(item).not.toHaveClass("bg-ctp-overlay");
  });

  it("highlightPlaying=false のとき playing=true でも再生アイコンが表示されない", () => {
    render(<TimelineItem {...defaultProps} playing={true} highlightPlaying={false} />);
    expect(screen.queryByText("▶")).not.toBeInTheDocument();
  });

  it("highlightPlaying=true（デフォルト）のとき playing=true でハイライトクラスが付く", () => {
    render(<TimelineItem {...defaultProps} playing={true} />);
    const item = screen.getByRole("listitem");
    expect(item).toHaveClass("bg-ctp-overlay");
  });
});

describe("TimelineItem - clip meta toggles", () => {
  it("showSpeakerName=true のとき話者名が表示される", () => {
    render(<TimelineItem {...defaultProps} showSpeakerName={true} />);
    expect(screen.getByText("話者A")).toBeInTheDocument();
  });

  it("showSpeakerName=false のとき話者名が表示されない", () => {
    render(<TimelineItem {...defaultProps} showSpeakerName={false} />);
    expect(screen.queryByText("話者A")).not.toBeInTheDocument();
  });

  it("showStyleName=true のときスタイル名が表示される", () => {
    render(<TimelineItem {...defaultProps} showStyleName={true} />);
    expect(screen.getByText("[スタイル1]")).toBeInTheDocument();
  });

  it("showStyleName=false のときスタイル名が表示されない", () => {
    render(<TimelineItem {...defaultProps} showStyleName={false} />);
    expect(screen.queryByText("[スタイル1]")).not.toBeInTheDocument();
  });

  it("showTimestamp=true のとき時刻が表示される", () => {
    render(<TimelineItem {...defaultProps} showTimestamp={true} />);
    const item = screen.getByRole("listitem");
    expect(item).toHaveTextContent(/\d{2}:\d{2}:\d{2}/);
  });

  it("showTimestamp=false のとき時刻が表示されない", () => {
    render(<TimelineItem {...defaultProps} showTimestamp={false} />);
    const item = screen.getByRole("listitem");
    expect(item).not.toHaveTextContent(/\d{2}:\d{2}:\d{2}/);
  });
});

describe("TimelineItem - error", () => {
  it("category=synthesis のとき合成エラーラベルが表示される", () => {
    const entry: TimelineEntry = {
      kind: "error",
      id: 2,
      category: "synthesis",
      message: "合成に失敗しました",
      timestamp: 1700000000000,
    };
    render(<TimelineItem {...defaultProps} entry={entry} />);
    expect(screen.getByText("合成エラー")).toBeInTheDocument();
    expect(screen.getByText("合成に失敗しました")).toBeInTheDocument();
  });

  it("category=file のときファイルエラーラベルが表示される", () => {
    const entry: TimelineEntry = {
      kind: "error",
      id: 3,
      category: "file",
      message: "ファイルが見つかりません",
      timestamp: 1700000000000,
    };
    render(<TimelineItem {...defaultProps} entry={entry} />);
    expect(screen.getByText("ファイルエラー")).toBeInTheDocument();
  });

  it("category=connection のとき接続エラーラベルが表示される", () => {
    const entry: TimelineEntry = {
      kind: "error",
      id: 4,
      category: "connection",
      message: "接続に失敗しました",
      timestamp: 1700000000000,
    };
    render(<TimelineItem {...defaultProps} entry={entry} />);
    expect(screen.getByText("接続エラー")).toBeInTheDocument();
  });

  it("error エントリの showSpeakerName=true のとき話者名が表示される", () => {
    const entry: TimelineEntry = {
      kind: "error",
      id: 5,
      category: "synthesis",
      message: "エラー",
      timestamp: 1700000000000,
      text: "エラー時のテキスト",
      speakerName: "話者B",
    };
    render(
      <TimelineItem {...defaultProps} entry={entry} showSpeakerName={true} />,
    );
    expect(screen.getByText("話者B")).toBeInTheDocument();
  });
});

describe("TimelineItem - scrollIntoView", () => {
  let scrollIntoViewMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    scrollIntoViewMock = vi.fn();
    HTMLElement.prototype.scrollIntoView = scrollIntoViewMock;
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("playing が false→true になると scrollIntoView が呼ばれる", () => {
    const { rerender } = render(
      <TimelineItem {...defaultProps} playing={false} />,
    );
    expect(scrollIntoViewMock).not.toHaveBeenCalled();

    rerender(<TimelineItem {...defaultProps} playing={true} />);
    expect(scrollIntoViewMock).toHaveBeenCalledTimes(1);
    expect(scrollIntoViewMock).toHaveBeenCalledWith({ block: "nearest" });
  });

  it("playing が true→true で再描画しても scrollIntoView は呼ばれない", () => {
    const { rerender } = render(
      <TimelineItem {...defaultProps} playing={true} />,
    );
    scrollIntoViewMock.mockClear();

    rerender(<TimelineItem {...defaultProps} playing={true} />);
    expect(scrollIntoViewMock).not.toHaveBeenCalled();
  });

  it("マウント時点で playing=true のとき scrollIntoView が呼ばれる", () => {
    render(<TimelineItem {...defaultProps} playing={true} />);
    expect(scrollIntoViewMock).toHaveBeenCalledTimes(1);
    expect(scrollIntoViewMock).toHaveBeenCalledWith({ block: "nearest" });
  });

  it("マウント時点で playing=false のとき scrollIntoView は呼ばれない", () => {
    render(<TimelineItem {...defaultProps} playing={false} />);
    expect(scrollIntoViewMock).not.toHaveBeenCalled();
  });
});
