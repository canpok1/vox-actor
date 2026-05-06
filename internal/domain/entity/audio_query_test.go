package entity_test

import (
	"testing"

	"github.com/canpok1/vox-actor/internal/domain/entity"
)

func TestAudioQuery_WithOverrides_AllNil(t *testing.T) {
	q := entity.AudioQuery{
		SpeedScale:      1.0,
		PitchScale:      0.0,
		IntonationScale: 1.0,
	}

	result := q.WithOverrides(entity.SynthOverrides{})

	if result.SpeedScale != 1.0 {
		t.Errorf("expected SpeedScale 1.0, got %f", result.SpeedScale)
	}
	if result.PitchScale != 0.0 {
		t.Errorf("expected PitchScale 0.0, got %f", result.PitchScale)
	}
	if result.IntonationScale != 1.0 {
		t.Errorf("expected IntonationScale 1.0, got %f", result.IntonationScale)
	}
}

func TestAudioQuery_WithOverrides_SpeedOnly(t *testing.T) {
	q := entity.AudioQuery{
		SpeedScale:      1.0,
		PitchScale:      0.0,
		IntonationScale: 1.0,
	}

	result := q.WithOverrides(entity.SynthOverrides{Speed: ptr(1.5)})

	if result.SpeedScale != 1.5 {
		t.Errorf("expected SpeedScale 1.5, got %f", result.SpeedScale)
	}
	if result.PitchScale != 0.0 {
		t.Errorf("expected PitchScale 0.0, got %f", result.PitchScale)
	}
	if result.IntonationScale != 1.0 {
		t.Errorf("expected IntonationScale 1.0, got %f", result.IntonationScale)
	}
}

func TestAudioQuery_WithOverrides_PitchOnly(t *testing.T) {
	q := entity.AudioQuery{
		SpeedScale:      1.0,
		PitchScale:      0.0,
		IntonationScale: 1.0,
	}

	result := q.WithOverrides(entity.SynthOverrides{Pitch: ptr(0.5)})

	if result.SpeedScale != 1.0 {
		t.Errorf("expected SpeedScale 1.0, got %f", result.SpeedScale)
	}
	if result.PitchScale != 0.5 {
		t.Errorf("expected PitchScale 0.5, got %f", result.PitchScale)
	}
	if result.IntonationScale != 1.0 {
		t.Errorf("expected IntonationScale 1.0, got %f", result.IntonationScale)
	}
}

func TestAudioQuery_WithOverrides_IntonationOnly(t *testing.T) {
	q := entity.AudioQuery{
		SpeedScale:      1.0,
		PitchScale:      0.0,
		IntonationScale: 1.0,
	}

	result := q.WithOverrides(entity.SynthOverrides{Intonation: ptr(2.0)})

	if result.SpeedScale != 1.0 {
		t.Errorf("expected SpeedScale 1.0, got %f", result.SpeedScale)
	}
	if result.PitchScale != 0.0 {
		t.Errorf("expected PitchScale 0.0, got %f", result.PitchScale)
	}
	if result.IntonationScale != 2.0 {
		t.Errorf("expected IntonationScale 2.0, got %f", result.IntonationScale)
	}
}

func TestAudioQuery_WithOverrides_AllParams(t *testing.T) {
	q := entity.AudioQuery{
		SpeedScale:      1.0,
		PitchScale:      0.0,
		IntonationScale: 1.0,
		VolumeScale:     1.0,
	}

	result := q.WithOverrides(entity.SynthOverrides{Speed: ptr(1.5), Pitch: ptr(0.5), Intonation: ptr(2.0)})

	if result.SpeedScale != 1.5 {
		t.Errorf("expected SpeedScale 1.5, got %f", result.SpeedScale)
	}
	if result.PitchScale != 0.5 {
		t.Errorf("expected PitchScale 0.5, got %f", result.PitchScale)
	}
	if result.IntonationScale != 2.0 {
		t.Errorf("expected IntonationScale 2.0, got %f", result.IntonationScale)
	}
	// 上書きしていないフィールドは元の値のまま
	if result.VolumeScale != 1.0 {
		t.Errorf("expected VolumeScale 1.0, got %f", result.VolumeScale)
	}
}

func TestAudioQuery_WithOverrides_DoesNotModifyOriginal(t *testing.T) {
	q := entity.AudioQuery{
		SpeedScale:      1.0,
		PitchScale:      0.0,
		IntonationScale: 1.0,
	}

	_ = q.WithOverrides(entity.SynthOverrides{Speed: ptr(1.5), Pitch: ptr(0.5), Intonation: ptr(2.0)})

	// 元のAudioQueryが変更されていないことを確認
	if q.SpeedScale != 1.0 {
		t.Errorf("original SpeedScale should not be modified, got %f", q.SpeedScale)
	}
	if q.PitchScale != 0.0 {
		t.Errorf("original PitchScale should not be modified, got %f", q.PitchScale)
	}
	if q.IntonationScale != 1.0 {
		t.Errorf("original IntonationScale should not be modified, got %f", q.IntonationScale)
	}
}
