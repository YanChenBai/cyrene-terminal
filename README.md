# Cyrene Terminal

面向 coding agents 的原生桌面终端，使用 **Go + MyGo 0.7.4**，当前版本 **0.0.0**。采用黑色透明毛玻璃、沉浸式布局与可收起侧栏，终端由 MyGo 的 Ghostty / libghostty-vt 插件驱动。无 WebView，不依赖 Node.js、Vite 或 Vite+。

## 功能

- 快速启动 Codex、Claude、Pi、OMP、OpenCode 或普通 shell；Agent CLI 需另行安装并加入 PATH。
- 文件夹分组可绑定工作目录，在分组内创建的终端自动进入对应目录，未分组终端进入用户主目录。侧栏路径跟随 shell 的 OSC 7 目录上报，默认 PowerShell 自动在当前会话内接入上报，不修改用户 profile；其他 shell 需支持 OSC 7。重新启动沿用最后上报的目录。
- 可拖动调整宽度的侧栏、分组折叠动画；未设置别名的会话使用当前文件夹名，并跟随目录变化。右键会话可设置优先显示的别名，留空恢复文件夹名；别名随当前会话保留。
- 控件 hover 与按下缩放反馈、从点击位置展开且轻微回弹的模态框、以触发按钮为缩放起点的启动菜单。
- 系统字体支持搜索选择或直接输入名称，颜色使用原生颜色选择器并保留十六进制输入。自定义字体、字号、背景浓度、玻璃清晰度及背景、文字、光标颜色；字号、透明度使用一体式减号 / 数值与单位 / 加号按钮组调整。
- 关闭窗口收起到托盘，终端继续运行；点击托盘恢复，**托盘右键 → 退出应用**才结束应用及其终端。关闭单个终端会先请求确认。
- 终端进程退出后保留输出并提供重新启动；右键会话也可重新启动，保留目录、分组及别名。
- Windows 启动或重试终端时刷新用户与系统 PATH，支持 npm 的 `.cmd` / `.ps1` CLI 入口。
- Windows 发行包携带 Chafa，为 Yazi 提供 ANSI 彩色字符图片预览。

当前采用单终端视图，不提供分屏。会话进程仅在应用运行期间保留，退出应用或重启系统后不会恢复进程。

## 开发与构建

需要 **Go 1.27.3 或更高版本**。当前主要验证平台为 Windows amd64；其他桌面平台尚未完成实机验收。MyGo 固定版本及工具声明见 `go.mod`。

```powershell
git clone https://github.com/YanChenBai/cyrene-terminal.git
cd cyrene-terminal
go mod download
go tool mygo dev
```

从仓库根目录构建：

```powershell
go tool mygo build -skip-installer
# 或使用 Windows 工具链包装脚本
./scripts/build.ps1 -skip-installer
./scripts/dev.ps1
```

Windows 输出位于 `build/windows-amd64/`。运行 `Cyrene Terminal.exe`，分发时保留同目录的 DLL、图标和 `tools` 目录。省略 `-skip-installer` 可由 MyGo 生成安装程序。脚本优先使用 PATH 中的 Go，也支持 `%LOCALAPPDATA%/cyrene-terminal/tools/go` 下的便携工具链，不修改全局 PATH。

## 快捷键

| 快捷键 | 操作 |
| --- | --- |
| Ctrl+Shift+T | 在当前分组新建终端，未分组时使用用户目录 |
| Ctrl+Shift+B | 展开 / 收起侧栏 |
| Ctrl+Shift+W | 确认关闭当前终端 |
| Ctrl+Tab / Ctrl+Shift+Tab | 切换到下一个 / 上一个终端 |

全局 `+` 使用用户主目录；分组旁的 `+` 使用绑定目录。右键分组可编辑或移除分组，已有终端及其工作目录不受影响。侧栏收起后仍可通过快捷键打开启动菜单。

## 配置

Windows 配置文件位于 `%APPDATA%/cyrene-terminal/settings.json`，其他系统使用 Go 用户配置目录下的 `cyrene-terminal/settings.json`。外观及目录分组可在界面编辑并保存；自定义 CLI 可修改 `profiles`：

```json
{
  "profiles": [
    { "id": "codex", "name": "Codex", "command": ["codex"] },
    { "id": "custom", "name": "My Agent", "command": ["my-agent", "--flag"] },
    { "id": "shell", "name": "Terminal" }
  ]
}
```

`id` 必须唯一，省略 `command` 使用默认 shell。手动编辑后，从托盘退出再启动应用。Windows 默认优先选择 PowerShell 7，其次为 Windows PowerShell 和 cmd。

## 玻璃效果与图片预览

窗口内使用 MyGo Glass；系统背景材质在 Windows 11 22H2+ 使用 Acrylic，在 macOS 26+ 使用 Liquid Glass。实际效果取决于操作系统与合成器，未支持的平台使用背景色。

**Windows 本地 ConPTY 的 Kitty 图形序列透传尚未解决**，因此暂不能保证 Kitty 原生像素图片显示。当前 Yazi 使用 Chafa 的 ANSI 字符预览，不能等同于完整 Kitty 图形支持。开发环境可运行 `./scripts/setup-chafa.ps1` 安装 Chafa，不改动全局 PATH 或 Yazi 配置。已运行的终端需重建才能获得新环境。

## 项目结构

```text
cmd/cyrene-terminal/    应用入口
internal/app/          原生 UI、工作区、设置、进程管理及对应测试
assets/                可编辑 SVG、嵌入的托盘图片与 agent 标识
resources/             发布图标与按平台分发的工具
scripts/               开发、构建、图标生成及原生验收脚本
docs/                  验证记录
mygo.json              MyGo 构建配置
```

重新生成图标：`go run ./scripts/render-icon`。它使用 MyGo 原生 SVG 渲染器，将 `assets/cyrene.svg` 导出为透明应用图标与托盘图标。

## 验证

从仓库根目录运行：

```powershell
$env:CGO_ENABLED = '0'
go test ./...
go tool mygo vet .
go tool mygo build -skip-installer
```

Windows 原生交互验收另需 Python 3：

```powershell
go build -tags e2e -o .mygo/cyrene-isolated-smoke.exe ./cmd/cyrene-terminal
python scripts/native-smoke.py
```

该脚本使用独立配置与实例，不操作日常会话。结果写入忽略的 `test-results/`。测试覆盖目录绑定、标题、菜单、动画、窗口及托盘生命周期；验证范围与限制见 [验证记录](docs/validation.md)。

## 许可证

项目代码采用 [MIT](LICENSE)。第三方组件保留各自许可证；agent 标识的来源与许可见 [assets/agents/README.md](assets/agents/README.md)。随 Windows 包分发的 Chafa 是独立程序，采用 LGPL-3.0-or-later，其许可与对应源码地址保存在 [resources/windows-amd64/tools/chafa](resources/windows-amd64/tools/chafa)。项目 MIT 许可不改变这些组件的许可证或商标权利。

项目地址：[YanChenBai/cyrene-terminal](https://github.com/YanChenBai/cyrene-terminal) · [MyGo 文档](https://mygo.egoist.dev/docs)
