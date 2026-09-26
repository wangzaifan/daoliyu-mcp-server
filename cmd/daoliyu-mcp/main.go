package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/daoliyu/daoliyu-mcp/internal/app"
	"github.com/daoliyu/daoliyu-mcp/internal/plugin"
	"github.com/daoliyu/daoliyu-mcp/internal/web"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var version = "0.3.2"

func main() {
	port := flag.Int("port", envInt("DAOLIYU_MCP_PORT", 37421), "HTTP port")
	data := flag.String("data", defaultDataDir(), "data directory")
	stdio := flag.Bool("stdio", false, "run MCP over stdio")
	flag.Parse()
	store, err := app.NewStore(*data, *port)
	if err != nil {
		log.Fatal(err)
	}
	manager, err := plugin.NewManager(filepath.Join(*data, "plugins"), func(id string) bool { return store.Config.Plugins[id] }, func(id string, enabled bool) error { store.Config.Plugins[id] = enabled; return store.Save() })
	if err != nil {
		log.Fatal(err)
	}
	manager.Config = store.PluginConfigRaw
	if err := retireLegacySiyuan(manager); err != nil {
		log.Fatal(err)
	}
	tokens, err := app.NewTokenStore(*data, store.Config.AccessToken)
	if err != nil {
		log.Fatal(err)
	}
	if *stdio {
		server := newMCPServer(manager, nil)
		if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			log.Fatal(err)
		}
		return
	}
	ui := (&web.Server{Store: store, Plugins: manager, Tokens: tokens}).Handler()
	mcpHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		token, _ := tokens.Find(bearerValue(r))
		return newMCPServer(manager, scopeSet(token.Scopes))
	}, &mcp.StreamableHTTPOptions{JSONResponse: true, Stateless: true})
	mux := http.NewServeMux()
	mux.Handle("/mcp", requireMCPToken(tokens, mcpHandler))
	mux.Handle("/", ui)
	bind := os.Getenv("DAOLIYU_MCP_BIND")
	if bind == "" {
		bind = "0.0.0.0"
	}
	httpServer := &http.Server{Addr: bind + ":" + strconv.Itoa(store.Config.ListenPort), Handler: mux}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() { <-ctx.Done(); _ = httpServer.Shutdown(context.Background()) }()
	log.Printf("道理鱼 MCP 服务 %s listening on %s", version, httpServer.Addr)
	log.Printf("Application data directory: %s", *data)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func retireLegacySiyuan(manager *plugin.Manager) error {
	if _, err := manager.Get("siyuan"); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	// Keep the private data and config so an older connector can be restored.
	return manager.UninstallWithOptions("siyuan", false)
}

func requireMCPToken(tokens *app.TokenStore, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token, ok := tokens.Find(bearerValue(r)); !ok || token.Admin {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func bearerValue(r *http.Request) string {
	const prefix = "Bearer "
	value := r.Header.Get("Authorization")
	if len(value) < len(prefix) || value[:len(prefix)] != prefix {
		return ""
	}
	return value[len(prefix):]
}
func scopeSet(scopes []string) map[string]bool {
	out := map[string]bool{}
	for _, scope := range scopes {
		out[scope] = true
	}
	return out
}
func newMCPServer(manager *plugin.Manager, scopes map[string]bool) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "daoliyu-mcp", Version: version}, nil)
	if err := manager.RegisterExternalScoped(server, scopes); err != nil {
		panic(err)
	}
	return server
}
func defaultDataDir() string {
	if v := os.Getenv("DAOLIYU_MCP_DATA"); v != "" {
		return v
	}
	if v := os.Getenv("TRIM_PKGVAR"); v != "" {
		return filepath.Join(v, "data")
	}
	return filepath.Join(".", "data")
}
func envInt(name string, fallback int) int {
	if v, err := strconv.Atoi(os.Getenv(name)); err == nil && v > 0 {
		return v
	}
	return fallback
}
