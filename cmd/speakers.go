package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/canpok1/vox-actor/internal/domain/entity"
	"github.com/spf13/cobra"
)

// SpeakersDeps は speakers コマンドの依存を保持する。
type SpeakersDeps struct {
	// AssetsDirFunc はアセットの配置先ルートディレクトリを返す。nil の場合はワークスペース解決を使う。
	AssetsDirFunc func() (string, error)
}

// SpeakerListItem は speakers list コマンドの出力要素を表す。
type SpeakerListItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func makeSpeakersCmd(deps *SpeakersDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "speakers",
		Short: "スピーカー管理コマンド",
		Long:  "利用可能なキャラクター（スピーカー）を管理するサブコマンド群。",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(makeSpeakersListCmd(deps))
	return cmd
}

func makeSpeakersListCmd(deps *SpeakersDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "利用可能なキャラクター一覧をJSON形式で出力する",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSpeakersList(cmd, deps)
		},
	}
}

func runSpeakersList(cmd *cobra.Command, deps *SpeakersDeps) error {
	assetsDir, err := resolveAssetsDirForSpeakers(deps)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(assetsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return outputSpeakerList(cmd, []SpeakerListItem{})
		}
		return fmt.Errorf("failed to read assets dir: %w", err)
	}

	var items []SpeakerListItem
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id := entry.Name()
		speakerJSONPath := filepath.Join(assetsDir, id, "speaker.json")

		data, err := os.ReadFile(speakerJSONPath)
		if err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: skipping %s: failed to read speaker.json: %v\n", id, err)
			continue
		}

		s, err := entity.ParseSpeakerJSON(data)
		if err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: skipping %s: failed to parse speaker.json: %v\n", id, err)
			continue
		}

		items = append(items, SpeakerListItem{
			ID:   id,
			Name: s.GetSpeakerName(),
		})
	}

	return outputSpeakerList(cmd, items)
}

func resolveAssetsDirForSpeakers(deps *SpeakersDeps) (string, error) {
	if deps != nil && deps.AssetsDirFunc != nil {
		return deps.AssetsDirFunc()
	}
	ws, err := resolveWorkspaceWithFallback()
	if err != nil {
		return "", err
	}
	return filepath.Join(ws, "assets"), nil
}

func outputSpeakerList(cmd *cobra.Command, items []SpeakerListItem) error {
	if items == nil {
		items = []SpeakerListItem{}
	}
	data, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("failed to marshal speaker list: %w", err)
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(data)); err != nil {
		return err
	}
	return nil
}
