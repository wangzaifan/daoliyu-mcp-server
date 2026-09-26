package plugin

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
)

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,63}$`)
var prefixPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)

type Manifest struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Version      string          `json:"version"`
	Description  string          `json:"description"`
	Icon         string          `json:"icon"`
	Type         string          `json:"type,omitempty"`
	Entry        string          `json:"entry"`
	Provider     string          `json:"provider"`
	Endpoint     string          `json:"endpoint,omitempty"`
	ToolPrefix   string          `json:"toolPrefix,omitempty"`
	ConfigSchema json.RawMessage `json:"configSchema,omitempty"`
	Tools        []ToolManifest  `json:"tools"`
}

type ToolManifest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

func (m Manifest) Validate() error {
	if !idPattern.MatchString(m.ID) {
		return fmt.Errorf("invalid plugin id")
	}
	if m.Name == "" || m.Version == "" {
		return fmt.Errorf("plugin name and version are required")
	}
	if m.Type != "" && m.Type != "mcp-http" {
		return fmt.Errorf("unsupported plugin type %q", m.Type)
	}
	if m.Type == "mcp-http" {
		u, err := url.Parse(m.Endpoint)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
			return fmt.Errorf("invalid MCP HTTP endpoint")
		}
		if !prefixPattern.MatchString(m.ToolPrefix) {
			return fmt.Errorf("invalid MCP HTTP tool prefix")
		}
	}
	if m.Entry == "" {
		m.Entry = "plugin"
	}
	for _, tool := range m.Tools {
		if !idPattern.MatchString(tool.Name) {
			return fmt.Errorf("invalid tool name %q", tool.Name)
		}
	}
	return nil
}
