package main

import (
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
	}

	if err := cmd.Execute(deps); err != nil {
		os.Exit(1)
	}
}
