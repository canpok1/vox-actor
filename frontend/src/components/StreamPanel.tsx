import type { ClipEvent } from "../types/api";
import { Timeline } from "./Timeline";
import { TimelineControls } from "./TimelineControls";

interface StreamPanelProps {
  hidden: boolean;
  clips: ClipEvent[];
  playingClipId: number | null;
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

export function StreamPanel(props: StreamPanelProps) {
  const {
    hidden,
    clips,
    playingClipId,
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
    <section
      id="panel-stream"
      role="tabpanel"
      aria-labelledby="tab-stream"
      hidden={hidden}
      className="mt-2"
    >
      <TimelineControls
        historySize={historySize}
        historySizeOptions={historySizeOptions}
        onHistorySizeChange={onHistorySizeChange}
        showSpeakerName={showSpeakerName}
        showStyleName={showStyleName}
        showTimestamp={showTimestamp}
        onShowSpeakerNameChange={onShowSpeakerNameChange}
        onShowStyleNameChange={onShowStyleNameChange}
        onShowTimestampChange={onShowTimestampChange}
      />
      <Timeline
        clips={clips}
        playingClipId={playingClipId}
        showSpeakerName={showSpeakerName}
        showStyleName={showStyleName}
        showTimestamp={showTimestamp}
      />
    </section>
  );
}
