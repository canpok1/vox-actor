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

// ScriptWriter は entity.Script をファイルへ書き出すインターフェース。
type ScriptWriter interface {
	// Write は script を path に書き出す。
	// 拡張子で書き出し形式を判定する: .json / .jsonl は感情制御パラメータも含めて、それ以外は本文のみ書き出す。
	// 既存ファイルがある場合はリネームしてユニーク化する。
	// 戻り値は実際の書き出し先パス。
	Write(path string, script entity.Script) (string, error)
}

// ScriptBulkWriter は複数の entity.Script をファイルへ一括書き出すインターフェース。
type ScriptBulkWriter interface {
	// WriteAll は scripts を path に上書き書き出す。
	// 既存ファイルがある場合は上書きされる。
	// 戻り値は実際の書き出し先パス。
	WriteAll(path string, scripts []entity.Script) (string, error)
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

// GitCloner は git リポジトリを clone するインターフェース。
type GitCloner interface {
	// Clone は repoURL を destDir へ clone する。destDir は Clone が作成する。
	Clone(ctx context.Context, repoURL string, destDir string) error
}

// WavSaver は WAV データをファイルに保存するインターフェース。
type WavSaver interface {
	// Save は wavData を path に保存する。
	// 親ディレクトリが存在しない場合は自動作成する。
	// 既存ファイルがある場合は上書きする。
	Save(path string, wavData []byte) error
}

// VoicevoxClient はVOICEVOXエンジンとの通信を抽象化するインターフェース。
type VoicevoxClient interface {
	// HealthCheck はエンジンの疎通確認を行う。
	HealthCheck(ctx context.Context) error

	// CreateQuery はテキストから音声合成用クエリを生成する。
	CreateQuery(ctx context.Context, text string, speakerID int) (*entity.AudioQuery, error)

	// Synthesize は音声合成を実行し、WAV形式のバイト列を返す。
	Synthesize(ctx context.Context, query *entity.AudioQuery, speakerID int) ([]byte, error)

	// GetSpeakers はエンジンに登録された話者一覧（話者名・スタイルID・スタイル名）を取得する。
	GetSpeakers(ctx context.Context) ([]entity.Speaker, error)
}
