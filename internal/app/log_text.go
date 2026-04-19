package app

import (
	"strconv"
	"strings"
)

// logTextMaxRunes はログ出力時にtextを切り詰めるrune数の上限。
const logTextMaxRunes = 15

// truncateAndEscapeText はログ表示用に text を整形する。
// rune数が logTextMaxRunes を超える場合は先頭をその長さで切り詰め末尾に "..." を付与し、
// 残った文字列に含まれる改行(\n, \r)を可視化のためバックスラッシュ表記へ置換する。
func truncateAndEscapeText(text string) string {
	runes := []rune(text)
	truncated := false
	if len(runes) > logTextMaxRunes {
		runes = runes[:logTextMaxRunes]
		truncated = true
	}
	s := string(runes)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	if truncated {
		s += "..."
	}
	return s
}

// formatFloatPtr は *float64 をログ表示用文字列に整形する。
// nil の場合は "default" を返す。
func formatFloatPtr(v *float64) string {
	if v == nil {
		return "default"
	}
	return strconv.FormatFloat(*v, 'g', -1, 64)
}

// dryRunPlaybackAttrs はdry-runモードの playback completed ログに付与する属性セットを返す。
// 呼び出し側で path などの属性を前後に追加できるよう []any を返す。
func dryRunPlaybackAttrs(text string, speakerID int, speed, pitch, intonation *float64) []any {
	return []any{
		"text", truncateAndEscapeText(text),
		"speaker", speakerID,
		"speed", formatFloatPtr(speed),
		"pitch", formatFloatPtr(pitch),
		"intonation", formatFloatPtr(intonation),
	}
}
