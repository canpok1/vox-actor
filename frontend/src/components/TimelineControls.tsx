interface TimelineControlsProps {
  historySize: number;
  historySizeOptions: readonly number[];
  onHistorySizeChange: (value: number) => void;
  showSpeakerName: boolean;
  showStyleName: boolean;
  showTimestamp: boolean;
  onShowSpeakerNameChange: (value: boolean) => void;
  onShowStyleNameChange: (value: boolean) => void;
  onShowTimestampChange: (value: boolean) => void;
}

interface CheckboxToggleProps {
  label: string;
  checked: boolean;
  onChange: (value: boolean) => void;
}

function CheckboxToggle({ label, checked, onChange }: CheckboxToggleProps) {
  return (
    <label className="inline-flex items-center gap-1">
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
      />
      {label}
    </label>
  );
}

export function TimelineControls(props: TimelineControlsProps) {
  const {
    historySize,
    historySizeOptions,
    onHistorySizeChange,
    showSpeakerName,
    showStyleName,
    showTimestamp,
    onShowSpeakerNameChange,
    onShowStyleNameChange,
    onShowTimestampChange,
  } = props;
  return (
    <div className="my-2 flex flex-wrap items-center gap-x-[0.6rem] gap-y-[0.35rem] text-[0.9rem] text-ctp-subtext sm:gap-x-4 sm:gap-y-2">
      <label htmlFor="history-size">履歴上限</label>
      <select
        id="history-size"
        value={historySize}
        onChange={(e) => onHistorySizeChange(Number(e.target.value))}
        className="rounded border border-ctp-overlay bg-ctp-base px-[0.4rem] py-[0.2rem] text-ctp-text"
      >
        {historySizeOptions.map((size) => (
          <option key={size} value={size}>
            {size}
          </option>
        ))}
      </select>
      <CheckboxToggle
        label="話者名"
        checked={showSpeakerName}
        onChange={onShowSpeakerNameChange}
      />
      <CheckboxToggle
        label="スタイル"
        checked={showStyleName}
        onChange={onShowStyleNameChange}
      />
      <CheckboxToggle
        label="時刻"
        checked={showTimestamp}
        onChange={onShowTimestampChange}
      />
    </div>
  );
}
