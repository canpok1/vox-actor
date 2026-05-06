package entity_test

import (
	"testing"

	"github.com/canpok1/vox-actor/internal/domain/entity"
)

func TestNewClip(t *testing.T) {
	speed := 1.2
	speakerID := entity.MustNewSpeakerID(3)

	clip := entity.NewClip("こんにちは", speakerID, &speed, nil, nil)

	if clip.Text != "こんにちは" {
		t.Errorf("expected text='こんにちは', got %q", clip.Text)
	}
	if clip.SpeakerID != speakerID {
		t.Errorf("expected speakerID=%v, got %v", speakerID, clip.SpeakerID)
	}
	if clip.Speed == nil || *clip.Speed != speed {
		t.Errorf("expected speed=1.2, got %v", clip.Speed)
	}
	if clip.Pitch != nil {
		t.Errorf("expected pitch=nil, got %v", clip.Pitch)
	}
	if clip.Intonation != nil {
		t.Errorf("expected intonation=nil, got %v", clip.Intonation)
	}
}
