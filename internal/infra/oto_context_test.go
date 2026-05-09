package infra

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ebitengine/oto/v3"
)

// TestGetOtoContext_FactoryCalledOnce はシングルトンの呼び出しが1回のみであることを確認する。
// 注意: sync.Once を操作するため t.Parallel() 不可。他のGetOtoContextテストと排他する必要がある。
func TestGetOtoContext_FactoryCalledOnce(t *testing.T) {
	origFactory := otoContextFactory
	defer func() {
		otoContextFactory = origFactory
		ResetOtoContext()
	}()
	ResetOtoContext()

	callCount := 0
	otoContextFactory = func() (*oto.Context, error) {
		callCount++
		return nil, fmt.Errorf("test error")
	}

	_, _ = getOtoContext()
	_, _ = getOtoContext()

	if callCount != 1 {
		t.Fatalf("expected factory called once, got %d", callCount)
	}
}

// TestGetOtoContext_PropagatesError はfactoryのエラーが呼び出し元に伝播することを確認する。
func TestGetOtoContext_PropagatesError(t *testing.T) {
	origFactory := otoContextFactory
	defer func() {
		otoContextFactory = origFactory
		ResetOtoContext()
	}()
	ResetOtoContext()

	expectedErr := fmt.Errorf("no audio device")
	otoContextFactory = func() (*oto.Context, error) {
		return nil, expectedErr
	}

	_, err := getOtoContext()
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

// TestGetOtoContext_ErrorCachedAcrossCalls はエラーが2回目の呼び出しでもキャッシュされることを確認する。
func TestGetOtoContext_ErrorCachedAcrossCalls(t *testing.T) {
	origFactory := otoContextFactory
	defer func() {
		otoContextFactory = origFactory
		ResetOtoContext()
	}()
	ResetOtoContext()

	expectedErr := fmt.Errorf("device init failed")
	otoContextFactory = func() (*oto.Context, error) {
		return nil, expectedErr
	}

	_, err1 := getOtoContext()
	_, err2 := getOtoContext()

	if !errors.Is(err1, expectedErr) {
		t.Fatalf("first call: expected %v, got %v", expectedErr, err1)
	}
	if !errors.Is(err2, expectedErr) {
		t.Fatalf("second call: expected %v, got %v", expectedErr, err2)
	}
}
