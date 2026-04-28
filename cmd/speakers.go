package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/canpok1/vox-actor/internal/domain/entity"
	"github.com/canpok1/vox-actor/internal/infra"
	"github.com/spf13/cobra"
)

// SpeakersDeps は speakers コマンドの依存を保持する。
type SpeakersDeps struct {
	// AssetsDirFunc はアセットの配置先ルートディレクトリを返す（後方互換）。nil の場合はワークスペース解決を使う。
	AssetsDirFunc func() (string, error)
	// AssetsDirsMergedFunc は優先順位付きのアセットディレクトリ一覧を返す（index 0 が最高優先）。
	// 設定されると AssetsDirFunc より優先される。
	AssetsDirsMergedFunc func() ([]string, error)
}

// SpeakerListItem は speakers list コマンドの出力要素を表す。
type SpeakerListItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SpeakerProfileOutput は speakers profile コマンドの出力形式を表す。
type SpeakerProfileOutput struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Pronoun      string         `json:"pronoun"`
	SpeechSuffix []string       `json:"speechSuffix"`
	Personality  []string       `json:"personality"`
	Speakers     map[string]int `json:"speakers"`
	Styles       []string       `json:"styles"`
	Description  string         `json:"description"`
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
	cmd.AddCommand(makeSpeakersProfileCmd(deps))
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
	charDirMap, err := buildCharacterDirMap(cmd, deps)
	if err != nil {
		return err
	}

	var items []SpeakerListItem
	for id, assetsDir := range charDirMap {
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
		items = append(items, SpeakerListItem{ID: id, Name: s.GetSpeakerName()})
	}

	return outputSpeakerList(cmd, items)
}

// buildCharacterDirMap はマージされた charID → assetsDir マップを返す。
// dirs は優先順位順（index 0 が最高優先）。同一 ID は先に登録された dir が優先される。
func buildCharacterDirMap(cmd *cobra.Command, deps *SpeakersDeps) (map[string]string, error) {
	dirs, err := resolveAssetsDirsForSpeakers(deps)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			if cmd != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: skipping assets dir %s: %v\n", dir, err)
			}
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			if _, exists := result[e.Name()]; !exists {
				result[e.Name()] = dir
			}
		}
	}
	return result, nil
}

// resolveAssetsDirsForSpeakers は優先順位付きのアセットディレクトリ一覧を返す（index 0 が最高優先）。
func resolveAssetsDirsForSpeakers(deps *SpeakersDeps) ([]string, error) {
	if deps != nil && deps.AssetsDirsMergedFunc != nil {
		return deps.AssetsDirsMergedFunc()
	}
	if deps != nil && deps.AssetsDirFunc != nil {
		dir, err := deps.AssetsDirFunc()
		if err != nil {
			return nil, err
		}
		return []string{dir}, nil
	}

	projectDir, err := infra.ResolveProjectAssetsPath()
	if err != nil {
		return nil, err
	}
	homeDir, err := infra.ResolveHomeAssetsPath()
	if err != nil {
		return nil, err
	}
	return []string{projectDir, homeDir}, nil
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

func makeSpeakersProfileCmd(deps *SpeakersDeps) *cobra.Command {
	var id string
	var name string

	cmd := &cobra.Command{
		Use:   "profile",
		Short: "指定したキャラクターのプロフィールをJSON形式で返す",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSpeakersProfile(cmd, deps, id, name)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "assets配下のディレクトリ名")
	cmd.Flags().StringVar(&name, "name", "", "speaker.json の name フィールド")
	return cmd
}

func runSpeakersProfile(cmd *cobra.Command, deps *SpeakersDeps, id string, name string) error {
	// Flag validation
	if (id != "" && name != "") || (id == "" && name == "") {
		return fmt.Errorf("either --id or --name must be specified (not both)")
	}

	charDirMap, err := buildCharacterDirMap(cmd, deps)
	if err != nil {
		return err
	}

	var characterID string
	var assetsDir string
	if id != "" {
		dir, exists := charDirMap[id]
		if !exists {
			return fmt.Errorf("character not found: %q", id)
		}
		characterID = id
		assetsDir = dir
	} else {
		// Search by name across merged dirs
		var matches []string
		for charID, charDir := range charDirMap {
			speakerJSONPath := filepath.Join(charDir, charID, "speaker.json")
			data, err := os.ReadFile(speakerJSONPath)
			if err != nil {
				continue
			}
			s, err := entity.ParseSpeakerJSON(data)
			if err != nil {
				continue
			}
			if s.GetSpeakerName() == name {
				matches = append(matches, charID)
			}
		}
		if len(matches) == 0 {
			return fmt.Errorf("character not found: %q", name)
		}
		if len(matches) > 1 {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "error: multiple characters found with name %q: %v\n", name, matches)
			return fmt.Errorf("duplicate character name")
		}
		characterID = matches[0]
		assetsDir = charDirMap[characterID]
	}

	speakerJSONPath := filepath.Join(assetsDir, characterID, "speaker.json")
	data, err := os.ReadFile(speakerJSONPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("character not found: %q", characterID)
		}
		return fmt.Errorf("failed to read speaker.json: %w", err)
	}

	speaker, err := entity.ParseSpeakerJSON(data)
	if err != nil {
		return fmt.Errorf("failed to parse speaker.json: %w", err)
	}

	output := SpeakerProfileOutput{
		ID:   characterID,
		Name: speaker.GetSpeakerName(),
	}

	if speaker.Profile != nil {
		output.Pronoun = speaker.Profile.Pronoun
		output.SpeechSuffix = speaker.Profile.SpeechSuffix
		output.Personality = speaker.Profile.Personality
		output.Speakers = speaker.Profile.Speakers

		if speaker.Profile.DescriptionPath != "" {
			descPath := filepath.Join(assetsDir, characterID, speaker.Profile.DescriptionPath)
			descData, err := os.ReadFile(descPath)
			if err != nil {
				if os.IsNotExist(err) {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: description file not found: %s\n", speaker.Profile.DescriptionPath)
				} else {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: failed to read description file: %v\n", err)
				}
			} else {
				output.Description = string(descData)
			}
		}
	}

	if output.Speakers != nil {
		for styleName := range output.Speakers {
			output.Styles = append(output.Styles, styleName)
		}
	}

	resultData, err := json.Marshal(output)
	if err != nil {
		return fmt.Errorf("failed to marshal profile: %w", err)
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(resultData)); err != nil {
		return err
	}
	return nil
}
