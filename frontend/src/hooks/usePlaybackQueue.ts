import {
  type RefObject,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import type { ClipEvent } from "../types/api";

export interface PreviewBody {
  text: string;
  speaker_id: number;
  speed?: number;
  pitch?: number;
  intonation?: number;
}

type QueueItem =
  | { kind: "timeline"; clip: ClipEvent }
  | { kind: "preview"; body: PreviewBody };

interface PlaybackQueue {
  playingClipTimestamp: number | null;
  enqueue: (clip: ClipEvent) => void;
  enqueuePreview: (body: PreviewBody) => void;
}

interface ReadyItem {
  item: QueueItem;
  objectUrl: string;
}

const WATCHDOG_EPSILON = 0.05;
const WATCHDOG_INTERVAL_MS = 250;

export function usePlaybackQueue(
  audioRef: RefObject<HTMLAudioElement>,
): PlaybackQueue {
  const waitingRef = useRef<QueueItem[]>([]);
  const prefetchedRef = useRef<ReadyItem | null>(null);
  const prefetchingAbortRef = useRef<AbortController | null>(null);
  const playingObjUrlRef = useRef<string | null>(null);
  const isPlayingRef = useRef(false);
  const [playingClipTimestamp, setPlayingClipTimestamp] = useState<
    number | null
  >(null);
  const playingClipTimestampRef = useRef<number | null>(null);
  playingClipTimestampRef.current = playingClipTimestamp;

  // startPrefetch は startNext から呼ばれ、完了後に startNext を呼ぶ。
  // 互いに依存する循環呼び出しを ref で解決する。
  const startNextRef = useRef<() => void>(() => {});
  const startPrefetchRef = useRef<() => void>(() => {});
  const watchdogCleanupRef = useRef<(() => void) | null>(null);

  const clearWatchdog = useCallback((): void => {
    if (watchdogCleanupRef.current) {
      watchdogCleanupRef.current();
      watchdogCleanupRef.current = null;
    }
  }, []);

  const startWatchdog = useCallback(
    (audio: HTMLAudioElement, item: QueueItem): void => {
      clearWatchdog();
      let fired = false;

      const checkEnded = (): void => {
        if (fired) return;
        const ended =
          audio.ended ||
          (Number.isFinite(audio.duration) &&
            audio.duration > 0 &&
            audio.currentTime >= audio.duration - WATCHDOG_EPSILON);
        if (!ended) return;

        fired = true;
        watchdogCleanupRef.current?.();
        watchdogCleanupRef.current = null;
        if (playingObjUrlRef.current) {
          URL.revokeObjectURL(playingObjUrlRef.current);
          playingObjUrlRef.current = null;
        }
        isPlayingRef.current = false;
        if (item.kind === "timeline") {
          playingClipTimestampRef.current = null;
          setPlayingClipTimestamp(null);
        }
        startNextRef.current();
      };

      audio.addEventListener("timeupdate", checkEnded);
      const intervalId = setInterval(checkEnded, WATCHDOG_INTERVAL_MS);

      watchdogCleanupRef.current = (): void => {
        audio.removeEventListener("timeupdate", checkEnded);
        clearInterval(intervalId);
      };
    },
    [clearWatchdog],
  );

  const startNext = useCallback((): void => {
    if (isPlayingRef.current) return;
    const audio = audioRef.current;
    if (!audio) return;

    const ready = prefetchedRef.current;
    if (ready) {
      prefetchedRef.current = null;
      const oldUrl = playingObjUrlRef.current;
      if (oldUrl) URL.revokeObjectURL(oldUrl);
      playingObjUrlRef.current = ready.objectUrl;
      isPlayingRef.current = true;
      if (ready.item.kind === "timeline") {
        const ts = ready.item.clip.timestamp;
        playingClipTimestampRef.current = ts;
        setPlayingClipTimestamp(ts);
      }
      audio.src = ready.objectUrl;
      audio.play().catch((err: unknown) => {
        console.error("play failed", err);
        URL.revokeObjectURL(ready.objectUrl);
        if (playingObjUrlRef.current === ready.objectUrl) {
          playingObjUrlRef.current = null;
        }
        clearWatchdog();
        isPlayingRef.current = false;
        if (ready.item.kind === "timeline") {
          playingClipTimestampRef.current = null;
          setPlayingClipTimestamp(null);
        }
        startNextRef.current();
      });
      startWatchdog(audio, ready.item);
      startPrefetchRef.current();
      return;
    }

    startPrefetchRef.current();
  }, [audioRef, clearWatchdog, startWatchdog]);

  const startPrefetch = useCallback((): void => {
    if (prefetchedRef.current) return;
    if (prefetchingAbortRef.current) return;
    const next = waitingRef.current.shift();
    if (!next) return;

    const abort = new AbortController();
    prefetchingAbortRef.current = abort;

    const fetchPromise: Promise<Response> =
      next.kind === "timeline"
        ? fetch(next.clip.url, { signal: abort.signal })
        : fetch("/api/preview-clip", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(next.body),
            signal: abort.signal,
          });

    fetchPromise
      .then((res) => {
        if (!res.ok) throw new Error(`fetch failed: ${res.status}`);
        return res.blob();
      })
      .then((blob) => {
        prefetchingAbortRef.current = null;
        prefetchedRef.current = {
          item: next,
          objectUrl: URL.createObjectURL(blob),
        };
        if (!isPlayingRef.current) {
          startNextRef.current();
        }
      })
      .catch((err: unknown) => {
        prefetchingAbortRef.current = null;
        if ((err as { name?: string }).name === "AbortError") return;
        console.error("prefetch failed", err);
        startPrefetchRef.current();
      });
  }, []);

  startNextRef.current = startNext;
  startPrefetchRef.current = startPrefetch;

  const enqueue = useCallback(
    (clip: ClipEvent): void => {
      waitingRef.current.push({ kind: "timeline", clip });
      startNext();
    },
    [startNext],
  );

  const enqueuePreview = useCallback(
    (body: PreviewBody): void => {
      waitingRef.current.push({ kind: "preview", body });
      startNext();
    },
    [startNext],
  );

  const stop = useCallback((): void => {
    clearWatchdog();
    const audio = audioRef.current;
    if (audio) {
      audio.pause();
      audio.removeAttribute("src");
      audio.load();
    }
    if (prefetchingAbortRef.current) {
      prefetchingAbortRef.current.abort();
      prefetchingAbortRef.current = null;
    }
    if (prefetchedRef.current) {
      URL.revokeObjectURL(prefetchedRef.current.objectUrl);
      prefetchedRef.current = null;
    }
    if (playingObjUrlRef.current) {
      URL.revokeObjectURL(playingObjUrlRef.current);
      playingObjUrlRef.current = null;
    }
    waitingRef.current = [];
    isPlayingRef.current = false;
    playingClipTimestampRef.current = null;
    setPlayingClipTimestamp(null);
  }, [audioRef, clearWatchdog]);

  useEffect(() => {
    return () => {
      stop();
    };
  }, [stop]);

  return { playingClipTimestamp, enqueue, enqueuePreview };
}
