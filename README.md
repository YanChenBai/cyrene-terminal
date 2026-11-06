# Cyrene Terminal

面向 coding agents 的原生桌面终端，使用 **Go + MyGo 0.7.4**，当前版本 **0.0.0**。采用透明磨砂玻璃、顶部工作区和带凸起光影的毛玻璃圆角窗格布局，不再提供左右侧栏，终端由 MyGo 的 Ghostty / libghostty-vt 插件驱动。应用外壳和终端为原生 UI，网站访问通过独立的 MyGo WebView 窗格提供，不依赖 Node.js、Vite 或 Vite+。

## 功能

- 快速启动 Codex、Claude、Pi、OMP、OpenCode 或普通 shell；Agent CLI 需另行安装并加入 PATH。
- 设置左侧的全局 `+` 创建工作区，默认依次命名为 `Workspace`、`Workspace2`；名称及编号重开后保持。工作区内部的 `+` 添加终端或空白网页，普通独立 tab 不显示内部 `+`。Agent 进程退出后，可在原标签点击“恢复会话”或通过该会话的右键菜单恢复，保留标签、别名、分组和当前目录；运行中的会话不会被此操作终止。
- 已绑定的 Agent 按确切 Session ID 直接恢复，不猜“最近一次”。Claude / Pi 新建时分配独立 ID；Codex 和 Claude 使用各窗格的启动 hook 捕获实际会话 ID，不改动全局配置。旧记录缺少 ID 时，Codex / Claude 在原窗格打开自带历史列表供选择，不要求复制粘贴 ID。自定义 Agent 无历史选择器时保留手工绑定入口。
- 从托盘退出整个应用时保存工作区；重开后还原终端目录、分组、别名、标签顺序、分屏比例和网页地址，Agent 按原 ID 续聊。普通 shell 会重新启动，内存状态和已结束的后台进程不会复活。
- 工作区标签位于 50-DIP 标题栏中，使用 34-DIP 胶囊分组，增大字号和内部间距，并平衡顶部与内容之间的留白。当前工作区有多个窗格时，通过连续弹簧动画向右展开全部子 tabs；点击子 tab 聚焦对应终端或网页，切换工作区平滑收起上一组。工作区和子 tab 的关闭按钮常显并占位；组内可拖拽排序，外部窗格或整个工作区拖入组内即合并，拖到组的外侧边缘则调整组顺序。顶部标签可拖动排序，多窗格工作区整体移动；过多时横向滚动并自动显示当前选择。
- 拖动子 tab、顶部标签或窗格标题到终端边缘，可向左、右、上、下分屏；拖动时显示半屏预览，松手后移动原终端，不重新启动进程。一个画面可混合同分组或不同分组的终端及网页，原分组、目录和别名不变。
- 终端窗格保留半透明底色、凸起边缘和更明显的底部回光，系统 Acrylic 提供背景毛玻璃；不使用液态折射。选中工作区和聚焦窗格有更明显的高亮，标题放大并显示可用的 Agent 图标；窗格操作按钮仅聚焦或悬停时动画出现，增加、移除和切换窗格都有过渡。不另画分割线，边界仍支持鼠标拖动和聚焦后的方向键调整。把窗格标题拖回顶部标签栏可移回独立标签，也可通过右键菜单或命令面板操作、均分布局；关闭单个窗格只结束该终端，关闭整张画面标签会明确确认所有包含的终端。
- 弹层和模态框使用真正的背景模糊、轻染色与凸起边缘，不靠降低透明度模拟玻璃。网页弹层期间临时使用页面当前画面参与模糊，关闭后回到同一原网页，不重新加载。
- Ctrl+Shift+P 打开可搜索的命令面板，支持方向键、Enter 和 Esc。分屏时可选择所有分组的已有窗格，或在目标窗格的目录和分组内新建终端、网页。Git diff 和文件预览仍待后续实现。
- 顶部标签右键可设置标签别名；命令面板中的“命名当前标签…”和“设置当前会话别名…”分别命名聚焦的画面和终端，不影响其他标签/会话。两类名称独立，清空后恢复默认名称，随工作区保存恢复。
- 支持 Ctrl+V / Ctrl+Shift+V 粘贴截图和本地文件。剪贴板 PNG 在后台保存为临时图片，再按拖入文件的方式传入路径；支持通过路径附图的 Agent 可读取图片，包括在 shell 内启动的 Agent。图片保留到对应终端关闭或重新启动，普通文本沿用原有粘贴行为。
- 文件夹分组可绑定工作目录，在分组内创建的终端自动进入对应目录，未分组终端进入用户主目录。终端路径跟随 shell 的 OSC 7 目录上报，默认 PowerShell 自动在当前会话内接入上报，不修改用户 profile；其他 shell 需支持 OSC 7。重新启动沿用最后上报的目录。
- 控件 hover 与按下缩放反馈、从点击位置展开且轻微回弹的模态框、以触发按钮为缩放起点的启动菜单。
- 系统字体支持搜索选择或直接输入名称，颜色使用原生颜色选择器并保留十六进制输入。自定义字体、字号、背景浓度、玻璃清晰度及背景、文字、光标颜色；字号、透明度使用一体式减号 / 数值与单位 / 加号按钮组调整。
- 关闭窗口收起到托盘，终端继续运行；点击托盘恢复，**托盘右键 → 退出应用**才结束应用及其终端。关闭单个终端会先请求确认。
- 终端进程退出后保留输出并提供重新启动；右键会话也可重新启动，保留目录、分组及别名。
- Windows 启动或重试终端时刷新用户与系统 PATH，支持 npm 的 `.cmd` / `.ps1` CLI 入口。
- Windows 发行包携带 Chafa，为 Yazi 提供 ANSI 彩色字符图片预览。

左侧列表管理会话，顶部标签管理画面布局；切换左侧会话会定位到包含它的画面。会话进程和分屏布局仅在应用运行期间保留，退出应用或重启系统后不会恢复进程。

## 网页窗格

从当前工作区内的圆形 `+` 选择“新建网页”，或在 Ctrl+Shift+P 搜索“打开网页”，直接进入空白网页窗格，在已聚焦的地址栏输入网址后回车。无需弹框，也不会预先访问网站。顶部不显示品牌名和命令面板独立图标，命令入口保留 Ctrl+Shift+P。支持 HTTP(S)、`localhost:3000` 和本地 IP；地址栏提供前进、后退、刷新，Ctrl+L 聚焦地址。也可在分屏命令中选择“新建网页”。

网页与终端共享分屏、子 tabs 和拖拽操作。切换工作区或重新停靠保留页面状态，关闭网页不结束旁边的终端。退出后保存网址，重开时重新加载；不恢复 DOM、表单内容或浏览历史。网页没有应用 Go bridge。Git diff 和原生文件预览暂未提供。

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
| Ctrl+L | 聚焦当前网页窗格地址栏 |
| Ctrl+Shift+W | 确认关闭当前终端 |
| Ctrl+Tab / Ctrl+Shift+Tab | 切换到下一个 / 上一个画面标签 |
| Ctrl+Shift+P | 打开命令面板；搜索分屏、窗格聚焦、移到独立标签、均分、切换会话或设置 |

工作区内 `+` 的弹层不显示分组选择，新终端或网页直接加入该工作区，继承打开位置的目录和分组。通过 Ctrl+Shift+T 或命令面板可新建独立终端；目录分组可从命令面板的“在分组名新建终端…”启动。分组创建、编辑、移除从命令面板进入，已有窗格及其工作目录不受影响。

## 配置

Windows 配置文件位于 `%APPDATA%/cyrene-terminal/settings.json`，其他系统使用 Go 用户配置目录下的 `cyrene-terminal/settings.json`。外观及目录分组可在界面编辑并保存；自定义 CLI 可修改 `profiles`：

```json
{
  "profiles": [
    { "id": "codex", "name": "Codex", "command": ["codex"] },
    { "id": "custom", "name": "My Agent", "command": ["my-agent", "--flag"], "resumeCommand": ["my-agent", "--resume", "{session}"] },
    { "id": "shell", "name": "Terminal" }
  ]
}
```

`id` 必须唯一，省略 `command` 使用默认 shell。手动编辑后，从托盘退出再启动应用。Windows 默认优先选择 PowerShell 7，其次为 Windows PowerShell 和 cmd。

标准 Agent 命令自动提供当前会话的恢复入口；自定义包装器可用 `resumeCommand` 指定恢复命令，`{session}` 替换为该终端绑定的 ID，省略占位符时将 ID 追加为最后一个参数。恢复在原终端标签内启动 Agent，携带原 ID 继续对话；不会终止其他运行中的终端，重开应用时会重启保存的 shell 并按原 ID 恢复已绑定的 Agent。Codex / Claude 的 SessionStart hook 可跟踪其后续会话切换；普通 shell 内手动启动的 Agent 不在此自动绑定范围；普通 shell 内存、后台任务和未绑定的 Agent 无法自动恢复。远程 SSH / WSL 中的 Agent 无法直接读取 Windows 临时图片路径，暂不支持图片跨环境上传。

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
$env:GOARCH = 'amd64'
$env:CGO_ENABLED = '0'
go test ./...
go tool mygo vet .
go tool mygo build -skip-installer
```

Windows 原生交互验收另需 Python 3：

```powershell
go build -tags e2e -o .mygo/cyrene-isolated-smoke.exe ./cmd/cyrene-terminal
python scripts/workbench-smoke.py
python scripts/workspace-restart-smoke.py
```

该脚本使用独立配置与实例，不操作日常会话。结果写入忽略的 `test-results/`。测试覆盖工作区子 tabs、嵌入式网页导航与停靠、混合分屏恢复及 Agent 原会话恢复；验证范围与限制见 [验证记录](docs/validation.md)。

## 许可证

项目代码采用 [MIT](LICENSE)。第三方组件保留各自许可证；文件图标使用 [vscode-icons](assets/file-icons/README.md) 的 CC BY-SA 原始资源，品牌标识保留各自权利；agent 标识的来源与许可见 [assets/agents/README.md](assets/agents/README.md)。随 Windows 包分发的 Chafa 是独立程序，采用 LGPL-3.0-or-later，其许可与对应源码地址保存在 [resources/windows-amd64/tools/chafa](resources/windows-amd64/tools/chafa)。项目 MIT 许可不改变这些组件的许可证或商标权利。

项目地址：[YanChenBai/cyrene-terminal](https://github.com/YanChenBai/cyrene-terminal) · [MyGo 文档](https://mygo.egoist.dev/docs)

应用图标采用 wink 风格的 `>_` 记号，SVG 源文件为 `assets/cyrene.svg`，运行 `go run ./scripts/render-icon` 同步生成应用和托盘图标。
