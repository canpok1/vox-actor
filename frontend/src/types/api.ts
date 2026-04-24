// サーバー (internal/infra/http_stream_player.go) と対応する型定義。
// Go 側構造体 (clipEvent / speakerJSON) の JSON タグと一致させること。
// どちらか一方を変更した場合は両方を同時に修正する。

export interface ClipEvent {
  id: number;
  url: string;
  text: string;
  speakerName: string;
  styleName: string;
  // Unix ms (UTC)
  timestamp: number;
}

export interface Speaker {
  id: number;
  speakerName: string;
  styleName: string;
}

// SSE の "clip" イベントを扱えるよう EventSourceEventMap を拡張する。
// これにより es.addEventListener("clip", (ev) => ...) の ev が MessageEvent<string> になる。
declare global {
  interface EventSourceEventMap {
    clip: MessageEvent<string>;
  }
}

export function isClipEvent(value: unknown): value is ClipEvent {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const v = value as Record<string, unknown>;
  return (
    typeof v.id === "number" &&
    typeof v.url === "string" &&
    typeof v.text === "string" &&
    typeof v.speakerName === "string" &&
    typeof v.styleName === "string" &&
    typeof v.timestamp === "number"
  );
}

export function isSpeaker(value: unknown): value is Speaker {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const v = value as Record<string, unknown>;
  return (
    typeof v.id === "number" &&
    typeof v.speakerName === "string" &&
    typeof v.styleName === "string"
  );
}

export function isSpeakerArray(value: unknown): value is Speaker[] {
  return Array.isArray(value) && value.every(isSpeaker);
}

export interface ApiStatus {
  silent: boolean;
  // 無音モード時の理由文面（改行を含む）。通常モードでは空文字。
  silentReason: string;
  speakers: Speaker[];
}

export function isApiStatus(value: unknown): value is ApiStatus {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const v = value as Record<string, unknown>;
  return (
    typeof v.silent === "boolean" &&
    typeof v.silentReason === "string" &&
    isSpeakerArray(v.speakers)
  );
}
