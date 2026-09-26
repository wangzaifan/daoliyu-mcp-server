package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	sdk "github.com/daoliyu/daoliyu-mcp/sdk/plugin"
)

type config struct {
	Host          string `json:"host"`
	Port          int    `json:"port"`
	Profile       string `json:"profile"`
	Scheme        string `json:"scheme"`
	AllowInsecure bool   `json:"allowInsecureWs"`
	TLSInsecure   bool   `json:"tlsInsecure"`
}

type input struct {
	Path     string   `json:"path"`
	Mode     string   `json:"mode"`
	Key      string   `json:"key"`
	Paths    []string `json:"paths"`
	App      string   `json:"app"`
	Keyword  string   `json:"keyword"`
	ID       int      `json:"id"`
	Disk     string   `json:"disk"`
	Query    string   `json:"query"`
	Limit    int      `json:"limit"`
	PhotoID  int      `json:"photoId"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	TotpCode string   `json:"totpCode"`
}

func main() {
	server := sdk.NewServer()
	for name := range toolNames {
		name := name
		server.Handle(name, func(ctx context.Context, args, rawConfig json.RawMessage) (any, error) {
			return call(ctx, name, args, rawConfig)
		})
	}
	if err := server.Run(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var toolNames = map[string]bool{
	"fnos_login": true, "fnos_logout": true, "fnos_status": true,
	"fnos_system_info": true, "fnos_monitor_cpu": true, "fnos_monitor_memory": true,
	"fnos_file_list": true, "fnos_file_search": true, "fnos_file_info": true,
	"fnos_app_list": true, "fnos_app_status": true, "fnos_app_check_update": true,
	"fnos_docker_containers": true, "fnos_docker_stats": true,
	"fnos_download_list": true, "fnos_download_info": true, "fnos_download_stat": true,
	"fnos_storage_overview": true, "fnos_storage_pools": true, "fnos_storage_disks": true,
	"fnos_storage_health": true, "fnos_storage_smart": true,
	"fnos_photos_search": true, "fnos_photos_info": true,
	"fnos_media_stats": true, "fnos_media_search": true,
}

func call(ctx context.Context, name string, rawArgs, rawConfig json.RawMessage) (string, error) {
	var in input
	if len(rawArgs) > 0 && string(rawArgs) != "null" {
		if err := json.Unmarshal(rawArgs, &in); err != nil {
			return "", err
		}
	}
	var cfg config
	if len(rawConfig) > 0 && string(rawConfig) != "null" {
		if err := json.Unmarshal(rawConfig, &cfg); err != nil {
			return "", fmt.Errorf("invalid fnOS configuration: %w", err)
		}
	}
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.Port == 0 {
		cfg.Port = 5666
	}
	if cfg.Profile == "" {
		cfg.Profile = "default"
	}
	if name == "fnos_login" {
		if in.Username == "" || in.Password == "" {
			return "", fmt.Errorf("username and password are required")
		}
		args := []string{"login", "-u", in.Username}
		if in.TotpCode != "" {
			args = append(args, "--totp-code", in.TotpCode)
		}
		return runCLI(ctx, cfg, args, in.Password+"\n")
	}
	if name == "fnos_logout" {
		return runCLI(ctx, cfg, []string{"logout"}, "")
	}
	args, err := command(name, in)
	if err != nil {
		return "", err
	}
	return runCLI(ctx, cfg, args, "")
}

func command(name string, in input) ([]string, error) {
	switch name {
	case "fnos_status":
		return []string{"+status"}, nil
	case "fnos_system_info":
		return []string{"system", "info"}, nil
	case "fnos_monitor_cpu":
		return []string{"monitor", "cpu"}, nil
	case "fnos_monitor_memory":
		return []string{"monitor", "memory"}, nil
	case "fnos_file_list":
		return append([]string{"file", "ls"}, optional(in.Path)...), nil
	case "fnos_file_search":
		if in.Key == "" {
			return nil, fmt.Errorf("key is required")
		}
		return append([]string{"file", "search", in.Key}, in.Paths...), nil
	case "fnos_file_info":
		if in.Path == "" {
			return nil, fmt.Errorf("path is required")
		}
		mode := in.Mode
		if mode == "" {
			mode = "prop"
		}
		if mode != "prop" && mode != "size" && mode != "download-url" {
			return nil, fmt.Errorf("mode must be prop, size or download-url")
		}
		return []string{"file", mode, in.Path}, nil
	case "fnos_app_list":
		return []string{"app", "list"}, nil
	case "fnos_app_status":
		return required("app", in.App, "app", "status")
	case "fnos_app_check_update":
		return []string{"app", "check-update"}, nil
	case "fnos_docker_containers":
		return []string{"docker", "container", "ls"}, nil
	case "fnos_docker_stats":
		return []string{"docker", "stats"}, nil
	case "fnos_download_list":
		return append([]string{"download", "ls"}, optional(in.Keyword)...), nil
	case "fnos_download_info":
		return requiredInt(in.ID, "id", "download", "info")
	case "fnos_download_stat":
		return []string{"download", "stat"}, nil
	case "fnos_storage_overview":
		return []string{"storage", "overview"}, nil
	case "fnos_storage_pools":
		return []string{"storage", "pools"}, nil
	case "fnos_storage_disks":
		return []string{"storage", "disks"}, nil
	case "fnos_storage_health":
		return required("disk", in.Disk, "storage", "health")
	case "fnos_storage_smart":
		return required("disk", in.Disk, "storage", "smart")
	case "fnos_photos_search":
		if in.Query == "" {
			return nil, fmt.Errorf("query is required")
		}
		args := []string{"photos", "search", in.Query}
		if in.Limit > 0 {
			args = append(args, "--limit", strconv.Itoa(in.Limit))
		}
		return args, nil
	case "fnos_photos_info":
		return requiredInt(in.PhotoID, "photoId", "photos", "info")
	case "fnos_media_stats":
		return []string{"media", "stats"}, nil
	case "fnos_media_search":
		if in.Query == "" {
			return nil, fmt.Errorf("query is required")
		}
		return []string{"media", "search", in.Query}, nil
	default:
		return nil, fmt.Errorf("unknown fnOS tool %q", name)
	}
}

func optional(value string) []string {
	if value == "" {
		return nil
	}
	return []string{value}
}

func required(label, value string, prefix ...string) ([]string, error) {
	if value == "" {
		return nil, fmt.Errorf("%s is required", label)
	}
	return append(prefix, value), nil
}

func requiredInt(value int, label string, prefix ...string) ([]string, error) {
	if value <= 0 {
		return nil, fmt.Errorf("%s must be positive", label)
	}
	return append(prefix, strconv.Itoa(value)), nil
}

func runCLI(ctx context.Context, cfg config, args []string, stdin string) (string, error) {
	bin, err := binaryPath()
	if err != nil {
		return "", err
	}
	global := []string{"--host", cfg.Host, "--port", strconv.Itoa(cfg.Port), "--profile", cfg.Profile}
	if cfg.Scheme != "" && cfg.Scheme != "auto" {
		global = append(global, "--scheme", cfg.Scheme)
	}
	if cfg.AllowInsecure {
		global = append(global, "--allow-insecure-ws")
	}
	if cfg.TLSInsecure {
		global = append(global, "--tls-insecure")
	}
	cmd := exec.CommandContext(ctx, bin, append(global, args...)...)
	dataDir := os.Getenv("DAOLIYU_PLUGIN_DATA")
	if dataDir == "" {
		return "", fmt.Errorf("DAOLIYU_PLUGIN_DATA is not set")
	}
	cmd.Env = append(os.Environ(), "TRIM_CLI_CONFIG_DIR="+filepath.Join(dataDir, "trim-cli"), "TRIM_CLI_SESSION_STORAGE=file")
	cmd.Stdin = strings.NewReader(stdin)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("trim-cli: %s", message)
	}
	return string(out), nil
}

func binaryPath() (string, error) {
	standard := filepath.Join(os.Getenv("DAOLIYU_PLUGIN_DIR"), "bin", "trim-cli")
	if _, err := os.Stat(standard); err == nil {
		return standard, nil
	}
	arch := map[string]string{"amd64": "x64", "arm64": "arm64"}[runtime.GOARCH]
	if arch == "" {
		return "", fmt.Errorf("unsupported Linux architecture %s", runtime.GOARCH)
	}
	path := filepath.Join(os.Getenv("DAOLIYU_PLUGIN_DIR"), "bin", "trim-cli-"+arch)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("bundled fnOS V2 CLI is missing: %w", err)
	}
	return path, nil
}
