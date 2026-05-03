package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/canpok1/vox-actor/internal/infra"
	"github.com/spf13/cobra"
)

// PlaybackWaitDeps は playback wait コマンドの依存を保持する。
type PlaybackWaitDeps struct {
	LockPathResolver func() (string, error)
}

func resolvePlaybackWaitLockPath(deps *PlaybackWaitDeps) (string, error) {
	if deps != nil {
		return resolveViewerLockPathWith(deps.LockPathResolver)
	}
	return resolveViewerLockPathWith(nil)
}

func makePlaybackCmd(deps *PlaybackWaitDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "playback",
		Short: "再生状態を管理する",
	}
	cmd.SilenceUsage = true
	cmd.AddCommand(makePlaybackWaitCmd(deps))
	return cmd
}

func makePlaybackWaitCmd(deps *PlaybackWaitDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wait <id>",
		Short: "再生が完了するまで待機する",
		Long:  "playback_id を指定して再生が完了するまでポーリングする。local_playback を渡すと即時終了する。",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return fmt.Errorf("%w: %s", ErrUsage, err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlaybackWait(cmd, args, deps)
		},
	}
	registerViewerURLFlag(cmd)
	cmd.Flags().Duration("server-down-timeout", 30*time.Second, "サーバー接続失敗を fail 扱いにするまでの連続失敗時間")
	return cmd
}

func runPlaybackWait(cmd *cobra.Command, args []string, deps *PlaybackWaitDeps) error {
	id := args[0]
	if id == "local_playback" {
		return nil
	}

	viewerURL, _ := cmd.Flags().GetString("viewer-url")
	serverDownTimeout, _ := cmd.Flags().GetDuration("server-down-timeout")

	var vc *infra.ViewerAPIClient
	if viewerURL != "" {
		vc = infra.NewViewerAPIClientFromURL(viewerURL)
	} else {
		lockPath, lockErr := resolvePlaybackWaitLockPath(deps)
		if lockErr == nil {
			if addr, ok := infra.DetectViewer(lockPath); ok {
				vc = infra.NewViewerAPIClient(addr)
			}
		}
	}

	if vc == nil {
		return fmt.Errorf("viewer not found: specify --viewer-url or start viewer")
	}

	return pollPlaybackCompletion(cmd.Context(), cmd, vc, id, serverDownTimeout)
}

func pollPlaybackCompletion(ctx context.Context, cmd *cobra.Command, vc *infra.ViewerAPIClient, id string, serverDownTimeout time.Duration) error {
	const (
		initialInterval = 200 * time.Millisecond
		maxInterval     = 2 * time.Second
	)

	interval := initialInterval
	var serverDownStart *time.Time

	for {
		resp, err := vc.GetPlayback(ctx, id)
		if err != nil {
			now := time.Now()
			if serverDownStart == nil {
				serverDownStart = &now
			} else if now.Sub(*serverDownStart) >= serverDownTimeout {
				return fmt.Errorf("server unreachable for %v: %w", serverDownTimeout, err)
			}
		} else {
			serverDownStart = nil
			switch resp.Status {
			case "completed":
				return nil
			case "failed":
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), resp.FailedReason)
				return fmt.Errorf("playback failed: %s", resp.FailedReason)
			case "unknown":
				return fmt.Errorf("playback not found: %s", id)
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}

		if interval < maxInterval {
			interval *= 2
			if interval > maxInterval {
				interval = maxInterval
			}
		}
	}
}
