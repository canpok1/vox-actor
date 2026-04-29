import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ClipEvent } from "../types/api";
import { usePlaybackQueue } from "./usePlaybackQueue";

function makeClip(n: number): ClipEvent {
  return {
    url: `http://example.com/clip/${n}.wav`,
    text: `clip ${n}`,
    speakerName: "speaker",
    styleName: "normal",
    timestamp: n * 1000,
  };
}

describe("usePlaybackQueue", () => {
  let audio: HTMLAudioElement;
  let audioRef: { current: HTMLAudioElement };
  let mockPlay: ReturnType<typeof vi.spyOn>;
  let mockPause: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    audio = document.createElement("audio");
    audioRef = { current: audio };
    mockPlay = vi
      .spyOn(HTMLMediaElement.prototype, "play")
      .mockResolvedValue(undefined);
    mockPause = vi
      .spyOn(HTMLMediaElement.prototype, "pause")
      .mockImplementation(() => {});
    vi.spyOn(HTMLMediaElement.prototype, "load").mockImplementation(() => {});
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("enqueue で audio.play が呼ばれ playingClipTimestamp が更新される", async () => {
    const { result } = renderHook(() => usePlaybackQueue(audioRef, true));
    expect(result.current.playingClipTimestamp).toBeNull();
    await act(async () => {
      result.current.enqueue(makeClip(1));
    });
    expect(mockPlay).toHaveBeenCalled();
    expect(result.current.playingClipTimestamp).toBe(1000);
  });

  it("ended イベント発火で次クリップが再生されキューが空なら null", async () => {
    const { result } = renderHook(() => usePlaybackQueue(audioRef, true));
    await act(async () => {
      result.current.enqueue(makeClip(1));
      result.current.enqueue(makeClip(2));
    });
    expect(result.current.playingClipTimestamp).toBe(1000);
    await act(async () => {
      audio.dispatchEvent(new Event("ended"));
    });
    expect(result.current.playingClipTimestamp).toBe(2000);
    await act(async () => {
      audio.dispatchEvent(new Event("ended"));
    });
    expect(result.current.playingClipTimestamp).toBeNull();
  });

  it("active=false への遷移で pause とキュー破棄と playingClipTimestamp=null になる", async () => {
    const { result, rerender } = renderHook(
      ({ active }: { active: boolean }) => usePlaybackQueue(audioRef, active),
      { initialProps: { active: true } },
    );
    await act(async () => {
      result.current.enqueue(makeClip(1));
    });
    expect(result.current.playingClipTimestamp).toBe(1000);
    await act(async () => {
      rerender({ active: false });
    });
    expect(mockPause).toHaveBeenCalled();
    expect(result.current.playingClipTimestamp).toBeNull();
  });

  it("active=false の間に enqueue してもキューに積まれない", async () => {
    const { result } = renderHook(() => usePlaybackQueue(audioRef, false));
    await act(async () => {
      result.current.enqueue(makeClip(1));
    });
    expect(mockPlay).not.toHaveBeenCalled();
    expect(result.current.playingClipTimestamp).toBeNull();
  });

  it("audio.play reject 時に playingClipTimestamp が null に戻り次の再生試行が走る", async () => {
    mockPlay
      .mockRejectedValueOnce(new Error("play failed"))
      .mockResolvedValue(undefined);
    vi.spyOn(console, "error").mockImplementation(() => {});
    const { result } = renderHook(() => usePlaybackQueue(audioRef, true));
    await act(async () => {
      result.current.enqueue(makeClip(1));
      result.current.enqueue(makeClip(2));
    });
    expect(result.current.playingClipTimestamp).toBe(2000);
  });

  it("アンマウントで ended リスナーが解除される", () => {
    const removeEventListenerSpy = vi.spyOn(audio, "removeEventListener");
    const { unmount } = renderHook(() => usePlaybackQueue(audioRef, true));
    unmount();
    expect(removeEventListenerSpy).toHaveBeenCalledWith(
      "ended",
      expect.any(Function),
    );
  });
});
