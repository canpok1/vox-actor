package entity

// Script は読み込んだ台本テキストを表す。
type Script struct {
	// Path はファイルのパス。
	Path string
	// Text はファイルの内容。
	Text string
	// IsEmpty はファイルが空であるかを示す。
	IsEmpty bool
}
