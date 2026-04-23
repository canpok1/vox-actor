interface VolumeControlsProps {
  volume: number;
  muted: boolean;
  onVolumeChange: (value: number) => void;
  onMuteChange: (muted: boolean) => void;
}

export function VolumeControls({ volume, muted, onVolumeChange, onMuteChange }: VolumeControlsProps) {
  const icon = muted || volume === 0 ? "🔇" : "🔊";
  return (
    <div className="mt-4 mb-1 flex flex-wrap items-center gap-4">
      <label className="flex flex-1 items-center gap-3">
        音量
        <input
          type="range"
          min={0}
          max={100}
          value={volume}
          onChange={(e) => onVolumeChange(Number(e.target.value))}
          className="flex-1"
        />
        <span>{icon}</span>
      </label>
      <label className="inline-flex items-center gap-1">
        <input
          type="checkbox"
          checked={muted}
          onChange={(e) => onMuteChange(e.target.checked)}
        />
        消音
      </label>
    </div>
  );
}
