package main

import (
	"embed"
	"io/fs"
)

//go:embed frontend/dist
var streamAssetsRoot embed.FS

// streamAssetsFS は frontend/dist を剥がしたビルド成果物の読み取り専用 FS を返す。
// Vite の出力する index.html と assets/<hash>.{js,css} を配信するために使う。
func streamAssetsFS() (fs.FS, error) {
	return fs.Sub(streamAssetsRoot, "frontend/dist")
}
