package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/canpok1/vox-actor/cmd"
	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/domain/entity"
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
			StreamPlayerFactory: func(addr string, logger *slog.Logger, speakerLookup map[int]entity.SpeakerStyleInfo, client app.VoicevoxClient) (app.StreamPlayer, error) {
				return infra.NewHTTPStreamPlayer(addr,
					infra.WithHTTPStreamLogger(logger),
					infra.WithSpeakerLookup(speakerLookup),
					infra.WithVoicevoxClient(client),
				)
			},
		},
		Say: &cmd.SayDeps{
			ClientFactory: clientFactory,
			Player:        player,
		},
		AudioCheck: &cmd.AudioCheckDeps{
			CheckFunc: func() (string, error) {
				return infra.CheckAudioDevice(infra.NewOtoAudioProbe())
			},
		},
	}

	if err := cmd.Execute(deps); err != nil {
		if errors.Is(err, cmd.ErrUsage) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}
