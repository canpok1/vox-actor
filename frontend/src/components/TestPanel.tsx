import { useMemo } from "react";
import type { Speaker, CharacterEntry } from "../types/api";
import { TestControls } from "./TestControls";
import { LipSyncImage } from "./LipSyncImage";

interface TestPanelProps {
  hidden: boolean;
  speakers: Speaker[];
  selectedSpeakerId: string;
  onSpeakerChange: (id: string) => void;
  onPlay: () => void;
  error: string;
  charactersEnabled: boolean;
  characters: CharacterEntry[];
  volume: number;
}

export function TestPanel({
  hidden,
  speakers,
  selectedSpeakerId,
  onSpeakerChange,
  onPlay,
  error,
  charactersEnabled,
  characters,
  volume,
}: TestPanelProps) {

  const selectedSpeaker = useMemo(
    () => speakers.find((s) => s.id === Number(selectedSpeakerId)) ?? null,
    [speakers, selectedSpeakerId],
  );

  const matchedCharacter = useMemo(
    () =>
      selectedSpeaker
        ? (characters.find(
            (c) =>
              c.speakerName === selectedSpeaker.speakerName &&
              c.styleName === selectedSpeaker.styleName,
          ) ?? null)
        : null,
    [selectedSpeaker, characters],
  );

  return (
    <section
      id="panel-test"
      role="tabpanel"
      aria-labelledby="tab-test"
      hidden={hidden}
      className={charactersEnabled ? "flex h-full flex-col gap-4" : "mt-2"}
    >
      <TestControls
        speakers={speakers}
        selectedSpeakerId={selectedSpeakerId}
        onSpeakerChange={onSpeakerChange}
        onPlay={onPlay}
        error={error}
      />
      {charactersEnabled && (
        <div className="flex min-h-0 flex-1 items-center justify-center rounded-md bg-ctp-surface p-4">
          {matchedCharacter ? (
            <LipSyncImage
              mouthClosedUrl={`/assets/images/characters/${encodeURIComponent(matchedCharacter.mouthClosed)}`}
              mouthOpenUrl={`/assets/images/characters/${encodeURIComponent(matchedCharacter.mouthOpen)}`}
              volume={volume}
            />
          ) : (
            <div
              className="h-full w-auto max-w-full"
              style={{
                aspectRatio: "3 / 4",
                backgroundColor: "var(--ctp-base)",
              }}
            />
          )}
        </div>
      )}
    </section>
  );
}
