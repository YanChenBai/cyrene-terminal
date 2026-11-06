package app

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
)

// Only the selected workspace exposes its pane tabs.
func (a *application) contentTabBar(c *ui.Context) {
	height := max(workspaceTitleHeight, c.TitleBar().Height)
	tabHeight := float32(34)
	active := a.work.activeScreen()
	reveal := active != nil && (active.ID != a.revealedScreen || active.Focused != a.revealedPane)
	if active != nil {
		a.revealedScreen, a.revealedPane = active.ID, active.Focused
	}
	ui.Row(c.Key("content-tabs")).Height(height).Grow(1).Basis(0).MinWidth(0).Clip().AlignItems(ui.Stretch).Children(func() {
		strip := ui.ScrollHorizontal(c.Key("content-tabs-scroll")).TrackScroll(&a.tabScroll).Grow(1).Basis(0).MinWidth(0).Height(height).Clip()
		strip.HandleInput(func(ev ui.InputEvent) bool {
			if ev.Kind == ui.InputScroll && ev.DX == 0 && ev.DY != 0 && ev.Mods&ui.Ctrl == 0 && a.tabScroll.MaxX > 0 {
				a.tabScroll.X = min(max(0, a.tabScroll.X+ev.DY), a.tabScroll.MaxX)
				return true
			}
			return false
		})
		if value, dropped := ui.Drop[terminalDrag](strip); dropped {
			a.pendingTabDrop = &tabDropRequest{Drag: value}
		}
		_, overStrip := ui.DragOver[terminalDrag](strip)
		strip.Children(func() {
			ui.Row(c).Height(height).AlignItems(ui.Center).Shrink(0).Gap(6).Padding(2, 0, 0, 0).Children(func() {
				for _, s := range a.work.Screens {
					label := a.screenLabel(s)
					row := ui.Row(c.Key(s.ID)).Shrink(0).Height(tabHeight).Radius(17).Clip().AlignItems(ui.Center)
					x, _, _ := row.PointerPosition()
					before := x < row.Bounds().W/2
					drag, over := ui.DragOver[terminalDrag](row)
					over = over && drag.ScreenID != s.ID
					if value, dropped := ui.Drop[terminalDrag](row); dropped {
						if value.ScreenID == 0 || (x > 12 && x < row.Bounds().W-12) {
							a.pendingGroupDrop = &groupDropRequest{Drag: value, Screen: s.ID}
						} else {
							a.pendingTabDrop = &tabDropRequest{Drag: value, Target: s.ID, Before: before}
						}
					}
					hovered := row.Hovered() || s.Close.Focused(c)
					alpha := float32(0)
					if a.work.activeScreen() == s {
						alpha = .065
					}
					if hovered {
						alpha += .03
					}
					joining := over && (drag.ScreenID == 0 || x > 12 && x < row.Bounds().W-12)
					if joining {
						alpha += .08
					}
					alpha = row.Animate("tab-hover", alpha, 150*time.Millisecond)
					naturalWidth := float32(0)
					if len(s.orderedPanes()) > 1 {
						for _, p := range s.orderedPanes() {
							if t := a.work.sessionForPane(p.ID); t != nil {
								textWidth, _ := c.MeasureText(0, ui.Span{Text: t.label(), Size: 13})
								naturalWidth += min(140, textWidth) + 73
							}
						}
					}
					width := float32(0)
					if active == s {
						width = naturalWidth
					}

					expansion := max(0, row.Spring("children-width", width, ui.SpringOptions{Duration: 180 * time.Millisecond, Precision: .2}))
					beveledSurface(row, a.config.Clarity, alpha, row.Animate("selected-rim", func() float32 {
						if active == s {
							return .30
						}
						return .06
					}(), 160*time.Millisecond)).Children(func() {
						ui.Row(c.Key("workspace-main")).Shrink(0).Height(tabHeight).AlignItems(ui.Center).Children(func() {
							b := ui.ButtonBase(c.Key("select")).Bind(&s.Select).Label(fmt.Sprintf("内容标签 %d", s.ID)).Height(tabHeight).Radius(17).MinWidth(70).MaxWidth(210).Padding(3, 12).FocusRing(true).
								Justify(ui.Start).Drag(terminalDrag{PaneID: s.Focused, ScreenID: s.ID}).Tooltip(label).OnClick(func() { a.selectScreen(s) })
							if reveal && s == active {
								b.ScrollIntoView()
							}
							b.Children(func() { ui.Text(c, label).FontSize(13).SingleLine().Clip() })
							b.ContextMenu(func(m *ui.Menu) {
								if m.Item("设置标签别名…").Chosen() {
									a.selectScreen(s)
									a.editScreenAlias(s, s.Select.Bounds(c))
								}
							})
							ui.ButtonBase(c.Key("close")).Bind(&s.Close).Label(fmt.Sprintf("关闭内容标签 %d", s.ID)).Size(24, 24).Margin(0, 5, 0, 0).Padding(5).Children(func() { ui.Icon(c, closeIcon).Size(12, 12) }).OnClick(func() { a.requestStopScreen(s, s.Close.Bounds(c)) })
						})
						if expansion > 0 || (active == s && len(s.orderedPanes()) > 1) {
							ui.Row(c.Key("children-viewport")).Label(fmt.Sprintf("工作区子标签 %d", s.ID)).Width(expansion).Shrink(0).MinWidth(0).Clip().Opacity(min(1, expansion/max(1, naturalWidth))).Disabled(active != s).Children(func() {
								ui.Row(c.Key("children-strip")).Shrink(0).AlignItems(ui.Center).Children(func() {
									for _, child := range s.orderedPanes() {
										owner := a.work.sessionForPane(child.ID)
										if owner == nil {
											continue
										}
										label := owner.label()
										childRow := ui.Row(c.Key(fmt.Sprintf("child-%d", child.ID))).Height(tabHeight-8).Margin(3, 4, 3, 0).Padding(2, 10).Gap(7).MaxWidth(209).Shrink(0).Radius(11).AlignItems(ui.Center)
										px, _, _ := childRow.PointerPosition()
										insertBefore := px < childRow.Bounds().W/2
										if value, dropped := ui.Drop[terminalDrag](childRow); dropped {
											a.pendingGroupDrop = &groupDropRequest{Drag: value, Screen: s.ID, Pane: child.ID, Before: insertBefore}
										}
										drag, childOver := ui.DragOver[terminalDrag](childRow)
										childOver = childOver && drag.PaneID != child.ID
										childAlpha := float32(0)
										if s.Focused == child.ID {
											childAlpha = .12
										}
										if childRow.Hovered() {
											childAlpha += .025
										}
										if childOver {
											childAlpha += .08
										}
										edge := float32(0)
										if s.Focused == child.ID {
											edge = .30
										}
										edge = childRow.Animate("selected-edge", edge, 160*time.Millisecond)
										childRow.Border(.5, ui.Transparent).BorderLinearGradient(ui.LinearGradient{From: ui.RGBA(250, 248, 255, edge), To: ui.RGBA(240, 235, 255, edge*.4), Angle: 180})
										childRow.Material(frostedSurface{Clarity: a.config.Clarity, Light: childRow.Animate("child-hover", childAlpha, 160*time.Millisecond)}).Transition(ui.ElementTransition{Duration: 180 * time.Millisecond, Position: true, Enter: &ui.Motion{Opacity: 0, Y: 3}, Exit: &ui.Motion{Opacity: 0, Scale: .96}}).Children(func() {
											childButton := ui.ButtonBase(c.Key("select")).Bind(&child.ChildTab).Label(fmt.Sprintf("子标签 %d", child.ID)).Tooltip(label).Height(22).Grow(1).MinWidth(0).Padding(0).MaxWidth(161).Drag(terminalDrag{PaneID: child.ID}).OnClick(func() { a.focusPane(child) })
											if (reveal || expansion != width) && s.Focused == child.ID {
												childRow.ScrollIntoView()
											}
											childButton.Children(func() {
												ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
													icon := commandsIcon
													if child.Browser != nil {
														icon = webIcon
													}
													ui.Icon(c, icon).Size(14, 14)
													ui.Text(c, label).FontSize(13).SingleLine().Clip().MaxWidth(140)
												})
											})
											ui.ButtonBase(c.Key("close")).Bind(&child.ChildClose).Label(fmt.Sprintf("关闭子标签 %d", child.ID)).Size(20, 20).Shrink(0).Padding(3).Children(func() { ui.Icon(c, closeIcon).Size(11, 11) }).OnClick(func() { a.requestStop(owner, child.ChildClose.Bounds(c)) })
											if childOver {
												cue := ui.Box(c.Key("insert")).Absolute().Top(3).Bottom(3).Width(2).Background(c.Theme().Text).PassThrough()
												if insertBefore {
													cue.Left(0)
												} else {
													cue.Right(0)
												}
											}
										})
									}
								})
							})
						}
						if active == s && s.WorkspaceIndex > 0 {
							a.workspaceAddButton(c, s)
						}
						if over && !joining {
							cue := ui.Box(c.Key("tab-insert")).Absolute().Top(0).Bottom(0).Width(2).Background(c.Theme().Text).PassThrough()
							if before {
								cue.Left(0)
							} else {
								cue.Right(0)
							}
						}
					})
				}
			})
			if overStrip {
				ui.Box(c.Key("tab-append")).Absolute().Right(1).Top(0).Bottom(0).Width(2).Background(c.Theme().Text).PassThrough()
			}
		})
	})
}
