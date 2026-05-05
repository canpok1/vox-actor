package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/domain/entity"
	"github.com/canpok1/vox-actor/internal/infra"
	"github.com/spf13/cobra"
)

// 無音モード通知用の固定文面（#284）。
// silentLogMessage は WARN ログ、silentReason* はフロントに返す `silentReason` 本文。
const (
	silentLogMessage = "VOICEVOX engine unreachable, continuing in silent mode (no audio will be played)"

	silentReasonHealthCheckFailed = `VOICEVOXに接続できないため無音モードで起動しました。
音を再生したい場合はVOICEVOXに接続できる状態で起動し直してください。`

	silentReasonGetSpeakersFailed = `VOICEVOXから話者一覧を取得できなかったため無音モードで起動しました。
音を再生したい場合はVOICEVOXを正常に応答する状態にして起動し直してください。`
)

// startStreamPlayer は HealthCheck → /speakers 取得 → lookup 構築 → StreamPlayer 起動を行う。
// VOICEVOX への接続失敗時は無音モードにフォールバックして起動を継続する（#284）。
// 成功した場合、呼び出し元は defer sp.Shutdown を責務とする。
func startStreamPlayer(
	ctx context.Context,
	addr string,
	logger *slog.Logger,
	client app.VoicevoxClient,
	factory func(string, *slog.Logger, map[int]entity.SpeakerStyleInfo, []int, app.VoicevoxClient) (app.StreamPlayer, error),
) (sp app.StreamPlayer, silent bool, err error) {
	var (
		lookup       map[int]entity.SpeakerStyleInfo
		orderedIDs   []int
		silentReason string
	)
	if hcErr := client.HealthCheck(ctx); hcErr != nil {
		silent = true
		silentReason = silentReasonHealthCheckFailed
		logger.Warn(silentLogMessage, "error", fmt.Errorf("health check failed: %w", hcErr))
	} else {
		speakers, gsErr := client.GetSpeakers(ctx)
		if gsErr != nil {
			silent = true
			silentReason = silentReasonGetSpeakersFailed
			logger.Warn(silentLogMessage, "error", fmt.Errorf("get speakers failed: %w", gsErr))
		} else {
			lookup, orderedIDs = entity.BuildSpeakerStyleLookupWithOrder(speakers)
			logger.Info("speakers loaded", "speakerCount", len(speakers), "styleCount", len(lookup))
		}
	}
	if silent {
		lookup = map[int]entity.SpeakerStyleInfo{}
		orderedIDs = nil
	}

	sp, err = factory(addr, logger, lookup, orderedIDs, client)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create stream player: %w", err)
	}
	if silent {
		sp.SetSilent(silentReason)
	}
	if err = sp.Start(ctx); err != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = sp.Shutdown(shutdownCtx)
		return nil, false, fmt.Errorf("failed to start stream server: %w", err)
	}
	return sp, silent, nil
}

// ViewerDeps はviewerコマンドの依存を保持する。
type ViewerDeps struct {
	ClientFactory       func(engineURL string) app.VoicevoxClient
	StreamPlayerFactory func(addr string, logger *slog.Logger, speakerLookup map[int]entity.SpeakerStyleInfo, orderedSpeakerIDs []int, client app.VoicevoxClient) (app.StreamPlayer, error)
	LockPathResolver    func() (string, error)
}

func makeViewerCmd(deps *ViewerDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "viewer",
		Short: "HTTPサーバーを起動してブラウザUIで音声配信を行う",
		Long:  "HTTPサーバーとブラウザUIを起動して音声配信を行う。ディレクトリ監視は 'vox-actor watch' を併用してください。",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runViewer(cmd, deps)
		},
	}

	registerBaseFlags(cmd)
	cmd.Flags().String("host", "127.0.0.1", "HTTPサーバーのバインドホスト")
	cmd.Flags().Int("port", 8080, "HTTPサーバーのバインドポート")
	registerSaveWavDirFlag(cmd)

	return cmd
}

func resolveViewerLockPathWith(resolver func() (string, error)) (string, error) {
	if resolver != nil {
		return resolver()
	}
	return infra.ResolveHomeViewerLockPath()
}

func resolveViewerLockPath(deps *ViewerDeps) (string, error) {
	if deps != nil {
		return resolveViewerLockPathWith(deps.LockPathResolver)
	}
	return resolveViewerLockPathWith(nil)
}

func runViewer(cmd *cobra.Command, deps *ViewerDeps) error {
	lockPath, err := resolveViewerLockPath(deps)
	if err != nil {
		return err
	}
	viewerLock, err := infra.AcquireViewerLock(lockPath)
	if err != nil {
		return err
	}
	defer viewerLock.Release() //nolint:errcheck

	host, _ := cmd.Flags().GetString("host")
	port, _ := cmd.Flags().GetInt("port")
	engineURL, _ := cmd.Flags().GetString("engine-url")
	saveWavDir, _ := cmd.Flags().GetString("save-wav-dir")

	if port < 1 || port > 65535 {
		return fmt.Errorf("%w: invalid port: %d", ErrUsage, port)
	}

	if deps == nil || deps.ClientFactory == nil || deps.StreamPlayerFactory == nil {
		return fmt.Errorf("viewer command dependencies are not initialized")
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := buildLoggerFromFlags(cmd)
	addr := fmt.Sprintf("%s:%d", host, port)

	client := deps.ClientFactory(engineURL)
	if client == nil {
		return fmt.Errorf("failed to create VoicevoxClient for %s", engineURL)
	}

	factory := deps.StreamPlayerFactory
	if saveWavDir != "" {
		saver := infra.NewWavSaver()
		origFactory := deps.StreamPlayerFactory
		factory = func(addr string, logger *slog.Logger, speakerLookup map[int]entity.SpeakerStyleInfo, orderedSpeakerIDs []int, client app.VoicevoxClient) (app.StreamPlayer, error) {
			sp, err := origFactory(addr, logger, speakerLookup, orderedSpeakerIDs, client)
			if err != nil {
				return nil, err
			}
			if httpSP, ok := sp.(*infra.HTTPStreamPlayer); ok {
				infra.WithSaveWavDir(saveWavDir, saver)(httpSP)
			}
			return sp, nil
		}
	}

	sp, silent, err := startStreamPlayer(ctx, addr, logger, client, factory)
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownErr := sp.Shutdown(shutdownCtx); shutdownErr != nil {
			logger.Error("stream server shutdown error", "error", shutdownErr)
		}
	}()
	logger.Info("stream server listening", "addr", sp.Addr(), "silent", silent)

	if err := viewerLock.WriteAddr(sp.Addr()); err != nil {
		logger.Warn("failed to write addr to viewer.lock", "error", err)
	}

	<-ctx.Done()
	return nil
}
