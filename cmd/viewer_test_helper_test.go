package cmd

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/canpok1/vox-actor/internal/infra"
)

// setupFakeViewer sets up a fake viewer server with a lock file and returns the lock path.
// Both say and act tests use this helper.
func setupFakeViewer(t *testing.T, handler http.Handler) (lockPath string, srv *httptest.Server) {
	t.Helper()
	srv = httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	lockPath = filepath.Join(dir, "viewer.lock")
	vl, err := infra.AcquireViewerLock(lockPath)
	if err != nil {
		t.Fatalf("AcquireViewerLock: %v", err)
	}
	t.Cleanup(func() { _ = vl.Release() })

	if err := vl.WriteAddr(srv.Listener.Addr().String()); err != nil {
		t.Fatalf("WriteAddr: %v", err)
	}
	return lockPath, srv
}
