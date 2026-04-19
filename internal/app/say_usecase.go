package app

import (
	"context"
	"log/slog"
)

// SayParams はsayユースケースのパラメータ。
type SayParams struct {
	Text       string
	SpeakerID  int
	Speed      *float64
	Pitch      *float64
	Intonation *float64
	// DryRun が true の場合、VOICEVOXエンジン/音声再生を一切呼ばずログ出力のみ行う。
	DryRun bool
}

// SayOption はSayUsecaseの生成時に指定するオプション。
type SayOption func(*SayUsecase)

// WithSayLogger はロガーを設定するオプション。
func WithSayLogger(logger *slog.Logger) SayOption {
	return func(u *SayUsecase) {
		if logger != nil {
			u.logger = logger
		}
	}
}

// SayUsecase はsayサブコマンドのユースケース。
type SayUsecase struct {
	client VoicevoxClient
	player AudioPlayer
	logger *slog.Logger
}

// NewSayUsecase は新しいSayUsecaseを生成する。
func NewSayUsecase(client VoicevoxClient, player AudioPlayer, opts ...SayOption) *SayUsecase {
	u := &SayUsecase{
		client: client,
		player: player,
		logger: discardLogger(),
	}
	for _, opt := range opts {
		opt(u)
	}
	return u
}

// Run はsayユースケースを実行する。
// 疎通確認→合成→再生の一連フローを実行する。
func (u *SayUsecase) Run(ctx context.Context, params SayParams) error {
	u.logger.Debug("say starting", "text", params.Text, "speakerID", params.SpeakerID,
		"speed", params.Speed, "pitch", params.Pitch, "intonation", params.Intonation)

	if params.DryRun {
		logDryRunSynthesize(u.logger, "", params.Text, params.SpeakerID, params.Speed, params.Pitch, params.Intonation)
		return nil
	}

	if err := u.client.HealthCheck(ctx); err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return err
	}
	u.logger.Info("engine health check passed")

	query, err := u.client.CreateQuery(ctx, params.Text, params.SpeakerID)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return err
	}
	u.logger.Debug("query created")

	q := query.WithOverrides(params.Speed, params.Pitch, params.Intonation)
	wavData, err := u.client.Synthesize(ctx, &q, params.SpeakerID)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return err
	}
	u.logger.Info("synthesis completed", "wavSize", len(wavData))

	if err := u.player.Play(ctx, wavData); err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return err
	}
	u.logger.Info("playback completed")

	return nil
}
