# 第三方插件开发

面向插件作者的完整入门文档是 [`plugin-author-guide.md`](plugin-author-guide.md)。它包含小白可执行的目录模板、图标尺寸、JSON 配置、Go 示例、MCP HTTP 示例、x86/ARM 打包、安装验证和 GitHub 发布流程。

宿主规则是固定的：第三方插件不能使用 `provider`，不能声明或修改 Token、权限、数据目录、FPK 生命周期、统一网关和其他插件的数据；插件只定义自己的 MCP 工具、参数、描述和私有配置。

1. 创建目录并写入 `plugin.json`。
2. 编译与目标 NAS 架构匹配的 `plugin` 可执行文件。
3. 将两者 ZIP 打包，在管理 UI 点击“安装插件包”。
4. 安装默认停用，确认来源和权限后再启用；启停状态在服务下次启动时加载。
5. 若插件声明 `configSchema`，平台会在该插件卡片的“配置”弹窗中渲染字段，并把配置写入该插件自己的命名空间。
6. handler 从调用请求的 `config` 字段读取配置，把可恢复数据写入 `DAOLIYU_PLUGIN_DATA`；不要写程序目录。

图标必须是 ZIP 根目录中的 PNG，正方形，64-512 像素，文件不超过 512 KiB；超过范围会被宿主拒绝。`repository` 和 `homepage` 可选，但必须是 `http` 或 `https` 地址。

最小 Go 入口：

```go
package main

import (
    "context"
    "encoding/json"
    "os"

    sdk "github.com/daoliyu/daoliyu-mcp/sdk/plugin"
)

func main() {
    server := sdk.NewServer()
    server.Handle("echo", func(_ context.Context, arguments, config json.RawMessage) (any, error) {
        return map[string]any{"arguments": arguments, "config": config}, nil
    })
    if err := server.Run(context.Background(), os.Stdin, os.Stdout); err != nil { panic(err) }
}
```

每个架构分别编译：`CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o plugin` 或 `GOARCH=arm64`，再把 `plugin`、`plugin.json` 和图标放在 ZIP 根目录。

插件不要访问飞牛内部 socket，也不要假设统一网关存在。需要文件共享或 NAS 权限时，先在后续版本的宿主能力清单中申请；当前协议只传递 MCP 工具调用及当前插件配置。

卸载插件时默认可选择保留私有数据和配置；重新安装同 ID 插件会恢复 `DAOLIYU_PLUGIN_DATA`。选择彻底清除时，宿主删除插件私有数据和该插件配置。
