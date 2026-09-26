# 思源笔记 fnOS FPK

该发行包把思源官方无头 Docker 发行内容封装为 fnOS 原生 FPK，不包含 Docker 守护进程，也不依赖飞牛统一网关。

## 安装行为

- 安装向导设置独立服务端口，默认 `6806`。
- 首次安装必须设置思源锁屏密码；该密码不是 API Token。
- 工作空间和配置分别保存到持久 `data-share`：`daoliyu.siyuan/workspace`、`daoliyu.siyuan/config`。
- 升级不会覆盖工作空间、端口或锁屏密码。
- 卸载默认保留全部笔记和配置；只有明确选择“彻底清除”才删除。
- 桌面入口为独立浏览器 `url`，不使用 `iframe`、`gatewayPrefix` 或 `gatewaySocket`。
- FPK 同时启动 Sisyphus `0.6.7`，只监听 `127.0.0.1:36806/mcp`；它从思源持久配置读取现有 API Token，Token 不进入命令行或日志。

安装后通过 `http://<FNOS-IP>:<安装端口>/` 写作和管理思源。思源 FPK 内已经运行仅监听本机的 Sisyphus；在道理鱼 MCP 上传并启用 `sisyphus-0.6.7.zip` 只是登记这条连接，默认连接 `http://127.0.0.1:36806/mcp`，无需填入思源 API Token。道理鱼不会再安装一份思源；旧直连适配器不随正式构建发布，历史数据会在升级时保留。

## 构建

```sh
SIYUAN_VERSION=3.8.4 PACKAGE_VERSION=3.8.4-0003 ./scripts/build-siyuan-fpk.sh
./scripts/check-siyuan-fpk.sh dist/siyuan/3.8.4-0003/*.fpk
```

构建脚本从官方 `b3log/siyuan` 多架构镜像提取 `/opt/siyuan`、musl 加载器、CA 和时区数据，从官方 `node:20-alpine` 提取同架构 Node 运行时，并嵌入校验过 SHA-256 的原版 Sisyphus 包。三方许可证随包保留；x86_64 与 ARM64 分别构建，不混用二进制。

## 边界

这是思源服务端浏览器版本。按照思源官方 Docker 说明，它不等同于桌面客户端，部分桌面专属导入导出能力不可用。一个工作空间只能由一个写入实例运行；备份前应先停止服务，不能只复制运行中的 SQLite 索引文件。
