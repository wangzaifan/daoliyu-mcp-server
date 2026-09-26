package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandRejectsMissingRequiredValues(t *testing.T) {
	if _, err := command("fnos_file_search", input{}); err == nil {
		t.Fatal("expected missing search key to fail")
	}
	if _, err := command("fnos_storage_smart", input{}); err == nil {
		t.Fatal("expected missing disk to fail")
	}
	if _, err := command("fnos_file_info", input{Path: "/vol1/a", Mode: "rm"}); err == nil {
		t.Fatal("expected unsupported file info mode to fail")
	}
}

func TestRunCLIUsesPluginDataAndConfig(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(binDir, "trim-cli")
	script := "#!/bin/sh\nprintf '%s' \"$TRIM_CLI_CONFIG_DIR|$TRIM_CLI_SESSION_STORAGE|$*\"\n"
	if err := os.WriteFile(bin, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DAOLIYU_PLUGIN_DIR", dir)
	t.Setenv("DAOLIYU_PLUGIN_DATA", filepath.Join(dir, "data"))
	out, err := runCLI(context.Background(), config{Host: "10.0.0.2", Port: 5666, Profile: "nas", Scheme: "wss", TLSInsecure: true}, []string{"system", "info"}, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"/data/trim-cli|file", "--host 10.0.0.2", "--profile nas", "--scheme wss", "--tls-insecure", "system info"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output %q missing %q", out, want)
		}
	}
}
