package app

import "context"

// AudioPlayer はWAVバイト列を再生するインターフェース。
type AudioPlayer interface {
	// Play はWAVバイト列をスピーカーから再生する。
	// 再生完了まで同期的に待機し、再生失敗時はエラーを返す。
	Play(ctx context.Context, wavData []byte) error
}
