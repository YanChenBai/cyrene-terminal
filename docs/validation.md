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

## 会话与控件交互调整

- 关闭图标仅在会话行悬停或关闭控件键盘聚焦时绘制，保留布局占位。
- 按钮增加按下反馈；输入框统一为 7 DIP 圆角；字号、背景浓度、玻璃清晰度使用一体式加减按钮组，分段按下变色，外框不缩放。
- 可通过会话右键菜单编辑别名或重新启动；未设置别名时使用当前目录名，跟随 OSC 7 更新。默认 PowerShell 接入会话内目录上报，重启沿用最后上报目录。退出后保留输出，重启按钮置于底部居中的玻璃操作条。
- 模态动画回弹峰值约 1.02，启动菜单以按钮中心为缩放来源。模态输入范围内保留顶部拖动窗口区域。
- Go 测试、MyGo vet、独立 Windows 发布目录构建通过。原生冒烟测试第一次在新终端输入时超时，复跑通过工作目录、标题、设置、会话与托盘生命周期；Kitty 透传仍未通过。通知区菜单鼠标操作与模态打开后的实际桌面拖动尚未进行人工验收。

## 字体与颜色

系统字体由 MyGo FontFamilies 在后台读取；可编辑下拉框支持搜索选择与手动输入。颜色使用原生 ColorWell 并保留 HEX 输入；不完整的 HEX 编辑不会触发解析崩溃。字体选择、自由输入、颜色弹出及目录上报相关测试通过。

最新原生冒烟尝试曾因字体控件由 textField 变为 comboBox 导致测试定位超时，定位已更新；后续复跑在 Chafa 命令输入步骤超时。当前完整原生冒烟未全程通过，不将此前成功记录当作最新端到端结果。Kitty 透传仍未解决。

## Windows 失焦背景

暂保留 MyGo 默认行为，不启用窗口子类兼容适配。MyGo v0.7.4 的 SetActiveAppearance 在 Windows 后端为空操作；失焦背景效果留待后续处理。

## 只读文件树侧栏（2026-11-04）

Git 面板、Git 操作与提交图实现已按当前范围移除。右侧栏改为 MyGo 原生 CollapsibleBase 组成的文件树，目录优先排序、展开时后台读取。文件仅可选中；无打开、编辑或写入动作。符号链接不递归，单层最多显示 2000 项。

图标复用固定版本 vscode-icons 的原始 SVG，来源与 CC BY-SA 声明位于 assets/file-icons/README.md。测试覆盖目录排序、图标映射、文件选中、目录关闭后后代隐藏、重新展开以及侧栏完全收起。

原生 Windows 窗口验收已通过右侧栏开关、目录按需读取、文件只读选中、目录折叠和刷新；结果位于忽略的 test-results/file-tree-native.json，截图为 file-tree-native.png。Go 全部测试、MyGo vet 和 Windows 发布构建通过。


### 文件树滚动与调宽

修正右侧栏外层与内容列的高度约束，滚动范围限制在标题和目录路径下方。右侧左边缘提供无分隔线的 6 DIP 拖拽区域，支持方向键微调，收起再展开保留宽度；两侧同时展开时为终端保留可用空间。

120 文件目录回归测试覆盖滚轮与精确滚动到末尾；另有拖拽调宽与收起后宽度保留测试。隔离原生 Windows 窗口验证滚轮到末尾、边缘拖拽与宽度保留均通过，记录见 test-results/file-tree-native.json，截图为 file-tree-scroll-native.png。Go 全部测试、MyGo vet 通过。
