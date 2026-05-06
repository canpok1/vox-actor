package entity

import (
	"fmt"
	"time"
)

// PlaybackStatus は再生バッチの状態を表す。
type PlaybackStatus string

const (
	PlaybackStatusPending   PlaybackStatus = "pending"
	PlaybackStatusPlaying   PlaybackStatus = "playing"
	PlaybackStatusCompleted PlaybackStatus = "completed"
	PlaybackStatusFailed    PlaybackStatus = "failed"
)

// Playback は1再生バッチの状態遷移を管理するエンティティ。
type Playback struct {
	status         PlaybackStatus
	reason         string
	createdAt      time.Time
	clipCount      int
	completedClips int
	startedAt      *int64 // Unix milliseconds
	finishedAt     *int64 // Unix milliseconds
}

// NewPlayback は pending 状態の Playback を生成する。
func NewPlayback(clipCount int, now time.Time) *Playback {
	return &Playback{
		status:    PlaybackStatusPending,
		clipCount: clipCount,
		createdAt: now,
	}
}

// MarkPlaying は pending → playing へ遷移する。他の状態からはエラーを返す。
func (p *Playback) MarkPlaying(nowMs int64) error {
	if p.status != PlaybackStatusPending {
		return fmt.Errorf("cannot transition from %s to playing", p.status)
	}
	p.status = PlaybackStatusPlaying
	p.startedAt = &nowMs
	return nil
}

// MarkCompleted は pending または playing → completed へ遷移する。
// pending からの遷移は無音モード（silent）での即時完了を想定する。
func (p *Playback) MarkCompleted(nowMs int64) error {
	switch p.status {
	case PlaybackStatusPending, PlaybackStatusPlaying:
		p.status = PlaybackStatusCompleted
		p.finishedAt = &nowMs
		return nil
	default:
		return fmt.Errorf("cannot transition from %s to completed", p.status)
	}
}

// MarkFailed は pending または playing → failed へ遷移する。
func (p *Playback) MarkFailed(reason string, nowMs int64) error {
	switch p.status {
	case PlaybackStatusCompleted, PlaybackStatusFailed:
		return fmt.Errorf("cannot transition from %s to failed", p.status)
	default:
		p.status = PlaybackStatusFailed
		p.reason = reason
		p.finishedAt = &nowMs
		return nil
	}
}

// IncrementCompletedClips は完了クリップ数を 1 加算する。
func (p *Playback) IncrementCompletedClips() {
	p.completedClips++
}

func (p *Playback) Status() PlaybackStatus { return p.status }
func (p *Playback) Reason() string         { return p.reason }
func (p *Playback) CreatedAt() time.Time   { return p.createdAt }
func (p *Playback) ClipCount() int         { return p.clipCount }
func (p *Playback) CompletedClips() int    { return p.completedClips }
func (p *Playback) StartedAt() *int64      { return p.startedAt }
func (p *Playback) FinishedAt() *int64     { return p.finishedAt }
