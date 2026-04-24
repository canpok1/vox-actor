package app

import "context"

// StreamErrorCategory は配信画面に表示するサーバー側エラーの種別。
type StreamErrorCategory string

const (
	// StreamErrorCategorySynthesis は CreateQuery / Synthesize / Play など合成・再生系エラー。
	StreamErrorCategorySynthesis StreamErrorCategory = "synthesis"
	// StreamErrorCategoryFile は監視対象スクリプトファイルの読込・移動・削除の失敗。
	StreamErrorCategoryFile StreamErrorCategory = "file"
	// StreamErrorCategoryConnection は VOICEVOX エンジンへの接続に関わる失敗（HealthCheck 等）。
	StreamErrorCategoryConnection StreamErrorCategory = "connection"
)

// StreamError は配信画面に通知するサーバー側エラーの情報。
// Category が StreamErrorCategorySynthesis の場合のみ Text / SpeakerID が使われ、
// broadcaster 側で speakerLookup を参照して speakerName / styleName が解決される。
// file / connection カテゴリでは該当しないフィールドは無視される。
type StreamError struct {
	Category  StreamErrorCategory
	Message   string
	Path      string
	Text      string
	SpeakerID int
}

// ErrorBroadcaster は配信画面向けにサーバー側エラーをブロードキャストする。
// StreamPlayer 実装が同時に実装することを想定し、WatchUsecase からは
// textPlayer と同じく type assertion で取得する。
type ErrorBroadcaster interface {
	BroadcastError(e StreamError)
}

// PlayMeta は AudioPlayer.Play に付随させる補助情報。
// 再生の実装（ローカルスピーカー/HTTPストリーム等）によっては利用しない場合もある。
type PlayMeta struct {
	// Text は再生対象のセリフ本文。ブラウザUI等での表示に使用される。
	Text string
	// SpeakerID は再生に使ったVOICEVOXのスタイルID（解決後の値）。
	// HTTPStreamPlayer では話者名/スタイル名の解決キーとして利用される。
	SpeakerID int
}

// AudioPlayer はWAVバイト列を再生するインターフェース。
type AudioPlayer interface {
	// Play はWAVバイト列をスピーカーから再生する。
	// 再生完了まで同期的に待機し、再生失敗時はエラーを返す。
	// meta は再生に付随する補助情報（セリフ本文など）。実装により利用の有無が異なる。
	Play(ctx context.Context, wavData []byte, meta PlayMeta) error
}

// StreamPlayer は HTTP サーバー等のライフサイクルを持つ AudioPlayer のインターフェース。
type StreamPlayer interface {
	AudioPlayer
	// Start はサーバーを起動する。Play() を呼ぶ前に必ず呼び出す必要がある。
	Start(ctx context.Context) error
	// Shutdown はサーバーを停止する。
	Shutdown(ctx context.Context) error
	// Addr はサーバーがリッスンしているアドレスを返す。Start 前の値は未定義。
	Addr() string
	// SetSilent は無音モードと silentReason を設定する。Start 前に呼び出すこと。
	// reason が空文字なら通常モードとして扱う。
	SetSilent(reason string)
	// PlayText は WAV 合成をスキップしたセリフ配信を行う。
	// 無音モードでの利用を想定し、内部で固定待機してから return することで
	// タイムラインが一瞬で流れ去らないようにする。
	PlayText(ctx context.Context, meta PlayMeta) error
}
