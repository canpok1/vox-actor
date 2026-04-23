import { useEffect, useRef } from "react";
import type { ClipEvent } from "../types/api";

interface TimelineItemProps {
  clip: ClipEvent;
  playing: boolean;
  showSpeakerName: boolean;
  showStyleName: boolean;
  showTimestamp: boolean;
}

function formatTimestamp(ms: number): string {
  if (!Number.isFinite(ms) || ms <= 0) return "";
  return new Date(ms).toLocaleTimeString("ja-JP", { hour12: false });
}

export function TimelineItem({
  clip,
  playing,
  showSpeakerName,
  showStyleName,
  showTimestamp,
}: TimelineItemProps) {
  const ref = useRef<HTMLLIElement>(null);
  const prevPlaying = useRef(playing);

  useEffect(() => {
    if (playing && !prevPlaying.current) {
      ref.current?.scrollIntoView({ block: "nearest" });
    }
    prevPlaying.current = playing;
  }, [playing]);

  const base =
    "my-1 flex flex-col gap-[0.15rem] rounded break-words px-2 py-1 sm:px-[0.6rem] sm:py-[0.4rem]";
  const playingCls = playing ? "bg-ctp-overlay font-bold" : "";
  const showMeta = showSpeakerName || showStyleName || showTimestamp;

  return (
    <li
      ref={ref}
      data-clip-id={clip.id}
      className={`${base} ${playingCls} text-ctp-text`}
    >
      {showMeta && (
        <div className="flex flex-wrap items-baseline gap-2 text-[0.8rem] text-ctp-subtext">
          {showTimestamp && (
            <span className="tabular-nums">{formatTimestamp(clip.timestamp)}</span>
          )}
          {showSpeakerName && (
            <span className="font-semibold text-ctp-text">{clip.speakerName}</span>
          )}
          {showStyleName && clip.styleName && (
            <span className="text-ctp-blue">[{clip.styleName}]</span>
          )}
        </div>
      )}
      <div className="flex items-start gap-2">
        <span aria-hidden className="inline-block w-[1em] flex-none text-ctp-blue">
          {playing ? "▶" : ""}
        </span>
        <span>{clip.text}</span>
      </div>
    </li>
  );
}
