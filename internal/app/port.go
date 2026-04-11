package app

import (
	"context"

	"github.com/canpok1/vox-actor/internal/domain/entity"
)

// ScriptReader は指定パスから台本を読み込むインターフェース。
type ScriptReader interface {
	// Read は指定パスから台本を読み込む。
	// パスがファイルの場合はそのファイルのみ、ディレクトリの場合は対象拡張子(.txt, .json)のファイルを辞書順で返す。
	// .jsonファイルの場合は感情制御パラメータを含むScriptを返す。
	Read(path string) ([]entity.Script, error)
}

// FileMover はファイルを処理済みディレクトリに移動または削除するインターフェース。
type FileMover interface {
	// MoveToDone はファイルを親ディレクトリのdone/サブディレクトリに移動する。
	MoveToDone(path string) error
	// Delete はファイルを削除する。
	Delete(path string) error
}

// DirWatcher はディレクトリを監視して新規ファイルを検知するインターフェース。
type DirWatcher interface {
	// Watch はディレクトリを監視し、新規ファイルのパスをチャネルで返す。
	// コンテキストがキャンセルされると監視を停止してチャネルをクローズする。
	Watch(ctx context.Context, dir string) (<-chan string, <-chan error)
}

// VoicevoxClient はVOICEVOXエンジンとの通信を抽象化するインターフェース。
type VoicevoxClient interface {
	// HealthCheck はエンジンの疎通確認を行う。
	HealthCheck(ctx context.Context) error

	// CreateQuery はテキストから音声合成用クエリを生成する。
	CreateQuery(ctx context.Context, text string, speakerID int) (*entity.AudioQuery, error)

	// Synthesize は音声合成を実行し、WAV形式のバイト列を返す。
	Synthesize(ctx context.Context, query *entity.AudioQuery, speakerID int) ([]byte, error)
}
