package main

import (
	"io/fs"
	"testing"
)

func TestStreamAssetsFS_Embeds(t *testing.T) {
	assetsFS, err := streamAssetsFS()
	if err != nil {
		t.Fatalf("streamAssetsFS returned error: %v", err)
	}
	if assetsFS == nil {
		t.Fatal("streamAssetsFS returned nil FS")
	}
	entries, err := fs.ReadDir(assetsFS, ".")
	if err != nil {
		t.Fatalf("fs.ReadDir failed: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one embedded entry in frontend/dist (e.g. .gitkeep or built assets), got 0")
	}
}
