package entity_test

import (
	"testing"

	"github.com/canpok1/vox-actor/internal/domain/entity"
)

func TestNewHistoryEntry_StoresAllFields(t *testing.T) {
	speakerID := entity.MustNewSpeakerID(3)
	speed := ptr(1.2)
	pitch := ptr(0.05)
	intonation := ptr(1.5)

	entry := entity.NewHistoryEntry("テスト", "ずんだもん", "ノーマル", 1000, speakerID, speed, pitch, intonation)

	if entry.Text != "テスト" {
		t.Errorf("Text: got %q, want %q", entry.Text, "テスト")
	}
	if entry.SpeakerName != "ずんだもん" {
		t.Errorf("SpeakerName: got %q, want %q", entry.SpeakerName, "ずんだもん")
	}
	if entry.StyleName != "ノーマル" {
		t.Errorf("StyleName: got %q, want %q", entry.StyleName, "ノーマル")
	}
	if entry.Timestamp != 1000 {
		t.Errorf("Timestamp: got %d, want %d", entry.Timestamp, 1000)
	}
	if entry.SpeakerID != speakerID {
		t.Errorf("SpeakerID: got %v, want %v", entry.SpeakerID, speakerID)
	}
	if entry.Speed != speed {
		t.Errorf("Speed: got %v, want %v", entry.Speed, speed)
	}
	if entry.Pitch != pitch {
		t.Errorf("Pitch: got %v, want %v", entry.Pitch, pitch)
	}
	if entry.Intonation != intonation {
		t.Errorf("Intonation: got %v, want %v", entry.Intonation, intonation)
	}
}

// TestNewHistoryEntry_LegacyFormat は旧形式（SpeakerID=0, Speed/Pitch/Intonation=nil）が
// 有効な HistoryEntry として表現できることを確認する。
func TestNewHistoryEntry_LegacyFormat(t *testing.T) {
	speakerID := entity.MustNewSpeakerID(0)

	entry := entity.NewHistoryEntry("旧テキスト", "話者#0", "", 500, speakerID, nil, nil, nil)

	if entry.SpeakerID.Value() != 0 {
		t.Errorf("SpeakerID.Value(): got %d, want 0", entry.SpeakerID.Value())
	}
	if entry.Speed != nil {
		t.Errorf("Speed: got %v, want nil", entry.Speed)
	}
	if entry.Pitch != nil {
		t.Errorf("Pitch: got %v, want nil", entry.Pitch)
	}
	if entry.Intonation != nil {
		t.Errorf("Intonation: got %v, want nil", entry.Intonation)
	}
}
