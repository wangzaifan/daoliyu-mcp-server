package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/daoliyu/daoliyu-mcp/internal/app"
	"github.com/daoliyu/daoliyu-mcp/internal/plugin"
)

func TestMCPRejectsAdminAndAcceptsCallToken(t *testing.T) {
	tokens, err := app.NewTokenStore(t.TempDir(), "admin")
	if err != nil {
		t.Fatal(err)
	}
	call, err := tokens.Create("client", []string{"*"})
	if err != nil {
		t.Fatal(err)
	}
	h := requireMCPToken(tokens, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	for value, want := range map[string]int{"admin": 401, call.Token: 204} {
		r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		r.Header.Set("Authorization", "Bearer "+value)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("token %q: got %d want %d", value, w.Code, want)
		}
	}
}

func TestRetireLegacySiyuanPreservesData(t *testing.T) {
	d := t.TempDir()
	pluginDir := filepath.Join(d, "siyuan")
	if err := os.MkdirAll(filepath.Join(pluginDir, "data"), 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := `{"id":"siyuan","name":"旧思源","version":"0.2.0","provider":"siyuan"}`
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "data", "keep.json"), []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	manager, err := plugin.NewManager(d, func(string) bool { return true }, func(string, bool) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if err := retireLegacySiyuan(manager); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Get("siyuan"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy plugin still installed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(d, "siyuan.data", "keep.json")); err != nil {
		t.Fatalf("preserved data missing: %v", err)
	}
}
