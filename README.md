# 道理鱼 MCP 服务

这是一个独立端口运行的 MCP 平台壳：本地插件和上游 Streamable HTTP MCP 都以可安装插件接入，可配置、停用或卸载；平台提供调用 Token 范围鉴权、插件健康与工具检查，以及 x86/ARM fnOS FPK。

## 快速运行

```sh
go run ./cmd/daoliyu-mcp -data ./data
open http://127.0.0.1:37421/
```

本应用只使用自定义端口提供独立浏览器页面，不使用 fnOS `iframe` 桌面窗口或统一网关。插件后台为 `/`，Token 管理是独立的 `/tokens` 页面；进入页面即可直接操作，不需要额外管理密钥。MCP 地址是 `http://<主机>:37421/mcp`，请求使用 `Authorization: Bearer <调用 Token>`。当前版本不使用飞牛统一网关，也不申请飞牛开放 API scope。

Token、插件包、启用状态和插件配置全部保存在应用数据目录。升级和重新安装默认保留并自动恢复；卸载向导可选择保留，或明确选择“彻底清除全部应用数据”。

## 目录

- `cmd/`：可执行入口
- `internal/`：平台核心、插件管理与 UI；旧思源直连适配器源码仅作迁移参考
- `docs/`：架构、开发、第三方插件和思源说明
- `fpk/`：fnOS 包骨架（不含编译产物）
- `sdk/plugin/`：第三方 Go 插件最小运行 SDK
- `plugins/sisyphus/`：通用 MCP HTTP 形式的 Sisyphus connector
- `plugins/siyuan/`：旧思源知识库直连适配器源码（不再随正式构建发布）
- `siyuan-fpk/`：思源无头服务、Node 与本机 Sisyphus sidecar 的 fnOS FPK 骨架
- `scripts/build-fpk.sh`：交叉编译 x86/ARM FPK 和 MCP 连接器 ZIP
- `scripts/build-siyuan-fpk.sh`：从思源官方多架构镜像构建独立思源 FPK

## 当前边界

正式链路只有一条：思源 FPK 内运行 Sisyphus sidecar，道理鱼安装一个普通的 MCP 连接器负责发现、转发和 Token 范围鉴权。道理鱼不会再次安装思源，也不复制底层思源 API、权限和严格写入预检。旧思源直连适配器仅保留源码和数据迁移能力，不进入正式插件列表。第三方既可使用一行 JSON stdin/stdout 协议，也可用 `mcp-http` manifest 直接代理现有 MCP；卸载可保留并在重装时恢复，也可彻底清除。
