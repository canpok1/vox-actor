import type { TimelineEntry } from "../types/api";
import { TimelineItem } from "./TimelineItem";

interface TimelineProps {
  entries: TimelineEntry[];
  playingClipId: number | null;
  showSpeakerName: boolean;
  showStyleName: boolean;
  showTimestamp: boolean;
}

export function Timeline({
  entries,
  playingClipId,
  showSpeakerName,
  showStyleName,
  showTimestamp,
}: TimelineProps) {
  return (
    <div className="rounded-md bg-ctp-base px-3 py-[0.75rem] sm:px-4">
      <ol
        aria-live="polite"
        className="m-0 max-h-[65vh] list-none overflow-y-auto p-0 md:max-h-[60vh]"
      >
        {entries.map((entry) => (
          <TimelineItem
            key={`${entry.kind}-${entry.id}`}
            entry={entry}
            playing={entry.kind === "clip" && entry.id === playingClipId}
            showSpeakerName={showSpeakerName}
            showStyleName={showStyleName}
            showTimestamp={showTimestamp}
          />
        ))}
      </ol>
    </div>
  );
}
