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

	dirWatcherFactory := func(logger *slog.Logger) app.DirWatcher {
		return infra.NewPollingDirWatcher(
			infra.PollInterval,
			infra.WithPollingDirWatcherLogger(logger),
		)
	}

	reader := infra.NewFileReader()
	mover := infra.NewFileMover()

	streamPlayerFactory := func(addr string, logger *slog.Logger, speakerLookup map[int]entity.SpeakerStyleInfo, orderedSpeakerIDs []int, client app.VoicevoxClient) (app.StreamPlayer, error) {
		staticFS, err := streamAssetsFS()
		if err != nil {
			return nil, fmt.Errorf("failed to resolve stream assets: %w", err)
		}
		opts := []infra.HTTPStreamOption{
			infra.WithHTTPStreamLogger(logger),
			infra.WithSpeakerLookup(speakerLookup),
			infra.WithOrderedSpeakerIDs(orderedSpeakerIDs),
			infra.WithVoicevoxClient(client),
		}
		var assetsDirs []string
		if projectAssets, err := infra.ResolveProjectAssetsPath(); err == nil {
			assetsDirs = append(assetsDirs, projectAssets)
		}
		if homeAssets, err := infra.ResolveHomeAssetsPath(); err == nil {
			assetsDirs = append(assetsDirs, homeAssets)
		}
		if len(assetsDirs) > 0 {
			opts = append(opts, infra.WithAssetsDirs(assetsDirs))
		}
		if historyDir, err := infra.ResolveHomeViewerHistoryPath(); err == nil {
			opts = append(opts, infra.WithHistoryDir(historyDir))
		}
		return infra.NewHTTPStreamPlayer(addr, staticFS, opts...)
	}

	audioProbe := func() error {
		_, err := infra.CheckAudioDevice(infra.NewOtoAudioProbe())
		return err
	}

	deps := &cmd.Deps{
		Assets: &cmd.AssetsDownloadDeps{
			Cloner: &infra.GitCloner{},
		},
		Act: &cmd.ActDeps{
			Reader:        reader,
			ClientFactory: clientFactory,
			Player:        player,
			AudioProbe:    audioProbe,
		},
		Watch: &cmd.WatchDeps{
			Reader:            reader,
			ClientFactory:     clientFactory,
			Player:            player,
			Mover:             mover,
			DirWatcherFactory: dirWatcherFactory,
			AudioProbe:        audioProbe,
		},
		Viewer: &cmd.ViewerDeps{
			ClientFactory:       clientFactory,
			StreamPlayerFactory: streamPlayerFactory,
		},
		Say: &cmd.SayDeps{
			ClientFactory: clientFactory,
			Player:        player,
			AudioProbe:    audioProbe,
		},
		ScriptAppend: &cmd.ScriptAppendDeps{
			WriterFactory: func(logger *slog.Logger) app.ScriptWriter {
				return infra.NewFileWriter(infra.WithFileWriterLogger(logger))
			},
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
