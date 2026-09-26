package siyuan

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestKnowledgeCaptureRunsThroughMCPAndSiYuanAPI(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/filetree/createDocWithMd":
			_, _ = w.Write([]byte(`{"code":0,"msg":"","data":"doc-id"}`))
		case "/api/attr/setBlockAttrs":
			_, _ = w.Write([]byte(`{"code":0,"msg":"","data":null}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	Register(mcpServer, func() (string, string) { return server.URL, "token" })
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := mcpServer.Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	clientSession, err := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1"}, nil).Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{Name: "siyuan_kb_capture", Arguments: map[string]any{
		"notebook": "box-id", "title": "API/约定", "content": "正文", "category": "decision", "type": "decision", "status": "verified", "related": []string{"other-id"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, `"id":"doc-id"`) || strings.Join(paths, ",") != "/api/filetree/createDocWithMd,/api/attr/setBlockAttrs" {
		t.Fatalf("result=%s paths=%v", text, paths)
	}
	_ = clientSession.Close()
	_ = serverSession.Wait()
}

func TestKnowledgeMetadataAndPath(t *testing.T) {
	in := KnowledgeInput{Title: "A/B", Category: "concept", Type: "concept", Status: "active", Tags: []string{"MCP", "MCP"}, Related: []string{"id-1"}}
	path, err := knowledgePath(in)
	if err != nil || path != "/AI知识库/20_概念/A／B" {
		t.Fatalf("path=%q err=%v", path, err)
	}
	markdown, err := buildKnowledgeMarkdown(in)
	if err != nil || markdown != "" {
		t.Fatalf("markdown=%q err=%v", markdown, err)
	}
	attrs := knowledgeAttrs(in)
	related := []string{}
	if err := json.Unmarshal([]byte(attrs["custom-ai-related"]), &related); err != nil || len(related) != 1 {
		t.Fatalf("attrs=%v err=%v", attrs, err)
	}
}
