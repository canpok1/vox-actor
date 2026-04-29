//go:build e2e

package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createSpeakerInDir(t *testing.T, assetsDir, id, jsonContent string) {
	t.Helper()
	dir := filepath.Join(assetsDir, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "speaker.json"), []byte(jsonContent), 0o644); err != nil {
		t.Fatalf("write speaker.json for %s: %v", id, err)
	}
}

func createProjectSpeaker(t *testing.T, workspace, id, jsonContent string) {
	t.Helper()
	createSpeakerInDir(t, filepath.Join(workspace, "assets"), id, jsonContent)
}

func createHomeSpeaker(t *testing.T, homeDir, id, jsonContent string) {
	t.Helper()
	createSpeakerInDir(t, filepath.Join(homeDir, ".vox-actor", "assets"), id, jsonContent)
}

func simpleSpeakerJSON(name string) string {
	type entry struct {
		Name   string   `json:"name"`
		Styles []string `json:"styles"`
	}
	b, _ := json.Marshal(entry{Name: name, Styles: []string{}})
	return string(b)
}

func speakersEnv(workspace, homeDir string) map[string]string {
	return map[string]string{
		"VOX_ACTOR_WORKSPACE": workspace,
		"HOME":                homeDir,
	}
}

func assertWarningContains(t *testing.T, stderr, keyword string) {
	t.Helper()
	line := extractLineContaining(t, stderr, keyword)
	if !strings.Contains(line, "warning") {
		t.Errorf("expected warning in stderr containing %q\nline: %s", keyword, line)
	}
}

// --- Group A: 親コマンド / ヘルプ / 引数バリデーション ---

func TestSpeakersE2E_NoSubcmd_ShowsHelp(t *testing.T) {
	t.Parallel()

	stdout, stderr, exitCode := runCLI(t, nil, "speakers")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}
	for _, want := range []string{"speakers", "list", "profile"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected stdout to contain %q\nstdout:\n%s", want, stdout)
		}
	}
}

func TestSpeakersE2E_ListHelp_ExitZero(t *testing.T) {
	t.Parallel()

	_, stderr, exitCode := runCLI(t, nil, "speakers", "list", "--help")
	if exitCode != 0 {
		t.Fatalf("expected exit 0 for speakers list --help, got %d\nstderr:\n%s", exitCode, stderr)
	}
}

func TestSpeakersE2E_ProfileHelp_ExitZero(t *testing.T) {
	t.Parallel()

	_, stderr, exitCode := runCLI(t, nil, "speakers", "profile", "--help")
	if exitCode != 0 {
		t.Fatalf("expected exit 0 for speakers profile --help, got %d\nstderr:\n%s", exitCode, stderr)
	}
}

func TestSpeakersE2E_ProfileNoFlags_ExitNonZero(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	_, _, exitCode := runCLI(t,
		map[string]string{"VOX_ACTOR_WORKSPACE": workspace},
		"speakers", "profile",
	)
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit when neither --id nor --name specified")
	}
}

func TestSpeakersE2E_ProfileBothFlags_ExitNonZero(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	_, _, exitCode := runCLI(t,
		map[string]string{"VOX_ACTOR_WORKSPACE": workspace},
		"speakers", "profile", "--id", "zundamon", "--name", "ずんだもん",
	)
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit when both --id and --name specified")
	}
}

func TestSpeakersE2E_ListExtraArg_ExitZero(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	homeDir := t.TempDir()
	_, stderr, exitCode := runCLI(t, speakersEnv(workspace, homeDir), "speakers", "list", "extra-arg")
	if exitCode != 0 {
		t.Fatalf("expected exit 0 for extra positional arg, got %d\nstderr:\n%s", exitCode, stderr)
	}
}

// --- Group B: 一覧・プロフィール表示（実ファイル経由） ---

func TestSpeakersE2E_List_EmptyAssetsDir_ReturnsEmptyArray(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	homeDir := t.TempDir()
	stdout, stderr, exitCode := runCLI(t, speakersEnv(workspace, homeDir), "speakers", "list")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	var items []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &items); err != nil {
		t.Fatalf("expected valid JSON array, got: %v\nstdout:\n%s", err, stdout)
	}
	if len(items) != 0 {
		t.Errorf("expected empty array, got %d items", len(items))
	}
}

func TestSpeakersE2E_List_OneSpeaker_InOutput(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	homeDir := t.TempDir()
	createProjectSpeaker(t, workspace, "zundamon", simpleSpeakerJSON("ずんだもん"))

	stdout, stderr, exitCode := runCLI(t, speakersEnv(workspace, homeDir), "speakers", "list")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}
	if !strings.Contains(stdout, "zundamon") {
		t.Errorf("expected stdout to contain 'zundamon'\nstdout:\n%s", stdout)
	}
}

func TestSpeakersE2E_List_MultipleSpeakers_SortedByID(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	homeDir := t.TempDir()
	createProjectSpeaker(t, workspace, "zundamon", simpleSpeakerJSON("ずんだもん"))
	createProjectSpeaker(t, workspace, "metan", simpleSpeakerJSON("四国めたん"))
	createProjectSpeaker(t, workspace, "kiritan", simpleSpeakerJSON("東北きりたん"))

	stdout, stderr, exitCode := runCLI(t, speakersEnv(workspace, homeDir), "speakers", "list")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	var items []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &items); err != nil {
		t.Fatalf("expected valid JSON, got: %v\nstdout:\n%s", err, stdout)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d\nstdout:\n%s", len(items), stdout)
	}
	// ID 昇順: kiritan < metan < zundamon
	if items[0].ID != "kiritan" || items[1].ID != "metan" || items[2].ID != "zundamon" {
		t.Errorf("expected sorted by ID (kiritan, metan, zundamon), got: %+v", items)
	}
}

func TestSpeakersE2E_List_SpeakerNameReflected(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	homeDir := t.TempDir()
	createProjectSpeaker(t, workspace, "zundamon", simpleSpeakerJSON("ずんだもん"))

	stdout, stderr, exitCode := runCLI(t, speakersEnv(workspace, homeDir), "speakers", "list")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}
	if !strings.Contains(stdout, "ずんだもん") {
		t.Errorf("expected stdout to contain speaker name 'ずんだもん'\nstdout:\n%s", stdout)
	}
}

func TestSpeakersE2E_Profile_ByID_AllFields(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	homeDir := t.TempDir()

	descContent := "# ずんだもん キャラクター設定"
	speakerJSON := `{
		"name": "ずんだもん",
		"profile": {
			"pronoun": "ボク",
			"speechSuffix": ["〜のだ", "〜なのだ"],
			"personality": ["元気", "明るい"],
			"speakers": {"ノーマル": 3, "ツンツン": 7},
			"descriptionPath": "character.md"
		},
		"styles": []
	}`
	createProjectSpeaker(t, workspace, "zundamon", speakerJSON)
	writeTempFile(t, filepath.Join(workspace, "assets", "zundamon"), "character.md", descContent)

	stdout, stderr, exitCode := runCLI(t, speakersEnv(workspace, homeDir), "speakers", "profile", "--id", "zundamon")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %v\nstdout:\n%s", err, stdout)
	}
	if result["id"] != "zundamon" {
		t.Errorf("expected id 'zundamon', got %v", result["id"])
	}
	if result["name"] != "ずんだもん" {
		t.Errorf("expected name 'ずんだもん', got %v", result["name"])
	}
	if result["pronoun"] != "ボク" {
		t.Errorf("expected pronoun 'ボク', got %v", result["pronoun"])
	}
	if result["description"] != descContent {
		t.Errorf("expected description content %q, got %v", descContent, result["description"])
	}
}

// --- Group C: 階層マージ・優先順位 ---

func TestSpeakersE2E_List_ProjectOnly_ReturnsSpeakers(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	homeDir := t.TempDir()
	createProjectSpeaker(t, workspace, "zundamon", simpleSpeakerJSON("ずんだもん"))

	stdout, stderr, exitCode := runCLI(t, speakersEnv(workspace, homeDir), "speakers", "list")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}
	if !strings.Contains(stdout, "zundamon") {
		t.Errorf("expected stdout to contain 'zundamon'\nstdout:\n%s", stdout)
	}
}

func TestSpeakersE2E_List_HomeOnly_ReturnsSpeakers(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	homeDir := t.TempDir()
	createHomeSpeaker(t, homeDir, "metan", simpleSpeakerJSON("四国めたん"))

	stdout, stderr, exitCode := runCLI(t, speakersEnv(workspace, homeDir), "speakers", "list")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}
	if !strings.Contains(stdout, "metan") {
		t.Errorf("expected stdout to contain 'metan'\nstdout:\n%s", stdout)
	}
}

func TestSpeakersE2E_List_ProjectAndHome_ReturnsBoth(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	homeDir := t.TempDir()
	createProjectSpeaker(t, workspace, "zundamon", simpleSpeakerJSON("ずんだもん"))
	createHomeSpeaker(t, homeDir, "metan", simpleSpeakerJSON("四国めたん"))

	stdout, stderr, exitCode := runCLI(t, speakersEnv(workspace, homeDir), "speakers", "list")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}
	for _, want := range []string{"zundamon", "metan"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected stdout to contain %q\nstdout:\n%s", want, stdout)
		}
	}
}

func TestSpeakersE2E_List_SameID_ProjectPriority(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	homeDir := t.TempDir()
	createProjectSpeaker(t, workspace, "shared", simpleSpeakerJSON("プロジェクト版"))
	createHomeSpeaker(t, homeDir, "shared", simpleSpeakerJSON("ホーム版"))

	stdout, stderr, exitCode := runCLI(t, speakersEnv(workspace, homeDir), "speakers", "list")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	var items []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &items); err != nil {
		t.Fatalf("expected valid JSON, got: %v\nstdout:\n%s", err, stdout)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item (merged), got %d\nstdout:\n%s", len(items), stdout)
	}
	if items[0].Name != "プロジェクト版" {
		t.Errorf("expected project name 'プロジェクト版' to take priority, got %q", items[0].Name)
	}
}

func TestSpeakersE2E_Profile_SameID_ProjectPriority(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	homeDir := t.TempDir()
	createProjectSpeaker(t, workspace, "shared", simpleSpeakerJSON("プロジェクト版"))
	createHomeSpeaker(t, homeDir, "shared", simpleSpeakerJSON("ホーム版"))

	stdout, stderr, exitCode := runCLI(t, speakersEnv(workspace, homeDir), "speakers", "profile", "--id", "shared")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %v\nstdout:\n%s", err, stdout)
	}
	if result["name"] != "プロジェクト版" {
		t.Errorf("expected project name 'プロジェクト版' to take priority, got %v", result["name"])
	}
}

// --- Group D: profile --name 検索 ---

func TestSpeakersE2E_Profile_ByName_UniqueMatch_ReturnsProfile(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	homeDir := t.TempDir()
	createProjectSpeaker(t, workspace, "zundamon", simpleSpeakerJSON("ずんだもん"))

	stdout, stderr, exitCode := runCLI(t, speakersEnv(workspace, homeDir), "speakers", "profile", "--name", "ずんだもん")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}
	if !strings.Contains(stdout, "zundamon") {
		t.Errorf("expected stdout to contain 'zundamon'\nstdout:\n%s", stdout)
	}
}

func TestSpeakersE2E_Profile_ByName_DuplicateName_ExitNonZero(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	homeDir := t.TempDir()
	createProjectSpeaker(t, workspace, "char1", simpleSpeakerJSON("重複名前"))
	createProjectSpeaker(t, workspace, "char2", simpleSpeakerJSON("重複名前"))

	_, stderr, exitCode := runCLI(t, speakersEnv(workspace, homeDir), "speakers", "profile", "--name", "重複名前")
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit for duplicate name")
	}
	if !strings.Contains(stderr, "char1") && !strings.Contains(stderr, "char2") {
		t.Errorf("expected stderr to mention matched IDs\nstderr:\n%s", stderr)
	}
}

func TestSpeakersE2E_Profile_ByName_NotFound_ExitNonZero(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	homeDir := t.TempDir()

	_, _, exitCode := runCLI(t, speakersEnv(workspace, homeDir), "speakers", "profile", "--name", "存在しない名前")
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit for non-existent name")
	}
}

// --- Group E: profile の description ファイル ---

func TestSpeakersE2E_Profile_DescriptionFileExists_InOutput(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	homeDir := t.TempDir()

	descContent := "# ずんだもん キャラクター設定\n\nずんだ餅の精。"
	speakerJSON := `{"name": "ずんだもん", "profile": {"descriptionPath": "character.md"}, "styles": []}`
	createProjectSpeaker(t, workspace, "zundamon", speakerJSON)
	writeTempFile(t, filepath.Join(workspace, "assets", "zundamon"), "character.md", descContent)

	stdout, stderr, exitCode := runCLI(t, speakersEnv(workspace, homeDir), "speakers", "profile", "--id", "zundamon")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %v\nstdout:\n%s", err, stdout)
	}
	if result["description"] != descContent {
		t.Errorf("expected description to contain file content %q, got %v", descContent, result["description"])
	}
}

func TestSpeakersE2E_Profile_DescriptionFileMissing_EmptyDescriptionWithWarning(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	homeDir := t.TempDir()
	createProjectSpeaker(t, workspace, "zundamon",
		`{"name": "ずんだもん", "profile": {"descriptionPath": "nonexistent.md"}, "styles": []}`)

	stdout, stderr, exitCode := runCLI(t, speakersEnv(workspace, homeDir), "speakers", "profile", "--id", "zundamon")
	if exitCode != 0 {
		t.Fatalf("expected exit 0 (continue with warning), got %d\nstderr:\n%s", exitCode, stderr)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %v\nstdout:\n%s", err, stdout)
	}
	if result["description"] != "" {
		t.Errorf("expected empty description when file missing, got: %v", result["description"])
	}
	if !strings.Contains(stderr, "nonexistent.md") {
		t.Errorf("expected warning about missing file in stderr\nstderr:\n%s", stderr)
	}
}

// --- Group F: 異常系・エラーハンドリング ---

func TestSpeakersE2E_List_AssetsDirMissing_EmptyList(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	homeDir := t.TempDir()

	stdout, stderr, exitCode := runCLI(t, speakersEnv(workspace, homeDir), "speakers", "list")
	if exitCode != 0 {
		t.Fatalf("expected exit 0 (skip missing dir), got %d\nstderr:\n%s", exitCode, stderr)
	}

	var items []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &items); err != nil {
		t.Fatalf("expected valid JSON, got: %v\nstdout:\n%s", err, stdout)
	}
	if len(items) != 0 {
		t.Errorf("expected empty list, got %d items", len(items))
	}
}

func TestSpeakersE2E_List_SpeakerJSONMissing_WarningAndSkip(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	homeDir := t.TempDir()

	// speaker.json なしのディレクトリのみ作成（createProjectSpeaker は speaker.json も書くため使えない）
	if err := os.MkdirAll(filepath.Join(workspace, "assets", "zundamon"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	stdout, stderr, exitCode := runCLI(t, speakersEnv(workspace, homeDir), "speakers", "list")
	if exitCode != 0 {
		t.Fatalf("expected exit 0 (warning + skip), got %d\nstderr:\n%s", exitCode, stderr)
	}

	var items []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &items); err != nil {
		t.Fatalf("expected valid JSON, got: %v\nstdout:\n%s", err, stdout)
	}
	if len(items) != 0 {
		t.Errorf("expected empty list (skipped), got %d items", len(items))
	}
	assertWarningContains(t, stderr, "zundamon")
}

func TestSpeakersE2E_List_SpeakerJSONInvalid_WarningAndSkip(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	homeDir := t.TempDir()
	createProjectSpeaker(t, workspace, "zundamon", "{invalid json}")

	stdout, stderr, exitCode := runCLI(t, speakersEnv(workspace, homeDir), "speakers", "list")
	if exitCode != 0 {
		t.Fatalf("expected exit 0 (warning + skip), got %d\nstderr:\n%s", exitCode, stderr)
	}

	var items []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &items); err != nil {
		t.Fatalf("expected valid JSON, got: %v\nstdout:\n%s", err, stdout)
	}
	if len(items) != 0 {
		t.Errorf("expected empty list (skipped), got %d items", len(items))
	}
	assertWarningContains(t, stderr, "zundamon")
}

func TestSpeakersE2E_Profile_ByID_NotFound_ExitNonZero(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	homeDir := t.TempDir()

	_, _, exitCode := runCLI(t, speakersEnv(workspace, homeDir), "speakers", "profile", "--id", "nonexistent")
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit for non-existent speaker ID")
	}
}
