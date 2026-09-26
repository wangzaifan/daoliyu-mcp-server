# 思源知识库直连适配器（历史参考）

> 正式部署不再安装本适配器。当前链路是思源 FPK 内的 Sisyphus MCP，加上道理鱼的 `mcp-http` 连接器；本文件保留知识库工具契约和迁移参考。

思源以普通可卸载插件接入道理鱼 MCP。插件只调用思源公开 HTTP API，不直接读写工作空间中的 `.sy` 文件，也不经过飞牛统一网关。默认连接 `http://127.0.0.1:6806`，Token 来自思源的 API Token 设置。

## AI 知识结构

AI 知识放在用户指定的思源笔记本中，默认文档根目录为 `/AI知识库`：

| category | 目录 | 用途 |
| --- | --- | --- |
| `inbox` | `00_收件箱` | 尚未整理的临时沉淀 |
| `fact` | `10_事实` | 可核验事实和数据 |
| `concept` | `20_概念` | 概念、方法和模型 |
| `decision` | `30_决策` | 决策、理由和取舍 |
| `project` | `40_项目` | 项目状态和交接 |
| `procedure` | `50_操作手册` | 可重复执行的步骤 |
| `reference` | `60_参考资料` | 外部资料摘要与来源 |
| `archive` | `90_归档` | 不再活跃但需保留的内容 |

类型为 `note/fact/concept/decision/project/procedure/source/question`，状态为 `inbox/active/verified/archived`。分类、类型、状态、标签、来源和关联 ID 保存为文档自定义属性 `custom-ai-*`；正文仍是普通 Markdown，用户可直接在思源独立页面中继续编辑。`related` 保存思源文档或块 ID，AI 返回结果同时带 `id/root_id/notebook/hpath`，便于后续引用和追溯。

## MCP 工具

- `siyuan_kb_schema`：读取上述分类约定；AI 首次使用时先调用。
- `siyuan_kb_capture`：把知识写入指定笔记本；未指定路径时自动进入对应分类目录。
- `siyuan_kb_search`：按正文、笔记本和 AI 元数据搜索，返回稳定引用、片段和属性。
- `siyuan_get_document`：按文档 ID 导出完整 Markdown。
- `siyuan_list_notebooks`：选择写入目标笔记本。
- `siyuan_health`、`siyuan_search`、`siyuan_create_document`：保留基础诊断和通用操作。

推荐调用顺序：`health → list_notebooks → kb_schema → kb_search/get_document`；需要沉淀时调用 `kb_capture`，并填写 `source` 与 `related`。未经用户要求不自动删除、移动或覆盖文档。

## 数据与独立页面

```text
道理鱼 MCP 应用数据
${TRIM_PKGVAR}/data/
├── config.json              # 端口、插件启用状态、插件配置
├── auth-tokens.json         # MCP 调用 Token
└── plugins/
    ├── siyuan/              # 插件 manifest
    └── <id>.data/           # 卸载时选择保留的插件私有数据

思源工作空间
由思源自身管理，包含笔记本、.sy、资源和索引；道理鱼 MCP 不直接修改。
```

思源网页本身使用思源独立端口访问，不依赖道理鱼 MCP 页面或飞牛统一网关。本文件描述的是历史直连适配器的工具契约与知识库结构；正式部署使用思源 FPK 内的 Sisyphus MCP，再由道理鱼的 `mcp-http` 连接器接入。不要把本适配器和思源 FPK 同时安装成两套思源工具。

公开 API 边界：`/api/notebook/lsNotebooks`、`/api/query/sql`、`/api/export/exportMdContent`、`/api/filetree/createDocWithMd`、`/api/attr/getBlockAttrs`、`/api/attr/setBlockAttrs`。不调用未公开的 `/api/transactions`。
