package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestWorkspaceNamesAndStandaloneCreation(t *testing.T) {
	a := &application{config: defaultSettings(), start: func(*pane) {}}
	a.work.add(profile{ID: "shell"}, t.TempDir())
	tt := ui.NewTester(a.view, 1100, 720)
	tt.Advance(time.Second)
	if _, ok := tt.Find("新建终端"); ok {
		t.Fatal("standalone tab exposes a workspace-local add button")
	}
	for _, name := range []string{"Workspace", "Workspace2"} {
		if err := tt.Click("新建工作区"); err != nil {
			t.Fatal(err)
		}
		tt.Advance(time.Second)
		if err := tt.Click("启动 Terminal"); err != nil {
			t.Fatal(err)
		}
		tt.Advance(time.Second)
		s := a.work.activeScreen()
		if a.screenLabel(s) != name || s.WorkspaceIndex == 0 {
			t.Fatalf("new workspace name: %s", a.screenLabel(s))
		}
		if _, ok := tt.Find("新建终端"); !ok {
			t.Fatal("explicit workspace has no local add button")
		}
	}
	b := &application{config: a.config}
	if err := b.restoreWorkspace(a.snapshotWorkspace()); err != nil {
		t.Fatal(err)
	}
	if b.screenLabel(b.work.activeScreen()) != "Workspace2" {
		t.Fatal("default name changed across restart")
	}
	tt.Key(ui.Ctrl|ui.Shift, ui.KeyT)
	tt.Advance(time.Second)
	if err := tt.Click("启动 Terminal"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	if a.work.activeScreen().WorkspaceIndex != 0 {
		t.Fatal("independent shortcut created a workspace")
	}
	if _, ok := tt.Find("新建终端"); ok {
		t.Fatal("independent tab shows local +")
	}
	newPane := b.work.add(profile{ID: "shell"}, t.TempDir())
	b.work.markWorkspace(b.work.screenForPane(newPane.ID))
	if b.screenLabel(b.work.activeScreen()) != "Workspace3" {
		t.Fatal("workspace sequence was not persisted")
	}
}

func TestTabSpacingAndBrowserToolbarAlignment(t *testing.T) {
	a, first, second := tilingTestApp(t)
	second.Browser = &browserState{}
	a.work.dock(second.ID, first.ID, splitRight)
	tt := ui.NewTester(a.view, 1100, 720)
	tt.Advance(time.Second)
	tab, _ := tt.Find(fmt.Sprintf("内容标签 %d", a.work.activeScreen().ID))
	pane, _ := tt.Find(fmt.Sprintf("窗格 %d", first.ID))
	if top, bottom := tab.Y, pane.Y-(tab.Y+tab.H); top < 9 || bottom-top > 2 || top-bottom > 2 {
		t.Fatalf("unbalanced tab breathing room: top=%v bottom=%v", top, bottom)
	}
	icon, _ := tt.Find(fmt.Sprintf("窗格 %d", second.ID))
	address, _ := tt.Find("当前网页地址")
	back, _ := tt.Find("网页后退")
	center := address.Y + address.H/2
	for _, r := range []ui.Rect{icon, back} {
		if d := r.Y + r.H/2 - center; d > 1 || d < -1 {
			t.Fatalf("browser controls not centered: %v %v", r, address)
		}
	}
	saveInteractionFrame(t, tt, "presentation-spacing-browser")
}
