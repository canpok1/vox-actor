package app

import (
	"context"
	"fmt"
)

// ActParams はactユースケースのパラメータ。
type ActParams struct {
	Path       string
	SpeakerID  int
	Speed      *float64
	Pitch      *float64
	Intonation *float64
}

// ActUsecase はactサブコマンドのユースケース。
type ActUsecase struct {
	reader ScriptReader
	client VoicevoxClient
	player AudioPlayer
}

// NewActUsecase は新しいActUsecaseを生成する。
func NewActUsecase(reader ScriptReader, client VoicevoxClient, player AudioPlayer) *ActUsecase {
	return &ActUsecase{
		reader: reader,
		client: client,
		player: player,
	}
}

// Run はactユースケースを実行する。
// 疎通確認→読込→合成→再生の一連フローを実行する。
func (u *ActUsecase) Run(ctx context.Context, params ActParams) error {
	if err := u.client.HealthCheck(ctx); err != nil {
		return err
	}

	scripts, err := u.reader.Read(params.Path)
	if err != nil {
		return err
	}

	for _, script := range scripts {
		if script.IsEmpty {
			continue
		}

		query, err := u.client.CreateQuery(ctx, script.Text, params.SpeakerID)
		if err != nil {
			return fmt.Errorf("create query for %s: %w", script.Path, err)
		}

		wavData, err := u.client.Synthesize(ctx, query, params.SpeakerID, params.Speed, params.Pitch, params.Intonation)
		if err != nil {
			return fmt.Errorf("synthesize %s: %w", script.Path, err)
		}

		if err := u.player.Play(ctx, wavData); err != nil {
			return fmt.Errorf("play %s: %w", script.Path, err)
		}
	}

	return nil
}
