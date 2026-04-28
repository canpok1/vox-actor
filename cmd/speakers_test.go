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

// speakers profile コマンド テストリスト
// TODO: profileサブコマンドがspeakersに登録されている
// TODO: --idフラグで指定したキャラクターのプロフィールをJSON形式で返す
// TODO: --nameフラグで指定したキャラクターのプロフィールをJSON形式で返す
// TODO: descriptionPathのmdファイル内容がdescriptionフィールドに含まれる
// TODO: --idと--nameを両方指定した場合は usage error
// TODO: --idと--nameを両方指定しなかった場合は usage error
// TODO: --nameで複数キャラクターが一致した場合は exit 1 エラー（stderrにidリスト）
// TODO: 存在しないキャラクターは "character not found" エラー（exit 1）
// TODO: descriptionPathが存在しない場合は description="" で継続し stderr に警告

func findSpeakersProfileCmdFromRoot(t *testing.T, rootCmd *cobra.Command) *cobra.Command {
	t.Helper()
	for _, c := range rootCmd.Commands() {
		if c.Name() == "speakers" {
			for _, sc := range c.Commands() {
				if sc.Name() == "profile" {
					return sc
				}
			}
		}
	}
	t.Fatal("speakers profile subcommand not found")
	return nil
}

func TestSpeakersProfileCmd_RegisteredUnderSpeakers(t *testing.T) {
	rootCmd := makeRootCmd()
	_ = findSpeakersProfileCmdFromRoot(t, rootCmd)
}

func TestSpeakersProfileCmd_WithID_ReturnsProfile(t *testing.T) {
	assetsDir := t.TempDir()

	// zundamon のキャラクター設定
	zundaDir := filepath.Join(assetsDir, "zundamon")
	if err := os.MkdirAll(zundaDir, 0755); err != nil {
		t.Fatal(err)
	}

	descriptionFile := filepath.Join(zundaDir, "character.md")
	descContent := "# ずんだもん キャラクター設定\n\n## 口調の特徴\n\n- 語尾に「のだ」を付ける"
	if err := os.WriteFile(descriptionFile, []byte(descContent), 0644); err != nil {
		t.Fatal(err)
	}

	speakerJSON := `{
		"name": "ずんだもん",
		"profile": {
			"pronoun": "ボク",
			"speechSuffix": ["〜のだ", "〜なのだ"],
			"personality": ["元気", "明るい"],
			"speakers": {
				"ノーマル": 3,
				"ツンツン": 7
			},
			"descriptionPath": "character.md"
		},
		"styles": []
	}`

	if err := os.WriteFile(filepath.Join(zundaDir, "speaker.json"), []byte(speakerJSON), 0644); err != nil {
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
	rootCmd.SetArgs([]string{"speakers", "profile", "--id", "zundamon"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %v (output: %s)", err, buf.String())
	}

	if result["id"] != "zundamon" {
		t.Errorf("expected id 'zundamon', got %v", result["id"])
	}
	if result["name"] != "ずんだもん" {
		t.Errorf("expected name 'ずんだもん', got %v", result["name"])
	}
	if result["description"] != descContent {
		t.Errorf("expected description content to be included, got: %v", result["description"])
	}
}

func TestSpeakersProfileCmd_WithName_ReturnsProfile(t *testing.T) {
	assetsDir := t.TempDir()

	// zundamon のキャラクター設定
	zundaDir := filepath.Join(assetsDir, "zundamon")
	if err := os.MkdirAll(zundaDir, 0755); err != nil {
		t.Fatal(err)
	}

	descriptionFile := filepath.Join(zundaDir, "character.md")
	descContent := "# ずんだもん"
	if err := os.WriteFile(descriptionFile, []byte(descContent), 0644); err != nil {
		t.Fatal(err)
	}

	speakerJSON := `{
		"name": "ずんだもん",
		"profile": {
			"pronoun": "ボク",
			"speechSuffix": ["〜のだ"],
			"personality": ["元気"],
			"speakers": {"ノーマル": 3},
			"descriptionPath": "character.md"
		},
		"styles": []
	}`

	if err := os.WriteFile(filepath.Join(zundaDir, "speaker.json"), []byte(speakerJSON), 0644); err != nil {
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
	rootCmd.SetArgs([]string{"speakers", "profile", "--name", "ずんだもん"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %v", err)
	}

	if result["id"] != "zundamon" {
		t.Errorf("expected id 'zundamon', got %v", result["id"])
	}
	if result["name"] != "ずんだもん" {
		t.Errorf("expected name 'ずんだもん', got %v", result["name"])
	}
}

func TestSpeakersProfileCmd_BothFlagsSpecified_UsageError(t *testing.T) {
	assetsDir := t.TempDir()

	deps := &Deps{
		Speakers: &SpeakersDeps{
			AssetsDirFunc: func() (string, error) { return assetsDir, nil },
		},
	}
	rootCmd := makeRootCmd(deps)
	errBuf := new(bytes.Buffer)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"speakers", "profile", "--id", "zundamon", "--name", "ずんだもん"})

	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error when both --id and --name specified")
	}
}

func TestSpeakersProfileCmd_NoFlagsSpecified_UsageError(t *testing.T) {
	assetsDir := t.TempDir()

	deps := &Deps{
		Speakers: &SpeakersDeps{
			AssetsDirFunc: func() (string, error) { return assetsDir, nil },
		},
	}
	rootCmd := makeRootCmd(deps)
	errBuf := new(bytes.Buffer)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"speakers", "profile"})

	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error when neither --id nor --name specified")
	}
}

func TestSpeakersProfileCmd_CharacterNotFound_Error(t *testing.T) {
	assetsDir := t.TempDir()

	deps := &Deps{
		Speakers: &SpeakersDeps{
			AssetsDirFunc: func() (string, error) { return assetsDir, nil },
		},
	}
	rootCmd := makeRootCmd(deps)
	rootCmd.SetArgs([]string{"speakers", "profile", "--id", "nonexistent"})

	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error for nonexistent character")
	}
}

func TestSpeakersProfileCmd_NameDuplicate_Error(t *testing.T) {
	assetsDir := t.TempDir()

	// zundamon
	zundaDir := filepath.Join(assetsDir, "zundamon")
	if err := os.MkdirAll(zundaDir, 0755); err != nil {
		t.Fatal(err)
	}
	speakerJSON := `{"name": "ずんだもん", "profile": {}, "styles": []}`
	if err := os.WriteFile(filepath.Join(zundaDir, "speaker.json"), []byte(speakerJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// another dir with same name
	otherDir := filepath.Join(assetsDir, "other")
	if err := os.MkdirAll(otherDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(otherDir, "speaker.json"), []byte(speakerJSON), 0644); err != nil {
		t.Fatal(err)
	}

	deps := &Deps{
		Speakers: &SpeakersDeps{
			AssetsDirFunc: func() (string, error) { return assetsDir, nil },
		},
	}
	rootCmd := makeRootCmd(deps)
	errBuf := new(bytes.Buffer)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"speakers", "profile", "--name", "ずんだもん"})

	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error when multiple characters have same name")
	}
	if !strings.Contains(errBuf.String(), "zundamon") && !strings.Contains(errBuf.String(), "other") {
		t.Errorf("expected matched IDs in stderr, got: %s", errBuf.String())
	}
}

func TestSpeakersListCmd_HomeOnly_ReturnsSpeakers(t *testing.T) {
	homeDir := t.TempDir()

	dir := filepath.Join(homeDir, "homechar")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "speaker.json"), []byte(`{"name": "ホームキャラ", "styles": []}`), 0644); err != nil {
		t.Fatal(err)
	}

	deps := &Deps{
		Speakers: &SpeakersDeps{
			AssetsDirsMergedFunc: func() ([]string, error) {
				return []string{homeDir}, nil
			},
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
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].ID != "homechar" {
		t.Errorf("expected id 'homechar', got %q", items[0].ID)
	}
}

func TestSpeakersListCmd_MergeProjectAndHome_ReturnsBoth(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()

	// project: charA
	dirA := filepath.Join(projectDir, "charA")
	if err := os.MkdirAll(dirA, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirA, "speaker.json"), []byte(`{"name": "キャラA", "styles": []}`), 0644); err != nil {
		t.Fatal(err)
	}

	// home: charB
	dirB := filepath.Join(homeDir, "charB")
	if err := os.MkdirAll(dirB, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirB, "speaker.json"), []byte(`{"name": "キャラB", "styles": []}`), 0644); err != nil {
		t.Fatal(err)
	}

	deps := &Deps{
		Speakers: &SpeakersDeps{
			AssetsDirsMergedFunc: func() ([]string, error) {
				return []string{projectDir, homeDir}, nil
			},
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
		t.Fatalf("expected 2 items, got %d: %+v", len(items), items)
	}
	found := map[string]string{}
	for _, item := range items {
		found[item.ID] = item.Name
	}
	if found["charA"] != "キャラA" {
		t.Errorf("expected charA name 'キャラA', got %q", found["charA"])
	}
	if found["charB"] != "キャラB" {
		t.Errorf("expected charB name 'キャラB', got %q", found["charB"])
	}
}

func TestSpeakersListCmd_ProjectPriority_WhenSameID(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()

	// Both have "shared" but different names
	for dir, name := range map[string]string{projectDir: "プロジェクト版", homeDir: "ホーム版"} {
		d := filepath.Join(dir, "shared")
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
		content := `{"name": "` + name + `", "styles": []}`
		if err := os.WriteFile(filepath.Join(d, "speaker.json"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	deps := &Deps{
		Speakers: &SpeakersDeps{
			AssetsDirsMergedFunc: func() ([]string, error) {
				return []string{projectDir, homeDir}, nil
			},
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
	if len(items) != 1 {
		t.Fatalf("expected 1 item (merged), got %d: %+v", len(items), items)
	}
	if items[0].Name != "プロジェクト版" {
		t.Errorf("expected project name 'プロジェクト版', got %q", items[0].Name)
	}
}

func TestSpeakersProfileCmd_ProjectPriority_WhenSameID(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()

	for dir, name := range map[string]string{projectDir: "プロジェクト版", homeDir: "ホーム版"} {
		d := filepath.Join(dir, "shared")
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
		content := `{"name": "` + name + `", "styles": []}`
		if err := os.WriteFile(filepath.Join(d, "speaker.json"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	deps := &Deps{
		Speakers: &SpeakersDeps{
			AssetsDirsMergedFunc: func() ([]string, error) {
				return []string{projectDir, homeDir}, nil
			},
		},
	}
	rootCmd := makeRootCmd(deps)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"speakers", "profile", "--id", "shared"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %v", err)
	}
	if result["name"] != "プロジェクト版" {
		t.Errorf("expected project name 'プロジェクト版', got %v", result["name"])
	}
}

func TestSpeakersProfileCmd_DescriptionFileMissing_ContinuesWithWarning(t *testing.T) {
	assetsDir := t.TempDir()

	zundaDir := filepath.Join(assetsDir, "zundamon")
	if err := os.MkdirAll(zundaDir, 0755); err != nil {
		t.Fatal(err)
	}

	speakerJSON := `{
		"name": "ずんだもん",
		"profile": {
			"pronoun": "ボク",
			"speechSuffix": ["〜のだ"],
			"personality": ["元気"],
			"speakers": {"ノーマル": 3},
			"descriptionPath": "nonexistent.md"
		},
		"styles": []
	}`

	if err := os.WriteFile(filepath.Join(zundaDir, "speaker.json"), []byte(speakerJSON), 0644); err != nil {
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
	rootCmd.SetArgs([]string{"speakers", "profile", "--id", "zundamon"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected no error (should continue with warning), got: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %v", err)
	}

	if result["description"] != "" {
		t.Errorf("expected empty description when file missing, got: %v", result["description"])
	}

	if !strings.Contains(errBuf.String(), "nonexistent.md") {
		t.Errorf("expected warning about missing file in stderr, got: %s", errBuf.String())
	}
}
