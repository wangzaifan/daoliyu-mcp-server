package plugin

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Installed struct {
	Manifest
	Enabled bool   `json:"enabled"`
	Builtin bool   `json:"builtin"`
	Dir     string `json:"-"`
	DataDir string `json:"-"`
}

type Manager struct {
	Dir        string
	Enabled    func(string) bool
	SetEnabled func(string, bool) error
	Config     func(string) json.RawMessage
	Providers  map[string]ProviderFunc
}

type ProviderFunc func(*mcp.Server)

type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type Status struct {
	State     string     `json:"state"`
	ToolCount int        `json:"toolCount"`
	Tools     []ToolInfo `json:"tools"`
	LatencyMS int64      `json:"latencyMs"`
}

func NewManager(dir string, enabled func(string) bool, setEnabled func(string, bool) error) (*Manager, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &Manager{Dir: dir, Enabled: enabled, SetEnabled: setEnabled, Providers: map[string]ProviderFunc{}}, nil
}

func (m *Manager) RegisterProvider(name string, provider ProviderFunc) { m.Providers[name] = provider }

func (m *Manager) Seed(manifest Manifest) error {
	if err := manifest.Validate(); err != nil {
		return err
	}
	dst := filepath.Join(m.Dir, manifest.ID)
	if _, err := os.Stat(dst + ".removed"); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if b, err := os.ReadFile(filepath.Join(dst, "plugin.json")); err == nil {
		var current Manifest
		if json.Unmarshal(b, &current) != nil || manifest.Provider == "" || current.Provider != manifest.Provider {
			return nil
		}
		return writeManifest(dst, manifest)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(dst, 0o700); err != nil {
		return err
	}
	return writeManifest(dst, manifest)
}

func writeManifest(dst string, manifest Manifest) error {
	b, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dst, "plugin.json"), append(b, '\n'), 0o600)
}

func (m *Manager) List() ([]Installed, error) {
	ents, err := os.ReadDir(m.Dir)
	if err != nil {
		return nil, err
	}
	result := make([]Installed, 0, len(ents))
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(m.Dir, e.Name(), "plugin.json"))
		if err != nil {
			continue
		}
		var manifest Manifest
		if json.Unmarshal(b, &manifest) != nil || manifest.Validate() != nil {
			continue
		}
		dir := filepath.Join(m.Dir, e.Name())
		result = append(result, Installed{Manifest: manifest, Enabled: m.Enabled(manifest.ID), Builtin: manifest.Provider != "", Dir: dir, DataDir: filepath.Join(dir, "data")})
	}
	return result, nil
}

func (m *Manager) Get(id string) (Installed, error) {
	if !idPattern.MatchString(id) {
		return Installed{}, fmt.Errorf("invalid plugin id")
	}
	b, err := os.ReadFile(filepath.Join(m.Dir, id, "plugin.json"))
	if err != nil {
		return Installed{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(b, &manifest); err != nil {
		return Installed{}, err
	}
	if err := manifest.Validate(); err != nil {
		return Installed{}, err
	}
	dir := filepath.Join(m.Dir, id)
	return Installed{Manifest: manifest, Enabled: m.Enabled(id), Builtin: manifest.Provider != "", Dir: dir, DataDir: filepath.Join(dir, "data")}, nil
}

func (m *Manager) Toggle(id string, enabled bool) error { return m.SetEnabled(id, enabled) }

func (m *Manager) Status(ctx context.Context, id string) (Status, error) {
	item, err := m.Get(id)
	if err != nil {
		return Status{}, err
	}
	if !item.Enabled {
		return Status{State: "disabled", Tools: manifestTools(item)}, nil
	}
	started := time.Now()
	if item.Type == "mcp-http" {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		session, err := m.connectHTTP(ctx, item)
		if err != nil {
			return Status{}, err
		}
		defer session.Close()
		tools := []ToolInfo{}
		for tool, err := range session.Tools(ctx, nil) {
			if err != nil {
				return Status{}, err
			}
			tools = append(tools, ToolInfo{Name: item.ToolPrefix + tool.Name, Description: tool.Description})
		}
		return Status{State: "ready", ToolCount: len(tools), Tools: tools, LatencyMS: time.Since(started).Milliseconds()}, nil
	}
	if item.Provider == "" {
		entry := item.Entry
		if entry == "" {
			entry = "plugin"
		}
		info, err := os.Stat(filepath.Join(item.Dir, entry))
		if err != nil {
			return Status{}, err
		}
		if info.Mode()&0o111 == 0 {
			return Status{}, fmt.Errorf("plugin entry is not executable")
		}
	}
	tools := manifestTools(item)
	return Status{State: "ready", ToolCount: len(tools), Tools: tools, LatencyMS: time.Since(started).Milliseconds()}, nil
}

func manifestTools(item Installed) []ToolInfo {
	tools := make([]ToolInfo, 0, len(item.Tools))
	for _, tool := range item.Tools {
		tools = append(tools, ToolInfo{Name: tool.Name, Description: tool.Description})
	}
	return tools
}

func (m *Manager) Uninstall(id string) error {
	return m.UninstallWithOptions(id, false)
}

func (m *Manager) UninstallWithOptions(id string, purgeData bool) error {
	if !idPattern.MatchString(id) {
		return fmt.Errorf("invalid plugin id")
	}
	pluginDir := filepath.Join(m.Dir, id)
	dataDir := filepath.Join(pluginDir, "data")
	preserved := filepath.Join(m.Dir, id+".data")
	if purgeData {
		if err := os.RemoveAll(preserved); err != nil {
			return err
		}
	} else if _, err := os.Stat(dataDir); err == nil {
		if err := os.RemoveAll(preserved); err != nil {
			return err
		}
		if err := os.Rename(dataDir, preserved); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.RemoveAll(pluginDir); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(m.Dir, id+".removed"), []byte("removed\n"), 0o600); err != nil {
		return err
	}
	return m.SetEnabled(id, false)
}

func (m *Manager) Install(r *http.Request) (Manifest, error) {
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		return Manifest{}, err
	}
	file, _, err := r.FormFile("bundle")
	if err != nil {
		return Manifest{}, fmt.Errorf("bundle is required: %w", err)
	}
	defer file.Close()
	tmp, err := os.CreateTemp(m.Dir, ".upload-*.zip")
	if err != nil {
		return Manifest{}, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := io.CopyN(tmp, file, 8<<20+1); err != nil && err != io.EOF {
		tmp.Close()
		return Manifest{}, err
	}
	if err := tmp.Close(); err != nil {
		return Manifest{}, err
	}
	z, err := zip.OpenReader(tmpName)
	if err != nil {
		return Manifest{}, fmt.Errorf("invalid zip: %w", err)
	}
	defer z.Close()
	var manifest Manifest
	for _, f := range z.File {
		if filepath.ToSlash(f.Name) == "plugin.json" {
			b, e := readZipFile(f, 1<<20)
			if e != nil {
				return Manifest{}, e
			}
			if e := json.Unmarshal(b, &manifest); e != nil {
				return Manifest{}, e
			}
			break
		}
	}
	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}
	if manifest.Provider != "" {
		return Manifest{}, fmt.Errorf("provider is reserved for built-in plugins")
	}
	if manifest.Icon == "" {
		return Manifest{}, fmt.Errorf("plugin icon is required")
	}
	stage, err := os.MkdirTemp(m.Dir, ".stage-*")
	if err != nil {
		return Manifest{}, err
	}
	defer os.RemoveAll(stage)
	for _, f := range z.File {
		name := filepath.Clean(filepath.FromSlash(f.Name))
		if name == "." || filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, ".."+string(os.PathSeparator)) {
			return Manifest{}, fmt.Errorf("unsafe zip path %q", f.Name)
		}
		dst := filepath.Join(stage, name)
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(dst, 0o700); err != nil {
				return Manifest{}, err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
			return Manifest{}, err
		}
		in, e := f.Open()
		if e != nil {
			return Manifest{}, e
		}
		out, e := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o700)
		if e == nil {
			_, e = io.CopyN(out, in, 8<<20+1)
			if e == io.EOF {
				e = nil
			}
			out.Close()
		}
		in.Close()
		if e != nil {
			return Manifest{}, e
		}
	}
	if manifest.Icon != "" {
		if err := validateIcon(stage, manifest.Icon); err != nil {
			return Manifest{}, err
		}
	}
	dst := filepath.Join(m.Dir, manifest.ID)
	preserved := filepath.Join(m.Dir, manifest.ID+".data")
	dataDir := filepath.Join(dst, "data")
	if _, err := os.Stat(dataDir); err == nil {
		if err := os.RemoveAll(preserved); err != nil {
			return Manifest{}, err
		}
		if err := os.Rename(dataDir, preserved); err != nil {
			return Manifest{}, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return Manifest{}, err
	}
	if err := os.RemoveAll(dst); err != nil {
		return Manifest{}, err
	}
	_ = os.Remove(dst + ".removed")
	if err := os.Rename(stage, dst); err != nil {
		return Manifest{}, err
	}
	if err := os.RemoveAll(filepath.Join(dst, "data")); err != nil {
		return Manifest{}, err
	}
	if err := os.Rename(preserved, filepath.Join(dst, "data")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Manifest{}, err
	}
	if err := m.SetEnabled(manifest.ID, false); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func validateIcon(root, name string) error {
	clean := filepath.Clean(name)
	if filepath.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("invalid plugin icon path")
	}
	if strings.ToLower(filepath.Ext(clean)) != ".png" {
		return fmt.Errorf("plugin icon must be a PNG")
	}
	file, err := os.Open(filepath.Join(root, clean))
	if err != nil {
		return fmt.Errorf("plugin icon is required: %w", err)
	}
	defer file.Close()
	if info, err := file.Stat(); err != nil || info.Size() > 512<<10 {
		return fmt.Errorf("plugin icon must be at most 512 KiB")
	}
	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return fmt.Errorf("invalid plugin icon: %w", err)
	}
	if config.Width != config.Height || config.Width < 64 || config.Width > 512 {
		return fmt.Errorf("plugin icon must be square and between 64x64 and 512x512")
	}
	return nil
}

func readZipFile(f *zip.File, max int64) ([]byte, error) {
	r, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(io.LimitReader(r, max))
}

func (m *Manager) RegisterExternal(s *mcp.Server) error {
	return m.RegisterExternalScoped(s, nil)
}

func (m *Manager) RegisterExternalScoped(s *mcp.Server, scopes map[string]bool) error {
	items, err := m.List()
	if err != nil {
		return err
	}
	for _, item := range items {
		if !item.Enabled {
			continue
		}
		if scopes != nil && !scopes[item.ID] && !scopes["*"] {
			continue
		}
		if item.Provider != "" {
			if provider := m.Providers[item.Provider]; provider != nil {
				provider(s)
			}
			continue
		}
		if item.Type == "mcp-http" {
			if err := m.registerHTTPTools(s, item); err != nil {
				log.Printf("MCP upstream %s unavailable: %v", item.ID, err)
			}
			continue
		}
		if item.Entry == "" {
			item.Entry = "plugin"
		}
		for _, tool := range item.Tools {
			registerExternalTool(s, item, tool, m.Config)
		}
	}
	return nil
}

type httpPluginConfig struct {
	Endpoint string `json:"endpoint"`
	Token    string `json:"token"`
}

func (m *Manager) registerHTTPTools(s *mcp.Server, item Installed) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	session, err := m.connectHTTP(ctx, item)
	if err != nil {
		return err
	}
	defer session.Close()
	for tool, err := range session.Tools(ctx, nil) {
		if err != nil {
			return err
		}
		upstreamName := tool.Name
		proxied := *tool
		proxied.Name = item.ToolPrefix + upstreamName
		s.AddTool(&proxied, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			session, err := m.connectHTTP(ctx, item)
			if err != nil {
				return nil, fmt.Errorf("MCP upstream %s: %w", item.ID, err)
			}
			defer session.Close()
			return session.CallTool(ctx, &mcp.CallToolParams{
				Name: upstreamName, Arguments: json.RawMessage(req.Params.Arguments),
				InputResponses: req.Params.InputResponses, RequestState: req.Params.RequestState,
			})
		})
	}
	return nil
}

func (m *Manager) connectHTTP(ctx context.Context, item Installed) (*mcp.ClientSession, error) {
	cfg := httpPluginConfig{Endpoint: item.Endpoint}
	if m.Config != nil {
		if raw := m.Config(item.ID); len(raw) > 0 {
			if err := json.Unmarshal(raw, &cfg); err != nil {
				return nil, fmt.Errorf("invalid plugin config: %w", err)
			}
			if cfg.Endpoint == "" {
				cfg.Endpoint = item.Endpoint
			}
		}
	}
	check := item.Manifest
	check.Endpoint = cfg.Endpoint
	if err := check.Validate(); err != nil {
		return nil, err
	}
	client := &http.Client{}
	if cfg.Token != "" {
		client.Transport = bearerTransport{token: cfg.Token}
	}
	transport := &mcp.StreamableClientTransport{
		Endpoint: cfg.Endpoint, HTTPClient: client, DisableStandaloneSSE: true, MaxRetries: -1,
	}
	return mcp.NewClient(&mcp.Implementation{Name: "daoliyu-mcp-proxy", Version: "1"}, nil).Connect(ctx, transport, nil)
}

type bearerTransport struct{ token string }

func (t bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(clone)
}

func registerExternalTool(s *mcp.Server, item Installed, tool ToolManifest, config func(string) json.RawMessage) {
	s.AddTool(&mcp.Tool{Name: tool.Name, Description: tool.Description, InputSchema: tool.InputSchema}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var input map[string]any
		if len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
				return nil, err
			}
		}
		payload := map[string]any{"method": "tools/call", "name": req.Params.Name, "arguments": input}
		if config != nil {
			if raw := config(item.ID); len(raw) > 0 {
				payload["config"] = raw
			}
		}
		b, _ := json.Marshal(payload)
		cmd := exec.CommandContext(ctx, filepath.Join(item.Dir, item.Entry))
		cmd.Env = append(os.Environ(),
			"DAOLIYU_MCP_DATA="+filepath.Dir(filepath.Dir(item.Dir)),
			"DAOLIYU_PLUGIN_ID="+item.ID,
			"DAOLIYU_PLUGIN_DIR="+item.Dir,
			"DAOLIYU_PLUGIN_DATA="+item.DataDir,
		)
		cmd.Stdin = strings.NewReader(string(b) + "\n")
		out, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("plugin %s: %w", item.ID, err)
		}
		var result mcp.CallToolResult
		if err := json.Unmarshal(out, &result); err != nil {
			return nil, fmt.Errorf("plugin %s returned invalid result: %w", item.ID, err)
		}
		return &result, nil
	})
}
