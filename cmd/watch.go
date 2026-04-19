package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/spf13/cobra"
)

// WatchDeps はwatchコマンドの依存を保持する。
type WatchDeps struct {
	Reader            app.ScriptReader
	ClientFactory     func(engineURL string) app.VoicevoxClient
	Player            app.AudioPlayer
	Mover             app.FileMover
	DirWatcherFactory func() app.DirWatcher
}

func makeWatchCmd(deps *WatchDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch <dir1> [<dir2> ...]",
		Short: "複数ディレクトリを監視してテキストファイルを読み上げる",
		Long:  "1つ以上のディレクトリを並列監視し、検知したファイルを順次読み上げる。",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
				return fmt.Errorf("%w: %s", ErrUsage, err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWatch(cmd, args, deps)
		},
	}

	registerCommonFlags(cmd)
	cmd.Flags().Bool("delete", false, "処理済みファイルを削除する（未指定時は各ディレクトリの done/ に移動）")

	return cmd
}

func runWatch(cmd *cobra.Command, args []string, deps *WatchDeps) error {
	for _, path := range args {
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrUsage, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("%w: %s is not a directory", ErrUsage, path)
		}
	}

	if deps == nil || deps.ClientFactory == nil || deps.Reader == nil || deps.Player == nil || deps.Mover == nil || deps.DirWatcherFactory == nil {
		return fmt.Errorf("watch command dependencies are not initialized")
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	deleteMode, _ := cmd.Flags().GetBool("delete")
	engineURL, _ := cmd.Flags().GetString("engine-url")
	speakerID, _ := cmd.Flags().GetInt("speaker")
	speed, _ := cmd.Flags().GetFloat64("speed")
	pitch, _ := cmd.Flags().GetFloat64("pitch")
	intonation, _ := cmd.Flags().GetFloat64("intonation")

	logger := buildLoggerFromFlags(cmd)

	client := deps.ClientFactory(engineURL)
	if client == nil {
		return fmt.Errorf("failed to create VoicevoxClient for %s", engineURL)
	}

	watcher := deps.DirWatcherFactory()
	opts := []app.WatchOption{app.WithWatchLogger(logger)}
	if deleteMode {
		opts = append(opts, app.WithDeleteMode())
	}

	uc := app.NewWatchUsecase(deps.Reader, client, deps.Player, deps.Mover, watcher, opts...)
	return uc.Run(ctx, app.WatchParams{
		Paths:      args,
		SpeakerID:  speakerID,
		Speed:      &speed,
		Pitch:      &pitch,
		Intonation: &intonation,
	})
}
