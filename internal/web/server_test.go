package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/daoliyu/daoliyu-mcp/internal/plugin"
)

func TestTokenManagerIsASeparatePage(t *testing.T) {
	h := (&Server{}).Handler()

	root := httptest.NewRecorder()
	h.ServeHTTP(root, httptest.NewRequest(http.MethodGet, "/", nil))
	if root.Code != http.StatusOK || strings.Contains(root.Body.String(), `id="tokenSettings"`) {
		t.Fatal("plugin page must not contain the token manager")
	}

	tokens := httptest.NewRecorder()
	h.ServeHTTP(tokens, httptest.NewRequest(http.MethodGet, "/tokens", nil))
	if tokens.Code != http.StatusOK || !strings.Contains(tokens.Body.String(), `id="tokenSettings"`) || strings.Contains(tokens.Body.String(), `id="adminToken"`) {
		t.Fatal("token manager page is missing")
	}
	if !strings.Contains(tokens.Body.String(), `id="tokenCount"`) {
		t.Fatal("token manager page is missing token count")
	}

	about := httptest.NewRecorder()
	h.ServeHTTP(about, httptest.NewRequest(http.MethodGet, "/api/about", nil))
	if about.Code != http.StatusOK || !strings.Contains(about.Body.String(), `"currentVersion":"0.3.3"`) {
		t.Fatalf("about=%d body=%s", about.Code, about.Body.String())
	}
}

func TestPluginStatusAlwaysIncludesLatency(t *testing.T) {
	manager, err := plugin.NewManager(t.TempDir(), func(string) bool { return true }, func(string, bool) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	manifest := plugin.Manifest{ID: "demo.provider", Name: "Demo", Version: "1", Provider: "demo", Tools: []plugin.ToolManifest{{Name: "demo_tool"}}}
	if err := manager.Seed(manifest); err != nil {
		t.Fatal(err)
	}
	h := (&Server{Plugins: manager}).Handler()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/plugins/demo.provider/status", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"latencyMs":0`) {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
