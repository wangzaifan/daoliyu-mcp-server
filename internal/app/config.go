package app

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	ListenPort  int    `json:"listenPort"`
	AccessToken string `json:"accessToken"`
	// Siyuan is retained for one-version migration from the original prototype.
	Siyuan        *SiyuanConfig              `json:"siyuan,omitempty"`
	Plugins       map[string]bool            `json:"plugins"`
	PluginConfigs map[string]json.RawMessage `json:"pluginConfigs,omitempty"`
}

type SiyuanConfig struct {
	BaseURL string `json:"baseURL"`
	Token   string `json:"token"`
}

type Store struct {
	Dir    string
	Path   string
	Config Config
}

func NewStore(dir string, port int) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	s := &Store{Dir: dir, Path: filepath.Join(dir, "config.json")}
	b, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		s.Config = Config{ListenPort: port, Plugins: map[string]bool{}, PluginConfigs: map[string]json.RawMessage{}}
		s.Config.AccessToken, err = newToken()
		if err != nil {
			return nil, err
		}
		if err := s.Save(); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(dir, "access-token"), []byte(s.Config.AccessToken+"\n"), 0o600); err != nil {
			return nil, err
		}
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &s.Config); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	if s.Config.Plugins == nil {
		s.Config.Plugins = map[string]bool{}
	}
	if s.Config.PluginConfigs == nil {
		s.Config.PluginConfigs = map[string]json.RawMessage{}
	}
	if _, ok := s.Config.PluginConfigs["siyuan"]; !ok && s.Config.Siyuan != nil {
		if raw, marshalErr := json.Marshal(s.Config.Siyuan); marshalErr == nil {
			s.Config.PluginConfigs["siyuan"] = raw
		}
	}
	if s.Config.ListenPort == 0 {
		s.Config.ListenPort = port
	}
	if s.Config.AccessToken == "" {
		s.Config.AccessToken, err = newToken()
		if err != nil {
			return nil, err
		}
		if err := s.Save(); err != nil {
			return nil, err
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "access-token"), []byte(s.Config.AccessToken+"\n"), 0o600); err != nil {
		return nil, err
	}
	return s, nil
}

func newToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Store) Save() error {
	b, err := json.MarshalIndent(s.Config, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path)
}

func (s *Store) PluginConfig(id string, out any) error {
	raw := s.Config.PluginConfigs[id]
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, out)
}

func (s *Store) PluginConfigRaw(id string) json.RawMessage {
	return append(json.RawMessage(nil), s.Config.PluginConfigs[id]...)
}

func (s *Store) UpdatePluginConfig(id string, in any) error {
	if id == "" {
		return fmt.Errorf("plugin id is required")
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return err
	}
	if s.Config.PluginConfigs == nil {
		s.Config.PluginConfigs = map[string]json.RawMessage{}
	}
	s.Config.PluginConfigs[id] = raw
	if id == "siyuan" {
		var legacy SiyuanConfig
		if json.Unmarshal(raw, &legacy) == nil {
			s.Config.Siyuan = &legacy
		}
	}
	return s.Save()
}

func (s *Store) DeletePluginConfig(id string) error {
	if id == "" {
		return fmt.Errorf("plugin id is required")
	}
	delete(s.Config.PluginConfigs, id)
	if id == "siyuan" {
		s.Config.Siyuan = nil
	}
	return s.Save()
}
