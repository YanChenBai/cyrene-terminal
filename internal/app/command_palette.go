package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

type paletteState struct {
	Open             bool
	Query, LastQuery string
	Selected         int
	Stage            string
	Target           int
	Edge             splitEdge
	Motion           overlayMotion
	Scroll           ui.ScrollState
}

type paletteCommand struct {
	ID, Label, Detail, Keywords string
	Run                         func()
}

func (a *application) openCommandPalette() {
	open := !a.palette.Open
	a.palette.Open, a.palette.Stage, a.palette.Query, a.palette.LastQuery, a.palette.Selected = open, "", "", "", 0
	a.palette.Scroll.Y = 0
	a.launcher = false
}

func (a *application) openSplitPalette(target int, edge splitEdge) {
	if t := a.work.sessionForPane(target); t != nil {
		a.selectTab(t)
	} else {
		return
	}
	a.palette.Open, a.palette.Stage, a.palette.Target, a.palette.Edge = true, "split", target, edge
	a.palette.Query, a.palette.LastQuery, a.palette.Selected, a.palette.Scroll.Y = "", "", 0, 0
	a.launcher = false
}

func (a *application) closePalette() {
	a.palette.Open = false
	a.focusActiveTerminal()
}

func (a *application) focusActiveTerminal() {
	if t := a.work.active(); t != nil {
		if n := t.Root.find(t.Focused); n != nil {
			n.Pane.Handle.Focus()
		}
	}
}

func (a *application) paletteCommands(c *ui.Context) []paletteCommand {
	var commands []paletteCommand
	add := func(id, label, detail, keywords string, run func()) {
		commands = append(commands, paletteCommand{id, label, detail, keywords, run})
	}
	if a.palette.Stage == "split" || a.palette.Stage == "switch" {
		split := a.palette.Stage == "split"
		target, edge := a.palette.Target, a.palette.Edge
		for _, t := range a.work.Tabs {
			if split && t.Focused == target {
				continue
			}
			label := t.label()
			detail := t.Name
			if group := a.groupLabel(t.GroupID); group != "" {
				detail = group + " · " + detail
			}
			add(fmt.Sprintf("session-%d", t.ID), label, detail, "terminal session switch move existing", func() {
				if split {
					if !a.splitFits(c, target, edge) {
						return
					}
					a.dockPane(t.Focused, target, edge)
				} else {
					a.selectTab(t)
				}
				a.closePalette()
			})
		}
		if split {
			add("new-web", "新建网页…", "使用目标窗格的目录和分组", "new create browser web url 网页", func() {
				if !a.splitFits(c, target, edge) {
					return
				}
				a.closePalette()
				a.newBrowser(target, edge)
			})
			// A plain terminal is first among new choices, independently of the
			// launcher's agent ordering. New splits inherit the target's directory.
			profiles := append([]profile(nil), a.config.Profiles...)
			for _, p := range profiles {
				if p.ID == "shell" {
					a.addNewSplitCommand(&commands, c, p, target, edge)
				}
			}
			for _, p := range profiles {
				if p.ID != "shell" {
					a.addNewSplitCommand(&commands, c, p, target, edge)
				}
			}
		}
		return filterPalette(commands, a.palette.Query)
	}
	if t := a.work.active(); t != nil {
		for _, edge := range []splitEdge{splitRight, splitBottom, splitLeft, splitTop} {
			keywords := map[splitEdge]string{splitRight: "split right horizontal", splitLeft: "split left horizontal", splitTop: "split up above vertical", splitBottom: "split down below vertical"}[edge]
			add(fmt.Sprintf("split-%d", edge), edge.label()+"…", "选择已有窗格或新建终端、网页", keywords, func() { a.openSplitPalette(t.Focused, edge) })
		}
		add("switch", "切换到窗格…", "所有工作区的终端和网页", "switch terminal session", func() {
			a.palette.Stage, a.palette.Query, a.palette.LastQuery, a.palette.Selected, a.palette.Scroll.Y = "switch", "", "", 0, 0
		})
		if s := a.work.activeScreen(); s != nil && len(s.Root.leaves()) > 1 {
			add("next-pane", "聚焦下一个窗格", "当前画面", "focus next pane", func() { a.cyclePane(1); a.closePalette() })
			add("previous-pane", "聚焦上一个窗格", "当前画面", "focus previous pane", func() { a.cyclePane(-1); a.closePalette() })
			add("detach", "将当前窗格移到独立标签", "保留终端进程", "detach pane unsplit tab", func() { a.work.detach(t.Focused); a.closePalette() })
			add("balance", "均分当前画面", "重置所有分隔线比例", "balance equal resize panes", func() { balanceLayout(s.Root); a.closePalette() })
		}
		add("rename-screen", "命名当前标签…", "仅修改聚焦画面的名称", "rename name tab screen alias 标签 别名", func() {
			a.closePalette()
			if s := a.work.activeScreen(); s != nil {
				a.editScreenAlias(s, s.Select.Bounds(c))
			}
		})
		add("rename-session", "设置当前会话别名…", "仅修改聚焦终端的别名", "rename name session terminal alias 会话 别名", func() {
			a.closePalette()
			if focused := a.work.active(); focused != nil {
				a.editSessionAlias(focused, focused.Select.Bounds(c))
			}
		})
		add("close", "关闭当前窗格…", "终端需确认，网页直接关闭", "close terminal pane", func() { a.closePalette(); a.requestStop(t, t.Select.Bounds(c)) })
	}
	add("new", "新建终端…", "打开启动菜单", "new terminal create", func() { a.closePalette(); a.openLauncher("") })
	add("web", "打开网页…", "在工作区中浏览网页或 localhost", "new open web browser url localhost 网页 网站", func() { a.closePalette(); a.newBrowser(0, splitRight) })
	add("new-group", "新建文件夹分组…", "绑定启动目录", "new create folder group project", func() { a.closePalette(); a.modalOrigin = ui.Rect{}; a.editGroup(nil) })
	for _, group := range a.config.Groups {
		add("launch-group-"+group.ID, "在 "+group.Name+" 新建终端…", group.Directory, "new terminal folder group", func() { a.closePalette(); a.openLauncher(group.ID) })
		add("edit-group-"+group.ID, "编辑分组 "+group.Name+"…", group.Directory, "edit folder group directory", func() { a.closePalette(); a.editGroup(&group) })
		add("remove-group-"+group.ID, "移除分组 "+group.Name, "保留已打开的终端和网页", "remove delete group", func() { a.closePalette(); a.removeGroup(group.ID) })
	}
	add("settings", "打开外观设置", "字体、颜色和玻璃效果", "settings preferences appearance", func() {
		a.closePalette()
		a.draft, a.preferences, a.preferencesError = a.config, true, ""
		a.modalOrigin = ui.Rect{}
		a.loadFonts()
	})
	return filterPalette(commands, a.palette.Query)
}

func (a *application) splitFits(c *ui.Context, target int, edge splitEdge) bool {
	t := a.work.sessionForPane(target)
	if t == nil {
		return false
	}
	p := t.Root.find(target).Pane
	if !roomForSplit(p.Surface.Bounds(c), edge) {
		a.status = "窗格空间不足，请先调整窗口或分隔线。"
		return false
	}
	return true
}

func (a *application) addNewSplitCommand(commands *[]paletteCommand, c *ui.Context, p profile, target int, edge splitEdge) {
	*commands = append(*commands, paletteCommand{ID: "new-" + p.ID, Label: "新建 " + p.Name, Detail: "使用目标窗格的目录和分组", Keywords: "new create terminal " + p.Name, Run: func() {
		if !a.splitFits(c, target, edge) {
			return
		}
		t := a.work.sessionForPane(target)
		dir, group := t.Root.find(target).Pane.directory(), t.GroupID
		leaf := a.work.add(p, dir)
		prepareAgentSession(leaf)
		a.work.active().GroupID = group
		a.dockPane(leaf.ID, target, edge)
		a.closePalette()
		if a.start != nil {
			a.start(leaf)
		}
	}})
}

func filterPalette(commands []paletteCommand, query string) []paletteCommand {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return commands
	}
	filtered := commands[:0]
	for _, command := range commands {
		text := strings.ToLower(command.Label + " " + command.Detail + " " + command.Keywords)
		match := true
		for _, word := range strings.Fields(query) {
			if !strings.Contains(text, word) {
				match = false
				break
			}
		}
		if match {
			filtered = append(filtered, command)
		}
	}
	return filtered
}

func (a *application) commandPaletteView(c *ui.Context) {
	p := &a.palette
	w, h := c.Size()
	origin := ui.Rect{X: w / 2, Y: 52, W: 1, H: 1}
	if p.Motion.step(c.Now(), p.Open, origin, .94) {
		c.AnimationFrame()
	}
	if !p.Motion.Visible {
		return
	}
	if p.Query != p.LastQuery {
		p.Selected, p.LastQuery, p.Scroll.Y = 0, p.Query, 0
	}
	commands := a.paletteCommands(c)
	p.Selected = max(0, min(p.Selected, len(commands)-1))
	width := min(float32(600), w-32)
	listHeight := min(float32(352), max(float32(88), h-200))
	ui.Overlay(c, func() {
		back := ui.Box(c.Key("command-backdrop")).Absolute().Left(0).Right(0).Top(0).Bottom(0).Modal().
			Background(ui.RGBA(0, 0, 0, .25*p.Motion.Opacity)).OnClick(a.closePalette)
		if back.OverlayShortcut(0, ui.KeyEscape) {
			a.closePalette()
		}
		back.Children(func() {
			raisedOverlay(ui.Column(c.Key("command-palette")).Absolute().Top(62).Left((w-width)/2).Width(width).Padding(10).Gap(6).Radius(16), .35).
				Opacity(p.Motion.Opacity).TransformOrigin(.5, 0).Scale(p.Motion.Scale, p.Motion.Scale).Disabled(!p.Motion.Target).OnClick(func() {}).Children(func() {
				placeholder := "输入命令…"
				if p.Stage == "split" {
					placeholder = p.Edge.label() + " · 选择会话或新建终端…"
				}
				if p.Stage == "switch" {
					placeholder = "搜索终端或分组…"
				}
				ui.TextInput(c.Key("command-query"), &p.Query).Label("命令搜索").Placeholder(placeholder).Height(40).Radius(8).AutoFocus().
					OnShortcut(0, ui.KeyDown, func() { a.movePaletteSelection(c, 1, listHeight) }).
					OnShortcut(0, ui.KeyUp, func() { a.movePaletteSelection(c, -1, listHeight) }).
					OnSubmit(func() {
						choices := a.paletteCommands(c)
						if len(choices) > 0 {
							choices[max(0, min(p.Selected, len(choices)-1))].Run()
						}
					})
				ui.Scroll(c.Key("command-list")).TrackScroll(&p.Scroll).MaxHeight(listHeight).Children(func() {
					ui.Column(c).Gap(0).Children(func() {
						for i, command := range commands {
							b := softButton(c.Key(command.ID), "命令 "+command.Label).Height(44).Grow(1).MinWidth(0).Radius(7).Padding(5, 11).
								OnClick(command.Run)
							if i == p.Selected {
								b.Background(ui.RGBA(210, 224, 246, .11))
							}
							b.Children(func() {
								ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
									ui.Text(c, command.Label).FontSize(12).SingleLine().Clip()
									ui.Text(c, command.Detail).FontSize(10).SingleLine().Clip().TextColor(c.Theme().TextMuted)
								})
							})
						}
						if len(commands) == 0 {
							ui.Text(c, "没有匹配的命令").FontSize(12).Padding(16).TextColor(c.Theme().TextMuted)
						}
					})
				})
				ui.Text(c, "↑ ↓ 选择    Enter 执行    Esc 关闭").FontSize(10).Padding(5, 10).TextColor(c.Theme().TextMuted)
			})
		})
	})
}

func (a *application) movePaletteSelection(c *ui.Context, delta int, height float32) {
	p := &a.palette
	count := len(a.paletteCommands(c))
	if count == 0 {
		return
	}
	p.Selected = (p.Selected + delta + count) % count
	top, bottom := float32(p.Selected*44), float32((p.Selected+1)*44)
	if top < p.Scroll.Y {
		p.Scroll.Y = top
	}
	if bottom > p.Scroll.Y+height {
		p.Scroll.Y = bottom - height
	}
}
