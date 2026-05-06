package entity_test

import (
	"testing"

	"github.com/canpok1/vox-actor/internal/domain/entity"
)

func ptr(f float64) *float64 { return &f }

func TestSynthOverrides_MergeWith_SelfTakesPrecedence(t *testing.T) {
	self := entity.SynthOverrides{Speed: ptr(1.5), Pitch: ptr(0.1), Intonation: ptr(1.2)}
	defaults := entity.SynthOverrides{Speed: ptr(1.0), Pitch: ptr(0.0), Intonation: ptr(1.0)}

	got := self.MergeWith(defaults)

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

func TestSynthOverrides_MergeWith_NilSelfUsesDefault(t *testing.T) {
	self := entity.SynthOverrides{}
	defaults := entity.SynthOverrides{Speed: ptr(1.0), Pitch: ptr(0.0), Intonation: ptr(1.0)}

	got := self.MergeWith(defaults)

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

func TestSynthOverrides_MergeWith_PartialSelfOverridesPartially(t *testing.T) {
	self := entity.SynthOverrides{Speed: ptr(1.5)}
	defaults := entity.SynthOverrides{Speed: ptr(1.0), Pitch: ptr(0.0), Intonation: ptr(1.0)}

	got := self.MergeWith(defaults)

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

func TestSynthOverrides_MergeWith_BothNilResultsInNil(t *testing.T) {
	self := entity.SynthOverrides{}
	defaults := entity.SynthOverrides{}

	got := self.MergeWith(defaults)

	if got.Speed != nil {
		t.Errorf("Speed = %v, want nil", got.Speed)
	}
	if got.Pitch != nil {
		t.Errorf("Pitch = %v, want nil", got.Pitch)
	}
	if got.Intonation != nil {
		t.Errorf("Intonation = %v, want nil", got.Intonation)
	}
}

func TestSynthOverrides_MergeWith_DoesNotModifyOriginal(t *testing.T) {
	self := entity.SynthOverrides{Speed: ptr(1.5)}
	defaults := entity.SynthOverrides{Speed: ptr(1.0), Pitch: ptr(0.0)}

	_ = self.MergeWith(defaults)

	if *self.Speed != 1.5 {
		t.Errorf("self.Speed modified, got %v", *self.Speed)
	}
	if *defaults.Speed != 1.0 {
		t.Errorf("defaults.Speed modified, got %v", *defaults.Speed)
	}
}
