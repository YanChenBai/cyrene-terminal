package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func assertUniquePanes(t *testing.T, w *workspace) {
	t.Helper()
	seen := map[int]int{}
	for _, s := range w.Screens {
		if s.Root == nil || s.Root.find(s.Focused) == nil {
			t.Fatal("screen has no valid focused pane")
		}
		for _, p := range s.Root.leaves() {
			seen[p.ID]++
		}
	}
	for _, session := range w.Tabs {
		for _, p := range session.Root.leaves() {
			if seen[p.ID] != 1 {
				t.Fatalf("pane %d appears %d times", p.ID, seen[p.ID])
			}
		}
	}
	if len(seen) != len(w.Tabs) {
		t.Fatal("orphan pane in screen")
	}
}

func TestDockRearrangeDetachAndClosePreserveSessionOwnership(t *testing.T) {
	w := workspace{}
	a := w.add(profile{ID: "shell"}, "alpha")
	first := w.active()
	first.GroupID, first.Alias = "alpha", "Build"
	b := w.add(profile{ID: "codex"}, "beta")
	second := w.active()
	second.GroupID, second.Alias = "beta", "Agent"
	c := w.add(profile{ID: "shell"}, "alpha")
	if !w.dock(b.ID, a.ID, splitRight) || !w.dock(c.ID, a.ID, splitBottom) {
		t.Fatal("cross-group dock failed")
	}
	if len(w.Tabs) != 3 || len(w.Screens) != 1 || first.Root.Pane != a || second.Root.Pane != b || second.GroupID != "beta" || b.Dir != "beta" {
		t.Fatal("docking changed session ownership")
	}
	assertUniquePanes(t, &w)
	if !w.dock(c.ID, b.ID, splitTop) {
		t.Fatal("rearranging a screen failed")
	}
	assertUniquePanes(t, &w)
	if w.dock(b.ID, b.ID, splitLeft) || w.dock(999, b.ID, splitLeft) {
		t.Fatal("invalid drag mutated layout")
	}
	if !w.detach(c.ID) || len(w.Screens) != 2 {
		t.Fatal("detach did not create a screen")
	}
	assertUniquePanes(t, &w)
	w.focusPane(b.ID)
	w.removeTab(second.ID)
	if w.active() != first || first.Alias != "Build" {
		t.Fatal("closing one pane did not focus a surviving neighbor")
	}
	assertUniquePanes(t, &w)
}

func tilingTestApp(t *testing.T) (*application, *pane, *pane) {
	t.Helper()
	a := &application{config: defaultSettings()}
	a.config.Groups = []folderGroup{{ID: "alpha", Name: "Alpha", Directory: t.TempDir()}, {ID: "beta", Name: "Beta", Directory: t.TempDir()}}
	first := a.work.add(profile{ID: "shell", Name: "Terminal"}, a.config.Groups[0].Directory)
	a.work.active().Alias, a.work.active().GroupID = "Build", "alpha"
	second := a.work.add(profile{ID: "codex", Name: "Codex"}, a.config.Groups[1].Directory)
	a.work.active().Alias, a.work.active().GroupID = "Review", "beta"
	a.work.focusPane(first.ID)
	return a, first, second
}

func TestDragSessionIntoRightHalfAndCancelDrag(t *testing.T) {
	a, first, second := tilingTestApp(t)
	starts := 0
	a.start = func(*pane) { starts++ }
	tt := ui.NewTester(a.view, 1100, 720)
	// The inactive center tab is also a drag source. The sidebar and pane
	// headers offer the same typed drag, without touching terminal selection.
	source, _ := tt.Find(fmt.Sprintf("内容标签 %d", a.work.sessionForPane(second.ID).ID))
	target, _ := tt.Find("正在启动终端…")
	if target.W == 0 {
		t.Fatal("target pane missing")
	}
	tt.Press(source.X+source.W/2, source.Y+source.H/2)
	tt.Move(1020, 350)
	tt.Advance(180 * time.Millisecond)
	if !tt.HasText("向右分屏") {
		t.Fatal("drag preview was not shown")
	}
	saveInteractionFrame(t, tt, "tiling-drop-preview")
	tt.Release(1020, 350)
	tt.Advance(300 * time.Millisecond)
	if len(a.work.Screens) != 1 || len(a.work.activeScreen().Root.leaves()) != 2 || starts != 0 {
		t.Fatal("drop must move the existing process without launching another")
	}
	root := a.work.activeScreen().Root
	if root.Vertical || root.First.Pane != first || root.Second.Pane != second {
		t.Fatal("drop used wrong split direction")
	}
	for range 4 {
		tt.Frame()
	}
	saveInteractionFrame(t, tt, "tiling-two-panes")
	header, _ := tt.Find(fmt.Sprintf("窗格 %d", second.ID))
	tt.Press(header.X+20, header.Y+10)
	tt.Move(280, 400)
	tt.Key(0, ui.KeyEscape)
	tt.Release(280, 400)
	if len(a.work.Screens) != 1 || root.First.Pane != first {
		t.Fatal("Escape must cancel a drag")
	}
}

func TestCommandPaletteKeyboardSplitAcrossGroups(t *testing.T) {
	a, first, second := tilingTestApp(t)
	tt := ui.NewTester(a.view, 1100, 720)
	tt.Key(ui.Ctrl|ui.Shift, ui.KeyP)
	tt.Advance(400 * time.Millisecond)
	tt.Type("split right")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.palette.Stage != "split" || a.palette.Target != first.ID {
		t.Fatalf("keyboard command did not open split chooser: open=%v stage=%q target=%d query=%q texts=%v", a.palette.Open, a.palette.Stage, a.palette.Target, a.palette.Query, tt.Texts())
	}
	tt.Type("Review")
	saveInteractionFrame(t, tt, "tiling-command-palette")
	tt.Key(0, ui.KeyEnter)
	tt.Advance(400 * time.Millisecond)
	if a.palette.Open || len(a.work.Screens) != 1 || a.work.activeScreen().Focused != second.ID {
		t.Fatal("keyboard split did not move the selected session")
	}
	tt.Key(ui.Ctrl|ui.Shift, ui.KeyP)
	tt.Advance(400 * time.Millisecond)
	tt.Type("detach")
	tt.Key(0, ui.KeyEnter)
	tt.Advance(400 * time.Millisecond)
	if len(a.work.Screens) != 2 || a.work.active().GroupID != "beta" {
		t.Fatal("keyboard detach lost process or group")
	}
	assertUniquePanes(t, &a.work)
}

func TestCloseScreenConfirmsAllAndClosePaneOnlyStopsOne(t *testing.T) {
	a, first, second := tilingTestApp(t)
	a.work.dock(second.ID, first.ID, splitRight)
	tt := ui.NewTester(a.view, 1100, 720)
	if err := tt.Click(fmt.Sprintf("关闭内容标签 %d", a.work.activeScreen().ID)); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	if a.stopScreen == nil || len(a.stopTabs()) != 2 {
		t.Fatal("screen close must confirm every terminal")
	}
	if err := tt.Click("取消"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	if err := tt.Click(fmt.Sprintf("关闭窗格 %d", second.ID)); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	if err := tt.Click("终止进程"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	if len(a.work.Tabs) != 1 || a.work.active().Root.Pane != first || len(a.work.Screens) != 1 {
		t.Fatal("closing a pane terminated another session")
	}
	for range 4 {
		tt.Frame()
	}
}

func TestPaletteArrowsNewSplitDirectoryAndMinimumWindow(t *testing.T) {
	a, first, _ := tilingTestApp(t)
	starts := 0
	a.start = func(p *pane) {
		starts++
		if p.Dir != first.Dir || a.work.sessionForPane(p.ID).GroupID != "alpha" {
			t.Fatal("new split lost target directory or group")
		}
	}
	tt := ui.NewTester(a.view, 1000, 720)
	tt.Key(ui.Ctrl|ui.Shift, ui.KeyP)
	tt.Advance(400 * time.Millisecond)
	tt.Key(0, ui.KeyDown)
	if a.palette.Selected != 1 {
		t.Fatal("Down must move selection without leaving search")
	}
	tt.Key(0, ui.KeyUp)
	if a.palette.Selected != 0 {
		t.Fatal("Up must return selection")
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Type("new Terminal")
	tt.Key(0, ui.KeyEnter)
	tt.Advance(400 * time.Millisecond)
	if starts != 1 || len(a.work.activeScreen().Root.leaves()) != 2 {
		t.Fatal("keyboard new split failed")
	}
	small := ui.NewTester(a.view, 720, 440)
	small.Key(ui.Ctrl|ui.Shift, ui.KeyP)
	small.Advance(400 * time.Millisecond)
	search, ok := small.Find("命令搜索")
	if !ok || search.X < 0 || search.Y < 32 || search.X+search.W > 720 || search.Y+search.H > 440 {
		t.Fatalf("palette does not fit minimum window: %+v", search)
	}
	saveInteractionFrame(t, small, "tiling-minimum-palette")
	small.Key(0, ui.KeyEscape)
	small.Advance(400 * time.Millisecond)
	if a.palette.Open {
		t.Fatal("Escape must close palette")
	}
}

func TestSplitDividerCanBeDraggedAndBalanced(t *testing.T) {
	a, first, second := tilingTestApp(t)
	a.work.dock(second.ID, first.ID, splitRight)
	tt := ui.NewTester(a.view, 1100, 720)
	for range 3 {
		tt.Frame()
	}
	divider, ok := tt.Find("Divider")
	if !ok {
		t.Fatal("native divider missing")
	}
	tt.Press(divider.X+divider.W/2, divider.Y+divider.H/2)
	tt.Move(divider.X-70, divider.Y+divider.H/2)
	tt.Advance(80 * time.Millisecond)
	tt.Release(divider.X-70, divider.Y+divider.H/2)
	if a.work.activeScreen().Root.Ratio >= 49 {
		t.Fatal("split divider did not resize")
	}
	balanceLayout(a.work.activeScreen().Root)
	tt.Frame()
	if a.work.activeScreen().Root.Ratio != 50 {
		t.Fatal("balance did not reset split")
	}
}
