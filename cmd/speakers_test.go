package cmd

// speakers list コマンド テストリスト
// DONE: speakersサブコマンドがrootに登録されている
// DONE: listサブコマンドがspeakersに登録されている
// DONE: アセットディレクトリが空の場合は空JSON配列を返す
// DONE: 正常なspeaker.jsonがあるキャラクターを列挙できる
// DONE: speaker.json読み込み失敗時は該当キャラクターをスキップしstderrに警告
// DONE: speaker.jsonのJSON形式が不正な場合も該当キャラクターをスキップしstderrに警告
// DONE: deps=nilでもワークスペース解決にフォールバックする（AssetsDirFuncのnillチェック）

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func findSpeakersListCmdFromRoot(t *testing.T, rootCmd *cobra.Command) *cobra.Command {
	t.Helper()
	for _, c := range rootCmd.Commands() {
		if c.Name() == "speakers" {
			for _, sc := range c.Commands() {
				if sc.Name() == "list" {
					return sc
				}
			}
		}
	}
	t.Fatal("speakers list subcommand not found")
	return nil
}

func TestSpeakersCmd_RegisteredAsSubcommand(t *testing.T) {
	rootCmd := makeRootCmd()
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "speakers" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'speakers' subcommand to be registered")
	}
}

func TestSpeakersListCmd_RegisteredUnderSpeakers(t *testing.T) {
	rootCmd := makeRootCmd()
	_ = findSpeakersListCmdFromRoot(t, rootCmd)
}

func TestSpeakersListCmd_EmptyAssetsDir_ReturnsEmptyArray(t *testing.T) {
	assetsDir := t.TempDir()

	deps := &Deps{
		Speakers: &SpeakersDeps{
			AssetsDirFunc: func() (string, error) { return assetsDir, nil },
		},
	}
	rootCmd := makeRootCmd(deps)
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"speakers", "list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var items []SpeakerListItem
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("expected valid JSON array, got parse error: %v (output: %s)", err, buf.String())
	}
	if len(items) != 0 {
		t.Errorf("expected empty array, got %d items", len(items))
	}
}

func TestSpeakersListCmd_ValidSpeakers_ListsAll(t *testing.T) {
	assetsDir := t.TempDir()

	speakers := map[string]string{
		"zundamon": `{"name": "ずんだもん", "styles": []}`,
		"metan":    `{"name": "四国めたん", "styles": []}`,
	}
	for id, content := range speakers {
		dir := filepath.Join(assetsDir, id)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "speaker.json"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	deps := &Deps{
		Speakers: &SpeakersDeps{
			AssetsDirFunc: func() (string, error) { return assetsDir, nil },
		},
	}
	rootCmd := makeRootCmd(deps)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"speakers", "list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var items []SpeakerListItem
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("expected valid JSON, got: %v (output: %s)", err, buf.String())
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
	found := map[string]string{}
	for _, item := range items {
		found[item.ID] = item.Name
	}
	if found["zundamon"] != "ずんだもん" {
		t.Errorf("expected zundamon name 'ずんだもん', got %q", found["zundamon"])
	}
	if found["metan"] != "四国めたん" {
		t.Errorf("expected metan name '四国めたん', got %q", found["metan"])
	}
}

func TestSpeakersListCmd_MissingSpeakerJSON_SkipsAndWarns(t *testing.T) {
	assetsDir := t.TempDir()

	// zundamonはspeaker.jsonなし、metanは正常
	if err := os.MkdirAll(filepath.Join(assetsDir, "zundamon"), 0755); err != nil {
		t.Fatal(err)
	}
	metanDir := filepath.Join(assetsDir, "metan")
	if err := os.MkdirAll(metanDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(metanDir, "speaker.json"), []byte(`{"name": "四国めたん", "styles": []}`), 0644); err != nil {
		t.Fatal(err)
	}

	deps := &Deps{
		Speakers: &SpeakersDeps{
			AssetsDirFunc: func() (string, error) { return assetsDir, nil },
		},
	}
	rootCmd := makeRootCmd(deps)
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"speakers", "list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var items []SpeakerListItem
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("expected valid JSON, got: %v (output: %s)", err, buf.String())
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item (metan), got %d", len(items))
	}
	if len(items) == 1 && items[0].ID != "metan" {
		t.Errorf("expected metan, got %q", items[0].ID)
	}

	if !strings.Contains(errBuf.String(), "zundamon") {
		t.Errorf("expected stderr warning for zundamon, got: %s", errBuf.String())
	}
}

func TestSpeakersListCmd_InvalidSpeakerJSON_SkipsAndWarns(t *testing.T) {
	assetsDir := t.TempDir()

	// zundamonは不正JSON
	zundaDir := filepath.Join(assetsDir, "zundamon")
	if err := os.MkdirAll(zundaDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(zundaDir, "speaker.json"), []byte(`{invalid json}`), 0644); err != nil {
		t.Fatal(err)
	}

	deps := &Deps{
		Speakers: &SpeakersDeps{
			AssetsDirFunc: func() (string, error) { return assetsDir, nil },
		},
	}
	rootCmd := makeRootCmd(deps)
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"speakers", "list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var items []SpeakerListItem
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("expected valid JSON, got: %v (output: %s)", err, buf.String())
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}

	if !strings.Contains(errBuf.String(), "zundamon") {
		t.Errorf("expected stderr warning for zundamon, got: %s", errBuf.String())
	}
}

func TestSpeakersListCmd_LegacySpeakerName_Listed(t *testing.T) {
	assetsDir := t.TempDir()

	// 古いスキーマ（speakerNameフィールド）
	dir := filepath.Join(assetsDir, "zundamon")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "speaker.json"), []byte(`{"speakerName": "ずんだもん", "styles": []}`), 0644); err != nil {
		t.Fatal(err)
	}

	deps := &Deps{
		Speakers: &SpeakersDeps{
			AssetsDirFunc: func() (string, error) { return assetsDir, nil },
		},
	}
	rootCmd := makeRootCmd(deps)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"speakers", "list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var items []SpeakerListItem
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("expected valid JSON, got: %v", err)
	}
	if len(items) != 1 || items[0].Name != "ずんだもん" {
		t.Errorf("expected legacy speakerName to work, got items: %+v", items)
	}
}
