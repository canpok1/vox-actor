package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/canpok1/vox-actor/cmd"
	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/infra"
)

func main() {
	player, err := infra.NewBeepPlayer(infra.NewRealSpeakerBackend())
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create audio player: %v\n", err)
		os.Exit(1)
	}

	clientFactory := func(engineURL string) app.VoicevoxClient {
		return infra.NewRetryableVoicevoxClient(infra.NewVoicevoxClient(engineURL))
	}

	dirWatcherFactory := func() app.DirWatcher {
		return infra.NewPollingDirWatcher(infra.PollInterval)
	}

	reader := infra.NewFileReader()
	mover := infra.NewFileMover()

	deps := &cmd.Deps{
		Act: &cmd.ActDeps{
			Reader:            reader,
			ClientFactory:     clientFactory,
			Player:            player,
			Mover:             mover,
			DirWatcherFactory: dirWatcherFactory,
		},
		Watch: &cmd.WatchDeps{
			Reader:            reader,
			ClientFactory:     clientFactory,
			Player:            player,
			Mover:             mover,
			DirWatcherFactory: dirWatcherFactory,
			StreamPlayerFactory: func(addr string, logger *slog.Logger) (app.StreamPlayer, error) {
				return infra.NewHTTPStreamPlayer(addr, infra.WithHTTPStreamLogger(logger))
			},
		},
		Say: &cmd.SayDeps{
			ClientFactory: clientFactory,
			Player:        player,
		},
	}

	if err := cmd.Execute(deps); err != nil {
		if errors.Is(err, cmd.ErrUsage) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}
