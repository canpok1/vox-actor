package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/infra"
	"github.com/spf13/cobra"
)

// SayDeps はsayコマンドの依存を保持する。
type SayDeps struct {
	ClientFactory    func(engineURL string) app.VoicevoxClient
	Player           app.AudioPlayer
	LockPathResolver func() (string, error)
	Logger           *slog.Logger
	// AudioProbe はローカル再生前に音声デバイス可用性を検査する関数。
	// nil の場合は検査をスキップする。--dry-run 時は呼ばれない。
	AudioProbe func() error
}

func makeSayCmd(deps *SayDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "say <text>",
		Short: "テキストを直接指定して読み上げる",
		Long:  "テキストを引数で指定し、VOICEVOXエンジンで音声合成して再生する。",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return fmt.Errorf("%w: %s", ErrUsage, err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSay(cmd, args, deps)
		},
	}

	registerCommonFlags(cmd)

	return cmd
}

func sayViaViewer(ctx context.Context, cmd *cobra.Command, vc *infra.ViewerAPIClient, clip infra.ViewerClip, logger *slog.Logger) error {
	resp, err := vc.Play(ctx, infra.ViewerPlayRequest{Clips: []infra.ViewerClip{clip}})
	if err != nil {
		return fmt.Errorf("say via viewer: %w", err)
	}
	if resp.Silent {
		logger.Warn("viewer is in silent mode", "reason", resp.SilentReason)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), resp.PlaybackID)
	return nil
}

func resolveSayLockPath(deps *SayDeps) (string, error) {
	if deps != nil {
		return resolveViewerLockPathWith(deps.LockPathResolver)
	}
	return resolveViewerLockPathWith(nil)
}

func runSay(cmd *cobra.Command, args []string, deps *SayDeps) error {
	if deps == nil || deps.ClientFactory == nil || deps.Player == nil {
		return fmt.Errorf("say command dependencies are not initialized")
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dryRun, _ := cmd.Flags().GetBool("dry-run")

	var logger *slog.Logger
	if deps.Logger != nil {
		logger = deps.Logger
	} else {
		logger = buildLoggerFromFlags(cmd)
	}

	engineURL, _ := cmd.Flags().GetString("engine-url")
	viewerURL, _ := cmd.Flags().GetString("viewer-url")
	speakerID, _ := cmd.Flags().GetInt("speaker")
	speed, _ := cmd.Flags().GetFloat64("speed")
	pitch, _ := cmd.Flags().GetFloat64("pitch")
	intonation, _ := cmd.Flags().GetFloat64("intonation")

	clip := infra.ViewerClip{
		Text:       args[0],
		SpeakerID:  speakerID,
		Speed:      &speed,
		Pitch:      &pitch,
		Intonation: &intonation,
	}

	if !dryRun {
		if viewerURL != "" {
			// 明示 URL: lockfile スキップ、AudioProbe スキップ、接続失敗時はエラー
			logger.Info("using explicit viewer URL", "url", viewerURL)
			vc := infra.NewViewerAPIClientFromURL(viewerURL)
			return sayViaViewer(ctx, cmd, vc, clip, logger)
		}

		lockPath, lockErr := resolveSayLockPath(deps)
		if lockErr != nil {
			logger.Debug("could not resolve viewer lock path, using local player", "error", lockErr)
		} else if addr, ok := infra.DetectViewer(lockPath); ok {
			logger.Info("viewer detected, sending audio via /api/play", "addr", addr)
			vc := infra.NewViewerAPIClient(addr)
			return sayViaViewer(ctx, cmd, vc, clip, logger)
		} else {
			logger.Debug("viewer not running, using local player")
		}
	}

	if err := runAudioProbe(deps.AudioProbe, dryRun); err != nil {
		return err
	}

	params := app.SayParams{
		Text:       args[0],
		SpeakerID:  speakerID,
		Speed:      &speed,
		Pitch:      &pitch,
		Intonation: &intonation,
		DryRun:     dryRun,
	}

	client := deps.ClientFactory(engineURL)
	if client == nil {
		return fmt.Errorf("failed to create VoicevoxClient for %s", engineURL)
	}

	uc := app.NewSayUsecase(client, deps.Player, app.WithSayLogger(logger))
	if err := uc.Run(ctx, params); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), localPlaybackID)
	return nil
}
