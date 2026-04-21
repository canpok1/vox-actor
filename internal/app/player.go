package app

import "context"

// AudioPlayer はWAVバイト列を再生するインターフェース。
type AudioPlayer interface {
	// Play はWAVバイト列をスピーカーから再生する。
	// 再生完了まで同期的に待機し、再生失敗時はエラーを返す。
	Play(ctx context.Context, wavData []byte) error
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
}
