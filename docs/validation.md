# 验证记录

## 0.0.0 仓库整理

版本统一重置为 0.0.0。入口迁移到 `cmd/cyrene-terminal`，实现及测试迁移到 `internal/app`，嵌入资源由 `assets` 包提供。MyGo 的 `main` 配置、脚本与验收入口同步更新。托盘提示名称改为 `Cyrene Terminal`。

2026-11-03，Windows amd64，Go 1.27.3，MyGo 0.7.4：

- `CGO_ENABLED=0 go test ./...`：全部通过。
- `go tool mygo vet .`：通过原生 UI 生命周期及协程捕获检查。
- `./scripts/build.ps1 -skip-installer`：成功生成 `build/windows-amd64/Cyrene Terminal.exe`、Ghostty DLL、图标及 Chafa 工具，发行元数据为 0.0.0。
- `go build -tags e2e -o .mygo/cyrene-isolated-smoke.exe ./cmd/cyrene-terminal` 与 `python scripts/native-smoke.py`：成功完成真实窗口、分组目录继承、OSC 标题、侧栏拖动及隐藏、设置保存、会话切换、关闭确认、窗口关闭后进程继续运行、再次启动恢复窗口、显式退出与子进程结束验收。
- Chafa 可用及 Yazi ANSI 图片预览检查通过。Kitty 经 Windows ConPTY 显示/删除检查仍为 false，保留为已知限制，未将其计为通过。
- 真实窗口验收使用独立配置与单实例标识；未关闭已有用户会话。托盘退出验证调用与生产托盘菜单相同的回调，不等同于通知区右键菜单的鼠标验收。托盘提示文本在构建源码中设置为 `Cyrene Terminal`。
- Git 索引包含源码、资源及第三方许可证，排除构建目录、原生工具缓存、模板备份、node_modules 与测试输出。

## 已知限制

Windows 本地 ConPTY 尚未完成 Kitty 图形序列透传；Yazi 使用 Chafa ANSI 字符预览。其他桌面平台未完成实机验收。退出应用后不会恢复已结束的进程。

## 早期记录

[整理前验证历史](validation-history.md)保留此前开发迭代及限制的记录，其中旧版本号、路径和命令属于历史，不代表当前发行版本。当前使用方式以仓库根目录 README 为准。
