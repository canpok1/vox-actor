package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
)

// ActParams はactユースケースのパラメータ。
type ActParams struct {
	Path       string
	SpeakerID  int
	Speed      *float64
	Pitch      *float64
	Intonation *float64
}

// ActOption はActUsecaseの生成時に指定するオプション。
type ActOption func(*ActUsecase)

// WithLogger はロガーを設定するオプション。
func WithLogger(logger *slog.Logger) ActOption {
	return func(u *ActUsecase) {
		u.logger = logger
	}
}

// discardLogger はログを破棄するロガーを返す。
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// ActUsecase はactサブコマンドのユースケース。
type ActUsecase struct {
	reader ScriptReader
	client VoicevoxClient
	player AudioPlayer
	logger *slog.Logger
}

// NewActUsecase は新しいActUsecaseを生成する。
func NewActUsecase(reader ScriptReader, client VoicevoxClient, player AudioPlayer, opts ...ActOption) *ActUsecase {
	u := &ActUsecase{
		reader: reader,
		client: client,
		player: player,
		logger: discardLogger(),
	}
	for _, opt := range opts {
		opt(u)
	}
	return u
}

// Run はactユースケースを実行する。
// 疎通確認→読込→合成→再生の一連フローを実行する。
func (u *ActUsecase) Run(ctx context.Context, params ActParams) error {
	u.logger.Debug("act starting", "path", params.Path, "speakerID", params.SpeakerID,
		"speed", params.Speed, "pitch", params.Pitch, "intonation", params.Intonation)

	if err := u.client.HealthCheck(ctx); err != nil {
		return err
	}
	u.logger.Info("engine health check passed")

	scripts, err := u.reader.Read(params.Path)
	if err != nil {
		return err
	}
	u.logger.Info("scripts loaded", "path", params.Path, "count", len(scripts))

	for _, script := range scripts {
		if ctx.Err() != nil {
			return nil
		}

		if script.IsEmpty {
			u.logger.Debug("skipping empty script", "path", script.Path)
			continue
		}

		u.logger.Info("processing script", "path", script.Path)

		query, err := u.client.CreateQuery(ctx, script.Text, params.SpeakerID)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("create query for %s: %w", script.Path, err)
		}
		u.logger.Debug("query created", "path", script.Path)

		wavData, err := u.client.Synthesize(ctx, query, params.SpeakerID, params.Speed, params.Pitch, params.Intonation)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("synthesize %s: %w", script.Path, err)
		}
		u.logger.Debug("synthesis completed", "path", script.Path, "wavSize", len(wavData))

		if err := u.player.Play(ctx, wavData); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("play %s: %w", script.Path, err)
		}
		u.logger.Info("playback completed", "path", script.Path)
	}

	u.logger.Info("all scripts processed")
	return nil
}
