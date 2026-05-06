package entity_test

import (
	"testing"
	"time"

	"github.com/canpok1/vox-actor/internal/domain/entity"
)

func TestNewPlayback(t *testing.T) {
	now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	p := entity.NewPlayback(3, now)

	if p.Status() != entity.PlaybackStatusPending {
		t.Errorf("expected status=pending, got %s", p.Status())
	}
	if p.ClipCount() != 3 {
		t.Errorf("expected clipCount=3, got %d", p.ClipCount())
	}
	if p.CompletedClips() != 0 {
		t.Errorf("expected completedClips=0, got %d", p.CompletedClips())
	}
	if !p.CreatedAt().Equal(now) {
		t.Errorf("expected createdAt=%v, got %v", now, p.CreatedAt())
	}
	if p.StartedAt() != nil {
		t.Errorf("expected startedAt=nil, got %v", p.StartedAt())
	}
	if p.FinishedAt() != nil {
		t.Errorf("expected finishedAt=nil, got %v", p.FinishedAt())
	}
}

func TestPlayback_MarkPlaying(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*entity.Playback)
		wantErr    bool
		wantStatus entity.PlaybackStatus
	}{
		{
			name:       "pending→playing は成功する",
			setup:      func(_ *entity.Playback) {},
			wantErr:    false,
			wantStatus: entity.PlaybackStatusPlaying,
		},
		{
			name: "playing→playing はエラー",
			setup: func(p *entity.Playback) {
				_ = p.MarkPlaying(1000)
			},
			wantErr:    true,
			wantStatus: entity.PlaybackStatusPlaying,
		},
		{
			name: "completed→playing はエラー",
			setup: func(p *entity.Playback) {
				_ = p.MarkCompleted(1000)
			},
			wantErr:    true,
			wantStatus: entity.PlaybackStatusCompleted,
		},
		{
			name: "failed→playing はエラー",
			setup: func(p *entity.Playback) {
				_ = p.MarkFailed("reason", 1000)
			},
			wantErr:    true,
			wantStatus: entity.PlaybackStatusFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := entity.NewPlayback(1, time.Now())
			tt.setup(p)

			err := p.MarkPlaying(2000)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if p.StartedAt() == nil {
					t.Error("expected startedAt to be set")
				} else if *p.StartedAt() != 2000 {
					t.Errorf("expected startedAt=2000, got %d", *p.StartedAt())
				}
			}
			if p.Status() != tt.wantStatus {
				t.Errorf("expected status=%s, got %s", tt.wantStatus, p.Status())
			}
		})
	}
}

func TestPlayback_MarkCompleted(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*entity.Playback)
		wantErr    bool
		wantStatus entity.PlaybackStatus
	}{
		{
			name:       "pending→completed は成功する（無音モード）",
			setup:      func(_ *entity.Playback) {},
			wantErr:    false,
			wantStatus: entity.PlaybackStatusCompleted,
		},
		{
			name: "playing→completed は成功する",
			setup: func(p *entity.Playback) {
				_ = p.MarkPlaying(1000)
			},
			wantErr:    false,
			wantStatus: entity.PlaybackStatusCompleted,
		},
		{
			name: "completed→completed はエラー",
			setup: func(p *entity.Playback) {
				_ = p.MarkCompleted(1000)
			},
			wantErr:    true,
			wantStatus: entity.PlaybackStatusCompleted,
		},
		{
			name: "failed→completed はエラー",
			setup: func(p *entity.Playback) {
				_ = p.MarkFailed("reason", 1000)
			},
			wantErr:    true,
			wantStatus: entity.PlaybackStatusFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := entity.NewPlayback(1, time.Now())
			tt.setup(p)

			err := p.MarkCompleted(2000)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if p.FinishedAt() == nil {
					t.Error("expected finishedAt to be set")
				} else if *p.FinishedAt() != 2000 {
					t.Errorf("expected finishedAt=2000, got %d", *p.FinishedAt())
				}
			}
			if p.Status() != tt.wantStatus {
				t.Errorf("expected status=%s, got %s", tt.wantStatus, p.Status())
			}
		})
	}
}

func TestPlayback_MarkFailed(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*entity.Playback)
		wantErr    bool
		wantStatus entity.PlaybackStatus
	}{
		{
			name:       "pending→failed は成功する",
			setup:      func(_ *entity.Playback) {},
			wantErr:    false,
			wantStatus: entity.PlaybackStatusFailed,
		},
		{
			name: "playing→failed は成功する",
			setup: func(p *entity.Playback) {
				_ = p.MarkPlaying(1000)
			},
			wantErr:    false,
			wantStatus: entity.PlaybackStatusFailed,
		},
		{
			name: "completed→failed はエラー",
			setup: func(p *entity.Playback) {
				_ = p.MarkCompleted(1000)
			},
			wantErr:    true,
			wantStatus: entity.PlaybackStatusCompleted,
		},
		{
			name: "failed→failed はエラー",
			setup: func(p *entity.Playback) {
				_ = p.MarkFailed("first", 1000)
			},
			wantErr:    true,
			wantStatus: entity.PlaybackStatusFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := entity.NewPlayback(1, time.Now())
			tt.setup(p)

			err := p.MarkFailed("test reason", 2000)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if p.FinishedAt() == nil {
					t.Error("expected finishedAt to be set")
				} else if *p.FinishedAt() != 2000 {
					t.Errorf("expected finishedAt=2000, got %d", *p.FinishedAt())
				}
				if p.Reason() != "test reason" {
					t.Errorf("expected reason='test reason', got %q", p.Reason())
				}
			}
			if p.Status() != tt.wantStatus {
				t.Errorf("expected status=%s, got %s", tt.wantStatus, p.Status())
			}
		})
	}
}

func TestPlayback_IncrementCompletedClips(t *testing.T) {
	p := entity.NewPlayback(3, time.Now())
	_ = p.MarkPlaying(1000)

	p.IncrementCompletedClips()
	p.IncrementCompletedClips()

	if p.CompletedClips() != 2 {
		t.Errorf("expected completedClips=2, got %d", p.CompletedClips())
	}
}
