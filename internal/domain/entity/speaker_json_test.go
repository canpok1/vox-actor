package entity_test

import (
	"testing"

	"github.com/canpok1/vox-actor/internal/domain/entity"
)

func TestParseSpeakerJSON_Valid(t *testing.T) {
	input := `{
		"speakerName": "ずんだもん",
		"styles": [
			{ "styleName": "ノーマル", "mouthClosed": "normal/close.png", "mouthOpened": "normal/open.png" },
			{ "styleName": "あまあま", "mouthClosed": "amaama/close.png", "mouthOpened": "amaama/open.png" }
		]
	}`

	got, err := entity.ParseSpeakerJSON([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.SpeakerName != "ずんだもん" {
		t.Errorf("SpeakerName = %q, want %q", got.SpeakerName, "ずんだもん")
	}
	if len(got.Styles) != 2 {
		t.Fatalf("len(Styles) = %d, want 2", len(got.Styles))
	}
	if got.Styles[0].StyleName != "ノーマル" {
		t.Errorf("Styles[0].StyleName = %q, want %q", got.Styles[0].StyleName, "ノーマル")
	}
	if got.Styles[0].MouthClosed != "normal/close.png" {
		t.Errorf("Styles[0].MouthClosed = %q, want %q", got.Styles[0].MouthClosed, "normal/close.png")
	}
	if got.Styles[0].MouthOpened != "normal/open.png" {
		t.Errorf("Styles[0].MouthOpened = %q, want %q", got.Styles[0].MouthOpened, "normal/open.png")
	}
}

func TestParseSpeakerJSON_InvalidJSON(t *testing.T) {
	_, err := entity.ParseSpeakerJSON([]byte(`{invalid}`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestValidateSpeakerJSON_Valid(t *testing.T) {
	s := &entity.SpeakerJSON{
		SpeakerName: "ずんだもん",
		Styles: []entity.CharacterStyle{
			{StyleName: "ノーマル", MouthClosed: "normal/close.png", MouthOpened: "normal/open.png"},
		},
	}
	if err := s.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestValidateSpeakerJSON_MissingSpeakerName(t *testing.T) {
	s := &entity.SpeakerJSON{
		SpeakerName: "",
		Styles: []entity.CharacterStyle{
			{StyleName: "ノーマル", MouthClosed: "normal/close.png", MouthOpened: "normal/open.png"},
		},
	}
	if err := s.Validate(); err == nil {
		t.Error("expected validation error for missing speakerName, got nil")
	}
}

func TestValidateSpeakerJSON_MissingStyleName(t *testing.T) {
	s := &entity.SpeakerJSON{
		SpeakerName: "ずんだもん",
		Styles: []entity.CharacterStyle{
			{StyleName: "", MouthClosed: "normal/close.png", MouthOpened: "normal/open.png"},
		},
	}
	if err := s.Validate(); err == nil {
		t.Error("expected validation error for missing styleName, got nil")
	}
}

func TestValidateSpeakerJSON_MissingMouthClosed(t *testing.T) {
	s := &entity.SpeakerJSON{
		SpeakerName: "ずんだもん",
		Styles: []entity.CharacterStyle{
			{StyleName: "ノーマル", MouthClosed: "", MouthOpened: "normal/open.png"},
		},
	}
	if err := s.Validate(); err == nil {
		t.Error("expected validation error for missing mouthClosed, got nil")
	}
}

func TestValidateSpeakerJSON_MissingMouthOpened(t *testing.T) {
	s := &entity.SpeakerJSON{
		SpeakerName: "ずんだもん",
		Styles: []entity.CharacterStyle{
			{StyleName: "ノーマル", MouthClosed: "normal/close.png", MouthOpened: ""},
		},
	}
	if err := s.Validate(); err == nil {
		t.Error("expected validation error for missing mouthOpened, got nil")
	}
}

func TestValidateSpeakerJSON_DuplicateStyleName(t *testing.T) {
	s := &entity.SpeakerJSON{
		SpeakerName: "ずんだもん",
		Styles: []entity.CharacterStyle{
			{StyleName: "ノーマル", MouthClosed: "a/close.png", MouthOpened: "a/open.png"},
			{StyleName: "ノーマル", MouthClosed: "b/close.png", MouthOpened: "b/open.png"},
		},
	}
	if err := s.Validate(); err != nil {
		t.Errorf("duplicate styleName should be valid, got error: %v", err)
	}
}

func TestParseSpeakerJSON_NewSchema_WithProfile(t *testing.T) {
	input := `{
		"name": "ずんだもん",
		"profile": {
			"pronoun": "ボク",
			"speechSuffix": ["〜のだ", "〜なのだ"],
			"personality": ["元気", "明るい"],
			"speakers": {
				"ノーマル": 3,
				"あまあま": 1
			},
			"descriptionPath": "character.md"
		},
		"styles": [
			{ "styleName": "ノーマル", "mouthClosed": "normal/close.png", "mouthOpened": "normal/open.png" }
		]
	}`

	got, err := entity.ParseSpeakerJSON([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "ずんだもん" {
		t.Errorf("Name = %q, want %q", got.Name, "ずんだもん")
	}
	if got.Profile == nil {
		t.Fatal("Profile is nil, want non-nil")
	}
	if got.Profile.Pronoun != "ボク" {
		t.Errorf("Profile.Pronoun = %q, want %q", got.Profile.Pronoun, "ボク")
	}
	if len(got.Profile.SpeechSuffix) != 2 {
		t.Errorf("len(Profile.SpeechSuffix) = %d, want 2", len(got.Profile.SpeechSuffix))
	}
	if len(got.Profile.Personality) != 2 {
		t.Errorf("len(Profile.Personality) = %d, want 2", len(got.Profile.Personality))
	}
	if got.Profile.Speakers["ノーマル"] != 3 {
		t.Errorf("Profile.Speakers[ノーマル] = %d, want 3", got.Profile.Speakers["ノーマル"])
	}
	if got.Profile.DescriptionPath != "character.md" {
		t.Errorf("Profile.DescriptionPath = %q, want %q", got.Profile.DescriptionPath, "character.md")
	}
}

func TestGetSpeakerName_Fallback(t *testing.T) {
	tests := []struct {
		name     string
		sj       *entity.SpeakerJSON
		expected string
	}{
		{
			name:     "prefer Name over SpeakerName",
			sj:       &entity.SpeakerJSON{Name: "new", SpeakerName: "old"},
			expected: "new",
		},
		{
			name:     "fallback to SpeakerName when Name is empty",
			sj:       &entity.SpeakerJSON{Name: "", SpeakerName: "old"},
			expected: "old",
		},
		{
			name:     "return empty when both are empty",
			sj:       &entity.SpeakerJSON{Name: "", SpeakerName: ""},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.sj.GetSpeakerName()
			if got != tt.expected {
				t.Errorf("GetSpeakerName() = %q, want %q", got, tt.expected)
			}
		})
	}
}
