package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestScreenReorderPreservesWholeLayoutAndAlias(t *testing.T) {
	a, first, second := tilingTestApp(t)
	a.work.dock(second.ID, first.ID, splitRight)
	composite := a.work.activeScreen()
	composite.Alias = "Workspace"
	root := composite.Root
	a.work.add(profile{ID: "shell"}, t.TempDir())
	other := a.work.activeScreen()
	a.dropOnTabStrip(tabDropRequest{Drag: terminalDrag{PaneID: second.ID, ScreenID: composite.ID}, Target: other.ID, Before: false})
	if len(a.work.Screens) != 2 || a.work.Screens[1] != composite || composite.Root != root || composite.Alias != "Workspace" {
		t.Fatal("tab reorder detached a pane or lost screen identity")
	}
	if a.work.reorderScreen(composite.ID, 999, true) || a.work.Screens[1] != composite {
		t.Fatal("invalid target changed screen order")
	}
	assertUniquePanes(t, &a.work)
}

func TestDragPaneBackToTabsAndReorder(t *testing.T) {
	a, first, second := tilingTestApp(t)
	a.work.dock(second.ID, first.ID, splitRight)
	original := a.work.activeScreen()
	original.Alias = "Builds"
	a.work.add(profile{ID: "shell"}, t.TempDir())
	a.work.focusPane(first.ID)
	starts := 0
	a.start = func(*pane) { starts++ }
	tt := ui.NewTester(a.view, 1100, 720)
	header, ok := tt.Find(fmt.Sprintf("窗格 %d", second.ID))
	if !ok {
		t.Fatal("pane drag handle missing")
	}
	tt.Press(header.X+20, header.Y+12)
	tt.Move(950, 16)
	tt.Advance(150 * time.Millisecond)
	tt.Release(950, 16)
	tt.Advance(300 * time.Millisecond)
	detached := a.work.screenForPane(second.ID)
	if len(a.work.Screens) != 3 || detached == original || original.Alias != "Builds" || starts != 0 {
		t.Fatal("dragging pane to strip did not detach existing terminal")
	}
	from, _ := tt.Find(fmt.Sprintf("内容标签 %d", detached.ID))
	to, _ := tt.Find(fmt.Sprintf("内容标签 %d", original.ID))
	tt.Press(from.X+20, from.Y+16)
	tt.Move(to.X+2, to.Y+16)
	tt.Advance(150 * time.Millisecond)
	tt.Release(to.X+2, to.Y+16)
	tt.Advance(300 * time.Millisecond)
	if a.work.Screens[0] != detached || a.work.sessionForPane(second.ID).GroupID != "beta" || starts != 0 {
		t.Fatal("tab drag did not reorder without restarting")
	}
	assertUniquePanes(t, &a.work)
	saveInteractionFrame(t, tt, "tabs-flat-reordered")
}

func TestPaletteRenamesOnlyFocusedScreenOrSession(t *testing.T) {
	a, first, second := tilingTestApp(t)
	a.work.dock(second.ID, first.ID, splitRight)
	focused := a.work.activeScreen()
	firstSession, secondSession := a.work.sessionForPane(first.ID), a.work.sessionForPane(second.ID)
	tt := ui.NewTester(a.view, 1100, 720)
	rename := func(query, value string) {
		t.Helper()
		tt.Key(ui.Ctrl|ui.Shift, ui.KeyP)
		tt.Advance(400 * time.Millisecond)
		tt.Type(query)
		tt.Key(0, ui.KeyEnter)
		tt.Advance(400 * time.Millisecond)
		if !a.aliasEditor {
			t.Fatal("rename command did not open editor")
		}
		tt.Key(ui.Ctrl, ui.KeyA)
		tt.Type(value)
		if a.aliasDraft != value {
			t.Fatalf("rename field did not receive keyboard focus: %q", a.aliasDraft)
		}
		if err := tt.Click("保存"); err != nil {
			t.Fatal(err)
		}
		tt.Advance(400 * time.Millisecond)
	}
	rename("rename tab", "Development")
	if focused.Alias != "Development" || firstSession.Alias != "Build" || secondSession.Alias != "Review" {
		t.Fatal("tab rename changed session aliases")
	}
	a.focusPane(first)
	tt.Frame()
	rename("rename session", "Builder")
	if firstSession.Alias != "Builder" || secondSession.Alias != "Review" || focused.Alias != "Development" {
		t.Fatal("session rename changed another session or screen")
	}
	if a.screenLabel(focused) != "Development · 2" {
		t.Fatal("custom tab name follows pane focus")
	}
	for range 4 {
		tt.Frame()
	}
}

func TestCompactWorkspaceTabsAndSplitterHasNoPaintedLine(t *testing.T) {
	a, first, second := tilingTestApp(t)
	tt := ui.NewTester(a.view, 1100, 720)
	left, _ := tt.Find(fmt.Sprintf("内容标签 %d", a.work.sessionForPane(first.ID).ID))
	leftClose, _ := tt.Find(fmt.Sprintf("关闭内容标签 %d", a.work.sessionForPane(first.ID).ID))
	right, _ := tt.Find(fmt.Sprintf("内容标签 %d", a.work.sessionForPane(second.ID).ID))
	if left.Y < 9 || left.H != 33 || right.X-left.X-left.W < 4 || right.X-left.X-left.W > 70 || leftClose.X < left.X+left.W || leftClose.X+leftClose.W > right.X || leftClose.Y+leftClose.H/2 != left.Y+left.H/2 {
		t.Fatalf("workspace tabs or reserved close geometry is wrong: %+v %+v %+v", left, leftClose, right)
	}
	a.work.dock(second.ID, first.ID, splitRight)
	tt.Frame()
	tt.Move(900, 400)
	tt.Advance(300 * time.Millisecond)
	divider, ok := tt.Find("Divider")
	if !ok {
		t.Fatal("invisible resize handle disappeared")
	}
	img := tt.Image()
	y := int(divider.Y + divider.H*.7)
	center := img.RGBAAt(int(divider.X), y)
	if center != img.RGBAAt(int(divider.X+divider.W/2), y) {
		t.Fatal("splitter still paints a line")
	}
	saveInteractionFrame(t, tt, "panes-rounded-tabs-compact")
}
