import { useEffect, useRef, useState } from "react";

// onClip は ref 経由で呼び出されるため、呼び出し側が memo 化しなくても再接続は起きない。
export function useEventSource(
  url: string,
  onClip: (data: string) => void,
): boolean {
  const [connected, setConnected] = useState(false);
  const onClipRef = useRef(onClip);
  onClipRef.current = onClip;

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
      es.addEventListener("error", () => {
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
