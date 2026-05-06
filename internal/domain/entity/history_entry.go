package entity

// HistoryEntry は再生履歴1件のドメインオブジェクト。
// 旧形式（SpeakerID=0, Speed/Pitch/Intonation=nil）も有効な状態として受け入れる。
// 旧形式は永続化ファイルに SpeakerID/Speed/Pitch/Intonation が欠落していた場合を指し、
// ゼロ値・nil として扱うことで後方互換を維持する。
type HistoryEntry struct {
	Text        string
	SpeakerName string
	StyleName   string
	Timestamp   int64
	SpeakerID   SpeakerID
	Speed       *float64
	Pitch       *float64
	Intonation  *float64
}

// NewHistoryEntry は HistoryEntry を生成する。
func NewHistoryEntry(text, speakerName, styleName string, timestamp int64, speakerID SpeakerID, speed, pitch, intonation *float64) HistoryEntry {
	return HistoryEntry{
		Text:        text,
		SpeakerName: speakerName,
		StyleName:   styleName,
		Timestamp:   timestamp,
		SpeakerID:   speakerID,
		Speed:       speed,
		Pitch:       pitch,
		Intonation:  intonation,
	}
}
