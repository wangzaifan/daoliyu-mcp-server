# 第三方插件开发

1. 创建目录并写入 `plugin.json`。
2. 编译与目标 NAS 架构匹配的 `plugin` 可执行文件。
3. 将两者 ZIP 打包，在管理 UI 点击“安装插件包”。
4. 安装默认停用，确认来源和权限后再启用；启停状态在服务下次启动时加载。
5. 若插件声明 `configSchema`，平台会在该插件卡片的“配置”弹窗中渲染字段，并把配置写入该插件自己的命名空间。
6. handler 从调用请求的 `config` 字段读取配置，把可恢复数据写入 `DAOLIYU_PLUGIN_DATA`；不要写程序目录。

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
