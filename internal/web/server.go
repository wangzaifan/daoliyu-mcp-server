package web

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	"github.com/daoliyu/daoliyu-mcp/internal/app"
	"github.com/daoliyu/daoliyu-mcp/internal/plugin"
)

//go:embed static/*
var assets embed.FS

type Server struct {
	Store   *app.Store
	Plugins *plugin.Manager
	Tokens  *app.TokenStore
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.index)
	staticFS, _ := fs.Sub(assets, "static")
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(staticFS))))
	mux.HandleFunc("/api/health", s.health)
	mux.HandleFunc("/api/auth-tokens", s.authTokens)
	mux.HandleFunc("/api/auth-tokens/", s.authTokenAction)
	mux.HandleFunc("/api/plugins", s.plugins)
	mux.HandleFunc("/api/plugins/install", s.install)
	mux.HandleFunc("/api/plugins/", s.pluginAction)
	return mux
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	name := "static/index.html"
	if r.URL.Path == "/tokens" {
		name = "static/tokens.html"
	} else if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	b, _ := assets.ReadFile(name)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(b)
}
func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{"ok": true, "version": "0.3.2", "mcpPath": "/mcp"})
}

func (s *Server) authTokens(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, s.Tokens.List())
	case http.MethodPost:
		var in struct {
			Name   string   `json:"name"`
			Scopes []string `json:"scopes"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&in); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		valid := map[string]bool{"*": true}
		if list, err := s.Plugins.List(); err == nil {
			for _, item := range list {
				if item.Enabled {
					valid[item.ID] = true
				}
			}
		}
		for _, scope := range in.Scopes {
			if !valid[scope] {
				http.Error(w, "unknown plugin scope", 400)
				return
			}
		}
		token, err := s.Tokens.Create(in.Name, in.Scopes)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, token)
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func (s *Server) authTokenAction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 3 || r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", 405)
		return
	}
	if err := s.Tokens.Delete(parts[2]); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

type card struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Enabled     bool   `json:"enabled"`
	Builtin     bool   `json:"builtin"`
}

func (s *Server) plugins(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	list, err := s.Plugins.List()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	out := make([]card, 0, len(list))
	for _, p := range list {
		out = append(out, card{p.ID, p.Name, p.Version, p.Description, p.Icon, p.Enabled, p.Builtin})
	}
	writeJSON(w, out)
}

func (s *Server) pluginAction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.Error(w, "bad plugin path", 400)
		return
	}
	id := parts[2]
	if len(parts) == 4 && parts[3] == "config" {
		s.pluginConfig(w, r, id)
		return
	}
	if len(parts) == 4 && parts[3] == "toggle" {
		s.toggle(w, r, id)
		return
	}
	if len(parts) == 4 && parts[3] == "status" {
		s.pluginStatus(w, r, id)
		return
	}
	if len(parts) == 3 && r.Method == http.MethodDelete {
		s.uninstall(w, r, id)
		return
	}
	http.Error(w, "method not allowed", 405)
}

func (s *Server) pluginStatus(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	status, err := s.Plugins.Status(r.Context(), id)
	if err != nil {
		writeJSON(w, map[string]any{"state": "error", "error": err.Error()})
		return
	}
	writeJSON(w, status)
}

func (s *Server) pluginConfig(w http.ResponseWriter, r *http.Request, id string) {
	item, err := s.Plugins.Get(id)
	if err != nil {
		http.Error(w, "plugin not found", 404)
		return
	}
	switch r.Method {
	case http.MethodGet:
		var current map[string]any
		_ = s.Store.PluginConfig(id, &current)
		if current == nil {
			current = map[string]any{}
		}
		writeJSON(w, map[string]any{"id": item.ID, "name": item.Name, "schema": json.RawMessage(item.ConfigSchema), "config": redact(current, item.ConfigSchema)})
	case http.MethodPut:
		var incoming map[string]any
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&incoming); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		var current map[string]any
		_ = s.Store.PluginConfig(id, &current)
		if current == nil {
			current = map[string]any{}
		}
		mergeSecrets(current, incoming, item.ConfigSchema)
		if err := s.Store.UpdatePluginConfig(id, incoming); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, map[string]bool{"ok": true})
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func (s *Server) toggle(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var in struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if _, err := s.Plugins.Get(id); err != nil {
		http.Error(w, "plugin not found", 404)
		return
	}
	if err := s.Plugins.Toggle(id, in.Enabled); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) uninstall(w http.ResponseWriter, r *http.Request, id string) {
	purge := r.URL.Query().Get("purge") == "1" || strings.EqualFold(r.URL.Query().Get("purge"), "true")
	if err := s.Plugins.UninstallWithOptions(id, purge); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if purge && s.Store != nil {
		if err := s.Store.DeletePluginConfig(id); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}
	writeJSON(w, map[string]any{"ok": true})
}
func (s *Server) install(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<20)
	p, err := s.Plugins.Install(r)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	writeJSON(w, p)
}

func redact(current map[string]any, raw json.RawMessage) map[string]any {
	result := map[string]any{}
	for k, v := range current {
		result[k] = v
	}
	var schema struct {
		Properties map[string]struct {
			Format string `json:"format"`
		} `json:"properties"`
	}
	_ = json.Unmarshal(raw, &schema)
	for key, property := range schema.Properties {
		if property.Format == "password" {
			if _, ok := result[key]; ok {
				result[key] = ""
			}
		}
	}
	return result
}
func mergeSecrets(current, incoming map[string]any, raw json.RawMessage) {
	var schema struct {
		Properties map[string]struct {
			Format string `json:"format"`
		} `json:"properties"`
	}
	_ = json.Unmarshal(raw, &schema)
	for key, property := range schema.Properties {
		if property.Format == "password" && strings.TrimSpace(toString(incoming[key])) == "" {
			if value, ok := current[key]; ok {
				incoming[key] = value
			}
		}
	}
}
func toString(v any) string { value, _ := v.(string); return value }
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}
