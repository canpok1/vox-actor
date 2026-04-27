import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { Speaker, CharacterEntry } from "../types/api";
import { TestPanel } from "./TestPanel";

describe("TestPanel", () => {
  const mockSpeakers: Speaker[] = [
    { id: 1, speakerName: "Speaker A", styleName: "style1" },
    { id: 2, speakerName: "Speaker B", styleName: "" },
  ];

  const mockCallbacks = {
    onSpeakerChange: vi.fn(),
    onPlay: vi.fn(),
  };

  const baseProps = {
    speakers: mockSpeakers,
    selectedSpeakerId: "1",
    onSpeakerChange: mockCallbacks.onSpeakerChange,
    onPlay: mockCallbacks.onPlay,
    error: "",
    charactersEnabled: false,
    characters: [] as CharacterEntry[],
    volume: 0,
  };

  it("hidden=false で表示される", () => {
    render(<TestPanel hidden={false} {...baseProps} />);
    const panel = screen.getByRole("tabpanel", { hidden: false });
    expect(panel).toBeInTheDocument();
  });

  it("hidden=true で非表示になる", () => {
    render(<TestPanel hidden={true} {...baseProps} />);
    const panel = screen.queryByRole("tabpanel", { hidden: false });
    expect(panel).not.toBeInTheDocument();
  });

  it("TestControls に正しい props が渡される", () => {
    render(
      <TestPanel
        hidden={false}
        {...baseProps}
        selectedSpeakerId="2"
        error="エラーメッセージ"
      />,
    );
    const select = screen.getByRole("combobox");
    expect(select).toHaveValue("2");

    const speakerOptions = screen.getAllByRole("option");
    expect(speakerOptions).toHaveLength(2);
    expect(speakerOptions[0]).toHaveTextContent("Speaker A(style1)");
    expect(speakerOptions[1]).toHaveTextContent("Speaker B");

    const errorElement = screen.getByRole("status");
    expect(errorElement).toHaveTextContent("エラーメッセージ");
  });

  describe("キャラクター機能", () => {
    const mockCharacters: CharacterEntry[] = [
      {
        speakerName: "Speaker A",
        styleName: "style1",
        mouthClosed: "closed.png",
        mouthOpen: "open.png",
      },
    ];

    it("charactersEnabled=false のときキャラクター表示エリアが表示されない", () => {
      render(<TestPanel hidden={false} {...baseProps} />);
      expect(
        screen.queryByAltText("character mouth closed"),
      ).not.toBeInTheDocument();
    });

    it("charactersEnabled=true かつ一致するキャラがある場合に口パク画像が表示される", () => {
      render(
        <TestPanel
          hidden={false}
          {...baseProps}
          charactersEnabled={true}
          characters={mockCharacters}
          selectedSpeakerId="1"
        />,
      );
      expect(screen.getByAltText("character mouth closed")).toBeInTheDocument();
      expect(screen.getByAltText("character mouth open")).toBeInTheDocument();
    });

    it("charactersEnabled=true かつ一致するキャラがない場合にプレースホルダーが表示される", () => {
      render(
        <TestPanel
          hidden={false}
          {...baseProps}
          charactersEnabled={true}
          characters={mockCharacters}
          selectedSpeakerId="2"
        />,
      );
      expect(
        screen.queryByAltText("character mouth closed"),
      ).not.toBeInTheDocument();
    });
  });
});
