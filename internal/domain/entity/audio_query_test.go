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

	result := q.WithOverrides(nil, nil, nil)

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

	speed := 1.5
	result := q.WithOverrides(&speed, nil, nil)

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

	pitch := 0.5
	result := q.WithOverrides(nil, &pitch, nil)

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

	intonation := 2.0
	result := q.WithOverrides(nil, nil, &intonation)

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

	speed := 1.5
	pitch := 0.5
	intonation := 2.0
	result := q.WithOverrides(&speed, &pitch, &intonation)

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

	speed := 1.5
	pitch := 0.5
	intonation := 2.0
	_ = q.WithOverrides(&speed, &pitch, &intonation)

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
