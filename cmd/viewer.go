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
	"github.com/spf13/cobra"
)

// ViewerDeps はviewerコマンドの依存を保持する。
type ViewerDeps struct {
	Reader              app.ScriptReader
	ClientFactory       func(engineURL string) app.VoicevoxClient
	Mover               app.FileMover
	DirWatcherFactory   func(logger *slog.Logger) app.DirWatcher
	StreamPlayerFactory func(addr string, logger *slog.Logger, speakerLookup map[int]entity.SpeakerStyleInfo, client app.VoicevoxClient) (app.StreamPlayer, error)
	QueuePathResolver   func() (string, error)
}

func makeViewerCmd(deps *ViewerDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "viewer",
		Short: "HTTPサーバーを起動してブラウザUIで音声配信を行う",
		Long:  "HTTPサーバーとブラウザUIを起動して音声配信を行う。--watch や --watch-queue でディレクトリを監視できる。",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runViewer(cmd, deps)
		},
	}

	registerBaseFlags(cmd)
	cmd.Flags().StringArray("watch", nil, "監視対象ディレクトリ（複数回指定可）")
	cmd.Flags().Bool("watch-queue", false, "vox-actor config path.queue で解決される queue ディレクトリを監視対象に追加")
	cmd.Flags().Bool("delete", false, "処理済みファイルを削除する（未指定時は各ディレクトリの done/ に移動）")
	cmd.Flags().String("host", "127.0.0.1", "HTTPサーバーのバインドホスト")
	cmd.Flags().Int("port", 8080, "HTTPサーバーのバインドポート")

	return cmd
}

func runViewer(cmd *cobra.Command, deps *ViewerDeps) error {
	host, _ := cmd.Flags().GetString("host")
	port, _ := cmd.Flags().GetInt("port")
	watchDirs, _ := cmd.Flags().GetStringArray("watch")
	watchQueue, _ := cmd.Flags().GetBool("watch-queue")
	deleteMode, _ := cmd.Flags().GetBool("delete")
	engineURL, _ := cmd.Flags().GetString("engine-url")
	speakerID, _ := cmd.Flags().GetInt("speaker")
	speed, _ := cmd.Flags().GetFloat64("speed")
	pitch, _ := cmd.Flags().GetFloat64("pitch")
	intonation, _ := cmd.Flags().GetFloat64("intonation")

	if port < 1 || port > 65535 {
		return fmt.Errorf("%w: invalid port: %d", ErrUsage, port)
	}

	dirs, err := resolveViewerPaths(watchDirs, watchQueue, deps)
	if err != nil {
		return err
	}

	for _, path := range dirs {
		info, statErr := os.Stat(path)
		if statErr != nil {
			return fmt.Errorf("%w: %v", ErrUsage, statErr)
		}
		if !info.IsDir() {
			return fmt.Errorf("%w: %s is not a directory", ErrUsage, path)
		}
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

	sp, silent, err := startStreamPlayer(ctx, addr, logger, client, deps.StreamPlayerFactory)
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

	if len(dirs) == 0 {
		<-ctx.Done()
		return nil
	}

	if deps.Reader == nil || deps.Mover == nil || deps.DirWatcherFactory == nil {
		return fmt.Errorf("viewer command dependencies for watching are not initialized")
	}

	watcher := deps.DirWatcherFactory(logger)
	opts := []app.WatchOption{app.WithWatchLogger(logger)}
	if deleteMode {
		opts = append(opts, app.WithDeleteMode())
	}
	if silent {
		opts = append(opts, app.WithSilent())
	}

	uc := app.NewWatchUsecase(deps.Reader, client, sp, deps.Mover, watcher, opts...)
	return uc.Run(ctx, app.WatchParams{
		Paths:      dirs,
		SpeakerID:  speakerID,
		Speed:      &speed,
		Pitch:      &pitch,
		Intonation: &intonation,
	})
}

func resolveViewerPaths(watchDirs []string, watchQueue bool, deps *ViewerDeps) ([]string, error) {
	dirs := append([]string(nil), watchDirs...)

	if watchQueue {
		var resolver func() (string, error)
		if deps != nil {
			resolver = deps.QueuePathResolver
		}
		queuePath, err := resolveQueueDir(resolver)
		if err != nil {
			return nil, err
		}
		dirs = append(dirs, queuePath)
	}

	return dirs, nil
}
