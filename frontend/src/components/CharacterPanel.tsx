import type { CharacterEntry, TimelineEntry } from "../types/api";
import { LipSyncImage } from "./LipSyncImage";
import { useCharacterStage } from "../hooks/useCharacterStage";

interface CharacterPanelProps {
  hidden: boolean;
  characters: CharacterEntry[];
  playingClipId: number | null;
  entries: TimelineEntry[];
  volume: number;
}

const MULTI_SLOT_LAYOUT = [
  { slotIndex: 3, flip: true }, // 外側左
  { slotIndex: 1, flip: true }, // 内側左
  { slotIndex: 0, flip: false }, // 内側右
  { slotIndex: 2, flip: false }, // 外側右
] as const;

function getCharacterImageUrls(character: CharacterEntry): {
  closed: string;
  open: string;
} {
  return {
    closed: `/assets/images/${encodeURIComponent(
      character.mouthClosed,
    )}`,
    open: `/assets/images/${encodeURIComponent(
      character.mouthOpen,
    )}`,
  };
}

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
  volume,
}: CharacterPanelProps) {
  const { slots, isMultiSlot, lastClipText } = useCharacterStage(
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

  const placeholderStyle: React.CSSProperties = {
    aspectRatio: "3 / 4",
    backgroundColor: "var(--ctp-base)",
  };

  if (hidden) {
    return null;
  }

  const singleSlotChar = !isMultiSlot ? slots[0] : null;
  const singleSlotUrls = singleSlotChar
    ? getCharacterImageUrls(singleSlotChar.character)
    : null;

  // マルチスロット時に実際に表示するキャラのスロット情報を抽出
  const filledSlots = !isMultiSlot
    ? []
    : MULTI_SLOT_LAYOUT.filter(({ slotIndex }) => slots[slotIndex] !== null).map(
        ({ slotIndex, flip }) => ({
          slotIndex,
          flip,
          slot: slots[slotIndex]!,
        }),
      );

  return (
    <section
      id="panel-character"
      role="tabpanel"
      aria-labelledby="tab-character"
      className="flex h-full flex-col gap-4"
    >
      <div className="flex min-h-0 flex-1 items-center justify-center rounded-md bg-ctp-surface p-4">
        {!isMultiSlot ? (
          singleSlotUrls && singleSlotChar ? (
            <LipSyncImage
              mouthClosedUrl={singleSlotUrls.closed}
              mouthOpenUrl={singleSlotUrls.open}
              volume={isPlayingCharacter(singleSlotChar.character) ? volume : 0}
            />
          ) : (
            <div
              className="h-full w-auto max-w-full"
              style={placeholderStyle}
            />
          )
        ) : (
          <>
            {/* Desktop layout: horizontal flex layout */}
            <div className="hidden sm:flex h-full items-end justify-center gap-2">
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
            <div className="sm:hidden relative w-full h-full">
              {filledSlots.map((_, index) => {
                const { slotIndex, flip, slot } = filledSlots[index];
                const urls = getCharacterImageUrls(slot.character);
                const isPlaying = isPlayingCharacter(slot.character);
                const charCount = filledSlots.length;
                // 位置: n人表示の場合、画面幅を(n+1)分割して、(index+1)/(n+1)の位置に配置
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

      <div className="flex h-32 shrink-0 items-center overflow-y-auto rounded-md bg-ctp-surface p-4">
        {lastClipText !== null ? (
          <p className="break-words text-lg font-medium text-ctp-text">
            {lastClipText}
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
