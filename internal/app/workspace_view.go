package app

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

func svgIcon(path string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">` + path + `</svg>`))
}

var (
	splitDownIcon  = svgIcon(`<rect x="4" y="4" width="16" height="16" rx="2"/><path d="M4 12h16"/>`)
	panelIcon      = svgIcon(`<rect x="4" y="4" width="16" height="16" rx="2"/><path d="M10 4v16"/>`)
	plusIcon       = svgIcon(`<path d="M12 5v14M5 12h14"/>`)
	folderPlusIcon = svgIcon(`<path d="M3 7a2 2 0 0 1 2-2h5l2 3h7a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2zM12 11v6M9 14h6"/>`)
	chevronIcon    = svgIcon(`<path d="m9 5 7 7-7 7"/>`)
	closeIcon      = svgIcon(`<path d="m7 7 10 10M17 7 7 17"/>`)
	settingsIcon   = makeSettingsIcon()
	webIcon        = svgIcon(`<circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3c5 5 5 13 0 18-5-5-5-13 0-18z"/>`)
	commandsIcon   = svgIcon(`<path d="m5 7 5 5-5 5M13 17h6"/>`)
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
	if !b.Pressed() {
		alpha = b.Animate("hover", alpha, 150*time.Millisecond)
	}
	pressFeedback(b)
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
	pressFeedback(b)
	return b.Children(func() { ui.Text(c, label).FontSize(12).Bold() })
}

func iconButton(c *ui.Context, icon *ui.SVG, label string) ui.Element {
	b := softButton(c, label).Size(26, 26).Padding(5).Tooltip(label)
	b.Children(func() { ui.Icon(c, icon).Size(15, 15) })
	return b
}

func (a *application) content() mygo.Content { return ui.View(a.view) }

func (a *application) view(c *ui.Context) {
	a.syncOverlaySnapshots()
	defer func() {
		if a.syncOverlaySnapshots() {
			c.AnimationFrame()
		}
	}()
	defer func() { a.rendered = true }()
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
	c.OnShortcut(ui.Ctrl|ui.Shift, ui.KeyP, a.openCommandPalette)
	c.OnShortcut(ui.Ctrl|ui.Shift, ui.KeyW, func() {
		if t := a.work.active(); t != nil {
			a.requestStop(t, t.Close.Bounds(c))
		}
	})
	c.OnShortcut(ui.Ctrl|ui.Shift, ui.KeyTab, func() { a.cycleTab(-1) })
	c.OnShortcut(ui.Ctrl, ui.KeyTab, func() { a.cycleTab(1) })
	ui.Column(c).Fill().Children(func() {
		a.titlebar(c)
		ui.Column(c.Key("workspace-content")).Grow(1).MinHeight(0).MinWidth(0).Padding(0, 4, 4, 4).Children(func() {
			if screen := a.work.activeScreen(); screen != nil {
				a.layoutView(c, screen.Root, len(screen.Root.leaves()) > 1)
			} else {
				ui.Column(c).Grow(1).Center().Gap(12).Children(func() {
					ui.Icon(c, commandsIcon).Size(28, 28).Opacity(.4)
					textButton(c, "新建终端").OnClick(func() { a.openLauncher("") })
					textButton(c, "新建网页").OnClick(func() { a.newBrowser(0, splitRight) })
				})
			}
		})
		if a.status != "" {
			ui.Text(c, a.status).FontSize(11).Padding(6, 12).TextColor(c.Theme().TextMuted).Wrap()
		}
	})
	a.launcherView(c)
	a.groupEditorView(c)
	a.settingsView(c)
	a.confirmations(c)
	a.aliasEditorView(c)
	a.agentResumeView(c)
	a.commandPaletteView(c)
	if request := a.pendingDock; request != nil {
		a.pendingDock = nil
		a.dockPane(request.Source, request.Target, request.Edge)
	}
	if request := a.pendingGroupDrop; request != nil {
		a.pendingGroupDrop = nil
		a.dropInGroup(*request)
	}
	if request := a.pendingTabDrop; request != nil {
		a.pendingTabDrop = nil
		a.dropOnTabStrip(*request)
	}
}

func (a *application) cycleTab(direction int) {
	for i, s := range a.work.Screens {
		if s == a.work.activeScreen() {
			n := len(a.work.Screens)
			a.selectScreen(a.work.Screens[(i+direction+n)%n])
			return
		}
	}
}

func (a *application) selectTab(t *tab) {
	a.work.focusPane(t.Focused)
	if n := t.Root.find(t.Focused); n != nil {
		n.Pane.Handle.Focus()
	}
}

func (a *application) titlebar(c *ui.Context) {
	bar := c.TitleBar()
	ui.Row(c).Height(max(workspaceTitleHeight, bar.Height)).Padding(0, bar.Right+8, 0, bar.Left+8).Gap(6).AlignItems(ui.Center).DragWindow().Children(func() {
		title := "Cyrene Terminal"
		if screen := a.work.activeScreen(); screen != nil {
			title = a.screenLabel(screen)
		}
		if a.win != nil && a.windowTitle != title {
			a.windowTitle = title
			a.win.SetTitle(title)
		}
		a.contentTabBar(c)
		iconButton(c.Key("new-workspace"), plusIcon, "新建工作区").Bind(&a.newGroupHandle).OnClick(func() { a.openLauncher(""); a.launchWorkspace = true })
		iconButton(c.Key("preferences-button"), settingsIcon, "外观设置").Bind(&a.preferencesHandle).OnClick(func() {
			a.modalOrigin = a.preferencesHandle.Bounds(c)
			a.draft, a.preferences, a.preferencesError = a.config, true, ""
			a.loadFonts()
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

func (a *application) terminalView(c *ui.Context, p *pane) {
	if p.Browser != nil {
		a.browserView(c, p)
		return
	}
	if p.Term != nil {
		ui.Box(c.Key("terminal-surface")).Grow(1).MinHeight(0).Clip().Children(func() {
			terminal.View(c.Key(p.ID), p.Term).Grow(1).MinHeight(0).Bind(&p.Handle).AutoFocus().Description(acceptanceTerminalDescription(p.Term))
			select {
			case <-p.Term.Done():
				ui.Row(c.Key("restart-position")).Absolute().Left(16).Right(16).Bottom(20).Justify(ui.Center).Children(func() {
					ui.Row(c.Key("restart-bar")).Padding(8, 12).Gap(16).AlignItems(ui.Center).Radius(12).
						Material(frostedOverlay{Clarity: .35}).Background(ui.RGBA(20, 23, 28, .82)).Border(1, ui.RGBA(255, 255, 255, .12)).Children(func() {
						ui.Text(c, fmt.Sprintf("进程已结束 · 退出码 %d", p.Term.ExitCode())).FontSize(12).TextColor(c.Theme().TextMuted)
						if p.Profile.supportsResume() {
							dialogButton(c, "恢复会话", true).Bind(&p.ResumeHandle).Height(30).MinWidth(86).Padding(5, 12).OnClick(func() { a.requestAgentResume(p, p.ResumeHandle.Bounds(c)) })
						}
						dialogButton(c, "重新启动", true).Height(30).MinWidth(86).Padding(5, 12).Disabled(p.Starting).OnClick(func() { a.restartPane(p) })
					})
				})
			default:
			}
		})
		return
	}
	ui.Column(c.Key(p.ID)).Grow(1).Padding(16).Gap(10).Children(func() {
		if p.Error != "" {
			viewport, _ := c.Size()
			cardWidth := min(float32(420), max(float32(240), viewport-48))
			ui.Column(c).Grow(1).MinHeight(0).Center().Children(func() {
				ui.Column(c).Width(cardWidth).Padding(24).Gap(14).Radius(14).
					Material(frostedOverlay{Clarity: .35}).Border(1, ui.RGBA(255, 255, 255, .10)).Children(func() {
					title := "终端未能启动"
					if p.NeedsResume {
						title = "恢复 Agent 会话"
					}
					ui.Text(c, title).FontSize(18).Bold()
					ui.Text(c, p.Error).Wrap().FontSize(12).TextColor(c.Theme().TextMuted)
					ui.Row(c).Justify(ui.End).Children(func() {
						if p.NeedsResume {
							dialogButton(c, "恢复会话", true).Bind(&p.ResumeHandle).OnClick(func() { a.requestAgentResume(p, p.ResumeHandle.Bounds(c)) })
							return
						}
						dialogButton(c, "重试", true).Material(frostedOverlay{Clarity: .45}).TextColor(c.Theme().Text).
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
	anchor := c.Resolve(a.addHandle)
	if a.launchTarget == 0 {
		anchor = c.Resolve(a.newGroupHandle)
	}
	if !anchor.Valid() && a.launcherMotion.Visible {
		r := a.launcherMotion.Source
		anchor = ui.Box(c.Key("launcher-anchor")).Absolute().Left(r.X).Top(r.Y).Size(r.W, r.H).PassThrough().Opacity(0)
	}
	if !anchor.Valid() {
		return
	}
	m := &a.launcherMotion
	if m.step(c.Now(), a.launcher, anchor.Bounds(), 0.82) {
		c.AnimationFrame()
	}
	visible := m.Visible
	ui.PopoverBase(c.Key("launcher"), anchor, &visible, func(panel ui.Element) {
		pivotX, pivotY := min(float32(1), max(float32(0), m.Source.W/2/252)), float32(0)
		bounds := panel.Bounds()
		if bounds.W > 0 && bounds.H > 0 {
			pivotX = (m.Source.X + m.Source.W/2 - bounds.X) / bounds.W
			pivotY = (m.Source.Y + m.Source.H/2 - bounds.Y) / bounds.H
		}
		raisedOverlay(panel.Width(252).Padding(7).Gap(4).Radius(16).Margin(7, 0, 0, 0), .25).
			Opacity(m.Opacity).TransformOrigin(pivotX, pivotY).
			Transform(ui.Scaling(m.Scale, m.Scale).Then(ui.Translation(0, -5*(1-m.Position)))).Disabled(!m.Target)

		softButton(c.Key("launch-web"), "新建网页").Height(38).Gap(10).Justify(ui.Start).Children(func() {
			ui.Icon(c, webIcon).Size(17, 17)
			ui.Text(c, "新建网页").FontSize(13)
		}).OnClick(func() {
			a.launcher = false
			a.newBrowser(a.launchTarget, a.launchEdge)
			if a.launchWorkspace {
				a.work.markWorkspace(a.work.activeScreen())
			}
			a.launchWorkspace = false
		})
		_, h := c.Size()
		ui.Scroll(c).MaxHeight(max(100, h-120)).Children(func() {
			ui.Column(c).Gap(2).Children(func() {
				for i, p := range a.config.Profiles {
					b := softButton(c.Key(p.ID), "启动 "+p.Name).Width(236).Height(38).Radius(11).Gap(10).Justify(ui.Start).
						Children(func() {
							agentLogo(c, p.ID, 19)
							ui.Text(c, p.Name).FontSize(13).SingleLine()
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

func pressFeedback(b ui.Element) {
	scale := float32(1)
	if b.Pressed() {
		scale = .95
	}
	b.Scale(scale, scale)
}
func nStarting(t *tab) bool {
	for _, p := range t.Root.leaves() {
		if p.Starting && p.Browser == nil {
			return true
		}
	}
	return false
}
func (a *application) restartPane(p *pane) {
	a.restartPaneWithCommand(p, nil)
}

func (a *application) restartPaneWithCommand(p *pane, command []string) {
	if p.Starting {
		return
	}
	if term := p.Term; term != nil {
		p.Dir = p.directory()
		a.sessionsMu.Lock()
		delete(a.sessions, term)
		a.sessionsMu.Unlock()
		p.Term = nil
		store := p.clipboard
		go func() { term.Close(); store.close() }()
	}
	if len(command) > 0 {
		p.Command = append([]string(nil), command...)
	} else {
		p.Command, p.AgentSession, p.AgentToken, p.AgentRecord, p.NeedsResume = nil, "", "", "", false
		prepareAgentSession(p)
	}
	if a.start != nil {
		a.start(p)
	}
}
func (a *application) aliasEditorView(c *ui.Context) {
	a.dialog(c, "session-alias", &a.aliasEditor, 420, func() {
		label, placeholder := "会话别名", "留空使用当前文件夹名"
		if a.aliasScreen != nil {
			label, placeholder = "标签名称", "留空使用当前会话名"
		}
		ui.Text(c, label).FontSize(20).Bold()
		ui.TextInput(c.Key("session-alias-input"), &a.aliasDraft).Height(36).Radius(7).Label(label).Placeholder(placeholder).AutoFocus()
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			dialogButton(c, "取消", false).OnClick(func() { a.aliasEditor = false; a.focusActiveTerminal() })
			dialogButton(c, "保存", true).OnClick(func() {
				if a.aliasScreen != nil {
					if a.work.screenByID(a.aliasScreen.ID) == a.aliasScreen {
						a.aliasScreen.Alias = strings.TrimSpace(a.aliasDraft)
					}
				} else if a.aliasTarget != nil {
					a.aliasTarget.Alias = strings.TrimSpace(a.aliasDraft)
				}
				a.aliasEditor = false
				a.focusActiveTerminal()
			})
		})
	})
}

func (a *application) editScreenAlias(s *screen, origin ui.Rect) {
	if s == nil {
		return
	}
	a.aliasScreen, a.aliasTarget, a.aliasDraft, a.aliasEditor, a.modalOrigin = s, nil, s.Alias, true, origin
}

func (a *application) editSessionAlias(t *tab, origin ui.Rect) {
	if t == nil {
		return
	}
	a.aliasScreen, a.aliasTarget, a.aliasDraft, a.aliasEditor, a.modalOrigin = nil, t, t.Alias, true, origin
}

func (a *application) workspaceAddButton(c *ui.Context, s *screen) {
	softButton(c.Key("add-terminal"), "新建终端").Bind(&a.addHandle).
		Size(26, 26).Padding(6).Radius(13).Margin(3).Tooltip("添加终端或网页").
		Children(func() { ui.Icon(c, plusIcon).Size(14, 14) }).OnClick(func() {
		group := ""
		if s != nil {
			if owner := a.work.sessionForPane(s.Focused); owner != nil {
				group = owner.GroupID
			}
		}
		a.openLauncher(group)
		if s != nil {
			a.launchTarget = s.Focused
			a.launchEdge = splitRight
			if n := s.Root.find(s.Focused); n != nil {
				bounds := n.Pane.Surface.Bounds(c)
				if bounds.H > bounds.W {
					a.launchEdge = splitBottom
				}
			}
		}
	})
}
