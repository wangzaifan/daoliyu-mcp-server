package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreCreatesTokenAndPersists(t *testing.T) {
	d := t.TempDir()
	s, err := NewStore(d, 37421)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Config.AccessToken) < 32 {
		t.Fatal("token not generated")
	}
	s2, err := NewStore(d, 37421)
	if err != nil {
		t.Fatal(err)
	}
	if s2.Config.AccessToken != s.Config.AccessToken {
		t.Fatal("token changed")
	}
}

func TestStoreMigratesLegacySiyuanConfig(t *testing.T) {
	d := t.TempDir()
	if err := os.WriteFile(filepath.Join(d, "config.json"), []byte(`{"listenPort":37421,"accessToken":"token","plugins":{"siyuan":true},"siyuan":{"baseURL":"http://old:6806","token":"secret"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := NewStore(d, 37421)
	if err != nil {
		t.Fatal(err)
	}
	var cfg SiyuanConfig
	if err := s.PluginConfig("siyuan", &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL != "http://old:6806" || cfg.Token != "secret" {
		t.Fatalf("%+v", cfg)
	}
}
