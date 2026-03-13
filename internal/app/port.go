package app

import (
	"context"

	"github.com/canpok1/vox-actor/internal/domain/entity"
)

// VoicevoxClient はVOICEVOXエンジンとの通信を抽象化するインターフェース。
type VoicevoxClient interface {
	// HealthCheck はエンジンの疎通確認を行う。
	HealthCheck(ctx context.Context) error

	// CreateQuery はテキストから音声合成用クエリを生成する。
	CreateQuery(ctx context.Context, text string, speakerID int) (*entity.AudioQuery, error)

	// Synthesize は音声合成を実行し、WAV形式のバイト列を返す。
	// speed, pitch, intonation が指定された場合、AudioQueryの対応フィールドを上書きする。
	Synthesize(ctx context.Context, query *entity.AudioQuery, speakerID int, speed, pitch, intonation *float64) ([]byte, error)
}
