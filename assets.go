package main

import (
	"embed"
	"io/fs"
)

// all: プレフィックスにより、dot-prefix のファイル（.gitkeep 等）も埋め込み対象にする。
// これにより frontend/dist が未ビルドでも .gitkeep だけで embed 制約を満たし、
// `go run` / `go build` がフロントエンド未ビルドの状態で通るようになる。
//
//go:embed all:frontend/dist
var streamAssetsRoot embed.FS

// streamAssetsFS は frontend/dist を剥がしたビルド成果物の読み取り専用 FS を返す。
// Vite の出力する index.html と assets/<hash>.{js,css} を配信するために使う。
func streamAssetsFS() (fs.FS, error) {
	return fs.Sub(streamAssetsRoot, "frontend/dist")
}
