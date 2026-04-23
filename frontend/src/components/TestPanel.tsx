import type { Speaker } from "../types/api";
import { TestControls } from "./TestControls";

interface TestPanelProps {
  hidden: boolean;
  speakers: Speaker[];
  selectedSpeakerId: string;
  onSpeakerChange: (id: string) => void;
  onPlay: () => void;
  error: string;
}

export function TestPanel({
  hidden,
  speakers,
  selectedSpeakerId,
  onSpeakerChange,
  onPlay,
  error,
}: TestPanelProps) {
  return (
    <section
      id="panel-test"
      role="tabpanel"
      aria-labelledby="tab-test"
      hidden={hidden}
      className="mt-2"
    >
      <TestControls
        speakers={speakers}
        selectedSpeakerId={selectedSpeakerId}
        onSpeakerChange={onSpeakerChange}
        onPlay={onPlay}
        error={error}
      />
    </section>
  );
}
