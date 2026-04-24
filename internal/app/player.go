package app

import "context"

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
