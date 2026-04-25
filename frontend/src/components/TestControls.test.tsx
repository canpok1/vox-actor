import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { Speaker } from "../types/api";
import { TestControls } from "./TestControls";

describe("TestControls", () => {
  const mockSpeakers: Speaker[] = [
    { id: 1, speakerName: "Taro", styleName: "happy" },
    { id: 2, speakerName: "Hanako", styleName: "sad" },
    { id: 3, speakerName: "Jiro", styleName: "" },
  ];

  const mockCallbacks = {
    onSpeakerChange: vi.fn(),
    onPlay: vi.fn(),
  };

  it("speakers が空のときはオプションが表示されない", () => {
    render(
      <TestControls
        speakers={[]}
        selectedSpeakerId=""
        onSpeakerChange={mockCallbacks.onSpeakerChange}
        onPlay={mockCallbacks.onPlay}
        error=""
      />,
    );
    const options = screen.queryAllByRole("option");
    expect(options).toHaveLength(0);
  });

  it("speakers の各要素が option として描画される", () => {
    render(
      <TestControls
        speakers={mockSpeakers}
        selectedSpeakerId="1"
        onSpeakerChange={mockCallbacks.onSpeakerChange}
        onPlay={mockCallbacks.onPlay}
        error=""
      />,
    );
    const options = screen.getAllByRole("option");
    expect(options).toHaveLength(3);
    expect(options[0]).toHaveTextContent("Taro(happy)");
    expect(options[1]).toHaveTextContent("Hanako(sad)");
    expect(options[2]).toHaveTextContent("Jiro");
  });

  it("styleName が空のときはスタイル部分を表示しない", () => {
    render(
      <TestControls
        speakers={mockSpeakers}
        selectedSpeakerId="1"
        onSpeakerChange={mockCallbacks.onSpeakerChange}
        onPlay={mockCallbacks.onPlay}
        error=""
      />,
    );
    const jiroOption = screen.getByRole("option", { name: /^Jiro$/ });
    expect(jiroOption.textContent).not.toContain("(");
  });

  it("選択変更で onSpeakerChange が呼ばれる", async () => {
    const user = userEvent.setup();

    render(
      <TestControls
        speakers={mockSpeakers}
        selectedSpeakerId="1"
        onSpeakerChange={mockCallbacks.onSpeakerChange}
        onPlay={mockCallbacks.onPlay}
        error=""
      />,
    );

    const select = screen.getByRole("combobox") as HTMLSelectElement;
    await user.selectOptions(select, "2");

    expect(mockCallbacks.onSpeakerChange).toHaveBeenCalledWith("2");
  });

  it("テスト再生ボタンクリックで onPlay が呼ばれる", async () => {
    const user = userEvent.setup();

    render(
      <TestControls
        speakers={mockSpeakers}
        selectedSpeakerId="1"
        onSpeakerChange={mockCallbacks.onSpeakerChange}
        onPlay={mockCallbacks.onPlay}
        error=""
      />,
    );

    const button = screen.getByRole("button", { name: /テスト再生/ });
    await user.click(button);

    expect(mockCallbacks.onPlay).toHaveBeenCalledTimes(1);
  });

  it("エラーメッセージが渡されたときに表示される", () => {
    const errorMessage = "再生に失敗しました";
    render(
      <TestControls
        speakers={mockSpeakers}
        selectedSpeakerId="1"
        onSpeakerChange={mockCallbacks.onSpeakerChange}
        onPlay={mockCallbacks.onPlay}
        error={errorMessage}
      />,
    );

    const statusElement = screen.getByRole("status");
    expect(statusElement).toHaveTextContent(errorMessage);
  });

  it("エラーメッセージが空のときは何も表示されない", () => {
    render(
      <TestControls
        speakers={mockSpeakers}
        selectedSpeakerId="1"
        onSpeakerChange={mockCallbacks.onSpeakerChange}
        onPlay={mockCallbacks.onPlay}
        error=""
      />,
    );

    const statusElement = screen.getByRole("status");
    expect(statusElement).toHaveTextContent("");
  });
});
