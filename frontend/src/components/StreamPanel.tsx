import type { CharacterEntry, ClipEvent, TimelineEntry } from "../types/api";
import type { CharacterStageSlot } from "../hooks/useCharacterStage";
import { useCharacterStage } from "../hooks/useCharacterStage";
import { LipSyncImage } from "./LipSyncImage";
import { Timeline } from "./Timeline";
import { TimelineControls } from "./TimelineControls";

interface StreamPanelProps {
  hidden: boolean;
  entries: TimelineEntry[];
  playingClipTimestamp: number | null;
  historySize: number;
  historySizeOptions: readonly number[];
  onHistorySizeChange: (value: number) => void;
  showSpeakerName: boolean;
  showStyleName: boolean;
  showTimestamp: boolean;
  onShowSpeakerNameChange: (value: boolean) => void;
  onShowStyleNameChange: (value: boolean) => void;
  onShowTimestampChange: (value: boolean) => void;
  characters: CharacterEntry[];
  charactersEnabled: boolean;
  showCharacters: boolean;
  onShowCharactersChange: (value: boolean) => void;
  volume: number;
  onReplay?: (entry: ClipEvent & { kind: "clip" }) => void;
}

const MULTI_SLOT_LAYOUT = [
  { slotIndex: 3, flip: true }, // 外側左
  { slotIndex: 1, flip: true }, // 内側左
  { slotIndex: 0, flip: false }, // 内側右
  { slotIndex: 2, flip: false }, // 外側右
] as const;

type FilledSlot = { slotIndex: number; flip: boolean; slot: CharacterStageSlot };

const PLACEHOLDER_STYLE: React.CSSProperties = {
  aspectRatio: "3 / 4",
  backgroundColor: "var(--ctp-base)",
};

function getCharacterImageUrls(character: CharacterEntry): {
  closed: string;
  open: string;
} {
  return {
    closed: `/assets/images/${encodeURIComponent(character.mouthClosed)}`,
    open: `/assets/images/${encodeURIComponent(character.mouthOpen)}`,
  };
}

export function StreamPanel(props: StreamPanelProps) {
  const {
    hidden,
    entries,
    playingClipTimestamp,
    historySize,
    historySizeOptions,
    onHistorySizeChange,
    showSpeakerName,
    showStyleName,
    showTimestamp,
    onShowSpeakerNameChange,
    onShowStyleNameChange,
    onShowTimestampChange,
    characters,
    charactersEnabled,
    showCharacters,
    onShowCharactersChange,
    volume,
    onReplay,
  } = props;

  const { slots, isMultiSlot, lastClipTimestamp } = useCharacterStage(
    playingClipTimestamp,
    entries,
    characters,
  );

  const hasCharactersOnStage = slots.some((s) => s !== null);
  const isCharacterMode = showCharacters && hasCharactersOnStage;

  const playingClipData = playingClipTimestamp
    ? (entries.find(
        (e) => e.kind === "clip" && e.timestamp === playingClipTimestamp,
      ) ?? null)
    : null;

  function isPlayingCharacter(character: CharacterEntry): boolean {
    return (
      playingClipData !== null &&
      playingClipData.kind === "clip" &&
      character.speakerName === playingClipData.speakerName &&
      character.styleName === playingClipData.styleName
    );
  }

  const singleSlotChar = !isMultiSlot ? slots[0] : null;
  const singleSlotUrls = singleSlotChar
    ? getCharacterImageUrls(singleSlotChar.character)
    : null;

  const filledSlots: FilledSlot[] = !isMultiSlot
    ? []
    : MULTI_SLOT_LAYOUT.flatMap(({ slotIndex, flip }) => {
        const slot = slots[slotIndex];
        return slot ? [{ slotIndex, flip, slot }] : [];
      });

  return (
    <section
      id="panel-stream"
      role="tabpanel"
      aria-labelledby="tab-stream"
      hidden={hidden}
      className={isCharacterMode ? "mt-2 flex h-full flex-col gap-2" : "mt-2"}
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
        charactersEnabled={charactersEnabled}
        showCharacters={showCharacters}
        onShowCharactersChange={onShowCharactersChange}
      />
      {isCharacterMode && (
        <div className="flex min-h-0 flex-1 items-center justify-center rounded-md bg-ctp-surface p-4">
          {!isMultiSlot ? (
            singleSlotUrls && singleSlotChar ? (
              <LipSyncImage
                mouthClosedUrl={singleSlotUrls.closed}
                mouthOpenUrl={singleSlotUrls.open}
                volume={
                  isPlayingCharacter(singleSlotChar.character) ? volume : 0
                }
              />
            ) : (
              <div
                className="h-full w-auto max-w-full"
                style={PLACEHOLDER_STYLE}
              />
            )
          ) : (
            <>
              {/* Desktop layout: horizontal flex */}
              <div className="hidden h-full items-end justify-center gap-2 sm:flex">
                {filledSlots.map(({ slotIndex, flip, slot }) => {
                  const urls = getCharacterImageUrls(slot.character);
                  const wrapperStyle: React.CSSProperties = flip
                    ? { transform: "scaleX(-1)" }
                    : {};
                  const isPlaying = isPlayingCharacter(slot.character);
                  return (
                    <div
                      key={slotIndex}
                      className="relative h-full flex-shrink-0"
                      style={{
                        ...wrapperStyle,
                        maxWidth: "25%",
                        width: "auto",
                        zIndex: isPlaying ? 10 : 1,
                      }}
                    >
                      <LipSyncImage
                        mouthClosedUrl={urls.closed}
                        mouthOpenUrl={urls.open}
                        volume={isPlaying ? volume : 0}
                      />
                    </div>
                  );
                })}
              </div>

              {/* Mobile layout: absolute positioning with overlap */}
              <div className="relative h-full w-full sm:hidden">
                {filledSlots.map(({ slotIndex, flip, slot }, index) => {
                  const urls = getCharacterImageUrls(slot.character);
                  const isPlaying = isPlayingCharacter(slot.character);
                  const charCount = filledSlots.length;
                  const leftPercent = ((index + 1) / (charCount + 1)) * 100;
                  return (
                    <div
                      key={slotIndex}
                      className="absolute bottom-0 h-full"
                      style={{
                        left: `${leftPercent}%`,
                        transform: `translateX(-50%)${flip ? " scaleX(-1)" : ""}`,
                        maxWidth: "60%",
                        width: "auto",
                        zIndex: isPlaying ? 10 : 1,
                      }}
                    >
                      <LipSyncImage
                        mouthClosedUrl={urls.closed}
                        mouthOpenUrl={urls.open}
                        volume={isPlaying ? volume : 0}
                      />
                    </div>
                  );
                })}
              </div>
            </>
          )}
        </div>
      )}
      <Timeline
        entries={entries}
        playingClipTimestamp={playingClipTimestamp}
        lastPlayingClipTimestamp={lastClipTimestamp}
        showSpeakerName={showSpeakerName}
        showStyleName={showStyleName}
        showTimestamp={showTimestamp}
        isCharacterMode={isCharacterMode}
        onReplay={onReplay}
      />
    </section>
  );
}
