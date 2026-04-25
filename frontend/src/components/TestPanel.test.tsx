import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { Speaker } from "../types/api";
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

  it("hidden=false で表示される", () => {
    render(
      <TestPanel
        hidden={false}
        speakers={mockSpeakers}
        selectedSpeakerId="1"
        onSpeakerChange={mockCallbacks.onSpeakerChange}
        onPlay={mockCallbacks.onPlay}
        error=""
      />,
    );
    const panel = screen.getByRole("tabpanel", { hidden: false });
    expect(panel).toBeInTheDocument();
  });

  it("hidden=true で非表示になる", () => {
    render(
      <TestPanel
        hidden={true}
        speakers={mockSpeakers}
        selectedSpeakerId="1"
        onSpeakerChange={mockCallbacks.onSpeakerChange}
        onPlay={mockCallbacks.onPlay}
        error=""
      />,
    );
    const panel = screen.queryByRole("tabpanel", { hidden: false });
    expect(panel).not.toBeInTheDocument();
  });

  it("TestControls に正しい props が渡される", () => {
    render(
      <TestPanel
        hidden={false}
        speakers={mockSpeakers}
        selectedSpeakerId="2"
        onSpeakerChange={mockCallbacks.onSpeakerChange}
        onPlay={mockCallbacks.onPlay}
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
});
