import {
  type RefObject,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import type { ClipEvent } from "../types/api";

interface PlaybackQueue {
  playingClipId: number | null;
  enqueue: (clip: ClipEvent) => void;
}

// active=false の間は enqueue してもキューに積まれず、既存の再生は停止される。
export function usePlaybackQueue(
  audioRef: RefObject<HTMLAudioElement>,
  active: boolean,
): PlaybackQueue {
  const queueRef = useRef<ClipEvent[]>([]);
  const [playingClipId, setPlayingClipId] = useState<number | null>(null);
  const playingClipIdRef = useRef<number | null>(null);
  playingClipIdRef.current = playingClipId;
  const activeRef = useRef(active);
  activeRef.current = active;

  const playNext = useCallback((): void => {
    if (!activeRef.current) return;
    if (playingClipIdRef.current !== null) return;
    const audio = audioRef.current;
    if (!audio) return;
    const next = queueRef.current.shift();
    if (!next) return;
    playingClipIdRef.current = next.id;
    setPlayingClipId(next.id);
    audio.src = next.url;
    audio.play().catch((err: unknown) => {
      console.error("play failed", err);
      playingClipIdRef.current = null;
      setPlayingClipId(null);
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
    playingClipIdRef.current = null;
    setPlayingClipId(null);
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
      playingClipIdRef.current = null;
      setPlayingClipId(null);
      playNext();
    };
    audio.addEventListener("ended", onEnded);
    return () => {
      audio.removeEventListener("ended", onEnded);
    };
  }, [audioRef, playNext]);

  return { playingClipId, enqueue };
}
