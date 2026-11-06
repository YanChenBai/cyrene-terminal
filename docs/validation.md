## 标签留白与真实毛玻璃弹层（2026-11-05，当前）

本节取代下方历史记录中的 40/30-DIP 标签尺寸和普通 source-over 模糊材质。

- 标题栏 50 DIP、工作区 34 DIP、标签文字 13 DIP，增加子 tab 内部间距；顶部留白约 9 DIP，与下方内容 gap 平衡。激活态使用较低浓度的透明填充和细高光。浏览器工具栏 44 DIP，地址栏与统一 SVG 控件中心对齐，增加顶部内边距。
- 默认工作区名为 Workspace、Workspace2，索引及递增计数随工作区保存。全局 + 新建显式工作区；普通独立 tab 不显示内部 +，Ctrl+Shift+T 继续独立新增。用户别名优先。
- CPU 和 Windows D3D11 修复模糊合成：过滤结果按覆盖率替换原背景，保留 alpha，不把清晰文字再次混进去。弹层加轻染色、细边缘回光与外阴影。高对比条纹测试中，内部相邻像素对比由旧材质的 13 降至 0，同时保持透明度。维护方法见 [SDK 补丁](mygo-webview-patch.md)。
- 网页在弹层期间用异步当前帧参与模糊，关闭后原 View 回归；原生脚本验证隐藏/恢复、HTTP 请求次数不变及无 GPU shader 编译错误。检查了 test-results/workbench-native-launcher.png、workbench-native-modal.png，网页色块被柔化，文字不会清晰穿过弹层。
- go test ./...、go tool mygo vet .、工作区原生回归与隔离 Agent 退出重开回归通过。命名、独立 tab 添加入口、标签上下留白和浏览器控件中心均有针对性验证；Agent 续聊使用模拟 CLI，不发送真实模型请求。
- 新版输出：build/review-frosted-spacing/windows-amd64/Cyrene Terminal.exe。未终止用户进程，也未覆盖使用中的旧程序。

## 聚焦控件、凸起边缘、组内排序与原生圆角（2026-11-05，当前）

本节取代下方历史记录中的“平整窗格”“隐藏关闭按钮”“黑色圆角 backing”和“缺 ID 必须手填”等描述。

- 保留系统 Acrylic 毛玻璃，不使用液态折射。工作区选中态及聚焦窗格增强边缘/底色；窗格底边增加回光和内外阴影，中央透明度仍由配置决定。标题调整为 13 DIP，显示已提供的 Agent 图标；没有资源时不补虚构图标。
- 窗格操作按钮在聚焦或悬停时通过 170ms 宽度/透明度过渡显示，离开且未聚焦后动画隐藏。工作区及子 tab 的关闭按钮常显、各自占位，不嵌套在选择按钮中；真实可访问性树验证子 tab 关闭动作独立可用。设置左边增加全局新工作区 +，组内 + 保留当前组作用域。
- 子 tab 拖动排序独立于空间布局，并写入工作区快照。拖外部窗格或整个工作区进入组内会合并；整组子布局、终端/网页对象和会话所属目录/分组保留。组外侧 12 DIP 仍用于整组排序，空白栏用于拆回独立工作区。选中子 tab 的高亮有过渡；新增/移除窗格用匹配几何过渡，测试采样 60ms 中间尺寸，确认没有直接跳到终点。
- Windows 网页 visual 直接使用 DirectComposition 圆角裁切，去掉黑色底板。半径同时用于原生命中检测，支持 padding/border 归一化及内容缩放。补丁固定在 vendored MyGo v0.7.4，见 [补丁维护说明](mygo-webview-patch.md)。补丁在原版源码上的正向 check、当前 vendor 的反向 check 均通过。
- 已知 ID 的 Agent 自动精确恢复。Codex/Claude 缺关联时改为原窗格内的原生历史列表，不要求粘贴 ID，不自动选择同目录的最新会话。Claude 新增 per-launch SessionStart 捕获并保留显式 settings/原有 hooks；退出时再读一次记录，减少最后一刻写入的 ID 被遗漏。未保存关联的旧记录无法可靠反推原会话，需要在列表中选一次；普通 shell 内手动启动的 Agent 仍不自动绑定。
- `GOARCH=amd64 CGO_ENABLED=0 go test ./...`、`go tool mygo vet .` 和 `go tool mygo build -skip-installer -o build/review-frosted-focus` 通过。PowerShell hook 实际执行测试通过，不调用模型。
- `scripts/workbench-smoke.py` 原生回归通过：组内拖动、跨组加入、可访问子 tab 关闭、混合布局重开、网页导航、原网页停靠不重载、全局/局部新增作用域。可选 `CYRENE_CAPTURE_NATIVE=1 uv run --with pillow python scripts/workbench-smoke.py` 真实合成截图通过：角落 RGB=(36,37,38)，与相邻窗格完全一致；页面内部 RGB=(220,211,238)，没有黑色楔角。已检查 `test-results/workbench-native-corners.png`。
- `scripts/workspace-restart-smoke.py` 原生退出重开通过：使用隔离模拟 Codex/Claude CLI，原 ID、对话记忆标记、布局与别名保留，进程 PID 更新；没有发送真实模型请求。这不等于所有 Agent 版本或被禁用的 hooks 均已实测。
- 新版单独输出至 `build/review-frosted-focus/windows-amd64/Cyrene Terminal.exe`，不替换或终止用户正在运行的实例。重新启动新版前需从托盘退出旧版，以释放单实例锁。

# 验证记录

## 普通毛玻璃，移除液态效果（2026-11-05）

当前设计改为普通毛玻璃。窗口明确使用 VibrancyAcrylic；窗格和 tabs 仅保留平整半透明填色，移除倒角高光、渐变亮边及内阴影。透明半 DIP 边框保留原几何。命令面板、启动弹层、对话框和字体列表统一改为普通 backdrop blur + 中性底色，无 glass.Glass 折射或光圈。

保持背景 alpha、网页圆角遮罩、工作区内添加和连续 tabs 动画。原有 alpha 测试追加平面像素一致性断言；执行全量 Go 测试、MyGo vet、workbench 原生回归及独立发布构建。下方倒角记录为历史方案。

## 玻璃倒角、通透度及工作区内添加（2026-11-05）

- 标题栏 40 DIP、胶囊 30 DIP，上下保留约 5 DIP；移除标题栏下方额外 padding，tabs 到内容约 8 DIP。选中组内的圆形 + 为 24 DIP、半径 12。
- 玻璃倒角使用半 DIP 渐变亮边与上下浅内阴影。取消每窗格 backdrop blur 与全窗额外染色，避免重复采样已透明的背景。直接材质和完整 application.view（Vibrancy=true）的 alpha 测试均检查 40% 背景仍在约 40% 范围内，而不会叠成厚底。
- Windows 原生 WebView 为矩形合成层。网页视口内缩 4 DIP、页面半径 10 DIP，在 portal 后绘制实色 backing；页面内部由 hole 清空，只有圆角切口保持不透明，避免白页角落穿过透明 UI。测试检查 portal 中央 alpha=0、角落 alpha=255、周围窗格仍为约 40%。
- 添加弹层移除分组选择和目录行。保存打开按钮的窗格 ID，选择终端或空白网页后加入其工作区并继承目录/分组；即便焦点切换仍保持原目标。组启动功能移到命令面板。新分屏按目标窗格较宽/较高选择左右或上下方向。
- TestWorkspaceOwnsAddButtonWithTopBreathingRoom 同时覆盖按钮位置、紧凑间距、无分组选择、跨焦点终端与网页添加。原生 workbench-smoke.py 另验收同工作区新增真实终端、空白网页及工作区数量不变；仍运行 Agent 退出重开回归。
- 旧版截图库与下方记录是历史设计；当前 UI 渲染帧为 internal/app/test-results/glass-inset-tabs-launcher.png，原生结果保存在 test-results/workbench.json。桌面壁纸的实际色彩与系统合成有关，headless 截图只验证应用内容和 alpha，不模拟壁纸透射。

## 连续玻璃表面、tabs 动画与空白网页（2026-11-05）

- 移除顶部品牌名及命令面板独立图标，命令仍通过 Ctrl+Shift+P 调用。窗格与 tabs 使用无亮边的 backdrop blur 加柔和渐变，保留透明背景及用户颜色设置；不再用持续描边强调分屏。
- 子 tabs 宽度按 MyGo 原生字体度量计算，通过无回弹的弹簧和透明度连续展开/收起，支持反向切换并保留当前姿态与速度。测试覆盖中间帧、反向切换、最终宽度和自动滚动。
- 新建网页与分屏新建网页直接创建空状态，自动聚焦地址栏，无模态框、无初始网络请求、无 shell 进程。输入网址回车后才创建 WebView。显式保存 Browser 标记，空网页可跨重启恢复。
- 去掉浏览窗格的边框内缩，外层圆角与原生 WebView 视口统一为 12 DIP，并增加独立的视口裁剪；弱化地址栏边框。
- 原生验收入口仍为 scripts/workbench-smoke.py 和 scripts/workspace-restart-smoke.py；网页、分屏、停靠及 Agent 精确会话恢复的验证边界沿用下节。截图位于 internal/app/test-results/glass-workspace-subtabs.png，以及 test-results/workbench-browser.png；网页独立合成层依然需要单独 CapturePage，窗口 capture 不代表真实桌面壁纸透射效果。

## 无侧栏工作区、子 tabs 与网页窗格（2026-11-05）

本节为当前设计；下方侧栏及直角标签记录仅保留为历史验收，不代表当前 UI。旧 scripts/native-smoke.py 使用历史侧栏选择器，当前入口改为 scripts/workbench-smoke.py 和 scripts/workspace-restart-smoke.py。

- 移除左右侧栏和开关；分组绑定通过启动菜单选择、命令面板管理。工作区采用圆角透明分组，只有当前多窗格工作区展开全部子 tabs；子 tab 聚焦原窗格，支持拖出及停靠。保留工作区整体排序、独立别名、浮动关闭按钮及标签横向滚动。
- 终端与 MyGo 原生嵌入 WebView 共享窗格模型。网页支持 HTTP(S)/localhost、地址栏、前进、后退、刷新和 Ctrl+L。Untrusted 页面无应用桥接；网页新建不启动 shell。切换、停靠保留 WebView；关闭只销毁目标页面，退出重开重新加载保存的网址。
- Go 测试覆盖子标签展开/收起/聚焦、18 个子标签的滚动与自动显现、旧侧栏状态兼容、网址验证、混合布局持久化和启动中网页关闭。保留既有终端生命周期、图片粘贴、Agent 恢复和拖拽测试；删除已移除侧栏功能的 UI 断言。
- scripts/workbench-smoke.py 使用真实 Windows 窗口、独立配置、普通 PowerShell 与本机 HTTP fixture，验证网页 JavaScript 执行、标题、前进/后退/刷新、子 tabs 切换、拖出再停靠不发起额外加载、混合工作区重开及关闭网页保留 shell。报告：test-results/workbench.json。
- scripts/workspace-restart-smoke.py 再次通过退出重开后的 Codex/Claude 精确 ID、对话记忆标记及布局恢复；使用替身验证调用契约，无真实模型请求。
- 原生 UI capture 不包含嵌入网页的独立合成层；网页像素另通过 e2e 专用 WebView.CapturePage 保存。自动化实例在屏幕外，不使用桌面截图作为页面绘制证据。网页状态跨窗口隐藏保留；跨应用退出仅恢复 URL，不保证 DOM、表单或历史。
- 未重新执行旧侧栏版本的全量原生脚本、跨平台及屏幕阅读器验收。不新增 CLI 终端控制接口。真实 Agent ID hook 的先前独立验证范围见下方恢复记录。

## 退出重开恢复、标签溢出和 wink 图标（2026-11-05）

- 退出应用前在 UI 线程快照工作区，在后台原子写入 workspace.json；保存失败时保留程序和终端并显示原因。重开后还原目录、分组、会话/标签别名、标签顺序、分屏比例、焦点和侧栏状态。普通 shell 重新启动；不恢复已经结束的后台进程或 shell 内存状态。
- Claude 新建时分配 UUID，Codex 通过每窗格的 SessionStart hook 记录 ID，保留 CLI 自身的 hook 信任审核；不更改全局配置、Agent 身份或权限。仅按原 ID 恢复，缺失时显式绑定。旧版本已经丢失的会话关联无法追溯自动恢复。
- tabs 使用受限的横向滚动容器，保留右文件栏及窗口工具区宽度；35 标签测试覆盖普通滚轮、横向滚动和聚焦标签自动显现。
- scripts/workspace-restart-smoke.py 使用隔离 Agent CLI 替身做真实 Windows 退出重开，验证原会话 ID、会话记忆标记、别名、分屏与焦点保留。test-results/workspace-restart.json 记录通过；替身验证 CLI 调用契约，不代表真实模型完成了续聊。
- 实装 Codex CLI 0.166.0 通过本地无认证测试 provider 验证 SessionStart ID 记录与相同 ID 的 exec resume；仅连接本地拒绝端口，不调用真实模型。报告 test-results/codex-real-resume.json。生产保留首次 hook 信任提示；没有验证真实 Claude 模型对话。
- wink/terminal >_ SVG 已同步生成应用和托盘 PNG。分割线替代方案仅讨论，当前区域边框样式未改动。

## 紧凑直角标签、拖回标签与聚焦命名（2026-11-04）

- 按 Zed 参考改为直角标签；左右侧栏内容边界与中间标题栏底部使用 1-DIP、6.5% 淡色边框，侧栏顶部不画横线；左右侧栏各自从顶部到底部共用一层半透明背景；中间标题栏保持根节点背景。移除标签和关闭按钮的不透明底色，避免亚克力背景下的黑块；悬停时只裁剪关闭图标下的标签文本，不改变标签宽度。保留侧栏零宽收起和拖动热区。
- 顶部标签改为 32 DIP 高、上下贴齐、四角为直角、零间隔。关闭按钮悬浮覆盖，不占常态宽度。恢复分屏终端的细边框与圆角；在局部主题中隐藏 MyGo 原生分隔线的常态和悬停绘制，同时保留原生拖动/键盘调整热区，恢复子窗格的正常主题。
- 窗格标题拖到顶部空白处或标签边缘会拆回独立标签；顶部标签拖动只改变画面顺序，完整保留多窗格树、原画面 ID、名称、终端和分组。插入位置有临时指示。不存在的目标不会改变布局。
- 标签右键可设置独立名称，Ctrl+Shift+P 增加“命名当前标签…”和“设置当前会话别名…”。两个命令只命名聚焦对象，不显示选择其他对象的列表，也不批量改名。标签名称不随其内部窗格焦点改变；会话名称不改写标签自定义名称。保存/取消后请求恢复当前终端输入焦点。OS 窗口标题统一由标题栏计算，避免会话 OSC 标题或改名覆盖画面名称。
- 新增交互测试通过：拖回顶部、拖动排序、整个多窗格画面重排保持身份、命令只命名聚焦对象、两个名称相互独立、键盘焦点与额外帧、标签上下贴齐且相邻间距为零、分隔线像素不绘制但仍可拖动调整。常规 Go 全部测试和 MyGo vet 通过；真实 Windows 冒烟已通过命令改名、拖回顶部、标签排序和操作前后 shell PID 一致。
- 本轮新截图为忽略的 `test-results/tabs-08-compact-rounded.png` 和 `tabs-09-compact-reordered.png`。分屏终端保留圆角边框，标签之间不加分割线；未分组会话不显示分组占位。该早期样式回归尚未包括跨应用退出持久化；后续恢复验证见上方记录。
- 本轮首次原生回归在 Beta 终端命令写文件步骤超时；重跑含剪贴板与分屏的完整回归通过。最终完整原生回归也校验了命名后 OS 窗口标题仍保持画面自定义名。标准输出目录因运行中的程序占用而不能替换，使用 `go tool mygo build -skip-installer -o build/review-tabs-compact` 构建独立验收目录，不结束当前用户终端。

## 分屏、顶部标签与命令面板（2026-11-04）

- 会话继续拥有各自的终端进程、分组、目录和别名；画面单独持有布局引用。可跨分组拖入已有终端、在同一画面重排、移回独立标签，关闭一个窗格后自动折叠空分支。布局调整不重新启动进程。
- 标签合并到 32-DIP 标题栏，没有重复标题行或标签新增按钮。整张标签共享悬停背景，关闭区域没有独立底色；关闭图标在悬停或键盘聚焦时显示，悬浮覆盖，不保留固定占位。鼠标拖拽预览显示将占用的半屏，只有确认关闭才结束进程。
- Ctrl+Shift+P 命令面板支持中文和英文关键词搜索、方向键、Enter、Esc；包含四向分屏、跨分组选择会话、在目标目录新建终端、切换会话、窗格聚焦、移回独立标签、均分、侧栏、外观设置及关闭确认。屏幕空间不足时不继续细分。
- `GOARCH=amd64 CGO_ENABLED=0 go test ./...`、新增分隔线拖动测试、`go tool mygo vet .` 和 `go tool mygo build -skip-installer` 通过。测试覆盖同画面重排后每个会话只出现一次、跨组身份保留、关闭窗格和关闭画面的不同确认范围、取消拖拽、键盘搜索/选择/分屏/拆回、目标目录继承、整行悬停像素一致性，以及 720×440 下命令面板布局和 Escape。
- `CYRENE_SMOKE_CLIPBOARD=1 CYRENE_SMOKE_TILING=1 python scripts/native-smoke.py` 通过真实 Windows 窗口验收：跨分组拖入、分隔线调整、拖拽前后 shell PID 一致、键盘创建第三个终端且工作目录正确、移回独立标签、既有粘贴/目录/设置/托盘/关闭进程生命周期。首次原生脚本仍查找旧的重复标题，后续分隔线步骤缺少绘制帧，已分别改为检查标题栏标签和在拖动事件间等待帧；最终完整复跑通过。
- 已逐张检查本轮 `test-results/tiling-01-titlebar.png`、`tiling-02-drop-preview.png`、`tiling-03-command-palette.png`、`tiling-04-three-panes.png`、`tiling-05-detached.png`，未发现重叠或遮挡；步骤审查记录在忽略的 `test-results/tiling-ui-audit.md`。最小窗口和悬停状态另由 MyGo Tester 渲染检查。未做屏幕阅读器和其他操作系统的实机验收，不声称完整可访问性认证。
- Git diff / 文件预览仍未接入。该早期回归不包括布局跨退出保存，现已由上方工作区恢复实现补齐。Agent 通过 CLI 列出、读取或发送终端输入的方案仅记于 `terminal-cli-proposal.md`，没有启用生产控制接口。Kitty 经 Windows ConPTY 图形透传的既有限制未改变。

## 图片粘贴、Agent 恢复与中间标签（2026-11-04）

- 剪贴板 PNG 在后台写入每终端独立临时目录，通过文件列表和路径文本提供给 Agent。支持 Ctrl+V / Ctrl+Shift+V，普通文本保留 MyGo 默认粘贴。关闭或重启终端清理临时目录，过期任务不会向替换后的终端写入。
- `+` 启动器只新增。恢复位于已退出 Agent 会话的操作条和该会话右键菜单，明确携带该终端绑定的 ID：Codex `resume ID`，Claude / OMP `--resume ID`，Pi / OpenCode `--session ID`。新建 Claude / Pi 时分配独立 UUID；Codex 正常退出时解析自身恢复提示。无法取得 ID 时显示原会话 ID 绑定弹框，不回退到最近会话或全局选择器。自定义包装器通过含 `{session}` 的 `resumeCommand` 配置。
- 中间标签栏与左侧会话共享终端；增加、切换、关闭确认和独立按钮来源的启动弹层已接入。当前只承载终端，Git diff / 文件预览留待后续实现。
- Windows amd64 下 `CGO_ENABLED=0 go test ./...`、`go tool mygo vet .`、`go tool mygo build -skip-installer` 通过。测试覆盖恢复命令与目录继承、标签切换不新建进程、关闭确认、PNG 保留和清理以及保存/关闭并发。
- `CYRENE_SMOKE_CLIPBOARD=1 python scripts/native-smoke.py` 通过真实 Windows 位图剪贴板 → Ctrl+V → 终端可读 PNG 路径，以及中间标签切换和既有分组、设置、托盘、进程生命周期验收；测试结束恢复原剪贴板。结果见忽略的 `test-results/native-smoke.json`，截图见 `launcher.png` 和 `workspace.png`。
- Agent 恢复参数已对照本机各 CLI 的 `--help`；测试覆盖同目录不同会话独立 ID、原标签恢复不替换身份、缺少 ID 时必须绑定、启动菜单没有恢复入口。未提交真实模型请求，也未逐一验收每个 Agent 的附图显示或恢复后对话。尚不自动追踪 Agent 内部会话切换或普通 shell 中手动启动的 Agent，该早期阶段会话绑定未跨退出保存，现已由工作区记录补齐。路径兼容以现有本地拖入文件语义为基础，不支持跨 SSH / WSL 自动上传。Kitty 经 Windows ConPTY 图形序列透传仍为已知未通过项。
- `go test -race` 尝试被本机 Clang / MSVC 链接器的 LNK1143 阻止，未取得 race 通过结果；并发保存/关闭的普通回归测试通过。

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
