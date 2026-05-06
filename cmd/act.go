package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/domain/entity"
	"github.com/canpok1/vox-actor/internal/infra"
	"github.com/spf13/cobra"
)

// ActDeps はactコマンドの依存を保持する。
type ActDeps struct {
	Reader           app.ScriptReader
	ClientFactory    func(engineURL string) app.VoicevoxClient
	Player           app.AudioPlayer
	LockPathResolver func() (string, error)
	Logger           *slog.Logger
	// AudioProbe はローカル再生前に音声デバイス可用性を検査する関数。
	// nil の場合は検査をスキップする。--dry-run 時は呼ばれない。
	AudioProbe func() error
}

func makeActCmd(deps *ActDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "act <path>",
		Short: "テキストファイルを読み上げる",
		Long:  "テキストファイルを読み込み、VOICEVOXエンジンで音声合成して再生する。",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return fmt.Errorf("%w: %s", ErrUsage, err)
			}
			return nil
		},
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Flags().Changed("watch") {
				return fmt.Errorf("%w: --watch is removed; use 'vox-actor watch <dir>' instead", ErrUsage)
			}
			if cmd.Flags().Changed("watch-delete") {
				return fmt.Errorf("%w: --watch-delete is removed; use 'vox-actor watch --delete <dir>' instead", ErrUsage)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAct(cmd, args, deps)
		},
	}

	registerCommonFlags(cmd)
	// --watch / --watch-delete は廃止済み。廃止フラグが渡された場合は PreRunE でエラーを返す。
	cmd.Flags().Bool("watch", false, "")
	cmd.Flags().Bool("watch-delete", false, "")
	_ = cmd.Flags().MarkHidden("watch")
	_ = cmd.Flags().MarkHidden("watch-delete")

	return cmd
}

func actViaViewer(
	ctx context.Context,
	cmd *cobra.Command,
	vc *infra.ViewerAPIClient,
	scripts []entity.Script,
	speakerID entity.SpeakerID, speed, pitch, intonation float64,
	logger *slog.Logger,
) error {
	defaults := entity.SynthOverrides{Speed: &speed, Pitch: &pitch, Intonation: &intonation}
	var clips []infra.ViewerClip
	for _, script := range scripts {
		if script.IsEmpty {
			continue
		}
		resolved := script.ResolveOverrides(defaults)
		clips = append(clips, infra.ViewerClip{
			Text:       script.Text,
			SpeakerID:  script.ResolveSpeakerID(speakerID).Value(),
			Speed:      resolved.Speed,
			Pitch:      resolved.Pitch,
			Intonation: resolved.Intonation,
		})
	}
	if len(clips) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), localPlaybackID)
		return nil
	}
	if ctx.Err() != nil {
		return nil
	}
	resp, err := vc.Play(ctx, infra.ViewerPlayRequest{Clips: clips})
	if err != nil {
		return fmt.Errorf("act via viewer: %w", err)
	}
	if resp.Silent {
		logger.Warn("viewer is in silent mode", "reason", resp.SilentReason)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), resp.PlaybackID)
	return nil
}

func resolveActLockPath(deps *ActDeps) (string, error) {
	if deps != nil {
		return resolveViewerLockPathWith(deps.LockPathResolver)
	}
	return resolveViewerLockPathWith(nil)
}

func runAct(cmd *cobra.Command, args []string, deps *ActDeps) error {
	if deps == nil || deps.ClientFactory == nil || deps.Reader == nil || deps.Player == nil {
		return fmt.Errorf("act command dependencies are not initialized")
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	engineURL, _ := cmd.Flags().GetString("engine-url")
	speakerIDInt, _ := cmd.Flags().GetInt("speaker")
	speed, _ := cmd.Flags().GetFloat64("speed")
	pitch, _ := cmd.Flags().GetFloat64("pitch")
	intonation, _ := cmd.Flags().GetFloat64("intonation")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	speakerID, err := entity.NewSpeakerID(speakerIDInt)
	if err != nil {
		return fmt.Errorf("%w: --speaker: %v", ErrUsage, err)
	}

	var logger *slog.Logger
	if deps.Logger != nil {
		logger = deps.Logger
	} else {
		logger = buildLoggerFromFlags(cmd)
	}

	client := deps.ClientFactory(engineURL)
	if client == nil {
		return fmt.Errorf("failed to create VoicevoxClient for %s", engineURL)
	}

	if !dryRun {
		viewerURL, _ := cmd.Flags().GetString("viewer-url")

		if viewerURL != "" {
			// 明示 URL: lockfile スキップ、AudioProbe スキップ、接続失敗時はエラー
			logger.Info("using explicit viewer URL", "url", viewerURL)
			scripts, err := deps.Reader.Read(args[0])
			if err != nil {
				return err
			}
			vc := infra.NewViewerAPIClientFromURL(viewerURL)
			return actViaViewer(ctx, cmd, vc, scripts, speakerID, speed, pitch, intonation, logger)
		}

		lockPath, lockErr := resolveActLockPath(deps)
		if lockErr != nil {
			logger.Debug("could not resolve viewer lock path, using local player", "error", lockErr)
		} else if addr, ok := infra.DetectViewer(lockPath); ok {
			logger.Info("viewer detected, sending audio via /api/play", "addr", addr)
			scripts, err := deps.Reader.Read(args[0])
			if err != nil {
				return err
			}
			vc := infra.NewViewerAPIClient(addr)
			return actViaViewer(ctx, cmd, vc, scripts, speakerID, speed, pitch, intonation, logger)
		} else {
			logger.Debug("viewer not running, using local player")
		}
	}

	if err := runAudioProbe(deps.AudioProbe, dryRun); err != nil {
		return err
	}

	uc := app.NewActUsecase(deps.Reader, client, deps.Player, app.WithLogger(logger))
	if err := uc.Run(ctx, app.ActParams{
		Path:       args[0],
		SpeakerID:  speakerID,
		Speed:      &speed,
		Pitch:      &pitch,
		Intonation: &intonation,
		DryRun:     dryRun,
	}); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), localPlaybackID)
	return nil
}
