import { useEffect, useRef } from "react";
import type { TimelineEntry } from "../types/api";
import { TimelineItem } from "./TimelineItem";

interface TimelineProps {
  entries: TimelineEntry[];
  playingClipId: number | null;
  showSpeakerName: boolean;
  showStyleName: boolean;
  showTimestamp: boolean;
  isCharacterMode?: boolean;
}

export function Timeline({
  entries,
  playingClipId,
  showSpeakerName,
  showStyleName,
  showTimestamp,
  isCharacterMode = false,
}: TimelineProps) {
  const listRef = useRef<HTMLOListElement>(null);
  const prevIsCharacterMode = useRef(isCharacterMode);

  useEffect(() => {
    if (!isCharacterMode && prevIsCharacterMode.current) {
      const list = listRef.current;
      if (list) {
        list.scrollTop = list.scrollHeight;
      }
    }
    prevIsCharacterMode.current = isCharacterMode;
  }, [isCharacterMode]);

  const displayEntries = isCharacterMode
    ? playingClipId !== null
      ? entries.filter((e) => e.kind === "clip" && e.id === playingClipId)
      : []
    : entries;

  const listCls = isCharacterMode
    ? "m-0 h-[7rem] list-none overflow-y-auto p-0"
    : "m-0 max-h-[65vh] list-none overflow-y-auto p-0 md:max-h-[60vh]";

  return (
    <div className="rounded-md bg-ctp-base px-3 py-[0.75rem] sm:px-4">
      <ol ref={listRef} aria-live="polite" className={listCls}>
        {displayEntries.map((entry) => (
          <TimelineItem
            key={`${entry.kind}-${entry.id}`}
            entry={entry}
            playing={entry.kind === "clip" && entry.id === playingClipId}
            showSpeakerName={isCharacterMode ? true : showSpeakerName}
            showStyleName={isCharacterMode ? false : showStyleName}
            showTimestamp={isCharacterMode ? false : showTimestamp}
            highlightPlaying={!isCharacterMode}
          />
        ))}
      </ol>
    </div>
  );
}
