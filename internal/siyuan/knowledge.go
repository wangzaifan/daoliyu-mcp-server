package siyuan

import (
	"fmt"
	"strings"
)

var knowledgeCategories = map[string]string{
	"inbox":     "00_收件箱",
	"fact":      "10_事实",
	"concept":   "20_概念",
	"decision":  "30_决策",
	"project":   "40_项目",
	"procedure": "50_操作手册",
	"reference": "60_参考资料",
	"archive":   "90_归档",
}

var knowledgeTypes = map[string]bool{
	"note": true, "fact": true, "concept": true, "decision": true,
	"project": true, "procedure": true, "source": true, "question": true,
}

var knowledgeStatuses = map[string]bool{
	"inbox": true, "active": true, "verified": true, "archived": true,
}

type KnowledgeInput struct {
	Notebook string   `json:"notebook"`
	Path     string   `json:"path,omitempty"`
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Category string   `json:"category,omitempty"`
	Type     string   `json:"type,omitempty"`
	Status   string   `json:"status,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Source   string   `json:"source,omitempty"`
	Related  []string `json:"related,omitempty"`
}

func buildKnowledgeMarkdown(in KnowledgeInput) (string, error) {
	if strings.TrimSpace(in.Title) == "" {
		return "", fmt.Errorf("title is required")
	}
	if len(in.Content) > 4<<20 {
		return "", fmt.Errorf("content is too large")
	}
	if _, err := normalizeKnowledge(in.Category, knowledgeCategories, "inbox"); err != nil {
		return "", err
	}
	if _, err := normalizeKnowledge(in.Type, knowledgeTypes, "note"); err != nil {
		return "", err
	}
	if _, err := normalizeKnowledge(in.Status, knowledgeStatuses, "inbox"); err != nil {
		return "", err
	}
	return in.Content, nil
}

func knowledgePath(in KnowledgeInput) (string, error) {
	category, err := normalizeKnowledge(in.Category, knowledgeCategories, "inbox")
	if err != nil {
		return "", err
	}
	title := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(in.Title, "/", "／"), "\\", "＼"))
	if title == "" {
		return "", fmt.Errorf("title is required")
	}
	if in.Path != "" {
		if !strings.HasPrefix(in.Path, "/") || strings.Contains(in.Path, "..") {
			return "", fmt.Errorf("path must be an absolute SiYuan document path")
		}
		return in.Path, nil
	}
	return "/AI知识库/" + knowledgeCategories[category] + "/" + title, nil
}

func normalizeKnowledge[T any](value string, allowed map[string]T, fallback string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return fallback, nil
	}
	if _, ok := allowed[value]; !ok {
		return "", fmt.Errorf("unsupported knowledge value %q", value)
	}
	return value, nil
}

func cleanKnowledgeList(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}
