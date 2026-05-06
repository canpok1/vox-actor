package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/canpok1/vox-actor/internal/domain/entity"
)

// ActParams はactユースケースのパラメータ。
type ActParams struct {
	Path       string
	SpeakerID  int
	Speed      *float64
	Pitch      *float64
	Intonation *float64
	// DryRun が true の場合、VOICEVOXエンジン/音声再生を一切呼ばずログ出力のみ行う。
	DryRun bool
}

// ActOption はActUsecaseの生成時に指定するオプション。
type ActOption func(*ActUsecase)

// WithLogger はロガーを設定するオプション。
// nilが渡された場合はデフォルトのdiscardロガーを維持する。
func WithLogger(logger *slog.Logger) ActOption {
	return func(u *ActUsecase) {
		if logger != nil {
			u.logger = logger
		}
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
// 疎通確認→読込→（プリフェッチ付き）合成→再生の一連フローを実行する。
func (u *ActUsecase) Run(ctx context.Context, params ActParams) error {
	u.logger.Debug("act starting", "path", params.Path, "speakerID", params.SpeakerID,
		"speed", params.Speed, "pitch", params.Pitch, "intonation", params.Intonation)

	if !params.DryRun {
		if err := u.client.HealthCheck(ctx); err != nil {
			return err
		}
		u.logger.Info("engine health check passed")
	}

	scripts, err := u.reader.Read(params.Path)
	if err != nil {
		return err
	}
	u.logger.Info("scripts loaded", "path", params.Path, "count", len(scripts))

	// 非空スクリプトの総数をカウント（進捗表示用）
	total := 0
	for _, s := range scripts {
		if !s.IsEmpty {
			total++
		}
	}

	if params.DryRun {
		return u.runDryRun(ctx, scripts, total, params)
	}

	cfg := synthPipelineConfig{
		defaultSpeakerID: params.SpeakerID,
		defaultOverrides: entity.SynthOverrides{
			Speed:      params.Speed,
			Pitch:      params.Pitch,
			Intonation: params.Intonation,
		},
		synthesisLogLevel: slog.LevelInfo,
	}
	ch := startSynthPipeline(ctx, u.client, u.logger, scripts, cfg)

	for result := range ch {
		switch result.stage {
		case synthStageQueryFailed:
			return fmt.Errorf("create query for %s: %w", result.script.Path, result.err)
		case synthStageSynthesizeFailed:
			return fmt.Errorf("synthesize %s: %w", result.script.Path, result.err)
		}

		u.logger.Info(fmt.Sprintf("[%d/%d] processing script", result.index, total), "path", result.script.Path)

		speakerID := result.script.ResolveSpeakerID(params.SpeakerID)
		if err := u.player.Play(ctx, result.wav, PlayMeta{Text: result.script.Text, SpeakerID: speakerID}); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("play %s: %w", result.script.Path, err)
		}
		u.logger.Info("playback completed", "path", result.script.Path)

		// 現在のセリフ処理完了後にctxキャンセルを検知したら残りをスキップする。
		// プリフェッチ済みアイテムがバッファに残っていても再生しない。
		if ctx.Err() != nil {
			return nil
		}
	}

	u.logger.Info("all scripts processed")
	return nil
}

// runDryRun はDryRunモードのactユースケースを実行する。
// VOICEVOXエンジン/音声再生は一切呼ばず、既存のログ互換性を保ったまま逐次出力する。
func (u *ActUsecase) runDryRun(ctx context.Context, scripts []entity.Script, total int, params ActParams) error {
	current := 0
	for _, script := range scripts {
		if ctx.Err() != nil {
			return nil
		}
		if script.IsEmpty {
			u.logger.Debug("skipping empty script", "path", script.Path)
			continue
		}
		current++
		u.logger.Info(fmt.Sprintf("[%d/%d] processing script", current, total), "path", script.Path)

		speakerID := script.ResolveSpeakerID(params.SpeakerID)
		overrides := script.ResolveOverrides(entity.SynthOverrides{
			Speed:      params.Speed,
			Pitch:      params.Pitch,
			Intonation: params.Intonation,
		})

		u.logger.Info("synthesis completed", "path", script.Path, "wavSize", 0)
		attrs := append([]any{"path", script.Path}, dryRunPlaybackAttrs(script.Text, speakerID, overrides.Speed, overrides.Pitch, overrides.Intonation)...)
		u.logger.Info("playback completed", attrs...)
	}
	u.logger.Info("all scripts processed")
	return nil
}
