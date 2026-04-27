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

// FilterBySpeakers は speakers で指定した名前の asset エントリのみを返す。
// speakers が空の場合は全エントリを返す。
// 存在しない speaker 名が指定された場合はエラーを返す。
func (a *AssetsJSON) FilterBySpeakers(speakers []string) (map[string]AssetEntry, error) {
	if len(speakers) == 0 {
		return a.Assets, nil
	}
	result := make(map[string]AssetEntry)
	for _, name := range speakers {
		entry, ok := a.Assets[name]
		if !ok {
			return nil, fmt.Errorf("speaker %q not found in vox-actor-assets.json", name)
		}
		result[name] = entry
	}
	return result, nil
}
