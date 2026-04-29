//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// apiCharactersResp は GET /api/characters のレスポンス。
type apiCharactersResp struct {
	Enabled    bool                `json:"enabled"`
	Characters []apiCharacterEntry `json:"characters"`
}

// apiCharacterEntry は /api/characters の各エントリ。
type apiCharacterEntry struct {
	SpeakerName string `json:"speakerName"`
	StyleName   string `json:"styleName"`
	MouthClosed string `json:"mouthClosed"`
	MouthOpen   string `json:"mouthOpen"`
}

// characterStyleDef はテスト用キャラクタースタイルの定義。
type characterStyleDef struct {
	styleName   string
	mouthClosed string
	mouthOpen   string
}

// makeCharacterAssets はassetsDir配下にcharID名のサブディレクトリを作成し、
// speaker.json と各スタイルの画像ファイル（fake）を配置する。
func makeCharacterAssets(t *testing.T, assetsDir, charID, speakerName string, styles []characterStyleDef) {
	t.Helper()
	charDir := filepath.Join(assetsDir, charID)
	if err := os.MkdirAll(charDir, 0o755); err != nil {
		t.Fatalf("mkdir charDir %s: %v", charDir, err)
	}
	for _, s := range styles {
		for _, img := range []string{s.mouthClosed, s.mouthOpen} {
			if err := os.WriteFile(filepath.Join(charDir, img), []byte("fake-img"), 0o644); err != nil {
				t.Fatalf("write image %s: %v", img, err)
			}
		}
	}
	type styleJSON struct {
		StyleName   string `json:"styleName"`
		MouthClosed string `json:"mouthClosed"`
		MouthOpened string `json:"mouthOpened"`
	}
	type speakerJSONContent struct {
		Name   string      `json:"name"`
		Styles []styleJSON `json:"styles"`
	}
	sp := speakerJSONContent{Name: speakerName}
	for _, s := range styles {
		sp.Styles = append(sp.Styles, styleJSON{
			StyleName:   s.styleName,
			MouthClosed: s.mouthClosed,
			MouthOpened: s.mouthOpen,
		})
	}
	data, err := json.Marshal(sp)
	if err != nil {
		t.Fatalf("marshal speaker.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(charDir, "speaker.json"), data, 0o644); err != nil {
		t.Fatalf("write speaker.json: %v", err)
	}
}

// startViewerGetAddr はviewerを起動してアドレスを返す。
func startViewerGetAddr(t *testing.T, env map[string]string, port int) (*viewerProcess, string) {
	t.Helper()
	vp := startViewer(t, env, "--port", fmt.Sprintf("%d", port))
	line := vp.waitForStderr("stream server listening", 10*time.Second)
	addr := extractAddrFromLog(line)
	if addr == "" {
		t.Fatalf("failed to extract addr from log line: %q", line)
	}
	return vp, addr
}

// getCharactersResponse はGET /api/characters を実行してレスポンスを返す。
func getCharactersResponse(t *testing.T, addr string) (*http.Response, []byte) {
	t.Helper()
	url := fmt.Sprintf("http://%s/api/characters", addr)
	var (
		resp   *http.Response
		getErr error
	)
	for i := 0; i < 20; i++ {
		resp, getErr = http.Get(url) //nolint:noctx
		if getErr == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if getErr != nil {
		t.Fatalf("GET /api/characters: %v", getErr)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return resp, body
}

// getAPICharacters はGET /api/characters を実行してパース済みレスポンスを返す。
func getAPICharacters(t *testing.T, addr string) *apiCharactersResp {
	t.Helper()
	resp, body := getCharactersResponse(t, addr)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", resp.StatusCode, body)
	}
	var result apiCharactersResp
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("decode /api/characters: %v\nbody: %s", err, body)
	}
	return &result
}

func TestViewerE2E_Characters_NoAssetsDir(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	workspace := t.TempDir()
	port := freePort(t)

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
	}, port)

	result := getAPICharacters(t, addr)
	if result.Enabled {
		t.Errorf("expected enabled=false when no assets dir exists")
	}
	if len(result.Characters) != 0 {
		t.Errorf("expected empty characters, got %d entries", len(result.Characters))
	}
}

func TestViewerE2E_Characters_ProjectAssetsOnly(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	workspace := t.TempDir()
	port := freePort(t)

	projectAssetsDir := filepath.Join(workspace, "assets")
	makeCharacterAssets(t, projectAssetsDir, "zundamon", "ずんだもん", []characterStyleDef{
		{"ノーマル", "mouth-closed.png", "mouth-open.png"},
	})

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
	}, port)

	result := getAPICharacters(t, addr)
	if !result.Enabled {
		t.Errorf("expected enabled=true with project assets")
	}
	if len(result.Characters) != 1 {
		t.Fatalf("expected 1 character, got %d\ncharacters: %+v", len(result.Characters), result.Characters)
	}
	if result.Characters[0].SpeakerName != "ずんだもん" {
		t.Errorf("expected speakerName=ずんだもん, got %q", result.Characters[0].SpeakerName)
	}
}

func TestViewerE2E_Characters_HomeAssetsOnly(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	workspace := t.TempDir()
	port := freePort(t)

	homeAssetsDir := filepath.Join(homeDir, ".vox-actor", "assets")
	makeCharacterAssets(t, homeAssetsDir, "metan", "四国めたん", []characterStyleDef{
		{"ノーマル", "mouth-closed.png", "mouth-open.png"},
	})

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
	}, port)

	result := getAPICharacters(t, addr)
	if !result.Enabled {
		t.Errorf("expected enabled=true with home assets")
	}
	if len(result.Characters) != 1 {
		t.Fatalf("expected 1 character, got %d\ncharacters: %+v", len(result.Characters), result.Characters)
	}
	if result.Characters[0].SpeakerName != "四国めたん" {
		t.Errorf("expected speakerName=四国めたん, got %q", result.Characters[0].SpeakerName)
	}
}

func TestViewerE2E_Characters_ProjectPriorityOverHome(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	workspace := t.TempDir()
	port := freePort(t)

	// 同じ charID "zundamon" を project と home 両方に配置
	projectAssetsDir := filepath.Join(workspace, "assets")
	makeCharacterAssets(t, projectAssetsDir, "zundamon", "ずんだもんproject", []characterStyleDef{
		{"ノーマル", "mouth-closed.png", "mouth-open.png"},
	})

	homeAssetsDir := filepath.Join(homeDir, ".vox-actor", "assets")
	makeCharacterAssets(t, homeAssetsDir, "zundamon", "ずんだもんhome", []characterStyleDef{
		{"ノーマル", "mouth-closed.png", "mouth-open.png"},
	})

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
	}, port)

	result := getAPICharacters(t, addr)
	if !result.Enabled {
		t.Errorf("expected enabled=true")
	}
	if len(result.Characters) != 1 {
		t.Fatalf("expected 1 character (project wins over home), got %d\ncharacters: %+v", len(result.Characters), result.Characters)
	}
	if result.Characters[0].SpeakerName != "ずんだもんproject" {
		t.Errorf("expected project character to win, got speakerName=%q", result.Characters[0].SpeakerName)
	}
}

func TestViewerE2E_Characters_StylesExpanded(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	workspace := t.TempDir()
	port := freePort(t)

	// 2スタイルを持つスピーカー
	projectAssetsDir := filepath.Join(workspace, "assets")
	makeCharacterAssets(t, projectAssetsDir, "zundamon", "ずんだもん", []characterStyleDef{
		{"ノーマル", "mouth-closed.png", "mouth-open.png"},
		{"あまあま", "mouth-closed-ama.png", "mouth-open-ama.png"},
	})

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
	}, port)

	result := getAPICharacters(t, addr)
	if !result.Enabled {
		t.Errorf("expected enabled=true")
	}
	if len(result.Characters) != 2 {
		t.Fatalf("expected 2 entries (one per style), got %d\ncharacters: %+v", len(result.Characters), result.Characters)
	}
	styles := make(map[string]bool, len(result.Characters))
	for _, c := range result.Characters {
		styles[c.StyleName] = true
	}
	for _, want := range []string{"ノーマル", "あまあま"} {
		if !styles[want] {
			t.Errorf("expected style %q in characters\ncharacters: %+v", want, result.Characters)
		}
	}
}

func TestViewerE2E_Characters_InvalidSpeakerJSONSkipped(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	workspace := t.TempDir()
	port := freePort(t)

	projectAssetsDir := filepath.Join(workspace, "assets")

	// 有効なキャラクター
	makeCharacterAssets(t, projectAssetsDir, "zundamon", "ずんだもん", []characterStyleDef{
		{"ノーマル", "mouth-closed.png", "mouth-open.png"},
	})

	// 不正な speaker.json を持つキャラクター
	invalidCharDir := filepath.Join(projectAssetsDir, "invalid-char")
	if err := os.MkdirAll(invalidCharDir, 0o755); err != nil {
		t.Fatalf("mkdir invalid char: %v", err)
	}
	if err := os.WriteFile(filepath.Join(invalidCharDir, "speaker.json"), []byte("not-valid-json"), 0o644); err != nil {
		t.Fatalf("write invalid speaker.json: %v", err)
	}

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
	}, port)

	result := getAPICharacters(t, addr)
	if !result.Enabled {
		t.Errorf("expected enabled=true (valid character should be loaded)")
	}
	if len(result.Characters) != 1 {
		t.Fatalf("expected 1 character (invalid char skipped), got %d\ncharacters: %+v", len(result.Characters), result.Characters)
	}
	if result.Characters[0].SpeakerName != "ずんだもん" {
		t.Errorf("expected speakerName=ずんだもん, got %q", result.Characters[0].SpeakerName)
	}
}

func TestViewerE2E_Characters_ContentType(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	workspace := t.TempDir()
	port := freePort(t)

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
	}, port)

	resp, _ := getCharactersResponse(t, addr)
	ct := resp.Header.Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Errorf("expected Content-Type=application/json; charset=utf-8, got %q", ct)
	}
}
