package entity

import (
	"encoding/json"
	"fmt"
)

// AssetsJSON は vox-actor-assets.json のトップレベル構造を表す。
type AssetsJSON struct {
	Assets map[string]AssetEntry `json:"assets"`
}

// AssetEntry は assets マップの各エントリを表す。
type AssetEntry struct {
	Path string `json:"path"`
}

// ParseAssetsJSON は JSON バイト列を AssetsJSON にパースする。
func ParseAssetsJSON(data []byte) (*AssetsJSON, error) {
	var a AssetsJSON
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, fmt.Errorf("failed to parse assets JSON: %w", err)
	}
	return &a, nil
}

// Validate は AssetsJSON の必須フィールドを検証する。
func (a *AssetsJSON) Validate() error {
	for name, entry := range a.Assets {
		if entry.Path == "" {
			return fmt.Errorf("assets[%q].path is required", name)
		}
	}
	return nil
}
