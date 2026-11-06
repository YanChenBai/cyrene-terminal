package app

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
)

func (a *application) layoutView(c *ui.Context, n *layout, tiled bool) {
	if n == nil {
		return
	}
	if n.Pane == nil {
		// The native splitter retains its mouse/keyboard hit area. Only its
		// paint uses transparent colors; restore the regular theme in panes.
		theme := c.Theme()
		dividerTheme := *theme
		dividerTheme.Border, dividerTheme.Accent = ui.Transparent, ui.Transparent
		first := func() {
			ui.Themed(c, theme).Fill().Background(ui.Transparent).Children(func() { a.layoutView(c, n.First, true) })
		}
		second := func() {
			ui.Themed(c, theme).Fill().Background(ui.Transparent).Children(func() { a.layoutView(c, n.Second, true) })
		}
		ui.Themed(c.Key(fmt.Sprintf("split-%d", n.ID)), &dividerTheme).Fill().Background(ui.Transparent).Children(func() {
			if n.Vertical {
				ui.SplitVerticalPercent(c.Key(n.ID), &n.Ratio, first, second).Fill().MinWidth(0).MinHeight(0)
			} else {
				ui.SplitPercent(c.Key(n.ID), &n.Ratio, first, second).Fill().MinWidth(0).MinHeight(0)
			}
		})
		return
	}
	p := n.Pane
	if p.Handle.Focused(c) || p.Header.Focused(c) || (p.Browser != nil && p.Browser.Address.Focused(c)) {
		a.work.focusPane(p.ID)
	}
	surface := ui.Column(c.Key(fmt.Sprintf("pane-%d", p.ID))).Bind(&p.Surface).Grow(1).MinWidth(0).MinHeight(0).Clip()
	active := false
	if t := a.work.active(); t != nil {
		active = t.Focused == p.ID
	}
	light := float32(.008)
	if active {
		light = .032
	}
	light = surface.Animate("surface-light", light, 180*time.Millisecond)
	strength := float32(.10)
	if active {
		strength = .44
	}
	strength = surface.Animate("focus-rim", strength, 180*time.Millisecond)
	beveledSurface(surface.Margin(2, 3, 2, 3).Radius(12).Transition(paneTransition(p.ID)), a.config.Clarity, light, strength)
	if !p.Entered {
		surface.Jump("entrance", 1)
		if a.rendered {
			surface.Jump("entrance", 0)
		}
		p.Entered = true
	}
	surface.Opacity(surface.Animate("entrance", 1, 180*time.Millisecond))
	p.ShowActions = active || surface.Hovered() || p.Actions.FocusWithin(c)

	drag, over := ui.DragOver[terminalDrag](surface)
	x, y, _ := surface.PointerPosition()
	bounds := p.Surface.Bounds(c)
	valid := over && drag.PaneID != p.ID
	if valid {
		p.DropEdge = nearestSplitEdge(x, y, bounds.W, bounds.H)
	}
	if value, dropped := ui.Drop[terminalDrag](surface); dropped && value.PaneID != p.ID {
		edge := nearestSplitEdge(x, y, bounds.W, bounds.H)
		if roomForSplit(bounds, edge) {
			a.pendingDock = &dockRequest{value.PaneID, p.ID, edge}
		} else {
			a.status = "窗格空间不足，请先调整窗口或分隔线。"
		}
	}
	surface.Children(func() {
		if p.Browser == nil {
			a.paneHeader(c, p, active)
		}
		a.terminalView(c, p)
		alpha := float32(0)
		if valid {
			alpha = 1
		}
		alpha = surface.Animate("drop-preview", alpha, 110*time.Millisecond)
		if alpha > .001 {
			preview := ui.Box(c.Key("split-preview")).Absolute().Left(5).Top(5).Right(5).Bottom(5).Radius(8).
				Background(ui.RGBA(150, 178, 224, .15*alpha)).Border(1, ui.RGBA(170, 197, 240, .65*alpha)).PassThrough().Center()
			switch p.DropEdge {
			case splitLeft:
				preview.Right(bounds.W / 2)
			case splitRight:
				preview.Left(bounds.W / 2)
			case splitTop:
				preview.Bottom(bounds.H / 2)
			case splitBottom:
				preview.Top(bounds.H / 2)
			}
			label := p.DropEdge.label()
			if !roomForSplit(bounds, p.DropEdge) {
				label = "空间不足"
			}
			preview.Children(func() {
				ui.Text(c, label).FontSize(12).Padding(7, 12).Radius(7).Background(ui.RGBA(18, 23, 32, .85*alpha)).Opacity(alpha)
			})
		}
	})
}

func nearestSplitEdge(x, y, w, h float32) splitEdge {
	if w <= 0 || h <= 0 {
		return splitRight
	}
	x, y = x/w, y/h
	edge, distance := splitLeft, x
	if 1-x < distance {
		edge, distance = splitRight, 1-x
	}
	if y < distance {
		edge, distance = splitTop, y
	}
	if 1-y < distance {
		edge = splitBottom
	}
	return edge
}

func roomForSplit(bounds ui.Rect, edge splitEdge) bool {
	if edge == splitTop || edge == splitBottom {
		return bounds.H >= 240 && bounds.W >= 180
	}
	return bounds.W >= 360 && bounds.H >= 120
}

func (a *application) paneHeader(c *ui.Context, p *pane, active bool) {
	t := a.work.sessionForPane(p.ID)
	if t == nil {
		return
	}
	row := ui.Row(c.Key("pane-header")).Height(32).Shrink(0).Padding(0, 3).AlignItems(ui.Center)
	show := p.ShowActions || p.PaneClose.Focused(c)
	row.Children(func() {
		b := ui.ButtonBase(c.Key("pane-drag")).Bind(&p.Header).Label(fmt.Sprintf("窗格 %d", p.ID)).Drag(terminalDrag{PaneID: p.ID}).
			Height(32).Grow(1).MinWidth(0).Padding(0, 7).FocusRing(false).Justify(ui.Start).OnClick(func() { a.focusPane(p) })
		b.Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				iconID := p.Profile.agentName()
				if iconID == "" {
					iconID = p.Profile.ID
				}
				if agentLogos[iconID] != nil {
					agentLogo(c, iconID, 17)
				}
				text := ui.Text(c, t.label()).FontSize(13).SingleLine().Clip()
				if !active {
					text.TextColor(c.Theme().TextMuted)
				}
				if group := a.groupLabel(t.GroupID); group != "" {
					ui.Text(c, group).FontSize(10).TextColor(c.Theme().TextMuted).SingleLine().Clip()
				}
			})
		})
		b.ContextMenu(func(m *ui.Menu) {
			if m.Item("移到独立标签").Chosen() {
				a.work.detach(p.ID)
				a.focusPane(p)
			}
			if m.Item("均分当前画面").Chosen() {
				if s := a.work.screenForPane(p.ID); s != nil {
					balanceLayout(s.Root)
				}
			}
		})
		alpha := float32(0)
		if show {
			alpha = 1
		}
		alpha = row.Animate("actions-visible", alpha, 170*time.Millisecond)
		ui.Row(c.Key("pane-actions")).Bind(&p.Actions).Width(78 * alpha).Shrink(0).Clip().Opacity(alpha).Disabled(!show).Gap(2).Children(func() {
			iconButton(c.Key("split-right"), panelIcon, fmt.Sprintf("向右分屏 %d", p.ID)).Size(24, 24).OnClick(func() { a.openSplitPalette(p.ID, splitRight) })
			iconButton(c.Key("split-down"), splitDownIcon, fmt.Sprintf("向下分屏 %d", p.ID)).Size(24, 24).OnClick(func() { a.openSplitPalette(p.ID, splitBottom) })
			ui.ButtonBase(c.Key("pane-close")).Bind(&p.PaneClose).Label(fmt.Sprintf("关闭窗格 %d", p.ID)).Size(24, 24).Padding(5).Children(func() { ui.Icon(c, closeIcon).Size(12, 12) }).OnClick(func() { a.requestStop(t, p.PaneClose.Bounds(c)) })
		})
	})
}
