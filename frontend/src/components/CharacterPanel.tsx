import type { CharacterEntry, TimelineEntry } from "../types/api";
import { LipSyncImage } from "./LipSyncImage";
import { useAudioVolume } from "../hooks/useAudioVolume";
import { useCharacterStage } from "../hooks/useCharacterStage";

interface CharacterPanelProps {
  hidden: boolean;
  characters: CharacterEntry[];
  playingClipId: number | null;
  entries: TimelineEntry[];
  audioRef: React.RefObject<HTMLAudioElement>;
}

const MULTI_SLOT_LAYOUT = [
  { slotIndex: 3, flip: true }, // 外側左
  { slotIndex: 1, flip: true }, // 内側左
  { slotIndex: 0, flip: false }, // 内側右
  { slotIndex: 2, flip: false }, // 外側右
] as const;

/**
 * CharacterPanel はキャラクタータブのメインコンポーネント。
 * 現在再生中のクリップから話者を特定し、マッチするキャラクター画像を表示する。
 * 複数キャラの場合は4スロットレイアウト、1キャラの場合は中央寄せで表示。
 */
export function CharacterPanel({
  hidden,
  characters,
  playingClipId,
  entries,
  audioRef,
}: CharacterPanelProps) {
  const volume = useAudioVolume(audioRef, !hidden);
  const { slots, isMultiSlot } = useCharacterStage(
    playingClipId,
    entries,
    characters,
  );

  const playingClipEntry = playingClipId
    ? entries.find((e) => e.kind === "clip" && e.id === playingClipId)
    : null;

  const playingClipData =
    playingClipEntry && playingClipEntry.kind === "clip"
      ? playingClipEntry
      : null;

  function isPlayingCharacter(character: CharacterEntry): boolean {
    return (
      playingClipData !== null &&
      character.speakerName === playingClipData.speakerName &&
      character.styleName === playingClipData.styleName
    );
  }

  function getCharacterImageUrls(character: CharacterEntry): {
    closed: string;
    open: string;
  } {
    return {
      closed: `/assets/images/characters/${encodeURIComponent(
        character.mouthClosed,
      )}`,
      open: `/assets/images/characters/${encodeURIComponent(
        character.mouthOpen,
      )}`,
    };
  }

  const placeholderStyle: React.CSSProperties = {
    aspectRatio: "3 / 4",
    backgroundColor: "var(--ctp-base)",
  };

  if (hidden) {
    return null;
  }

  return (
    <section
      id="panel-character"
      role="tabpanel"
      aria-labelledby="tab-character"
      className="flex h-full flex-col gap-4"
    >
      <div className="flex min-h-0 flex-1 items-center justify-center rounded-md bg-ctp-surface p-4">
        {!isMultiSlot ? (
          // Single character center layout
          slots[0] ? (
            <LipSyncImage
              mouthClosedUrl={getCharacterImageUrls(slots[0].character).closed}
              mouthOpenUrl={getCharacterImageUrls(slots[0].character).open}
              volume={isPlayingCharacter(slots[0].character) ? volume : 0}
            />
          ) : (
            <div
              className="h-full w-auto max-w-full"
              style={placeholderStyle}
            />
          )
        ) : (
          // Multi-slot 4-character layout
          <div className="grid h-full w-full grid-cols-4 gap-2">
            {MULTI_SLOT_LAYOUT.map(({ slotIndex, flip }) => {
              const slot = slots[slotIndex];
              const wrapperStyle: React.CSSProperties = flip
                ? { transform: "scaleX(-1)" }
                : {};

              return (
                <div key={slotIndex} style={wrapperStyle}>
                  {slot ? (
                    <LipSyncImage
                      mouthClosedUrl={
                        getCharacterImageUrls(slot.character).closed
                      }
                      mouthOpenUrl={getCharacterImageUrls(slot.character).open}
                      volume={isPlayingCharacter(slot.character) ? volume : 0}
                    />
                  ) : (
                    <div className="h-full w-full" style={placeholderStyle} />
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>

      <div className="flex max-h-32 min-h-[4rem] shrink-0 items-center overflow-y-auto rounded-md bg-ctp-surface p-4">
        {playingClipData ? (
          <p className="break-words text-lg font-medium text-ctp-text">
            {playingClipData.text}
          </p>
        ) : (
          <p className="text-ctp-subtext">
            クリップを再生するとセリフが表示されます
          </p>
        )}
      </div>
    </section>
  );
}
