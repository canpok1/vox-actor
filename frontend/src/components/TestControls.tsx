import type { Speaker } from "../types/api";

interface TestControlsProps {
  speakers: Speaker[];
  selectedSpeakerId: string;
  onSpeakerChange: (id: string) => void;
  onPlay: () => void;
  error: string;
}

export function TestControls({
  speakers,
  selectedSpeakerId,
  onSpeakerChange,
  onPlay,
  error,
}: TestControlsProps) {
  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-2 rounded-md bg-ctp-base px-4 py-3">
      <label htmlFor="test-speaker">話者</label>
      <select
        id="test-speaker"
        value={selectedSpeakerId}
        onChange={(e) => onSpeakerChange(e.target.value)}
        className="min-w-0 flex-1 rounded border border-ctp-overlay bg-ctp-surface px-2 py-1 text-ctp-text sm:min-w-[12rem] sm:flex-none"
      >
        {speakers.map((speaker) => {
          const style = speaker.styleName ? `(${speaker.styleName})` : "";
          return (
            <option key={speaker.id} value={String(speaker.id)}>
              {speaker.speakerName}
              {style}
            </option>
          );
        })}
      </select>
      <button
        type="button"
        onClick={onPlay}
        className="cursor-pointer rounded border-0 bg-ctp-blue px-[0.85rem] py-[0.4rem] font-semibold text-ctp-base hover:bg-ctp-sky disabled:cursor-not-allowed disabled:bg-ctp-overlay disabled:text-ctp-subtext"
      >
        ▶ テスト再生
      </button>
      <span role="status" aria-live="polite" className="text-[0.85rem] text-ctp-red">
        {error}
      </span>
    </div>
  );
}
