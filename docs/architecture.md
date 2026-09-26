# 架构

```text
ChatGPT / MCP Client
        │ Streamable HTTP + 调用 Token
        ▼
独立端口 37421 ── MCP Server
        ├── MCP connector ── Sisyphus 127.0.0.1:36806 ── SiYuan :6806
        ├── Plugin Manager ── data/plugins/<id>
        └── Management UI ── plugin cards / upload / config / token scopes
```

首版不经过 fnOS gateway，不依赖 fnOS 统一会话，也不调用 fnOS 开放 API。FPK 的桌面入口必须使用 `type: "url"` 指向自定义端口；禁止改回 `iframe`、`gatewayPrefix` 或 `gatewaySocket`，这样统一网关失效不会影响独立页面和 MCP 端口。进入独立后台页即可执行管理操作；页面创建的调用 Token 只允许访问 MCP。每个 MCP 请求根据 Token 的 `*` 或插件 ID 范围动态注册已启用插件的工具，删除 Token 后立即失效。

Token、插件目录、启用状态和插件配置统一位于 `${TRIM_PKGVAR}/data`，升级/重新安装保留；卸载向导默认保留，只有明确选择彻底清除才删除。

通用 `mcp-http` 插件动态读取上游工具并转发调用，平台仍在入口执行调用 Token 的插件范围鉴权。思源 FPK 内的 Sisyphus sidecar 只监听 `127.0.0.1:36806`，外部客户端不能绕过道理鱼 Token 直接访问。道理鱼卡片只是连接器，不是第二个思源安装；旧 `provider: siyuan` 适配器仅保留源码和可恢复数据，不再默认安装。

插件卸载可保留或清除私有数据，重新安装同 ID 时恢复保留数据。FPK 只负责安装、生命周期和独立端口进程；`config/resource` 不声明 API scope。

文档分层：

- `docs/development.md` 是本项目维护者的构建与调试说明。
- `docs/operations.md` 是安装 FPK、读取令牌和升级运维说明。
- `docs/plugin-spec.md` 与 `docs/third-party.md` 是第三方插件作者契约。
- `docs/siyuan.md` 是思源插件的 API 边界与安全约束。
