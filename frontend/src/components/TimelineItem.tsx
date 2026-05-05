import { useEffect, useRef } from "react";
import type { ErrorCategory, TimelineEntry } from "../types/api";

interface TimelineItemProps {
  entry: TimelineEntry;
  playing: boolean;
  showSpeakerName: boolean;
  showStyleName: boolean;
  showTimestamp: boolean;
  highlightPlaying?: boolean;
  onReplay?: () => void;
}

function formatTimestamp(ms: number): string {
  if (!Number.isFinite(ms) || ms <= 0) return "";
  return new Date(ms).toLocaleTimeString("ja-JP", { hour12: false });
}

const ERROR_CATEGORY_LABEL: Record<ErrorCategory, string> = {
  synthesis: "合成エラー",
  file: "ファイルエラー",
  connection: "接続エラー",
};

export function TimelineItem({
  entry,
  playing,
  showSpeakerName,
  showStyleName,
  showTimestamp,
  highlightPlaying = true,
  onReplay,
}: TimelineItemProps) {
  const ref = useRef<HTMLLIElement>(null);
  // 初期値は false 固定。マウント時点で playing=true のケース（SSE で
  // clip 追加と再生開始が同一レンダーにコミットされる）も立ち上がりエッジと
  // して scrollIntoView を発火させる必要があるため。
  const prevPlaying = useRef(false);

  useEffect(() => {
    if (playing && !prevPlaying.current) {
      ref.current?.scrollIntoView({ block: "nearest" });
    }
    prevPlaying.current = playing;
  }, [playing]);

  const base =
    "my-1 flex flex-col gap-[0.15rem] rounded break-words px-2 py-1 sm:px-[0.6rem] sm:py-[0.4rem]";
  const showMeta = showSpeakerName || showStyleName || showTimestamp;

  if (entry.kind === "error") {
    const categoryLabel = ERROR_CATEGORY_LABEL[entry.category];
    return (
      <li
        ref={ref}
        data-error-id={entry.id}
        data-error-category={entry.category}
        className={`${base} text-ctp-red`}
      >
        <div className="flex flex-wrap items-baseline gap-2 text-[0.8rem] text-ctp-red">
          {showTimestamp && (
            <span className="tabular-nums">
              {formatTimestamp(entry.timestamp)}
            </span>
          )}
          <span className="font-semibold">{categoryLabel}</span>
          {entry.path && <span className="break-all">{entry.path}</span>}
        </div>
        <div className="flex items-start gap-2">
          <span aria-hidden className="inline-block w-[1em] flex-none">
            ⚠
          </span>
          <span>{entry.message}</span>
        </div>
        {entry.text && (
          <div className="flex flex-wrap items-baseline gap-2 text-[0.8rem] text-ctp-subtext">
            {showSpeakerName && entry.speakerName && (
              <span className="font-semibold text-ctp-text">
                {entry.speakerName}
              </span>
            )}
            {showStyleName && entry.styleName && (
              <span className="text-ctp-blue">[{entry.styleName}]</span>
            )}
            <span className="break-all">{entry.text}</span>
          </div>
        )}
      </li>
    );
  }

  const playingCls = playing && highlightPlaying ? "bg-ctp-overlay font-bold" : "";
  return (
    <li
      ref={ref}
      data-clip-timestamp={entry.timestamp}
      className={`${base} ${playingCls} text-ctp-text`}
    >
      {showMeta && (
        <div className="flex flex-wrap items-baseline gap-2 text-[0.8rem] text-ctp-subtext">
          {showTimestamp && (
            <span className="tabular-nums">
              {formatTimestamp(entry.timestamp)}
            </span>
          )}
          {showSpeakerName && (
            <span className="font-semibold text-ctp-text">
              {entry.speakerName}
            </span>
          )}
          {showStyleName && entry.styleName && (
            <span className="text-ctp-blue">[{entry.styleName}]</span>
          )}
        </div>
      )}
      <div className="flex items-start gap-2">
        <span
          aria-hidden
          className="inline-block w-[1em] flex-none text-ctp-blue"
        >
          {playing && highlightPlaying ? "▶" : ""}
        </span>
        <span>{entry.text}</span>
        {onReplay && (
          <button
            type="button"
            onClick={onReplay}
            className="ml-auto shrink-0 cursor-pointer rounded border-0 bg-ctp-surface px-2 py-0.5 text-[0.75rem] text-ctp-blue hover:bg-ctp-overlay"
          >
            再生
          </button>
        )}
      </div>
    </li>
  );
}
