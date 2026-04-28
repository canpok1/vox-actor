import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { CharacterEntry, TimelineEntry } from "../types/api";
import { StreamPanel } from "./StreamPanel";

const mockEntries: TimelineEntry[] = [
  {
    kind: "clip",
    id: 1,
    url: "http://example.com/audio.mp3",
    text: "クリップテキスト",
    speakerName: "話者A",
    styleName: "ノーマル",
    timestamp: 1700000000000,
  },
];

const mockCharacters: CharacterEntry[] = [
  {
    speakerName: "話者A",
    styleName: "ノーマル",
    mouthClosed: "closed.png",
    mouthOpen: "open.png",
  },
];

const defaultProps = {
  entries: [],
  playingClipId: null,
  historySize: 50,
  historySizeOptions: [10, 50, 100] as const,
  onHistorySizeChange: vi.fn(),
  showSpeakerName: false,
  showStyleName: false,
  showTimestamp: false,
  onShowSpeakerNameChange: vi.fn(),
  onShowStyleNameChange: vi.fn(),
  onShowTimestampChange: vi.fn(),
  characters: [],
  charactersEnabled: false,
  showCharacters: false,
  onShowCharactersChange: vi.fn(),
  volume: 0,
};

describe("StreamPanel", () => {
  it("hidden=false のときにパネルが表示される", () => {
    render(<StreamPanel {...defaultProps} hidden={false} />);
    const panel = screen.getByRole("tabpanel", { hidden: false });
    expect(panel).toBeInTheDocument();
  });

  it("hidden=true のときにパネルが非表示になる", () => {
    render(<StreamPanel {...defaultProps} hidden={true} />);
    const panel = screen.queryByRole("tabpanel", { hidden: false });
    expect(panel).not.toBeInTheDocument();
  });

  it("TimelineControls に historySize が渡される", () => {
    render(<StreamPanel {...defaultProps} hidden={false} historySize={50} />);
    const select = screen.getByRole("combobox");
    expect(select).toHaveValue("50");
  });

  it("TimelineControls に historySizeOptions が渡される", () => {
    render(
      <StreamPanel
        {...defaultProps}
        hidden={false}
        historySizeOptions={[10, 50, 100]}
      />,
    );
    const options = screen.getAllByRole("option");
    expect(options).toHaveLength(3);
    expect(options[0]).toHaveTextContent("10");
    expect(options[1]).toHaveTextContent("50");
    expect(options[2]).toHaveTextContent("100");
  });

  it("Timeline に entries が渡される", () => {
    render(
      <StreamPanel {...defaultProps} hidden={false} entries={mockEntries} />,
    );
    expect(screen.getByText("クリップテキスト")).toBeInTheDocument();
  });

  it("Timeline に playingClipId が渡され再生中エントリが強調される", () => {
    render(
      <StreamPanel
        {...defaultProps}
        hidden={false}
        entries={mockEntries}
        playingClipId={1}
      />,
    );
    expect(screen.getByText("▶")).toBeInTheDocument();
  });

  it("charactersEnabled=false のときキャラ画像チェックボックスが表示されない", () => {
    render(
      <StreamPanel
        {...defaultProps}
        hidden={false}
        charactersEnabled={false}
      />,
    );
    expect(
      screen.queryByLabelText("キャラ画像を表示"),
    ).not.toBeInTheDocument();
  });

  it("charactersEnabled=true のときキャラ画像チェックボックスが表示される", () => {
    render(
      <StreamPanel
        {...defaultProps}
        hidden={false}
        charactersEnabled={true}
        showCharacters={false}
      />,
    );
    expect(screen.getByLabelText("キャラ画像を表示")).toBeInTheDocument();
  });

  it("showCharacters=false のとき履歴モードで全エントリが表示される", () => {
    const entries: TimelineEntry[] = [
      {
        kind: "clip",
        id: 1,
        url: "http://example.com/1.mp3",
        text: "古いテキスト",
        speakerName: "話者A",
        styleName: "ノーマル",
        timestamp: 1700000000000,
      },
      {
        kind: "clip",
        id: 2,
        url: "http://example.com/2.mp3",
        text: "新しいテキスト",
        speakerName: "話者A",
        styleName: "ノーマル",
        timestamp: 1700000001000,
      },
    ];
    render(
      <StreamPanel
        {...defaultProps}
        hidden={false}
        entries={entries}
        showCharacters={false}
      />,
    );
    expect(screen.getByText("古いテキスト")).toBeInTheDocument();
    expect(screen.getByText("新しいテキスト")).toBeInTheDocument();
  });

  it("キャラモード時は Timeline が latestOnly で最新 1 件のみ表示される", () => {
    const entries: TimelineEntry[] = [
      {
        kind: "clip",
        id: 1,
        url: "http://example.com/1.mp3",
        text: "古いテキスト",
        speakerName: "話者A",
        styleName: "ノーマル",
        timestamp: 1700000000000,
      },
      {
        kind: "clip",
        id: 2,
        url: "http://example.com/2.mp3",
        text: "新しいテキスト",
        speakerName: "話者A",
        styleName: "ノーマル",
        timestamp: 1700000001000,
      },
    ];
    // showCharacters=true かつ characters が存在し playingClipId が設定されていると
    // useCharacterStage がスロットを返し、キャラモードになる
    render(
      <StreamPanel
        {...defaultProps}
        hidden={false}
        entries={entries}
        characters={mockCharacters}
        charactersEnabled={true}
        showCharacters={true}
        playingClipId={2}
      />,
    );
    expect(screen.queryByText("古いテキスト")).not.toBeInTheDocument();
    expect(screen.getByText("新しいテキスト")).toBeInTheDocument();
  });
});
