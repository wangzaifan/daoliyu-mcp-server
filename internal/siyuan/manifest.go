package siyuan

import (
	"encoding/json"

	"github.com/daoliyu/daoliyu-mcp/internal/plugin"
)

func Manifest() plugin.Manifest {
	return plugin.Manifest{
		ID: "siyuan", Name: "思源知识库", Version: "0.2.0",
		Description: "通过思源官方 API 提供检索、沉淀、分类和关联能力，可独立停用和卸载。",
		Icon:        "icon.png", Provider: "siyuan",
		ConfigSchema: json.RawMessage(`{"type":"object","properties":{"baseURL":{"type":"string","title":"思源地址","default":"http://127.0.0.1:6806"},"token":{"type":"string","title":"API Token","format":"password"}},"required":["baseURL"]}`),
		Tools: []plugin.ToolManifest{
			{Name: "siyuan_health", Description: "检查思源内核是否可访问"},
			{Name: "siyuan_list_notebooks", Description: "列出思源笔记本"},
			{Name: "siyuan_search", Description: "在思源文档块内容中搜索文本"},
			{Name: "siyuan_get_document", Description: "按文档 ID 导出 Markdown 内容"},
			{Name: "siyuan_create_document", Description: "在指定笔记本创建 Markdown 文档"},
			{Name: "siyuan_kb_schema", Description: "返回 AI 知识库分类、类型和目录约定"},
			{Name: "siyuan_kb_capture", Description: "按统一分类和元数据模板沉淀知识"},
			{Name: "siyuan_kb_search", Description: "检索知识并返回稳定引用和元数据"},
		},
	}
}
