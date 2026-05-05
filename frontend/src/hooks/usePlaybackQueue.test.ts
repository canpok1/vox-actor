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
    speakerId: 0,
  };
}

function makeFetchMock() {
  return vi.fn((url: string | URL | Request) => {
    const blobData = new Blob([`audio:${String(url)}`], { type: "audio/wav" });
    // jsdom では new Response(blob) が blob.stream() を呼び失敗するため
    // 実装が使う .blob() のみを持つ最小限のオブジェクトを返す
    return Promise.resolve({ blob: () => Promise.resolve(blobData) } as Response);
  });
}

function setAudioEnded(audio: HTMLAudioElement, value: boolean): void {
  Object.defineProperty(audio, "ended", {
    get: () => value,
    configurable: true,
  });
}

function setAudioDuration(audio: HTMLAudioElement, value: number): void {
  Object.defineProperty(audio, "duration", {
    get: () => value,
    configurable: true,
  });
}

function setAudioCurrentTime(audio: HTMLAudioElement, value: number): void {
  Object.defineProperty(audio, "currentTime", {
    get: () => value,
    configurable: true,
  });
}

describe("usePlaybackQueue", () => {
  let audio: HTMLAudioElement;
  let audioRef: { current: HTMLAudioElement };
  let mockPlay: ReturnType<typeof vi.spyOn>;
  let mockPause: ReturnType<typeof vi.spyOn>;
  let mockCreateObjectURL: ReturnType<typeof vi.fn>;
  let urlCounter: number;

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
    urlCounter = 0;
    // jsdom では URL.createObjectURL / revokeObjectURL が未実装のため直接定義する
    mockCreateObjectURL = vi.fn(() => `blob:mock-${++urlCounter}`);
    URL.createObjectURL = mockCreateObjectURL;
    URL.revokeObjectURL = vi.fn();
    vi.stubGlobal("fetch", makeFetchMock());
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
    vi.useRealTimers();
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

  it("enqueue で fetch が呼ばれ audio.src に blob URL が設定される", async () => {
    const { result } = renderHook(() => usePlaybackQueue(audioRef, true));
    await act(async () => {
      result.current.enqueue(makeClip(1));
    });
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      "http://example.com/clip/1.wav",
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    );
    expect(mockCreateObjectURL).toHaveBeenCalled();
    expect(audio.src).toMatch(/^blob:/);
  });

  it("timeupdate で audio.ended=true のとき次クリップが再生されキューが空なら null", async () => {
    const { result } = renderHook(() => usePlaybackQueue(audioRef, true));
    await act(async () => {
      result.current.enqueue(makeClip(1));
      result.current.enqueue(makeClip(2));
    });
    expect(result.current.playingClipTimestamp).toBe(1000);

    setAudioEnded(audio, true);
    await act(async () => {
      audio.dispatchEvent(new Event("timeupdate"));
    });
    expect(result.current.playingClipTimestamp).toBe(2000);

    setAudioEnded(audio, true);
    await act(async () => {
      audio.dispatchEvent(new Event("timeupdate"));
    });
    expect(result.current.playingClipTimestamp).toBeNull();
  });

  it("duration/currentTime 判定でウォッチドッグが発火して次クリップが再生される", async () => {
    const { result } = renderHook(() => usePlaybackQueue(audioRef, true));
    await act(async () => {
      result.current.enqueue(makeClip(1));
      result.current.enqueue(makeClip(2));
    });
    expect(result.current.playingClipTimestamp).toBe(1000);

    // currentTime が duration - EPSILON に到達した状態を作る
    setAudioDuration(audio, 3.0);
    setAudioCurrentTime(audio, 2.96);
    await act(async () => {
      audio.dispatchEvent(new Event("timeupdate"));
    });
    expect(result.current.playingClipTimestamp).toBe(2000);
  });

  it("setInterval でウォッチドッグが発火する", async () => {
    vi.useFakeTimers();
    const { result } = renderHook(() => usePlaybackQueue(audioRef, true));
    await act(async () => {
      result.current.enqueue(makeClip(1));
    });
    expect(result.current.playingClipTimestamp).toBe(1000);

    setAudioEnded(audio, true);
    await act(async () => {
      vi.advanceTimersByTime(250);
    });
    expect(result.current.playingClipTimestamp).toBeNull();
  });

  it("ウォッチドッグ発火時に先読み済みクリップが即再生される", async () => {
    const { result } = renderHook(() => usePlaybackQueue(audioRef, true));
    await act(async () => {
      result.current.enqueue(makeClip(1));
      result.current.enqueue(makeClip(2));
    });
    // clip1 再生中、clip2 が先読み済み
    expect(result.current.playingClipTimestamp).toBe(1000);
    const playCallsBefore = mockPlay.mock.calls.length;

    setAudioEnded(audio, true);
    await act(async () => {
      audio.dispatchEvent(new Event("timeupdate"));
    });
    // ウォッチドッグ発火直後に fetch を待たず即 play() が追加呼び出しされる
    expect(mockPlay.mock.calls.length).toBeGreaterThan(playCallsBefore);
    expect(result.current.playingClipTimestamp).toBe(2000);
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

  it("先読みエラー時は次クリップへスキップされる", async () => {
    const fetchMock = vi
      .fn()
      .mockRejectedValueOnce(new Error("network error"))
      .mockImplementation(makeFetchMock());
    vi.stubGlobal("fetch", fetchMock);
    vi.spyOn(console, "error").mockImplementation(() => {});
    const { result } = renderHook(() => usePlaybackQueue(audioRef, true));
    await act(async () => {
      result.current.enqueue(makeClip(1));
      result.current.enqueue(makeClip(2));
    });
    // clip1 の先読みが失敗し clip2 が再生される
    expect(result.current.playingClipTimestamp).toBe(2000);
  });

  it("active=false 遷移で進行中の先読み fetch がキャンセルされる", async () => {
    let capturedSignal: AbortSignal | null = null;
    const slowFetch = vi.fn((_url: string | URL | Request, init?: RequestInit) => {
      capturedSignal = init?.signal ?? null;
      return new Promise<Response>(() => {}); // 解決しない
    });
    vi.stubGlobal("fetch", slowFetch);

    const { result, rerender } = renderHook(
      ({ active }: { active: boolean }) => usePlaybackQueue(audioRef, active),
      { initialProps: { active: true } },
    );
    await act(async () => {
      result.current.enqueue(makeClip(1));
    });
    expect(capturedSignal).not.toBeNull();
    expect((capturedSignal as unknown as AbortSignal).aborted).toBe(false);

    await act(async () => {
      rerender({ active: false });
    });
    expect((capturedSignal as unknown as AbortSignal).aborted).toBe(true);
  });

  it("アンマウントでウォッチドッグがクリアされる", async () => {
    const clearIntervalSpy = vi.spyOn(global, "clearInterval");
    const { result, unmount } = renderHook(() =>
      usePlaybackQueue(audioRef, true),
    );
    await act(async () => {
      result.current.enqueue(makeClip(1));
    });
    unmount();
    expect(clearIntervalSpy).toHaveBeenCalled();
  });
});
