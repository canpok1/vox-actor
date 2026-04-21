package entity

// Speaker はVOICEVOXエンジンの /speakers から取得できる話者情報を表す。
type Speaker struct {
	// Name は話者名（例: "ずんだもん"）。
	Name string
	// Styles はこの話者が持つスタイル一覧。
	Styles []SpeakerStyle
}

// SpeakerStyle は話者のスタイル情報を表す。
type SpeakerStyle struct {
	// ID はスタイルID。VOICEVOXの speaker クエリパラメータで使う数値はこれ。
	ID int
	// Name はスタイル名（例: "ノーマル"）。
	Name string
}

// SpeakerStyleInfo は SpeakerID（= スタイルID）から解決した話者名・スタイル名のペア。
type SpeakerStyleInfo struct {
	SpeakerName string
	StyleName   string
}

// BuildSpeakerStyleLookup は []Speaker から SpeakerID（= スタイルID）→ SpeakerStyleInfo の
// マップを構築する。複数 Speaker に同じスタイルIDが存在する場合は最初に現れた値が優先される。
func BuildSpeakerStyleLookup(speakers []Speaker) map[int]SpeakerStyleInfo {
	lookup := make(map[int]SpeakerStyleInfo)
	for _, sp := range speakers {
		for _, st := range sp.Styles {
			if _, exists := lookup[st.ID]; exists {
				continue
			}
			lookup[st.ID] = SpeakerStyleInfo{
				SpeakerName: sp.Name,
				StyleName:   st.Name,
			}
		}
	}
	return lookup
}
