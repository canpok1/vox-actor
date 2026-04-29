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

// active=false の間は enqueue してもキューに積まれず、既存の再生は停止される。
export function usePlaybackQueue(
  audioRef: RefObject<HTMLAudioElement>,
  active: boolean,
): PlaybackQueue {
  const queueRef = useRef<ClipEvent[]>([]);
  const [playingClipTimestamp, setPlayingClipTimestamp] = useState<number | null>(null);
  const playingClipTimestampRef = useRef<number | null>(null);
  playingClipTimestampRef.current = playingClipTimestamp;
  const activeRef = useRef(active);
  activeRef.current = active;

  const playNext = useCallback((): void => {
    if (!activeRef.current) return;
    if (playingClipTimestampRef.current !== null) return;
    const audio = audioRef.current;
    if (!audio) return;
    const next = queueRef.current.shift();
    if (!next) return;
    playingClipTimestampRef.current = next.timestamp;
    setPlayingClipTimestamp(next.timestamp);
    audio.src = next.url;
    audio.play().catch((err: unknown) => {
      console.error("play failed", err);
      playingClipTimestampRef.current = null;
      setPlayingClipTimestamp(null);
      playNext();
    });
  }, [audioRef]);

  const enqueue = useCallback(
    (clip: ClipEvent): void => {
      if (!activeRef.current) return;
      queueRef.current.push(clip);
      playNext();
    },
    [playNext],
  );

  const stop = useCallback((): void => {
    const audio = audioRef.current;
    if (audio) {
      audio.pause();
      audio.removeAttribute("src");
      audio.load();
    }
    queueRef.current = [];
    playingClipTimestampRef.current = null;
    setPlayingClipTimestamp(null);
  }, [audioRef]);

  useEffect(() => {
    if (!active) {
      stop();
    }
  }, [active, stop]);

  useEffect(() => {
    const audio = audioRef.current;
    if (!audio) return;
    const onEnded = (): void => {
      playingClipTimestampRef.current = null;
      setPlayingClipTimestamp(null);
      playNext();
    };
    audio.addEventListener("ended", onEnded);
    return () => {
      audio.removeEventListener("ended", onEnded);
    };
  }, [audioRef, playNext]);

  return { playingClipTimestamp, enqueue };
}
