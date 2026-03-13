package app

import (
	"github.com/canpok1/vox-actor/internal/domain/entity"
)

// ScriptReader は指定パスから台本を読み込むインターフェース。
type ScriptReader interface {
	// Read は指定パスから台本を読み込む。
	// パスがファイルの場合はそのファイルのみ、ディレクトリの場合は.txtファイルを辞書順で返す。
	Read(path string) ([]entity.Script, error)
}
