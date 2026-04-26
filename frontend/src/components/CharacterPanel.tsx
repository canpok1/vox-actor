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
      className="panel"
      style={{ display: hidden ? "none" : "block" }}
    >
      <div className="flex flex-col gap-4">
        {/* キャラクター表示エリア: 3:4 アスペクト比 */}
        <div className="bg-ctp-surface rounded-md p-4">
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
            // マッチしないキャラクター時は背景のみ表示
            <div
              className="w-full"
              style={{
                aspectRatio: "3 / 4",
                backgroundColor: "var(--ctp-base)",
              }}
            />
          )}
        </div>

        {/* セリフ表示エリア */}
        <div className="bg-ctp-surface rounded-md p-4 min-h-[4rem] flex items-center">
          {playingClipData ? (
            <p className="text-lg font-medium text-ctp-text break-words">
              {playingClipData.text}
            </p>
          ) : (
            <p className="text-ctp-subtext">クリップを再生するとセリフが表示されます</p>
          )}
        </div>
      </div>
    </section>
  );
}
