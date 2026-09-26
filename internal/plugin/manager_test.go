package plugin

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestManifestValidate(t *testing.T) {
	if err := (Manifest{ID: "bad id", Name: "x", Version: "1"}).Validate(); err == nil {
		t.Fatal("expected invalid id")
	}
	if err := (Manifest{ID: "demo.plugin", Name: "x", Version: "1"}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (Manifest{ID: "demo.http", Name: "x", Version: "1", Type: "mcp-http", Endpoint: "http://127.0.0.1/mcp"}).Validate(); err == nil {
		t.Fatal("expected missing tool prefix to fail")
	}
}

func TestMCPHTTPPluginDiscoversPrefixesAndForwards(t *testing.T) {
	upstream := mcp.NewServer(&mcp.Implementation{Name: "upstream", Version: "1"}, nil)
	mcp.AddTool(upstream, &mcp.Tool{Name: "echo", Description: "Echo text"}, func(_ context.Context, _ *mcp.CallToolRequest, in struct {
		Text string `json:"text"`
	}) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: in.Text}}}, nil, nil
	})
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return upstream }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer upstream-secret" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	}))
	defer httpServer.Close()

	d := t.TempDir()
	dir := filepath.Join(d, "sisyphus")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{ID: "sisyphus", Name: "Sisyphus", Version: "0.6.7", Type: "mcp-http", Endpoint: httpServer.URL, ToolPrefix: "siyuan_"}
	if err := writeManifest(dir, manifest); err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager(d, func(string) bool { return true }, func(string, bool) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	manager.Config = func(string) json.RawMessage { return json.RawMessage(`{"token":"upstream-secret"}`) }
	status, err := manager.Status(context.Background(), "sisyphus")
	if err != nil {
		t.Fatal(err)
	}
	if status.State != "ready" || status.ToolCount != 1 || status.Tools[0].Name != "siyuan_echo" {
		t.Fatalf("status=%+v", status)
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "1"}, nil)
	if err := manager.RegisterExternal(server); err != nil {
		t.Fatal(err)
	}
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	clientSession, err := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "1"}, nil).Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{Name: "siyuan_echo", Arguments: map[string]any{"text": "ok"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Content[0].(*mcp.TextContent).Text; got != "ok" {
		t.Fatalf("result=%q", got)
	}
	_ = clientSession.Close()
	_ = serverSession.Wait()
}

func TestExternalPluginReceivesItsConfig(t *testing.T) {
	d := t.TempDir()
	dir := filepath.Join(d, "demo.config")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\ninput=$(cat)\ncase \"$input\" in *'\"suffix\":\"!\"'*) printf '%s' '{\"content\":[{\"type\":\"text\",\"text\":\"configured\"}]}' ;; *) exit 9 ;; esac\n"
	if err := os.WriteFile(filepath.Join(dir, "plugin"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := `{"id":"demo.config","name":"Demo","version":"1","entry":"plugin","tools":[{"name":"demo_call","description":"demo","inputSchema":{"type":"object"}}]}`
	if err := os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager(d, func(string) bool { return true }, func(string, bool) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	manager.Config = func(string) json.RawMessage { return json.RawMessage(`{"suffix":"!"}`) }
	server := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "1"}, nil)
	if err := manager.RegisterExternal(server); err != nil {
		t.Fatal(err)
	}
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	clientSession, err := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "1"}, nil).Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{Name: "demo_call", Arguments: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	if text := result.Content[0].(*mcp.TextContent).Text; !strings.Contains(text, "configured") {
		t.Fatalf("result=%q", text)
	}
	_ = clientSession.Close()
	_ = serverSession.Wait()
}

func TestPluginDataCanBePreservedRestoredOrPurged(t *testing.T) {
	d := t.TempDir()
	m, err := NewManager(d, func(string) bool { return true }, func(string, bool) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	installTestBundle(t, m, "demo.data")
	dataFile := filepath.Join(d, "demo.data", "data", "value.txt")
	if err := os.MkdirAll(filepath.Dir(dataFile), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dataFile, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := m.UninstallWithOptions("demo.data", false); err != nil {
		t.Fatal(err)
	}
	installTestBundle(t, m, "demo.data")
	if value, err := os.ReadFile(dataFile); err != nil || string(value) != "keep" {
		t.Fatalf("restored value=%q err=%v", value, err)
	}
	if err := m.UninstallWithOptions("demo.data", true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(d, "demo.data.data")); !os.IsNotExist(err) {
		t.Fatalf("purged data still exists: %v", err)
	}
}

func installTestBundle(t *testing.T, manager *Manager, id string) {
	t.Helper()
	var bundle bytes.Buffer
	zw := zip.NewWriter(&bundle)
	w, err := zw.Create("plugin.json")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte(`{"id":"` + id + `","name":"Demo","version":"1"}`))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("bundle", "plugin.zip")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(bundle.Bytes())
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/api/plugins", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if _, err := manager.Install(req); err != nil {
		t.Fatal(err)
	}
}

func TestSeedAndUninstallDoesNotReseed(t *testing.T) {
	d := t.TempDir()
	m, err := NewManager(d, func(string) bool { return true }, func(string, bool) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{ID: "demo.plugin", Name: "Demo", Version: "1", Provider: "demo"}
	if err := m.Seed(manifest); err != nil {
		t.Fatal(err)
	}
	if err := m.Uninstall(manifest.ID); err != nil {
		t.Fatal(err)
	}
	if err := m.Seed(manifest); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Get(manifest.ID); err == nil {
		t.Fatal("uninstalled plugin was reseeded")
	}
}

func TestSeedUpdatesProviderManifestWithoutReplacingData(t *testing.T) {
	d := t.TempDir()
	m, err := NewManager(d, func(string) bool { return true }, func(string, bool) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	old := Manifest{ID: "demo.provider", Name: "Demo", Version: "1", Provider: "demo"}
	if err := m.Seed(old); err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(d, old.ID, "data", "keep")
	if err := os.MkdirAll(filepath.Dir(data), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(data, []byte("yes"), 0o600); err != nil {
		t.Fatal(err)
	}
	updated := old
	updated.Version = "2"
	if err := m.Seed(updated); err != nil {
		t.Fatal(err)
	}
	got, err := m.Get(old.ID)
	if err != nil || got.Version != "2" {
		t.Fatalf("plugin=%+v err=%v", got, err)
	}
	if _, err := os.Stat(data); err != nil {
		t.Fatal(err)
	}
}
