package app

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

func svgIcon(path string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">` + path + `</svg>`))
}

var (
	panelIcon      = svgIcon(`<rect x="4" y="4" width="16" height="16" rx="2"/><path d="M10 4v16"/>`)
	plusIcon       = svgIcon(`<path d="M12 5v14M5 12h14"/>`)
	folderPlusIcon = svgIcon(`<path d="M3 7a2 2 0 0 1 2-2h5l2 3h7a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2zM12 11v6M9 14h6"/>`)
	chevronIcon    = svgIcon(`<path d="m9 5 7 7-7 7"/>`)
	closeIcon      = svgIcon(`<path d="m7 7 10 10M17 7 7 17"/>`)
	settingsIcon   = makeSettingsIcon()
)

func makeSettingsIcon() *ui.SVG {
	path := ""
	for tooth := 0; tooth < 8; tooth++ {
		for j, offset := range []float64{-18, -10, 10, 18} {
			angle := (float64(tooth)*45 + offset) * math.Pi / 180
			radius := 7.0
			if j == 1 || j == 2 {
				radius = 9.5
			}
			command := "L"
			if tooth == 0 && j == 0 {
				command = "M"
			}
			path += fmt.Sprintf("%s%.3f %.3f", command, 12+radius*math.Cos(angle), 12+radius*math.Sin(angle))
		}
	}
	return svgIcon(`<path d="` + path + `z"/><circle cx="12" cy="12" r="3"/>`)
}

// Base widgets keep native focus/keyboard semantics; colors ease both in and out.
func softButton(c *ui.Context, label string) ui.Element {
	b := ui.ButtonBase(c).Height(26).Radius(7).Padding(4, 8).Label(label).Clip().FocusRing(false)
	alpha := float32(0)
	if b.Hovered() || b.FocusVisible() {
		alpha = 0.075
	}
	if b.Pressed() {
		alpha = 0.13
	}
	alpha = b.Animate("hover", alpha, 150*time.Millisecond)
	return b.Background(ui.RGBA(230, 232, 240, alpha))
}

func textButton(c *ui.Context, label string) ui.Element {
	b := softButton(c, label)
	b.Children(func() { ui.Text(c, label).FontSize(12).SingleLine() })
	return b
}

func dialogButton(c *ui.Context, label string, primary bool) ui.Element {
	b := ui.ButtonBase(c.Key(label)).Label(label).Height(34).MinWidth(76).Padding(7, 18).Radius(8)
	if primary {
		alpha := float32(.9)
		if b.Hovered() {
			alpha = 1
		}
		b.Background(ui.RGBA(222, 227, 237, alpha)).TextColor(ui.Hex("#17191e"))
	} else {
		b.Background(ui.RGBA(230, 235, 245, .055)).Border(1, ui.RGBA(230, 235, 245, .12))
	}
	return b.Children(func() { ui.Text(c, label).FontSize(12).Bold() })
}

func iconButton(c *ui.Context, icon *ui.SVG, label string) ui.Element {
	b := softButton(c, label).Size(26, 26).Padding(5).Tooltip(label)
	b.Children(func() { ui.Icon(c, icon).Size(15, 15) })
	return b
}

func (a *application) content() mygo.Content { return ui.View(a.view) }

func (a *application) view(c *ui.Context) {
	c.SetTheme(a.config.theme())
	bg := ui.Hex(a.config.Background)
	if c.Vibrancy() {
		bg = ui.RGBA(bg.R, bg.G, bg.B, a.config.Opacity)
	}
	c.Root().Background(bg)
	c.OnShortcut(ui.Ctrl|ui.Shift, ui.KeyT, func() {
		id := ""
		if t := a.work.active(); t != nil {
			id = t.GroupID
		}
		a.openLauncher(id)
	})
	c.OnShortcut(ui.Ctrl|ui.Shift, ui.KeyB, a.toggleSidebar)
	c.OnShortcut(ui.Ctrl|ui.Shift, ui.KeyW, func() {
		if t := a.work.active(); t != nil {
			a.requestStop(t, t.Close.Bounds(c))
		}
	})
	c.OnShortcut(ui.Ctrl|ui.Shift, ui.KeyTab, func() { a.cycleTab(-1) })
	c.OnShortcut(ui.Ctrl, ui.KeyTab, func() { a.cycleTab(1) })
	viewportWidth, _ := c.Size()
	if a.sidebarWidth <= 0 {
		a.sidebarWidth = 244
	}
	a.sidebarWidth = min(a.sidebarWidth, max(200, viewportWidth-360))
	target := float32(1)
	if a.collapsed {
		target = 0
	}
	// Explicit frames match the requested popup motion even when the platform's
	// automatic widget transitions are disabled. Resizing stays immediate.
	if a.sidebarMotion.step(c.Now(), target) {
		c.AnimationFrame()
	}
	width := a.sidebarWidth * a.sidebarMotion.Value
	// One uninterrupted surface extends behind the title bar and the tab list.
	ui.Box(c.Key("sidebar-surface")).Absolute().Left(0).Top(0).Bottom(0).Width(width).
		Background(ui.RGBA(0, 0, 0, 0.12)).PassThrough()
	ui.Column(c).Fill().Children(func() {
		a.titlebar(c, width)
		ui.Row(c).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
			a.sidebar(c, width)
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				if t := a.work.active(); t != nil {
					if n := t.Root.find(t.Focused); n != nil {
						a.terminalView(c, n.Pane)
					}
				} else {
					ui.Box(c).Grow(1)
				}
				if a.status != "" {
					ui.Text(c, a.status).FontSize(11).Padding(8).TextColor(c.Theme().TextMuted).Wrap()
				}
			})
		})
	})
	if !a.collapsed {
		grip := ui.Box(c.Key("sidebar-resize")).Absolute().Left(width - 3).Top(32).Bottom(0).Width(6).
			Cursor(ui.CursorResizeEW).Label("调整侧栏宽度").Focusable()
		if dx, _, held := grip.Dragged(); held && dx != 0 {
			a.sidebarWidth = min(max(a.sidebarWidth+dx, 200), min(520, viewportWidth-360))
		}
		grip.OnShortcut(0, ui.KeyLeft, func() { a.sidebarWidth = max(200, a.sidebarWidth-12) })
		grip.OnShortcut(0, ui.KeyRight, func() { a.sidebarWidth = min(min(520, viewportWidth-360), a.sidebarWidth+12) })
	}
	a.launcherView(c)
	a.groupEditorView(c)
	a.settingsView(c)
	a.confirmations(c)
}

func (a *application) cycleTab(direction int) {
	for i, t := range a.work.Tabs {
		if t.ID == a.work.Active {
			n := len(a.work.Tabs)
			a.selectTab(a.work.Tabs[(i+direction+n)%n])
			return
		}
	}
}

func (a *application) selectTab(t *tab) {
	a.work.Active = t.ID
	if a.win != nil {
		a.win.SetTitle(t.label())
	}
	if n := t.Root.find(t.Focused); n != nil {
		n.Pane.Handle.Focus()
	}
}

func (a *application) toggleSidebar() { a.collapsed = !a.collapsed; a.launcher = false }

func (a *application) titlebar(c *ui.Context, width float32) {
	bar := c.TitleBar()
	ui.Row(c).Height(max(32, bar.Height)).Padding(0, bar.Right+6, 0, 8+bar.Left).Gap(4).AlignItems(ui.Center).DragWindow().Children(func() {
		iconButton(c.Key("sidebar-toggle"), panelIcon, "切换侧栏").Bind(&a.toggleHandle).OnClick(a.toggleSidebar)
		ui.Row(c).Width(max(0, width-42)).Shrink(0).Clip().AlignItems(ui.Center).Children(func() {
			if width > 0.1 {
				ui.Spacer(c)
				iconButton(c.Key("add-folder"), folderPlusIcon, "新建文件夹分组").Bind(&a.groupEditorHandle).Disabled(a.settingsBusy).OnClick(func() {
					a.modalOrigin = a.groupEditorHandle.Bounds(c)
					a.editGroup(nil)
				})
				iconButton(c.Key("add-terminal"), plusIcon, "新建终端").Bind(&a.addHandle).OnClick(func() { a.openLauncher("") })
			}
		})
		title := ""
		if t := a.work.active(); t != nil {
			title = t.label()
		}
		ui.Text(c, title).FontSize(12).TextColor(c.Theme().TextMuted).Margin(0, 0, 0, 16).Grow(1).MinWidth(0).SingleLine().Clip()
		iconButton(c.Key("preferences-button"), settingsIcon, "外观设置").Bind(&a.preferencesHandle).OnClick(func() {
			a.modalOrigin = a.preferencesHandle.Bounds(c)
			a.draft, a.preferences, a.preferencesError = a.config, true, ""
		})
	})
}

func (a *application) sidebar(c *ui.Context, width float32) {
	ui.Row(c.Key("sidebar")).Width(width).Shrink(0).MinHeight(0).AlignItems(ui.Stretch).Clip().Children(func() {
		if width < 0.1 {
			return
		}
		ui.Column(c).Width(a.sidebarWidth).Shrink(0).Disabled(a.collapsed).Children(func() {
			ui.Scroll(c.Key("tabs")).Grow(1).MinHeight(0).Children(func() {
				ui.Column(c).Padding(5, 5).Gap(5).Children(func() {
					for i, t := range a.work.Tabs {
						if t.GroupID == "" {
							a.tabRow(c, t, a.sidebarWidth-10, i)
						}
					}
					for _, g := range a.config.Groups {
						a.groupView(c, g, a.sidebarWidth-10)
					}
				})
			})
		})
	})
}

func (a *application) tabRow(c *ui.Context, t *tab, width float32, index int) {
	row := ui.Row(c.Key(t.ID)).Width(width).Height(44).Radius(7).AlignItems(ui.Center).Padding(0, 4).Clip()
	alpha := float32(0)
	if a.work.Active == t.ID {
		alpha = 0.105
	}
	if row.Hovered() {
		alpha += 0.045
	}
	row.Background(ui.RGBA(240, 242, 250, row.Animate("hover", alpha, 160*time.Millisecond))).Children(func() {
		b := ui.ButtonBase(c.Key(fmt.Sprintf("tab-%d", t.ID))).Label(t.label()).Bind(&t.Select).Clip().FocusRing(false).Grow(1).MinWidth(0).Height(42).Padding(5, 7).Justify(ui.Start).
			OnClick(func() { a.selectTab(t) })
		b.Children(func() {
			ui.Row(c).Grow(1).MinWidth(0).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
					ui.Text(c, t.label()).FontSize(13).SingleLine().Clip()
					path := ""
					if n := t.Root.find(t.Focused); n != nil {
						path = shortDirectory(n.Pane.Dir)
					}
					ui.Text(c, path).FontSize(10).SingleLine().Clip().TextColor(c.Theme().TextMuted)
				})
			})
		})
		b.ContextMenu(func(m *ui.Menu) {
			if m.Item("关闭终端…").Chosen() {
				a.requestStop(t, t.Select.Bounds(c))
			}
		})
		ui.ButtonBase(c.Key("close")).Label(fmt.Sprintf("关闭终端 %d", t.ID)).Bind(&t.Close).Size(24, 24).Padding(5).FocusRing(false).Children(func() {
			ui.Icon(c, closeIcon).Size(14, 14)
		}).OnClick(func() {
			a.requestStop(t, t.Close.Bounds(c))
		})
	})
}

func shortDirectory(dir string) string {
	home, _ := os.UserHomeDir()
	if strings.EqualFold(filepath.Clean(dir), filepath.Clean(home)) {
		return "~"
	}
	if rel, err := filepath.Rel(home, dir); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "~" + string(filepath.Separator) + rel
	}
	return dir
}

func (a *application) groupView(c *ui.Context, g folderGroup, width float32) {
	state := a.groupState(g.ID)
	target := float32(1)
	if state.Folded {
		target = 0
	}
	if state.Motion.stepDuration(c.Now(), target, .18) {
		c.AnimationFrame()
	}
	reveal := state.Motion.Value
	count := 0
	for _, tab := range a.work.Tabs {
		if tab.GroupID == g.ID {
			count++
		}
	}
	fullHeight := max(0, float32(count*48-4))
	ui.Column(c.Key("group-" + g.ID)).Width(width).Gap(2 * reveal).Children(func() {
		ui.Row(c).Height(30).AlignItems(ui.Center).Children(func() {
			b := softButton(c, "分组 "+g.Name).Bind(&state.Header).Grow(1).MinWidth(0).Justify(ui.Start).Padding(5, 6).Gap(7).
				Tooltip(g.Directory).OnClick(func() { state.Folded = !state.Folded })
			b.Children(func() {
				icon := ui.Icon(c, chevronIcon).Size(11, 11)
				icon.Rotate(90 * reveal)
				ui.Text(c, g.Name).FontSize(12).SingleLine().Clip()
			})
			b.ContextMenu(func(m *ui.Menu) {
				if m.Item("编辑分组…").Disabled(a.settingsBusy).Chosen() {
					a.modalOrigin = state.Header.Bounds(c)
					a.editGroup(&g)
				}
				if m.Item("移除分组（保留终端）").Disabled(a.settingsBusy).Chosen() {
					a.removeGroup(g.ID)
				}
			})
			iconButton(c, plusIcon, "在 "+g.Name+" 新建终端").Bind(&state.Add).OnClick(func() { a.openLauncher(g.ID) })
		})
		if reveal > 0 && count > 0 {
			ui.Box(c.Key("group-reveal")).Width(width).Height(fullHeight * reveal).Clip().IgnoreInput(state.Folded).Children(func() {
				ui.Row(c).Width(width).Height(fullHeight).Shrink(0).AlignItems(ui.Stretch).Children(func() {
					// Header padding 6 + half of its 11-DIP chevron = 11.5.
					// The 1-DIP guide is centered on that same axis.
					ui.Box(c.Key("group-guide")).Width(1).Margin(0, 8, 0, 11).
						Background(ui.RGBA(220, 224, 235, 0.16)).PassThrough()
					ui.Column(c).Width(width - 20).Gap(4).Children(func() {
						for i, t := range a.work.Tabs {
							if t.GroupID == g.ID {
								a.tabRow(c, t, width-20, i)
							}
						}
					})
				})
			})
		}
	})
}

func (a *application) terminalView(c *ui.Context, p *pane) {
	if p.Term != nil {
		terminal.View(c.Key(p.ID), p.Term).Grow(1).MinHeight(0).Bind(&p.Handle).AutoFocus().Description(acceptanceTerminalDescription(p.Term))
		return
	}
	ui.Column(c.Key(p.ID)).Grow(1).Padding(16).Gap(10).Children(func() {
		if p.Error != "" {
			viewport, _ := c.Size()
			cardWidth := min(float32(420), max(float32(240), viewport-a.sidebarWidth*a.sidebarMotion.Value-48))
			ui.Column(c).Grow(1).MinHeight(0).Center().Children(func() {
				ui.Column(c).Width(cardWidth).Padding(24).Gap(14).Radius(14).
					Material(glass.Glass{Clarity: .35}).Border(1, ui.RGBA(255, 255, 255, .10)).Children(func() {
					ui.Text(c, "终端未能启动").FontSize(18).Bold()
					ui.Text(c, p.Error).Wrap().FontSize(12).TextColor(c.Theme().TextMuted)
					ui.Row(c).Justify(ui.End).Children(func() {
						dialogButton(c, "重试", true).Material(glass.Glass{Clarity: .45}).TextColor(c.Theme().Text).
							Border(1, ui.RGBA(255, 255, 255, .18)).OnClick(func() {
							if a.start != nil {
								a.start(p)
							}
						})
					})
				})
			})
		} else {
			ui.Text(c, "正在启动终端…").FontSize(12).TextColor(c.Theme().TextMuted)
		}
	})
}

func (a *application) launcherView(c *ui.Context) {
	// A shortcut can still open the launcher while its toolbar is hidden.
	// Use the persistent collapse toggle as the visible anchor in that case.
	anchor := c.Resolve(a.addHandle)
	if a.collapsed {
		anchor = c.Resolve(a.toggleHandle)
	}
	if state := a.groupUI[a.launchGroup]; state != nil && !a.collapsed {
		if groupAnchor := c.Resolve(state.Add); groupAnchor.Valid() {
			anchor = groupAnchor
		}
	}
	if !anchor.Valid() {
		return
	}
	m := &a.launcherMotion
	if m.step(c.Now(), a.launcher, ui.Rect{}, 0.82) {
		c.AnimationFrame()
	}
	visible := m.Visible
	ui.PopoverBase(c.Key("launcher"), anchor, &visible, func(panel ui.Element) {
		panel.Width(236).Padding(6).Gap(3).Radius(12).Margin(5, 0, 0, 0).
			Material(glass.Glass{Clarity: 0.25}).Border(1, ui.RGBA(255, 255, 255, 0.1)).
			Opacity(m.Opacity).TransformOrigin(0.08, 0).
			Transform(ui.Scaling(m.Scale, m.Scale).Then(ui.Translation(0, -10*(1-m.Position)))).Disabled(!m.Target)
		ui.Text(c, shortDirectory(a.directoryForGroup(a.launchGroup))).FontSize(10).TextColor(c.Theme().TextMuted).Padding(6, 8).SingleLine().Clip()
		_, h := c.Size()
		ui.Scroll(c).MaxHeight(max(100, h-120)).Children(func() {
			ui.Column(c).Gap(2).Children(func() {
				for i, p := range a.config.Profiles {
					b := softButton(c.Key(p.ID), "启动 "+p.Name).Width(224).Height(32).Radius(10).Gap(9).Justify(ui.Start).
						Children(func() {
							agentLogo(c, p.ID, 19)
							ui.Text(c, p.Name).FontSize(12).SingleLine()
						}).OnClick(func() { a.launch(p) })
					if i == 0 {
						b.AutoFocus()
					}
				}
			})
		})
	})
	if m.Visible && !visible {
		a.launcher = false
		c.AnimationFrame()
	}
}
