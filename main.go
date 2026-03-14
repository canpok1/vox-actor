package main

import (
	"errors"
	"fmt"
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

	deps := &cmd.Deps{
		Act: &cmd.ActDeps{
			Reader:        infra.NewFileReader(),
			ClientFactory: clientFactory,
			Player:        player,
			Mover:         infra.NewFileMover(),
			DirWatcherFactory: func() app.DirWatcher {
				return infra.NewPollingDirWatcher(infra.PollInterval)
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
