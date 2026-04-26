import type { CharacterEntry } from "../types/api";
import { LipSyncImage } from "./LipSyncImage";
import { useAudioVolume } from "../hooks/useAudioVolume";

interface CharacterPanelProps {
  hidden: boolean;
  characters: CharacterEntry[];
  playingClipId: number | null;
  entries: Array<{ kind: "clip"; id: number; text: string; speakerName: string; styleName: string } | { kind: "error" }>;
  audioRef: React.RefObject<HTMLAudioElement>;
}

/**
 * CharacterPanel はキャラクタータブのメインコンポーネント。
 * 現在再生中のクリップから話者を特定し、マッチするキャラクター画像を
 * 音量に応じて口パクさせながら表示する。
 */
export function CharacterPanel({
  hidden,
  characters,
  playingClipId,
  entries,
  audioRef,
}: CharacterPanelProps) {
  const volume = useAudioVolume(audioRef, !hidden);

  const playingClip = playingClipId
    ? entries.find((e) => e.kind === "clip" && e.id === playingClipId)
    : null;

  const playingClipData =
    playingClip && playingClip.kind === "clip" ? playingClip : null;

  const matchedCharacter = playingClipData
    ? characters.find(
        (c) =>
          c.speakerName === playingClipData.speakerName &&
          c.styleName === playingClipData.styleName
      )
    : null;

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
        {matchedCharacter ? (
          <LipSyncImage
            mouthClosedUrl={`/assets/images/characters/${encodeURIComponent(
              matchedCharacter.mouthClosed
            )}`}
            mouthOpenUrl={`/assets/images/characters/${encodeURIComponent(
              matchedCharacter.mouthOpen
            )}`}
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

      <div className="flex max-h-32 min-h-[4rem] shrink-0 items-center overflow-y-auto rounded-md bg-ctp-surface p-4">
        {playingClipData ? (
          <p className="break-words text-lg font-medium text-ctp-text">
            {playingClipData.text}
          </p>
        ) : (
          <p className="text-ctp-subtext">クリップを再生するとセリフが表示されます</p>
        )}
      </div>
    </section>
  );
}
