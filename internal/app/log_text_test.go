package app

import "testing"

func TestTruncateAndEscapeText(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "短いテキストはそのまま返す",
			in:   "短いセリフ",
			want: "短いセリフ",
		},
		{
			name: "ちょうど15文字は省略記号を付けない",
			in:   "あいうえおかきくけこさしすせそ",
			want: "あいうえおかきくけこさしすせそ",
		},
		{
			name: "16文字は先頭15文字に切り詰めて省略記号を付ける",
			in:   "あいうえおかきくけこさしすせそた",
			want: "あいうえおかきくけこさしすせそ...",
		},
		{
			name: "空文字はそのまま空",
			in:   "",
			want: "",
		},
		{
			name: "LFはエスケープされる",
			in:   "abc\ndef",
			want: `abc\ndef`,
		},
		{
			name: "CRもエスケープされる",
			in:   "abc\rdef",
			want: `abc\rdef`,
		},
		{
			name: "CRLFは両方エスケープされる",
			in:   "abc\r\ndef",
			want: `abc\r\ndef`,
		},
		{
			name: "改行を含む長文はrune15文字で切ってからエスケープ",
			in:   "あいうえおかきくけこさし\nすせそたちつ",
			want: `あいうえおかきくけこさし\nすせ...`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := truncateAndEscapeText(c.in)
			if got != c.want {
				t.Errorf("truncateAndEscapeText(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
