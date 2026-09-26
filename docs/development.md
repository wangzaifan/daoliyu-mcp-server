# 开发说明

本项目只使用 Go 标准库加官方 `github.com/modelcontextprotocol/go-sdk`。本地运行：

```sh
go test ./...
go run ./cmd/daoliyu-mcp -data ./data
```

浏览器打开 `http://127.0.0.1:37421/` 管理插件，Token 管理独立页面为 `/tokens`。两个独立后台页面均不需要额外管理密钥。ChatGPT 等客户端必须使用页面生成的调用 Token。

交叉打包：

```sh
VERSION=0.3.2 ./scripts/build-fpk.sh
```

产物位于 `dist/<version>/`，生成 Sisyphus MCP connector ZIP；旧思源直连适配器源码不进入正式包。FPK 升级回调不清理 `${TRIM_PKGVAR}/data`，应用数据由 fnOS 持久目录承载，版本升级只替换程序文件。未在真实 x86/ARM fnOS 设备安装验证前，不把构建成功称为兼容性验证。
