package entity_test

import (
	"testing"

	"github.com/canpok1/vox-actor/internal/domain/entity"
)

func TestParseAssetsJSON_Valid(t *testing.T) {
	input := `{
		"assets": {
			"zundamon": { "path": "groups/zundamon" },
			"metan":    { "path": "groups/metan" }
		}
	}`

	got, err := entity.ParseAssetsJSON([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Assets) != 2 {
		t.Fatalf("len(Assets) = %d, want 2", len(got.Assets))
	}
	if got.Assets["zundamon"].Path != "groups/zundamon" {
		t.Errorf(`Assets["zundamon"].Path = %q, want "groups/zundamon"`, got.Assets["zundamon"].Path)
	}
	if got.Assets["metan"].Path != "groups/metan" {
		t.Errorf(`Assets["metan"].Path = %q, want "groups/metan"`, got.Assets["metan"].Path)
	}
}

func TestParseAssetsJSON_InvalidJSON(t *testing.T) {
	_, err := entity.ParseAssetsJSON([]byte(`{invalid}`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestValidateAssetsJSON_Valid(t *testing.T) {
	a := &entity.AssetsJSON{
		Assets: map[string]entity.AssetEntry{
			"zundamon": {Path: "groups/zundamon"},
		},
	}
	if err := a.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestValidateAssetsJSON_EmptyAssets(t *testing.T) {
	a := &entity.AssetsJSON{Assets: map[string]entity.AssetEntry{}}
	if err := a.Validate(); err != nil {
		t.Errorf("empty assets should be valid: %v", err)
	}
}

func TestValidateAssetsJSON_MissingPath(t *testing.T) {
	a := &entity.AssetsJSON{
		Assets: map[string]entity.AssetEntry{
			"zundamon": {Path: ""},
		},
	}
	if err := a.Validate(); err == nil {
		t.Error("expected validation error for missing path, got nil")
	}
}
