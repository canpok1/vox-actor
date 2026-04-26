import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useEventSource } from "./useEventSource";

class MockEventSource {
  static instances: MockEventSource[] = [];
  readonly url: string;
  private listeners: Map<string, Array<(event: Event) => void>> = new Map();
  close = vi.fn();

  constructor(url: string) {
    this.url = url;
    MockEventSource.instances.push(this);
  }

  addEventListener(type: string, handler: (event: Event) => void): void {
    const existing = this.listeners.get(type) ?? [];
    existing.push(handler);
    this.listeners.set(type, existing);
  }

  emit(type: string, event: Event): void {
    for (const handler of this.listeners.get(type) ?? []) {
      handler(event);
    }
  }
}

describe("useEventSource", () => {
  const URL = "http://localhost/events";

  beforeEach(() => {
    MockEventSource.instances = [];
    vi.stubGlobal("EventSource", MockEventSource);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  function latestEs(): MockEventSource {
    return MockEventSource.instances[MockEventSource.instances.length - 1];
  }

  it("open 受信で connected=true に遷移する", () => {
    const { result } = renderHook(() => useEventSource(URL, vi.fn(), vi.fn()));
    expect(result.current).toBe(false);
    act(() => {
      latestEs().emit("open", new Event("open"));
    });
    expect(result.current).toBe(true);
  });

  it("clip イベント受信で onClip(event.data) が呼ばれる", () => {
    const onClip = vi.fn();
    renderHook(() => useEventSource(URL, onClip, vi.fn()));
    act(() => {
      latestEs().emit("clip", new MessageEvent("clip", { data: "clip-data" }));
    });
    expect(onClip).toHaveBeenCalledWith("clip-data");
  });

  it("サーバー送信 error は onError を呼び再接続しない", () => {
    const onError = vi.fn();
    renderHook(() => useEventSource(URL, vi.fn(), onError));
    const countBefore = MockEventSource.instances.length;
    act(() => {
      latestEs().emit("error", new MessageEvent("error", { data: "err-data" }));
    });
    expect(onError).toHaveBeenCalledWith("err-data");
    expect(MockEventSource.instances.length).toBe(countBefore);
  });

  it("ネイティブ接続失敗で connected=false に遷移し 2秒後に再接続する", () => {
    vi.useFakeTimers();
    const { result } = renderHook(() => useEventSource(URL, vi.fn(), vi.fn()));
    act(() => {
      latestEs().emit("open", new Event("open"));
    });
    expect(result.current).toBe(true);
    const esBeforeReconnect = latestEs();
    act(() => {
      latestEs().emit("error", new Event("error"));
    });
    expect(result.current).toBe(false);
    expect(esBeforeReconnect.close).toHaveBeenCalled();
    const countBefore = MockEventSource.instances.length;
    act(() => {
      vi.advanceTimersByTime(2000);
    });
    expect(MockEventSource.instances.length).toBe(countBefore + 1);
  });

  it("アンマウントで close が呼ばれ再接続タイマーがキャンセルされる", () => {
    vi.useFakeTimers();
    const { unmount } = renderHook(() => useEventSource(URL, vi.fn(), vi.fn()));
    const es = latestEs();
    act(() => {
      es.emit("error", new Event("error"));
    });
    unmount();
    const countAfterUnmount = MockEventSource.instances.length;
    act(() => {
      vi.advanceTimersByTime(2000);
    });
    expect(MockEventSource.instances.length).toBe(countAfterUnmount);
    expect(es.close).toHaveBeenCalled();
  });

  it("onClip と onError を差し替えても再接続しない", () => {
    const onClip1 = vi.fn();
    const onClip2 = vi.fn();
    const { rerender } = renderHook(
      ({ clip }: { clip: (data: string) => void }) =>
        useEventSource(URL, clip, vi.fn()),
      { initialProps: { clip: onClip1 } },
    );
    const countBefore = MockEventSource.instances.length;
    rerender({ clip: onClip2 });
    expect(MockEventSource.instances.length).toBe(countBefore);
    act(() => {
      latestEs().emit("clip", new MessageEvent("clip", { data: "data" }));
    });
    expect(onClip2).toHaveBeenCalledWith("data");
    expect(onClip1).not.toHaveBeenCalled();
  });
});
