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

	deps := &cmd.ActDeps{
		Reader: infra.NewFileReader(),
		ClientFactory: func(engineURL string) app.VoicevoxClient {
			return infra.NewVoicevoxClient(engineURL)
		},
		Player: player,
		Mover:  infra.NewFileMover(),
		DirWatcherFactory: func() app.DirWatcher {
			return infra.NewPollingDirWatcher(infra.PollInterval)
		},
	}

	if err := cmd.Execute(deps); err != nil {
		if errors.Is(err, cmd.ErrUsage) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}
