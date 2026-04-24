import { useEffect, useRef, useState } from "react";

// onClip / onError は ref 経由で呼び出されるため、呼び出し側が memo 化しなくても再接続は起きない。
// "error" イベントは EventSource ネイティブの接続失敗イベントと衝突するため、
// MessageEvent（= サーバー送信の `event: error`）と Event（= ネイティブの接続失敗）を
// ハンドラ内で判別する。
export function useEventSource(
  url: string,
  onClip: (data: string) => void,
  onError: (data: string) => void,
): boolean {
  const [connected, setConnected] = useState(false);
  const onClipRef = useRef(onClip);
  onClipRef.current = onClip;
  const onErrorRef = useRef(onError);
  onErrorRef.current = onError;

  useEffect(() => {
    let es: EventSource | null = null;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    let cancelled = false;

    const connect = (): void => {
      es = new EventSource(url);
      es.addEventListener("open", () => {
        setConnected(true);
      });
      es.addEventListener("clip", (event) => {
        onClipRef.current(event.data);
      });
      es.addEventListener("error", (event) => {
        // サーバー送信の `event: error` は MessageEvent として配送される。
        // ネイティブの接続失敗は通常の Event（data を持たない）。
        if (event instanceof MessageEvent && typeof event.data === "string") {
          onErrorRef.current(event.data);
          return;
        }
        setConnected(false);
        es?.close();
        es = null;
        if (!cancelled) {
          reconnectTimer = setTimeout(connect, 2000);
        }
      });
    };

    connect();

    return () => {
      cancelled = true;
      if (reconnectTimer !== null) {
        clearTimeout(reconnectTimer);
      }
      es?.close();
    };
  }, [url]);

  return connected;
}
