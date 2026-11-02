# 验证记录

2026-11-03，cyrene-terminal 0.3.0，Windows amd64，MyGo 0.7.4，Go 1.27.3。

## 已通过

- `CGO_ENABLED=0 go test ./...`：配置保存与校验，分组持久化/编辑/移除，用户目录与分组目录规则，菜单位于按钮下方，最小窗口键盘导航，hover 渐变与恢复，侧栏顶部对齐及完全隐藏。
- 弹窗逐帧测试：源位置等于点击控件的已提交边界，从该位置展开，240ms 到 1.1 倍，440ms 已回到 1.0；关闭期间保留绘制，结束后卸载，恢复输入/焦点。下拉测试覆盖回弹、Escape、点击外部、关闭中重新打开时的位置连续性。
- `go tool mygo vet .`：元素生命周期与协程捕获检查。
- `go tool mygo build -skip-installer -o build/v0.3.0`：原生发布版与 Ghostty DLL/图标打包。
- `python scripts/native-smoke.py`：真实原生窗口和 PowerShell PTY，未分组进入用户主目录，两个分组分别继承绑定目录，OSC 标题同步到侧栏/顶栏/系统窗口，切换 tab 保留会话，按钮下方菜单，无分屏与独立后台会话入口，侧栏完全隐藏且折叠按钮位于顶栏左侧，设置保存，分组绑定持久化。
- 关闭确认取消保留会话；确认关闭 tab 后对应 shell 实际退出。系统 WM_CLOSE 隐藏窗口后 shell 继续执行；第二次启动恢复原窗口及已有会话；托盘退出处理函数结束应用并确认 shell 退出。
- 查看了原生窗口截图中的 32-DIP 顶栏、齿轮图标、侧栏顶部排列，以及确定性动画截图中的点击来源、1.1 倍峰值和恢复状态。

## Kitty 图形协议

`TestKittyGraphicsDrawDeleteAndTabSwitch` 通过：实际 Ghostty/MyGo 渲染器读取 Kitty APC RGBA 数据，在终端区域绘制超过 1000 个目标颜色像素；切换 tab 不泄漏图片，切回后保留，Kitty 删除命令清除图片。

本机 PowerShell → 系统 ConPTY → 终端的同类图片测试未显示图片。报告中的 `kitty_graphics_through_real_windows_pty` 和依赖该图片的删除验证为 false。这是已知未通过项，不能把本版描述为 Windows 本地 Kitty 图形协议完整支持。需要继续处理 PTY 对图形序列的透传。

## 验收隔离与证据

冒烟测试使用临时 APPDATA、普通无 profile 的 PowerShell、独立单实例标识，窗口移到屏幕外，避免手动操作影响脚本；不使用用户 agent 认证或配置。脚本等待 shell 提示符就绪才输入命令，不以光标停止闪烁作为动画完成条件。

- `test-results/native-smoke.json`：原生端到端结果，明确包含 Kitty 未通过项。
- `test-results/workspace.png`、`sidebar-collapsed.png`、`launcher.png`、`close-confirmation.png`、`restored.png`：实际窗口内容截图。
- `test-results/dialog-origin.png`、`dialog-expanding.png`、`dialog-overshoot.png`、`dialog-settled.png`、`dialog-closing.png`、`menu-overshoot.png`：`CYRENE_CAPTURE=1 go test ./...` 生成的动画帧。
- 托盘退出通过 e2e 构建暴露的同一原生菜单回调验收；没有完成 Windows 通知区图标右键选项的鼠标点击验收。生产构建不暴露测试菜单或自动化 socket。

机器限制：之前尝试 `CGO_ENABLED=1 CC=clang go test -race ./...`，Windows Clang/MSVC 链接报 LNK1143（COMDAT），没有获得 race detector 的运行结果。无 cgo 测试与发布构建通过。

未覆盖：各 agent CLI 的完整交互，系统文件夹选择器的实际选取流程，macOS/Linux 实机效果，系统重启后的会话恢复（不属于本版功能）。内容截图不包含桌面合成器 Acrylic；原生窗口的外观只做目视检查，未做像素级系统材质验证。

发布目录：`build/v0.3.0/windows-amd64/`；稳定入口：`build/windows-amd64/`。程序和同目录 DLL、图标一起分发。

## 0.4.0 交互与预览复验

- 弹出动作改用连续阻尼弹簧，模态框位移和缩放同步；约 1.1 倍峰值后自然收敛，520ms 后规范为终值。菜单同时缩放和短距离位移；保留中途反向的速度。新增逐 10ms 采样验证位移/缩放一致性。
- 菜单圆角 18 DIP，模态框圆角 22 DIP。侧栏关闭按钮去掉独立 hover 背景。
- 侧栏背景贯穿顶栏到底部，去掉分隔线，右缘支持拖动宽度；测试从 244 拖至约 330 DIP，折叠/展开后保持。折叠后隐藏创建按钮，Ctrl+Shift+T 仍可打开启动菜单。
- 侧栏移除文件夹和终端条目图标。分组竖线对齐折叠箭头中心；几何测试检查箭头到文字恰为 7 DIP，展开/折叠不改变文字 X 坐标。截图 `test-results/group-tree.png`。
- Agent Logo 只出现在新建菜单，SVG 彩色渲染已目视检查。截图 `test-results/agent-logos.png`。
- 本机安装官方 Chafa 1.18.3 便携版，路径仅注入 Cyrene 子终端，不改变全局 PATH/Yazi 配置。原生冒烟报告 `yazi_chafa_available=true`、`yazi_ansi_image_preview=true`；使用真实 Yazi 打开红/青测试图，各颜色在实际截图中超过 1000 像素（容许 2 级合成舍入），截图 `test-results/yazi-preview.png`。这是 Chafa ANSI 预览，不是 Kitty 原生图片透传。
- PowerShell/ConPTY 的 Kitty 图片测试仍为 false；保留原有限制说明。

本次发布目录为 `build/v0.4.0/windows-amd64/`，稳定入口仍为 `build/windows-amd64/`。上面的 0.3.0 记录保留为上次验收历史；其旧版两段动画时刻不再描述当前实现。

## 0.4.1 样式、动画与 PowerShell 7 复验

- 全部文本输入框调整到 36 DIP；Logo 等宽占位，并按品牌轮廓分别调整实际绘制尺寸。菜单圆角从 18 收至 12 DIP，模态框从 22 收至 14 DIP。
- 弹簧频率从 30 调至 21，开场与回落节奏放慢约 30%，740ms 后归位；退出淡出 220ms。侧栏改用独立的 380ms 显式过渡，不再依赖系统可能直接跳过的自动控件动画。关闭/展开中间帧测试通过，截图 sidebar-closing.png / sidebar-opening.png。
- 修复 Windows 的 Path 大小写兼容：原先追加 PATH 可能与继承的 Path 并存；现在匹配原键进行替换。回归测试覆盖 Path 混合大小写。
- Windows 发布包直接携带官方 Chafa 1.18.3、许可与源码链接，优先从程序旁 tools/chafa 查找。独立原生测试采用 pwsh.exe（PowerShell 7）、混合大小写 Path，并把 LOCALAPPDATA 指向空目录，验证不依赖之前手动安装的 Chafa。Yazi ANSI 红/青图像预览实际通过；原有分组、拖宽、托盘生命周期测试也通过。
- go test ./...（CGO_ENABLED=0）和 go tool mygo vet . 通过。发布目录 build/v0.4.1/windows-amd64。
- 旧版运行中的终端不会自动更新环境；需要从托盘退出旧进程后启动新版。Kitty 原生图片透传仍是已知未通过项。

## 0.4.2 启动与交互复验

- 标题与工具按钮之间增加 16 DIP 外边距，几何测试确认标题不贴加号。
- 设置和分组编辑统一“取消 / 保存”；两者高 34 DIP，保存采用高对比实底，取消采用次级描边。
- 分组列表以 180ms 高度动画展开/收起，箭头同步转动，退出期间保留内容并屏蔽交互；测试检查中间高度与相邻分组位置连续变化。侧栏由 380ms 缩至 220ms，采用前快后缓曲线。
- 失败状态改为终端区域居中的 Liquid Glass 卡片，重试按钮位于卡片右下；点击后刷新环境重新启动。截图 startup-error.png。
- Windows 每次新建/重试从 HKCU/HKLM 读取 PATH，合并到当前进程环境；命令定位与子进程使用同一环境，保留 Path 键大小写。默认 shell 显式定位，npm cmd/ps1 入口经 PowerShell 引用启动。
- CYRENE_LIVE_CLI=1 go test -run TestInstalledCommandLaunchWithStaleParentPath -v .：把继承 PATH 限制到 System32 后，实际启动 Codex 0.166.0、Claude 2.1.309、Pi 1.0.0、OMP 18.14.0、OpenCode 2.1.4、oh-my-posh、winget 的版本命令，全部通过；实际 PowerShell 用户 profile 加载后 $Error.Count=0 并输出成功标记。结果 test-results/cli-launch.json。该检查不进行 agent 登录或完整对话。
- CGO_ENABLED=0 go test ./...、go tool mygo vet .、原生 PowerShell 7 冒烟通过；Yazi ANSI 预览及托盘生命周期复验通过。Kitty 原生图片透传仍未通过。
- 发布目录 build/v0.4.2/windows-amd64。升级要从托盘退出旧版后启动新版；已有进程不会热更新。
