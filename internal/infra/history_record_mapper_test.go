package infra

import (
	"testing"

	"github.com/canpok1/vox-actor/internal/domain/entity"
)

func TestHistoryRecord_ToDomain_FullFields(t *testing.T) {
	speed := 1.2
	pitch := 0.05
	intonation := 1.5
	rec := historyRecord{
		Text:        "テスト",
		SpeakerName: "ずんだもん",
		StyleName:   "ノーマル",
		Timestamp:   1000,
		SpeakerID:   3,
		Speed:       &speed,
		Pitch:       &pitch,
		Intonation:  &intonation,
	}

	entry := rec.toDomain()

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
		t.Errorf("Timestamp: got %d, want 1000", entry.Timestamp)
	}
	if entry.SpeakerID.Value() != 3 {
		t.Errorf("SpeakerID: got %d, want 3", entry.SpeakerID.Value())
	}
	if entry.Speed == nil || *entry.Speed != speed {
		t.Errorf("Speed: got %v, want %v", entry.Speed, &speed)
	}
	if entry.Pitch == nil || *entry.Pitch != pitch {
		t.Errorf("Pitch: got %v, want %v", entry.Pitch, &pitch)
	}
	if entry.Intonation == nil || *entry.Intonation != intonation {
		t.Errorf("Intonation: got %v, want %v", entry.Intonation, &intonation)
	}
}

// TestHistoryRecord_ToDomain_LegacyFormat は旧形式（SpeakerID=0, Speed/Pitch/Intonation=nil）の
// 変換を検証する。旧形式ファイルは json.Unmarshal でゼロ値・nil として読み込まれる。
func TestHistoryRecord_ToDomain_LegacyFormat(t *testing.T) {
	rec := historyRecord{
		Text:        "旧テキスト",
		SpeakerName: "話者#0",
		StyleName:   "",
		Timestamp:   500,
		SpeakerID:   0,
		Speed:       nil,
		Pitch:       nil,
		Intonation:  nil,
	}

	entry := rec.toDomain()

	if entry.SpeakerID.Value() != 0 {
		t.Errorf("SpeakerID: got %d, want 0", entry.SpeakerID.Value())
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

func TestHistoryRecordFromDomain_FullFields(t *testing.T) {
	speed := 1.2
	pitch := 0.05
	intonation := 1.5
	entry := entity.NewHistoryEntry("テスト", "ずんだもん", "ノーマル", 1000, entity.MustNewSpeakerID(3), &speed, &pitch, &intonation)

	rec := historyRecordFromDomain(entry)

	if rec.Text != "テスト" {
		t.Errorf("Text: got %q, want %q", rec.Text, "テスト")
	}
	if rec.SpeakerName != "ずんだもん" {
		t.Errorf("SpeakerName: got %q, want %q", rec.SpeakerName, "ずんだもん")
	}
	if rec.StyleName != "ノーマル" {
		t.Errorf("StyleName: got %q, want %q", rec.StyleName, "ノーマル")
	}
	if rec.Timestamp != 1000 {
		t.Errorf("Timestamp: got %d, want 1000", rec.Timestamp)
	}
	if rec.SpeakerID != 3 {
		t.Errorf("SpeakerID: got %d, want 3", rec.SpeakerID)
	}
	if rec.Speed == nil || *rec.Speed != speed {
		t.Errorf("Speed: got %v, want %v", rec.Speed, &speed)
	}
	if rec.Pitch == nil || *rec.Pitch != pitch {
		t.Errorf("Pitch: got %v, want %v", rec.Pitch, &pitch)
	}
	if rec.Intonation == nil || *rec.Intonation != intonation {
		t.Errorf("Intonation: got %v, want %v", rec.Intonation, &intonation)
	}
}

func TestHistoryEntryFromClipEvent_ConvertsAllFields(t *testing.T) {
	speed := 1.2
	pitch := 0.05
	intonation := 1.5
	ev := clipEvent{
		Text:        "テスト",
		SpeakerName: "ずんだもん",
		StyleName:   "ノーマル",
		Timestamp:   1000,
		SpeakerID:   3,
		Speed:       &speed,
		Pitch:       &pitch,
		Intonation:  &intonation,
	}

	entry := historyEntryFromClipEvent(ev)

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
		t.Errorf("Timestamp: got %d, want 1000", entry.Timestamp)
	}
	if entry.SpeakerID.Value() != 3 {
		t.Errorf("SpeakerID: got %d, want 3", entry.SpeakerID.Value())
	}
	if entry.Speed == nil || *entry.Speed != speed {
		t.Errorf("Speed: got %v, want %v", entry.Speed, &speed)
	}
	if entry.Pitch == nil || *entry.Pitch != pitch {
		t.Errorf("Pitch: got %v, want %v", entry.Pitch, &pitch)
	}
	if entry.Intonation == nil || *entry.Intonation != intonation {
		t.Errorf("Intonation: got %v, want %v", entry.Intonation, &intonation)
	}
}
