// サーバー (internal/infra/http_stream_player.go) と対応する型定義。
// Go 側構造体 (clipEvent / errorEvent / speakerJSON) の JSON タグと一致させること。
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

// ErrorEventPayload は SSE "error" イベントで届くサーバー側エラーのペイロード。
// Go 側の errorEvent 構造体と対応する。synthesis/file/connection で含まれるフィールドが異なる。
export type ErrorCategory = "synthesis" | "file" | "connection";

export interface ErrorEventPayload {
  // clipEvent.id とは独立した連番。
  id: number;
  category: ErrorCategory;
  message: string;
  // Unix ms (UTC)
  timestamp: number;
  // 以下はカテゴリに応じて付与される（omitempty）。
  path?: string;
  text?: string;
  speakerName?: string;
  styleName?: string;
}

export interface Speaker {
  id: number;
  speakerName: string;
  styleName: string;
}

// SSE の "clip" / "error" イベントを扱えるよう EventSourceEventMap を拡張する。
// これにより es.addEventListener("clip" | "error", (ev) => ...) の ev が MessageEvent<string> になる。
// ※ "error" は EventSource ネイティブの接続失敗イベントとも衝突するため、
//   ハンドラ側で event instanceof MessageEvent か否かで分岐する必要がある。
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

export function isErrorEventPayload(
  value: unknown,
): value is ErrorEventPayload {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const v = value as Record<string, unknown>;
  if (
    typeof v.id !== "number" ||
    typeof v.message !== "string" ||
    typeof v.timestamp !== "number"
  ) {
    return false;
  }
  if (
    v.category !== "synthesis" &&
    v.category !== "file" &&
    v.category !== "connection"
  ) {
    return false;
  }
  if (v.path !== undefined && typeof v.path !== "string") return false;
  if (v.text !== undefined && typeof v.text !== "string") return false;
  if (v.speakerName !== undefined && typeof v.speakerName !== "string")
    return false;
  if (v.styleName !== undefined && typeof v.styleName !== "string")
    return false;
  return true;
}

// TimelineEntry はタイムラインに表示される項目の共通型。
// クリップとエラーを discriminated union で扱い、既存の履歴件数上限を共通の枠で適用する。
export type TimelineEntry =
  | ({ kind: "clip" } & ClipEvent)
  | ({ kind: "error" } & ErrorEventPayload);

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
