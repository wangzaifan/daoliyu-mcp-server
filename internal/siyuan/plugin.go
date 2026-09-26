package siyuan

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Config func() (baseURL, token string)

func Register(s *mcp.Server, config Config) {
	mcp.AddTool(s, &mcp.Tool{Name: "siyuan_health", Description: "检查思源内核是否可访问"}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		c := client(config)
		notebooks, err := c.ListNotebooks(ctx)
		if err != nil {
			return nil, nil, err
		}
		return textResult(map[string]any{"ok": true, "notebooks": len(notebooks)})
	})
	mcp.AddTool(s, &mcp.Tool{Name: "siyuan_list_notebooks", Description: "列出思源笔记本"}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		notebooks, err := client(config).ListNotebooks(ctx)
		if err != nil {
			return nil, nil, err
		}
		return textResult(notebooks)
	})
	type searchInput struct {
		Query  string `json:"query"`
		Limit  int    `json:"limit,omitempty"`
		Offset int    `json:"offset,omitempty"`
	}
	mcp.AddTool(s, &mcp.Tool{Name: "siyuan_search", Description: "在思源文档块内容中搜索文本；只返回有限字段和分页结果"}, func(ctx context.Context, _ *mcp.CallToolRequest, in searchInput) (*mcp.CallToolResult, any, error) {
		if strings.TrimSpace(in.Query) == "" {
			return nil, nil, fmt.Errorf("query is required")
		}
		if in.Limit <= 0 || in.Limit > 100 {
			in.Limit = 20
		}
		if in.Offset < 0 {
			in.Offset = 0
		}
		q := escapeLike(in.Query)
		stmt := "SELECT id, root_id, parent_id, hpath, content, type, subtype FROM blocks WHERE content LIKE '%" + q + "%' ESCAPE '\\\\' ORDER BY hpath,id LIMIT " + strconv.Itoa(in.Limit) + " OFFSET " + strconv.Itoa(in.Offset)
		var rows []map[string]any
		if err := client(config).Query(ctx, stmt, &rows); err != nil {
			return nil, nil, err
		}
		return textResult(rows)
	})
	type getInput struct {
		ID string `json:"id"`
	}
	mcp.AddTool(s, &mcp.Tool{Name: "siyuan_get_document", Description: "按文档 ID 导出 Markdown 内容"}, func(ctx context.Context, _ *mcp.CallToolRequest, in getInput) (*mcp.CallToolResult, any, error) {
		if in.ID == "" {
			return nil, nil, fmt.Errorf("id is required")
		}
		out, err := client(config).ExportDocument(ctx, in.ID)
		if err != nil {
			return nil, nil, err
		}
		return textResult(out)
	})
	type createInput struct {
		Notebook string `json:"notebook"`
		Path     string `json:"path"`
		Markdown string `json:"markdown"`
	}
	mcp.AddTool(s, &mcp.Tool{Name: "siyuan_create_document", Description: "在指定笔记本创建 Markdown 文档"}, func(ctx context.Context, _ *mcp.CallToolRequest, in createInput) (*mcp.CallToolResult, any, error) {
		if in.Notebook == "" || in.Path == "" || !strings.HasPrefix(in.Path, "/") {
			return nil, nil, fmt.Errorf("notebook and an absolute path are required")
		}
		if len(in.Markdown) > 4<<20 {
			return nil, nil, fmt.Errorf("markdown is too large")
		}
		id, err := client(config).CreateDocument(ctx, in.Notebook, in.Path, in.Markdown)
		if err != nil {
			return nil, nil, err
		}
		return textResult(map[string]string{"id": id})
	})
	mcp.AddTool(s, &mcp.Tool{Name: "siyuan_kb_schema", Description: "返回 AI 知识库支持的分类、类型、状态和推荐目录"}, func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, any, error) {
		return textResult(map[string]any{
			"root":       "/AI知识库",
			"categories": knowledgeCategories,
			"types":      mapKeys(knowledgeTypes),
			"statuses":   mapKeys(knowledgeStatuses),
			"relations":  "related 使用思源文档或块 ID；检索结果返回 id/rootID/notebook/path 作为稳定引用",
		})
	})
	mcp.AddTool(s, &mcp.Tool{Name: "siyuan_kb_capture", Description: "按统一分类和元数据模板把知识写入思源；默认落入 /AI知识库/00_收件箱"}, func(ctx context.Context, _ *mcp.CallToolRequest, in KnowledgeInput) (*mcp.CallToolResult, any, error) {
		if strings.TrimSpace(in.Notebook) == "" {
			return nil, nil, fmt.Errorf("notebook is required")
		}
		path, err := knowledgePath(in)
		if err != nil {
			return nil, nil, err
		}
		markdown, err := buildKnowledgeMarkdown(in)
		if err != nil {
			return nil, nil, err
		}
		c := client(config)
		id, err := c.CreateDocument(ctx, in.Notebook, path, markdown)
		if err != nil {
			return nil, nil, err
		}
		attrs := knowledgeAttrs(in)
		if err := c.SetBlockAttrs(ctx, id, attrs); err != nil {
			return textResult(map[string]any{"id": id, "path": path, "metadataSaved": false, "warning": err.Error()})
		}
		return textResult(map[string]any{"id": id, "path": path, "metadataSaved": true, "attributes": attrs})
	})
	type kbSearchInput struct {
		Query    string   `json:"query"`
		Notebook string   `json:"notebook,omitempty"`
		Category string   `json:"category,omitempty"`
		Type     string   `json:"type,omitempty"`
		Status   string   `json:"status,omitempty"`
		Tags     []string `json:"tags,omitempty"`
		Limit    int      `json:"limit,omitempty"`
		Offset   int      `json:"offset,omitempty"`
	}
	mcp.AddTool(s, &mcp.Tool{Name: "siyuan_kb_search", Description: "检索思源知识库并返回可追溯的文档 ID、路径、片段和 AI 元数据"}, func(ctx context.Context, _ *mcp.CallToolRequest, in kbSearchInput) (*mcp.CallToolResult, any, error) {
		if strings.TrimSpace(in.Query) == "" {
			return nil, nil, fmt.Errorf("query is required")
		}
		if in.Limit <= 0 || in.Limit > 50 {
			in.Limit = 20
		}
		if in.Offset < 0 {
			in.Offset = 0
		}
		if in.Category != "" {
			if _, err := normalizeKnowledge(in.Category, knowledgeCategories, "inbox"); err != nil {
				return nil, nil, err
			}
		}
		if in.Type != "" {
			if _, err := normalizeKnowledge(in.Type, knowledgeTypes, "note"); err != nil {
				return nil, nil, err
			}
		}
		if in.Status != "" {
			if _, err := normalizeKnowledge(in.Status, knowledgeStatuses, "inbox"); err != nil {
				return nil, nil, err
			}
		}
		where := "content LIKE '%" + escapeLike(in.Query) + "%' ESCAPE '\\\\'"
		if in.Notebook != "" {
			where += " AND box='" + strings.ReplaceAll(in.Notebook, "'", "''") + "'"
		}
		stmt := "SELECT id, root_id, box, hpath, content, type, subtype, updated FROM blocks WHERE " + where + " ORDER BY updated DESC,hpath,id LIMIT " + strconv.Itoa(in.Limit) + " OFFSET " + strconv.Itoa(in.Offset)
		var rows []map[string]any
		c := client(config)
		if err := c.Query(ctx, stmt, &rows); err != nil {
			return nil, nil, err
		}
		wantedTags := cleanKnowledgeList(in.Tags)
		result := make([]map[string]any, 0, len(rows))
		attrsByRoot := map[string]map[string]string{}
		for _, row := range rows {
			root, _ := row["root_id"].(string)
			attrs := attrsByRoot[root]
			if attrs == nil && root != "" {
				attrs, _ = c.GetBlockAttrs(ctx, root)
				attrsByRoot[root] = attrs
			}
			if !matchesKnowledge(attrs, in.Category, in.Type, in.Status, wantedTags) {
				continue
			}
			row["attributes"] = attrs
			result = append(result, row)
		}
		return textResult(map[string]any{"items": result, "count": len(result), "limit": in.Limit, "offset": in.Offset})
	})
}

func client(config Config) *Client { base, token := config(); return NewClient(base, token) }
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return strings.ReplaceAll(s, `'`, `''`)
}
func textResult(v any) (*mcp.CallToolResult, any, error) {
	b, _ := json.Marshal(v)
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, nil, nil
}

func knowledgeAttrs(in KnowledgeInput) map[string]string {
	category, _ := normalizeKnowledge(in.Category, knowledgeCategories, "inbox")
	typ, _ := normalizeKnowledge(in.Type, knowledgeTypes, "note")
	status, _ := normalizeKnowledge(in.Status, knowledgeStatuses, "inbox")
	b, _ := json.Marshal(cleanKnowledgeList(in.Related))
	return map[string]string{
		"custom-ai-category": category,
		"custom-ai-type":     typ,
		"custom-ai-status":   status,
		"custom-ai-tags":     strings.Join(cleanKnowledgeList(in.Tags), ","),
		"custom-ai-source":   strings.TrimSpace(in.Source),
		"custom-ai-related":  string(b),
	}
}

func matchesKnowledge(attrs map[string]string, category, typ, status string, tags []string) bool {
	if category != "" && attrs["custom-ai-category"] != strings.ToLower(strings.TrimSpace(category)) ||
		typ != "" && attrs["custom-ai-type"] != strings.ToLower(strings.TrimSpace(typ)) ||
		status != "" && attrs["custom-ai-status"] != strings.ToLower(strings.TrimSpace(status)) {
		return false
	}
	have := map[string]bool{}
	for _, tag := range strings.Split(attrs["custom-ai-tags"], ",") {
		have[strings.TrimSpace(tag)] = true
	}
	for _, tag := range tags {
		if !have[tag] {
			return false
		}
	}
	return true
}

func mapKeys[T any](items map[string]T) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
