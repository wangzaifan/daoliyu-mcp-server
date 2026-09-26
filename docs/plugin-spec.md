# 插件包协议 v0.2

插件包是 ZIP，根目录必须有 `plugin.json`，并可包含一个可执行 `entry`（默认 `plugin`）。示例：

```json
{
  "id": "demo.echo",
  "name": "Echo",
  "version": "0.1.0",
  "description": "示例插件",
  "icon": "icon.png",
  "entry": "plugin",
  "provider": "",
  "configSchema": {"type":"object","properties":{"endpoint":{"type":"string","title":"服务地址"}}},
  "tools": [{
    "name": "echo",
    "description": "回显文本",
    "inputSchema": {"type":"object","properties":{"text":{"type":"string"}}}
  }]
}
```

平台 provider 插件也使用同一份 manifest；它们仍然可以像普通插件一样停用或卸载。思源正式接入使用 `mcp-http` connector，思源本体和 Sisyphus 运行在独立的思源 FPK 内。

启用的插件进程按一次调用启动，从 stdin 读取一行 JSON：

```json
{"method":"tools/call","name":"echo","arguments":{"text":"hi"},"config":{"endpoint":"http://127.0.0.1:9000"}}
```

`config` 只包含当前插件卡片保存的配置；不会混入其他插件配置。插件向 stdout 输出一行 MCP `CallToolResult` JSON，stderr 可用于日志。插件必须在对应 NAS 架构下提供可执行文件。

宿主同时提供：

- `DAOLIYU_MCP_DATA`：平台数据根目录，只读使用。
- `DAOLIYU_PLUGIN_DIR`：当前插件程序目录。
- `DAOLIYU_PLUGIN_DATA`：当前插件持久化目录；插件只能把运行数据写到这里。
- `DAOLIYU_PLUGIN_ID`：当前插件 ID。

Go 插件可直接使用 `sdk/plugin`：注册 handler 后调用 `Run`。handler 同时接收 `arguments` 和本插件的 `config`。宿主已用端到端测试覆盖 MCP 调用、配置传递、stdout 结果解析、插件数据保留恢复和彻底清除。

## 上游 MCP HTTP 插件

已有 Streamable HTTP MCP 服务不需要再写进程适配器，只需打包 manifest：

```json
{
  "id": "example.upstream",
  "name": "Example MCP",
  "version": "1.0.0",
  "type": "mcp-http",
  "endpoint": "http://127.0.0.1:9000/mcp",
  "toolPrefix": "example_",
  "configSchema": {
    "type": "object",
    "properties": {
      "endpoint": {"type": "string", "title": "MCP 地址"},
      "token": {"type": "string", "title": "Bearer Token", "format": "password"}
    }
  },
  "tools": []
}
```

平台在 MCP 会话建立时调用上游 `tools/list`，把工具注册为 `toolPrefix + 上游名称`，调用时转发原始参数和结果。`endpoint` 与可选 `token` 可由插件配置覆盖；Token 只作为上游 `Authorization: Bearer` 头发送，不写入日志。上游暂时离线时只跳过该插件，不影响平台和其他插件。调用 Token 的插件范围仍按该 manifest 的 `id` 执行。
