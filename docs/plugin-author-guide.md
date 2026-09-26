# 道理鱼可插拔能力包开发指南

这份文档写给插件作者、第一次做 MCP 工具的人，以及需要把这份文档交给 AI 执行的人。按“准备目录 -> 写一个 JSON -> 写工具程序 -> 打 ZIP -> 上传验证”做完，就能把自己的能力作为一张可启用、可停用、可配置、可卸载的卡片接入。

如果你要让 AI 帮你做插件，可以直接把这句话发给它：

> 请严格按照道理鱼《可插拔能力包开发指南》制作一个插件。只实现我描述的工具和配置，不修改宿主规则；使用 512x512 PNG 图标；最后运行打包脚本并检查 ZIP。

## 先理解边界

道理鱼是宿主平台。宿主负责安全规则和生命周期，插件作者只负责“这个能力做什么、工具怎样调用”。下面这些规则由宿主固定，插件不能声明、修改或绕过：

- 调用 Token、插件范围和 MCP 入口端口。
- 插件启用、停用、卸载、升级，以及卸载时保留或清除数据。
- 插件配置的存储位置、权限边界和持久化恢复。
- `DAOLIYU_PLUGIN_DATA`、`DAOLIYU_PLUGIN_DIR` 等运行时目录的含义。
- fnOS 统一网关、内部 socket、系统管理令牌和其他插件的数据。
- 宿主的 FPK 生命周期、备份、回滚和数据清理规则。

插件只能在自己的目录中读写自己的数据，只能读取宿主传入的本插件配置，只能通过 MCP 工具返回结果。`provider` 是宿主内置能力的保留字段，第三方 ZIP 不能使用；插件也不能通过自定义字段申请新的宿主权限。

插件可以自己定义：

- 工具名称、描述、参数和返回内容。
- 插件自己的配置字段，例如服务地址、模型名称或显示选项。
- 连接方式：本地可执行程序，或已有的 Streamable HTTP MCP 服务。
- GitHub 仓库、官网和卡片图标。

## 你要交付什么

最小目录如下。所有文件都放在 ZIP 根目录，不要再套一层同名目录：

```text
demo-weather/
├── plugin.json
├── plugin              # Linux 可执行文件；mcp-http 插件不需要它
└── icon.png            # 64-512 的正方形 PNG，推荐 512x512
```

### 图标规则（硬限制）

图标是卡片的一部分，作者必须自己准备并测试好。平台只接受：

- PNG 文件，不能使用 SVG、WebP 或 JPG。
- 正方形，宽和高都在 `64` 到 `512` 像素之间。
- 文件不超过 `512 KiB`。
- 图标路径必须指向 ZIP 内的文件，不能使用 `../` 跳出插件目录。

超出任何一项，打包脚本或安装器都会拒绝。推荐使用透明背景的 `512x512` PNG，主体四周留出约 10% 空白；平台会在卡片中按统一圆角显示，作者不需要把圆角网页代码塞进图标。这样可以避免图标撑大卡片、裁切错位或破坏页面布局。

## 第一步：写 `plugin.json`

下面是本地程序插件模板。字段名可以直接复制，只有工具和配置内容需要换成你自己的：

```json
{
  "id": "demo.weather",
  "name": "天气查询",
  "version": "1.0.0",
  "description": "查询指定城市的当前天气",
  "icon": "icon.png",
  "repository": "https://github.com/your-name/demo-weather",
  "homepage": "https://example.com/demo-weather",
  "entry": "plugin",
  "configSchema": {
    "type": "object",
    "properties": {
      "endpoint": {
        "type": "string",
        "title": "天气服务地址",
        "default": "https://api.example.com"
      }
    }
  },
  "tools": [
    {
      "name": "weather_current",
      "description": "查询一个城市的当前天气",
      "inputSchema": {
        "type": "object",
        "properties": {
          "city": { "type": "string", "description": "城市名称" }
        },
        "required": ["city"]
      }
    }
  ]
}
```

字段说明：

| 字段 | 作用 |
| --- | --- |
| `id` | 全局唯一 ID，只能使用小写字母、数字、`.`、`_`、`-`，例如 `demo.weather`。发布后不要修改。 |
| `name`、`version`、`description` | 卡片显示的名称、版本和简介。 |
| `icon` | ZIP 内的 PNG 图标路径。第三方能力包必须提供它；格式和尺寸必须符合上面的硬限制。 |
| `repository`、`homepage` | 卡片上的仓库和主页链接，必须是 `http` 或 `https`。 |
| `entry` | 本地程序入口文件，默认是 `plugin`。 |
| `configSchema` | 只描述本插件自己的配置。配置会保存到本插件命名空间，密码字段使用 `"format": "password"`。 |
| `tools` | 本地程序插件公开的工具清单。工具名和参数由插件作者定义。 |

不要添加 `provider`。不要在 `configSchema` 里伪造权限、Token、数据目录或系统设置字段；这些字段不会获得宿主权限。

## 第二步：写本地插件程序

推荐使用仓库自带的最小 Go SDK。SDK 使用一行 JSON 输入、一行 JSON 输出，适合 fnOS 的受限运行环境。

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
    server.Handle("weather_current", func(_ context.Context, arguments, config json.RawMessage) (any, error) {
        var in struct { City string `json:"city"` }
        if err := json.Unmarshal(arguments, &in); err != nil { return nil, err }
        return map[string]any{
            "city": in.City,
            "config": json.RawMessage(config),
            "message": "在这里调用你的天气服务并返回结构化结果",
        }, nil
    })
    if err := server.Run(context.Background(), os.Stdin, os.Stdout); err != nil { panic(err) }
}
```

运行时平台会传入：

| 环境变量 | 含义 |
| --- | --- |
| `DAOLIYU_PLUGIN_ID` | 当前插件 ID。 |
| `DAOLIYU_PLUGIN_DIR` | 当前插件程序目录，只读。 |
| `DAOLIYU_PLUGIN_DATA` | 当前插件持久化目录，插件自己的缓存和数据只能写这里。 |
| `DAOLIYU_MCP_DATA` | 平台数据根目录，只读参考；不要写入，也不要遍历其他插件目录。 |

程序不得依赖 fnOS 统一网关、飞牛内部 socket、其他插件目录或系统管理令牌。需要网络时只访问自己声明的服务地址。

## 另一种方式：代理已有 MCP 服务

如果你已经有一个 Streamable HTTP MCP 服务，不需要写 `plugin` 程序，只需把 `plugin.json` 写成：

```json
{
  "id": "demo.remote",
  "name": "远程 MCP 能力",
  "version": "1.0.0",
  "description": "转发一个已有的 MCP 服务",
  "icon": "icon.png",
  "repository": "https://github.com/your-name/demo-remote",
  "type": "mcp-http",
  "endpoint": "http://127.0.0.1:9000/mcp",
  "toolPrefix": "demo_",
  "configSchema": {
    "type": "object",
    "properties": {
      "endpoint": { "type": "string", "title": "MCP 地址" },
      "token": { "type": "string", "title": "Bearer Token", "format": "password" }
    }
  },
  "tools": []
}
```

平台会发现上游的 `tools/list`，并以 `toolPrefix + 上游工具名` 暴露给 MCP 客户端。上游 Token 只作为请求头发送，不写日志；道理鱼调用 Token 仍然负责决定这个插件能不能被调用。

## 第三步：编译和打包

在插件目录中编译两个 fnOS 架构。不要把 macOS 或 Windows 可执行文件放进 ZIP：

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o plugin .
zip -q demo-weather-1.0.0-x86.zip plugin.json plugin icon.png

CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -o plugin .
zip -q demo-weather-1.0.0-arm.zip plugin.json plugin icon.png
```

如果插件是 `mcp-http`，只需要打包 `plugin.json` 和 `icon.png`。优先使用仓库脚本，它会检查清单、保留字段、图标格式、尺寸、大小和 ZIP 内容：

```sh
./scripts/package-plugin.sh ./demo-weather ./demo-weather-1.0.0-x86.zip
```

手工打包时也要检查 ZIP 是否正确：

```sh
unzip -l demo-weather-1.0.0-x86.zip
unzip -t demo-weather-1.0.0-x86.zip
```

输出中必须能直接看到 `plugin.json` 和 `icon.png`，不能是 `demo-weather/plugin.json` 这种多一层目录。

## 第四步：安装和验证

1. 打开道理鱼 MCP 服务首页，点击右上角设置，再选择“安装插件包”。
2. 选择对应架构的 ZIP。插件默认停用，这是正常的安全行为。
3. 打开插件配置，填写本插件需要的配置并保存。
4. 启用插件，查看卡片上的能力标签和状态。
5. 点击能力详情，确认工具名称和描述正确。
6. 到 Token 管理中创建一个只允许该插件的调用 Token，再用 MCP 客户端测试。

升级时使用相同 `id` 的新版本 ZIP。平台会替换程序文件并恢复原配置和 `DAOLIYU_PLUGIN_DATA`。卸载时选择“保留”会留下这些数据，选择“彻底清除”才会删除。

## 第五步：发布到 GitHub

公开仓库只提交源代码、`plugin.json`、图标、许可证和文档。不要提交 Token、密码、Cookie、`.env`、私钥、日志、截图、`build/`、`dist/` 或本机数据目录。

```sh
git init -b main
git add plugin.json icon.png .gitignore README.md LICENSE docs/ src/
git commit -m "feat: add demo weather MCP plugin"
git remote add origin https://github.com/your-name/demo-weather.git
git push -u origin main
```

发布前自查：

```sh
git status --short
git grep -n -I -E 'BEGIN .* PRIVATE KEY|gho_|sk-|PASSWORD=|Bearer [A-Za-z0-9._-]{20,}' || true
```

插件包可以作为 GitHub Release 附件发布，仓库源码保持轻量。每次升级同时修改 `version`、更新变更说明，并重新生成 x86 和 ARM 两个 ZIP。

## 常见错误

- **安装提示 provider 保留**：删除 `provider` 字段。它只属于宿主内置能力。
- **安装提示图标无效**：确认是 PNG、正方形、64-512 像素、不超过 512 KiB，并且图标路径没有 `../`。
- **卡片没有能力**：检查 `tools` 是否填写，或确认上游 MCP 的 `tools/list` 可访问。
- **启用后状态异常**：检查入口文件是否为 Linux 可执行文件，以及配置里的服务地址是否能从 NAS 访问。
- **升级后配置丢失**：确认插件只写 `DAOLIYU_PLUGIN_DATA`，不要把数据写进程序目录。
