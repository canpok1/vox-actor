import { useCallback, useEffect, useRef } from "react";
import type { TimelineEntry } from "../types/api";
import { TimelineItem } from "./TimelineItem";

interface TimelineProps {
  entries: TimelineEntry[];
  playingClipTimestamp: number | null;
  lastPlayingClipTimestamp?: number | null;
  showSpeakerName: boolean;
  showStyleName: boolean;
  showTimestamp: boolean;
  isCharacterMode?: boolean;
}

export function Timeline({
  entries,
  playingClipTimestamp,
  lastPlayingClipTimestamp = null,
  showSpeakerName,
  showStyleName,
  showTimestamp,
  isCharacterMode = false,
}: TimelineProps) {
  const listRef = useRef<HTMLOListElement>(null);
  const prevIsCharacterMode = useRef(isCharacterMode);
  const hasInitialScrolled = useRef(false);

  const scrollToBottom = useCallback(() => {
    const list = listRef.current;
    if (list) {
      list.scrollTop = list.scrollHeight;
    }
  }, []);

  useEffect(() => {
    if (!isCharacterMode && prevIsCharacterMode.current) {
      scrollToBottom();
    }
    prevIsCharacterMode.current = isCharacterMode;
  }, [isCharacterMode, scrollToBottom]);

  useEffect(() => {
    if (!hasInitialScrolled.current && entries.length > 0) {
      scrollToBottom();
      hasInitialScrolled.current = true;
    }
  }, [entries, scrollToBottom]);

  const characterModeClipTimestamp =
    playingClipTimestamp ?? lastPlayingClipTimestamp;
  const displayEntries = isCharacterMode
    ? characterModeClipTimestamp !== null
      ? entries.filter(
          (e) => e.kind === "clip" && e.timestamp === characterModeClipTimestamp,
        )
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
            key={`${entry.kind}-${entry.timestamp}`}
            entry={entry}
            playing={
              entry.kind === "clip" &&
              entry.timestamp === playingClipTimestamp
            }
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
