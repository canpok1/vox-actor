package entity_test

import (
	"testing"

	"github.com/canpok1/vox-actor/internal/domain/entity"
)

func TestScript_ResolveOverrides_ScriptTakesPrecedence(t *testing.T) {
	s := entity.Script{
		Overrides: entity.SynthOverrides{Speed: ptr(1.5), Pitch: ptr(0.1), Intonation: ptr(1.2)},
	}
	defaults := entity.SynthOverrides{Speed: ptr(1.0), Pitch: ptr(0.0), Intonation: ptr(1.0)}

	got := s.ResolveOverrides(defaults)

	if *got.Speed != 1.5 {
		t.Errorf("Speed = %v, want 1.5", *got.Speed)
	}
	if *got.Pitch != 0.1 {
		t.Errorf("Pitch = %v, want 0.1", *got.Pitch)
	}
	if *got.Intonation != 1.2 {
		t.Errorf("Intonation = %v, want 1.2", *got.Intonation)
	}
}

func TestScript_ResolveOverrides_NilScriptUsesDefaults(t *testing.T) {
	s := entity.Script{}
	defaults := entity.SynthOverrides{Speed: ptr(1.0), Pitch: ptr(0.0), Intonation: ptr(1.0)}

	got := s.ResolveOverrides(defaults)

	if *got.Speed != 1.0 {
		t.Errorf("Speed = %v, want 1.0", *got.Speed)
	}
	if *got.Pitch != 0.0 {
		t.Errorf("Pitch = %v, want 0.0", *got.Pitch)
	}
	if *got.Intonation != 1.0 {
		t.Errorf("Intonation = %v, want 1.0", *got.Intonation)
	}
}

func TestScript_ResolveOverrides_PartialOverride(t *testing.T) {
	s := entity.Script{
		Overrides: entity.SynthOverrides{Speed: ptr(1.5)},
	}
	defaults := entity.SynthOverrides{Speed: ptr(1.0), Pitch: ptr(0.0), Intonation: ptr(1.0)}

	got := s.ResolveOverrides(defaults)

	if *got.Speed != 1.5 {
		t.Errorf("Speed = %v, want 1.5", *got.Speed)
	}
	if *got.Pitch != 0.0 {
		t.Errorf("Pitch = %v, want 0.0", *got.Pitch)
	}
	if *got.Intonation != 1.0 {
		t.Errorf("Intonation = %v, want 1.0", *got.Intonation)
	}
}

func TestScript_ResolveSpeakerID_ScriptValueTakesPrecedence(t *testing.T) {
	id := 5
	s := entity.Script{SpeakerID: &id}

	got := s.ResolveSpeakerID(3)

	if got != 5 {
		t.Errorf("got %d, want 5", got)
	}
}

func TestScript_ResolveSpeakerID_NilUsesDefault(t *testing.T) {
	s := entity.Script{}

	got := s.ResolveSpeakerID(3)

	if got != 3 {
		t.Errorf("got %d, want 3", got)
	}
}
