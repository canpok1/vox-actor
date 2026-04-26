import { render, screen, act } from "@testing-library/react";
import { describe, expect, it, beforeEach, afterEach, vi } from "vitest";
import { LipSyncImage } from "./LipSyncImage";

describe("LipSyncImage", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.runOnlyPendingTimers();
    vi.useRealTimers();
  });

  it("初期状態では口が閉じている", () => {
    render(
      <LipSyncImage
        mouthClosedUrl="/closed.png"
        mouthOpenUrl="/open.png"
        volume={0}
      />
    );

    const closedImg = screen.getByAltText("character mouth closed");
    const openImg = screen.getByAltText("character mouth open");

    expect(closedImg).toHaveStyle({ display: "block" });
    expect(openImg).toHaveStyle({ display: "none" });
  });

  it("音量が動的閾値を超えると口が開く", () => {
    const { rerender } = render(
      <LipSyncImage
        mouthClosedUrl="/closed.png"
        mouthOpenUrl="/open.png"
        volume={0}
      />
    );

    // 高い音量を渡す（動的閾値より上）
    rerender(
      <LipSyncImage
        mouthClosedUrl="/closed.png"
        mouthOpenUrl="/open.png"
        volume={0.1}
      />
    );

    const closedImg = screen.getByAltText("character mouth closed");
    const openImg = screen.getByAltText("character mouth open");

    expect(closedImg).toHaveStyle({ display: "none" });
    expect(openImg).toHaveStyle({ display: "block" });
  });

  it("音量が動的閾値以下になるとヒステリシス後に口が閉じる", () => {
    const { rerender } = render(
      <LipSyncImage
        mouthClosedUrl="/closed.png"
        mouthOpenUrl="/open.png"
        volume={0.1}
      />
    );

    // 音量を低く
    rerender(
      <LipSyncImage
        mouthClosedUrl="/closed.png"
        mouthOpenUrl="/open.png"
        volume={0}
      />
    );

    const openImg = screen.getByAltText("character mouth open");
    // ヒステリシス前は口開
    expect(openImg).toHaveStyle({ display: "block" });

    // ヒステリシス時間進める（デフォルト 80ms）
    act(() => {
      vi.advanceTimersByTime(80);
    });

    // ヒステリシス後は口閉
    const closedImg = screen.getByAltText("character mouth closed");
    expect(closedImg).toHaveStyle({ display: "block" });
    expect(openImg).toHaveStyle({ display: "none" });
  });

  it("音量が低い場合、静音時は口が閉じたままになる", () => {
    render(
      <LipSyncImage
        mouthClosedUrl="/closed.png"
        mouthOpenUrl="/open.png"
        volume={0}
      />
    );

    // 変化なし
    vi.advanceTimersByTime(100);

    const closedImg = screen.getByAltText("character mouth closed");
    const openImg = screen.getByAltText("character mouth open");

    expect(closedImg).toHaveStyle({ display: "block" });
    expect(openImg).toHaveStyle({ display: "none" });
  });

  it("音量が高い時に低い時に戻る場合、ヒステリシスがリセットされる", () => {
    const { rerender } = render(
      <LipSyncImage
        mouthClosedUrl="/closed.png"
        mouthOpenUrl="/open.png"
        volume={0.1}
      />
    );

    // 音量を低く
    rerender(
      <LipSyncImage
        mouthClosedUrl="/closed.png"
        mouthOpenUrl="/open.png"
        volume={0}
      />
    );

    // 40ms進める（ヒステリシス80msの途中）
    vi.advanceTimersByTime(40);

    // 音量を高く戻す
    rerender(
      <LipSyncImage
        mouthClosedUrl="/closed.png"
        mouthOpenUrl="/open.png"
        volume={0.1}
      />
    );

    // タイマーがクリアされたので、さらに時間を進めても口は開いたまま
    vi.advanceTimersByTime(50);

    const openImg = screen.getByAltText("character mouth open");
    expect(openImg).toHaveStyle({ display: "block" });
  });

  it("カスタム hysteresisMs が機能する", () => {
    const { rerender } = render(
      <LipSyncImage
        mouthClosedUrl="/closed.png"
        mouthOpenUrl="/open.png"
        volume={0.1}
        hysteresisMs={200}
      />
    );

    // 音量を低く
    rerender(
      <LipSyncImage
        mouthClosedUrl="/closed.png"
        mouthOpenUrl="/open.png"
        volume={0}
        hysteresisMs={200}
      />
    );

    // 100ms進める（200ms未満）
    act(() => {
      vi.advanceTimersByTime(100);
    });

    const openImg = screen.getByAltText("character mouth open");
    expect(openImg).toHaveStyle({ display: "block" });

    // さらに 100ms進める（合計 200ms）
    act(() => {
      vi.advanceTimersByTime(100);
    });

    const closedImg = screen.getByAltText("character mouth closed");
    expect(closedImg).toHaveStyle({ display: "block" });
    expect(openImg).toHaveStyle({ display: "none" });
  });

  it("動的閾値により音節間の振幅低下で口が閉じる", () => {
    const { rerender } = render(
      <LipSyncImage
        mouthClosedUrl="/closed.png"
        mouthOpenUrl="/open.png"
        volume={0}
      />
    );

    // 高い音量: 動的閾値が設定される
    rerender(
      <LipSyncImage
        mouthClosedUrl="/closed.png"
        mouthOpenUrl="/open.png"
        volume={0.1}
      />
    );

    const openImg = screen.getByAltText("character mouth open");
    expect(openImg).toHaveStyle({ display: "block" });

    // 音量が少し低下（音節間の振幅低下をシミュレート）
    // 動的閾値は recentMax * 0.5 = 0.1 * 0.5 = 0.05
    // 0.02 < 0.05 なのでタイマーが開始
    rerender(
      <LipSyncImage
        mouthClosedUrl="/closed.png"
        mouthOpenUrl="/open.png"
        volume={0.02}
      />
    );

    expect(openImg).toHaveStyle({ display: "block" });

    // ヒステリシス後に口が閉じる
    act(() => {
      vi.advanceTimersByTime(80);
    });

    const closedImg = screen.getByAltText("character mouth closed");
    expect(closedImg).toHaveStyle({ display: "block" });
    expect(openImg).toHaveStyle({ display: "none" });
  });
});
