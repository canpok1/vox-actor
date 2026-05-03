import {
  type RefObject,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import type { ClipEvent } from "../types/api";

interface PlaybackQueue {
  playingClipTimestamp: number | null;
  enqueue: (clip: ClipEvent) => void;
}

interface ReadyClip {
  clip: ClipEvent;
  objectUrl: string;
}

// active=false の間は enqueue してもキューに積まれず、既存の再生は停止される。
export function usePlaybackQueue(
  audioRef: RefObject<HTMLAudioElement>,
  active: boolean,
): PlaybackQueue {
  const waitingRef = useRef<ClipEvent[]>([]);
  const prefetchedRef = useRef<ReadyClip | null>(null);
  const prefetchingAbortRef = useRef<AbortController | null>(null);
  const playingObjUrlRef = useRef<string | null>(null);
  const [playingClipTimestamp, setPlayingClipTimestamp] = useState<
    number | null
  >(null);
  const playingClipTimestampRef = useRef<number | null>(null);
  playingClipTimestampRef.current = playingClipTimestamp;
  const activeRef = useRef(active);
  activeRef.current = active;

  // startPrefetch は startNext から呼ばれ、完了後に startNext を呼ぶ。
  // 互いに依存する循環呼び出しを ref で解決する。
  const startNextRef = useRef<() => void>(() => {});
  const startPrefetchRef = useRef<() => void>(() => {});

  const startNext = useCallback((): void => {
    if (!activeRef.current) return;
    if (playingClipTimestampRef.current !== null) return;
    const audio = audioRef.current;
    if (!audio) return;

    const ready = prefetchedRef.current;
    if (ready) {
      prefetchedRef.current = null;
      const oldUrl = playingObjUrlRef.current;
      if (oldUrl) URL.revokeObjectURL(oldUrl);
      playingObjUrlRef.current = ready.objectUrl;
      playingClipTimestampRef.current = ready.clip.timestamp;
      setPlayingClipTimestamp(ready.clip.timestamp);
      audio.src = ready.objectUrl;
      audio.play().catch((err: unknown) => {
        console.error("play failed", err);
        URL.revokeObjectURL(ready.objectUrl);
        if (playingObjUrlRef.current === ready.objectUrl) {
          playingObjUrlRef.current = null;
        }
        playingClipTimestampRef.current = null;
        setPlayingClipTimestamp(null);
        startNextRef.current();
      });
      startPrefetchRef.current();
      return;
    }

    startPrefetchRef.current();
  }, [audioRef]);

  const startPrefetch = useCallback((): void => {
    if (prefetchedRef.current) return;
    if (prefetchingAbortRef.current) return;
    const next = waitingRef.current.shift();
    if (!next) return;

    const abort = new AbortController();
    prefetchingAbortRef.current = abort;

    fetch(next.url, { signal: abort.signal })
      .then((res) => res.blob())
      .then((blob) => {
        prefetchingAbortRef.current = null;
        if (!activeRef.current) {
          return;
        }
        prefetchedRef.current = {
          clip: next,
          objectUrl: URL.createObjectURL(blob),
        };
        if (playingClipTimestampRef.current === null) {
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
      if (!activeRef.current) return;
      waitingRef.current.push(clip);
      startNext();
    },
    [startNext],
  );

  const stop = useCallback((): void => {
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
    playingClipTimestampRef.current = null;
    setPlayingClipTimestamp(null);
  }, [audioRef]);

  useEffect(() => {
    if (!active) {
      stop();
    }
  }, [active, stop]);

  useEffect(() => {
    return () => {
      stop();
    };
  }, [stop]);

  useEffect(() => {
    const audio = audioRef.current;
    if (!audio) return;
    const onEnded = (): void => {
      if (playingObjUrlRef.current) {
        URL.revokeObjectURL(playingObjUrlRef.current);
        playingObjUrlRef.current = null;
      }
      playingClipTimestampRef.current = null;
      setPlayingClipTimestamp(null);
      startNextRef.current();
    };
    audio.addEventListener("ended", onEnded);
    return () => {
      audio.removeEventListener("ended", onEnded);
    };
  }, [audioRef]);

  return { playingClipTimestamp, enqueue };
}
