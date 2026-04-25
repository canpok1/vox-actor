import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { VolumeControls } from "./VolumeControls";

describe("VolumeControls", () => {
  it("volume 値がスライダーに反映される", () => {
    render(
      <VolumeControls
        volume={50}
        muted={false}
        onVolumeChange={() => {}}
        onMuteChange={() => {}}
      />,
    );
    const slider = screen.getByRole("slider");
    expect(slider).toHaveValue("50");
  });

  it("スライダー変更で onVolumeChange が数値で呼ばれる", () => {
    const onVolumeChange = vi.fn();
    render(
      <VolumeControls
        volume={50}
        muted={false}
        onVolumeChange={onVolumeChange}
        onMuteChange={() => {}}
      />,
    );

    const slider = screen.getByRole("slider") as HTMLInputElement;
    fireEvent.change(slider, { target: { value: "75" } });

    expect(onVolumeChange).toHaveBeenCalledWith(75);
  });

  it("muted=true のときにチェックボックスが checked 状態になる", () => {
    render(
      <VolumeControls
        volume={50}
        muted={true}
        onVolumeChange={() => {}}
        onMuteChange={() => {}}
      />,
    );
    const checkbox = screen.getByRole("checkbox");
    expect(checkbox).toBeChecked();
  });

  it("muted=false のときにチェックボックスが checked 状態でない", () => {
    render(
      <VolumeControls
        volume={50}
        muted={false}
        onVolumeChange={() => {}}
        onMuteChange={() => {}}
      />,
    );
    const checkbox = screen.getByRole("checkbox");
    expect(checkbox).not.toBeChecked();
  });

  it("ミュートチェックボックス操作で onMuteChange が呼ばれる", async () => {
    const user = userEvent.setup();
    const onMuteChange = vi.fn();
    render(
      <VolumeControls
        volume={50}
        muted={false}
        onVolumeChange={() => {}}
        onMuteChange={onMuteChange}
      />,
    );

    const checkbox = screen.getByRole("checkbox");
    await user.click(checkbox);

    expect(onMuteChange).toHaveBeenCalledWith(true);
  });

  it("muted=true または volume=0 のときにアイコンが🔇になる", () => {
    const { rerender } = render(
      <VolumeControls
        volume={50}
        muted={true}
        onVolumeChange={() => {}}
        onMuteChange={() => {}}
      />,
    );
    expect(screen.getByText("🔇")).toBeInTheDocument();

    rerender(
      <VolumeControls
        volume={0}
        muted={false}
        onVolumeChange={() => {}}
        onMuteChange={() => {}}
      />,
    );
    expect(screen.getByText("🔇")).toBeInTheDocument();
  });

  it("muted=false かつ volume > 0 のときにアイコンが🔊になる", () => {
    render(
      <VolumeControls
        volume={50}
        muted={false}
        onVolumeChange={() => {}}
        onMuteChange={() => {}}
      />,
    );
    expect(screen.getByText("🔊")).toBeInTheDocument();
  });
});
