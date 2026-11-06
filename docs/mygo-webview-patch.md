# MyGo Windows 网页圆角补丁

MyGo 固定为 `v0.7.4`，项目通过 Go 标准 `vendor/` 构建；保留上游 MIT LICENSE 和其他依赖的授权文件，不修改模块缓存。

上游网页在 UI 中挖出圆角 portal，但 Windows DirectComposition 页面仍是矩形。用黑色底板遮住圆角会破坏透明背景。本补丁把内容框四角半径传到原生布局、缩放及命中检测，并为实际网页 visual 设置 `IDCompositionRectangleClip`。网页滚动条也受此裁切，不再绘制实色角落补丁。半径与 UI 使用相同的归一化及边框、padding 扣减。

Windows COM 调用使用正确的 float32 ABI。MSVC 对同名重载的虚表顺序与头文件声明顺序相反，矩形裁切的 float setter 位于 animation setter 之后；不能直接按声明位置调用。裁切宽高按与 controller 相同的物理像素边界计算，避免 DPI 舍入产生一像素缝隙。关闭时释放 clip 引用。

最小变更保存在 [补丁文件](patches/mygo-v0.7.4-rounded-webview.patch)。刷新依赖时运行：

```powershell
./scripts/refresh-vendor.ps1
```

脚本重新生成 vendor 后应用补丁；版本变化导致上下文不匹配时直接失败，需要重新检查上游实现。仅执行 `go mod vendor` 会丢失本地补丁。

验证包括四角命中检测、带 padding/border 的半径、透明度测试和真实 Windows 页面。原生验收使用隔离实例及 localhost，不访问外部网站：

```powershell
$env:GOARCH='amd64'; $env:CGO_ENABLED='0'
go build -tags e2e -o .mygo/cyrene-isolated-smoke.exe ./cmd/cyrene-terminal
$env:CYRENE_CAPTURE_NATIVE='1'
uv run --with pillow python scripts/workbench-smoke.py
```

该可选模式短暂显示测试窗口，保持不激活，截图后恢复离屏位置。`test-results/native-corners.json` 记录网页圆角、相邻窗格和页面内部的像素；`workbench-native-corners.png` 包含实际网页合成结果。框架普通 UI capture 不包含独立的网页合成层，不能单凭它判断原生圆角。当前实机验证限于 Windows；不宣称其他平台已经实测。

## 透明背景的模糊合成

另一个最小补丁位于 [frosted-overlays](patches/mygo-v0.7.4-frosted-overlays.patch)。原有 blur 把经过模糊的半透明背景再次 source-over 到清晰原图，会残留锐利文字并累计 alpha。本地 `glass.Blur.Replace` 用形状覆盖率替换背景样本，保留过滤后的 alpha；仅用于无渐变 mask 的均匀模糊。

CPU 包含普通及变换路径；Windows D3D11 使用已有的双源混合，第二输出携带形状覆盖率，而不是经过模糊的像素 alpha。此模式目前只实现并验证 CPU / Windows D3D11，不声明 Metal / OpenGL 已支持。正常 blur 的默认行为不变。新增 HLSL 分支使用独立宏，保留 LF；shader 源码分段依赖 LF 标记，CRLF 会使 effect 定义顺序错误。原生脚本会拒绝 GPU effect 编译错误。

原生网页不在 UI 绘制列表中。弹层期间异步捕获当前页面，临时把画面作为 UI 图片绘制以参与同一模糊；原 WebView 保持加载状态。弹层退出动画结束后切回原 View，不导航、不重载。关闭、会话切换及快速重新打开时用 generation 和 View 身份检查拒绝过期结果。捕获失败保留原网页作为回退，不阻塞操作。

`refresh-vendor.ps1` 会依次重新应用两份补丁。回归覆盖高对比条纹不再清晰透过弹层、透明度不被叠成实心、原生菜单/模态框截图及弹层关闭后页面未重载。
