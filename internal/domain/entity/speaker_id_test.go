package entity_test

import (
	"testing"

	"github.com/canpok1/vox-actor/internal/domain/entity"
)

func TestNewSpeakerID_Valid(t *testing.T) {
	cases := []struct {
		name  string
		value int
	}{
		{"zero", 0},
		{"positive", 3},
		{"large", 9999},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := entity.NewSpeakerID(tc.value)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id.Value() != tc.value {
				t.Errorf("Value() = %d, want %d", id.Value(), tc.value)
			}
		})
	}
}

func TestNewSpeakerID_Negative(t *testing.T) {
	_, err := entity.NewSpeakerID(-1)
	if err == nil {
		t.Error("expected error for negative value, got nil")
	}
}

func TestSpeakerID_Value(t *testing.T) {
	id, err := entity.NewSpeakerID(5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Value() != 5 {
		t.Errorf("Value() = %d, want 5", id.Value())
	}
}
