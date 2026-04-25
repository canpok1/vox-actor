import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { TimelineControls } from "./TimelineControls";

const defaultProps = {
  historySize: 50,
  historySizeOptions: [10, 50, 100] as const,
  onHistorySizeChange: vi.fn(),
  showSpeakerName: false,
  showStyleName: false,
  showTimestamp: false,
  onShowSpeakerNameChange: vi.fn(),
  onShowStyleNameChange: vi.fn(),
  onShowTimestampChange: vi.fn(),
};

describe("TimelineControls - 履歴サイズ", () => {
  it("historySize が select の選択値に反映される", () => {
    render(<TimelineControls {...defaultProps} historySize={50} />);
    expect(screen.getByRole("combobox")).toHaveValue("50");
  });

  it("historySizeOptions が option として描画される", () => {
    render(
      <TimelineControls {...defaultProps} historySizeOptions={[10, 50, 100]} />,
    );
    const options = screen.getAllByRole("option");
    expect(options).toHaveLength(3);
    expect(options[0]).toHaveTextContent("10");
    expect(options[1]).toHaveTextContent("50");
    expect(options[2]).toHaveTextContent("100");
  });

  it("select の変更で onHistorySizeChange が数値で呼ばれる", () => {
    const onHistorySizeChange = vi.fn();
    render(
      <TimelineControls
        {...defaultProps}
        onHistorySizeChange={onHistorySizeChange}
      />,
    );
    const select = screen.getByRole("combobox") as HTMLSelectElement;
    fireEvent.change(select, { target: { value: "100" } });
    expect(onHistorySizeChange).toHaveBeenCalledWith(100);
    expect(onHistorySizeChange).toHaveBeenCalledTimes(1);
  });
});

describe("TimelineControls - チェックボックストグル", () => {
  it("showSpeakerName=true のとき話者名チェックボックスが checked になる", () => {
    render(<TimelineControls {...defaultProps} showSpeakerName={true} />);
    expect(screen.getByLabelText("話者名")).toBeChecked();
  });

  it("showSpeakerName=false のとき話者名チェックボックスが unchecked になる", () => {
    render(<TimelineControls {...defaultProps} showSpeakerName={false} />);
    expect(screen.getByLabelText("話者名")).not.toBeChecked();
  });

  it("話者名チェックボックスをクリックすると onShowSpeakerNameChange が呼ばれる", async () => {
    const user = userEvent.setup();
    const onShowSpeakerNameChange = vi.fn();
    render(
      <TimelineControls
        {...defaultProps}
        onShowSpeakerNameChange={onShowSpeakerNameChange}
      />,
    );
    await user.click(screen.getByLabelText("話者名"));
    expect(onShowSpeakerNameChange).toHaveBeenCalledWith(true);
    expect(onShowSpeakerNameChange).toHaveBeenCalledTimes(1);
  });

  it("showStyleName=true のときスタイルチェックボックスが checked になる", () => {
    render(<TimelineControls {...defaultProps} showStyleName={true} />);
    expect(screen.getByLabelText("スタイル")).toBeChecked();
  });

  it("スタイルチェックボックスをクリックすると onShowStyleNameChange が呼ばれる", async () => {
    const user = userEvent.setup();
    const onShowStyleNameChange = vi.fn();
    render(
      <TimelineControls
        {...defaultProps}
        onShowStyleNameChange={onShowStyleNameChange}
      />,
    );
    await user.click(screen.getByLabelText("スタイル"));
    expect(onShowStyleNameChange).toHaveBeenCalledWith(true);
    expect(onShowStyleNameChange).toHaveBeenCalledTimes(1);
  });

  it("showTimestamp=true のとき時刻チェックボックスが checked になる", () => {
    render(<TimelineControls {...defaultProps} showTimestamp={true} />);
    expect(screen.getByLabelText("時刻")).toBeChecked();
  });

  it("時刻チェックボックスをクリックすると onShowTimestampChange が呼ばれる", async () => {
    const user = userEvent.setup();
    const onShowTimestampChange = vi.fn();
    render(
      <TimelineControls
        {...defaultProps}
        onShowTimestampChange={onShowTimestampChange}
      />,
    );
    await user.click(screen.getByLabelText("時刻"));
    expect(onShowTimestampChange).toHaveBeenCalledWith(true);
    expect(onShowTimestampChange).toHaveBeenCalledTimes(1);
  });
});
