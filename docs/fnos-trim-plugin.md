# fnOS 管理工具预制插件

这个插件把 [fnOS trim-cli V2](https://github.com/techysy/fnos-trim-cli-skill) 接入道理鱼 MCP。它是连接器，不是第二个 fnOS 管理后台：道理鱼负责 MCP Token 和插件范围，trim-cli 负责通过 fnOS WebSocket 调用 NAS。

## 当前版本判断

`/Users/wangzaifan/Downloads/skill` 是 V2 内容。它与上游 `main` 当前文件对比一致，只缺少上游 README 和 Agent 指南；上游主分支最新提交为 `43abc51`（2026-08-13）。`manifest.json` 中的 `0.1.0` 是 CLI 包版本，不代表 V1/V2；V2 的判断依据是新增的 network、photos、media、profile 和 Windows ARM64 能力。

上游仓库当前没有声明可识别的开源许可证，因此本仓库不直接重新发布 `trim-cli` 二进制；打包脚本从你已取得的 V2 目录取用对应架构文件。

## 首版能力

首版只暴露查询和登录相关工具，避免 AI 误触发关机、格式化、删除文件、存储池变更、应用卸载等破坏性操作：

- 登录、退出、系统状态、系统信息、CPU、内存。
- 文件列目录、搜索、属性/大小/下载地址查询。
- 应用列表、应用状态、检查更新。
- Docker 容器和统计。
- 下载任务列表、详情和统计。
- 存储总览、存储池、磁盘、健康和 SMART。
- 照片搜索和详情，影视库统计和搜索。

配置页必须填写 fnOS 地址和 WebSocket 端口。默认是 `127.0.0.1:5666`，访问其他设备时改成 NAS 的地址；远程自签名环境才打开 `tlsInsecure`，远程明文 WS 默认关闭。

首次调用 `fnos_login` 时传入用户名、密码和可选 TOTP。会话只保存到宿主提供的 `DAOLIYU_PLUGIN_DATA`，不会写入程序目录、平台根目录或日志。

## 打包

V2 的 Linux x86 和 ARM 二进制不直接提交到本仓库。这样避免把没有明确开源许可证的第三方二进制重新发布。准备好 V2 目录后运行：

```sh
FNOS_SKILL_DIR=/path/to/skill ./scripts/build-fnos-trim-plugin.sh
```

脚本会验证 V2 清单和两个 Linux 二进制，分别编译道理鱼适配器，生成：

```text
dist/fnos-trim/fnos-trim-0.1.0-x86.zip
dist/fnos-trim/fnos-trim-0.1.0-arm.zip
```

生成的 ZIP 根目录包含 `plugin`、`plugin.json`、`icon.png` 和对应架构的 `bin/trim-cli`。上传到道理鱼设置页即可安装；安装后先配置地址，再启用、登录和调用查询工具。

## 安全边界

这是安全默认版。V2 中的写操作仍可由用户在 fnOS 官方界面或独立 CLI 中执行；后续如果要开放 MCP 写操作，应按工具逐个加入明确参数、二次确认和最小权限，不使用任意命令执行入口。
