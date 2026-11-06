package app

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestBrowserURLs(t *testing.T) {
	for raw, want := range map[string]string{
		"localhost:3000/demo":       "http://localhost:3000/demo",
		"127.0.0.1:8080":            "http://127.0.0.1:8080",
		"[::1]:8080":                "http://[::1]:8080",
		" example.com/docs?q=go ":   "https://example.com/docs?q=go",
		"https://example.com/#hash": "https://example.com/#hash",
	} {
		got, err := normalizeBrowserURL(raw)
		if err != nil || got != want {
			t.Fatalf("%q: %q, %v", raw, got, err)
		}
	}
	for _, raw := range []string{"", "two words", "javascript:alert(1)", "file:///C:/secret", "ftp://example.com", "https://", "https://reader@example.com"} {
		if _, err := normalizeBrowserURL(raw); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
}

func TestExpandedWorkspaceChildrenFollowSelection(t *testing.T) {
	a, first, second := tilingTestApp(t)
	a.work.dock(second.ID, first.ID, splitRight)
	composite := a.work.activeScreen()
	a.work.add(profile{ID: "shell"}, t.TempDir())
	other := a.work.activeScreen()
	a.selectScreen(composite)
	tt := ui.NewTester(a.view, 1100, 720)
	for _, p := range []*pane{first, second} {
		if _, ok := tt.Find(fmt.Sprintf("子标签 %d", p.ID)); !ok {
			t.Fatal("selected workspace did not expand")
		}
	}
	if err := tt.Click(fmt.Sprintf("子标签 %d", first.ID)); err != nil {
		t.Fatal(err)
	}
	if composite.Focused != first.ID || a.work.active() != a.work.sessionForPane(first.ID) {
		t.Fatal("child did not focus its pane")
	}
	if err := tt.Click(fmt.Sprintf("内容标签 %d", other.ID)); err != nil {
		t.Fatal(err)
	}
	tt.Advance(400 * time.Millisecond)
	if _, ok := tt.Find(fmt.Sprintf("子标签 %d", first.ID)); ok {
		t.Fatal("inactive workspace is still expanded")
	}
	if err := tt.Click(fmt.Sprintf("内容标签 %d", composite.ID)); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find(fmt.Sprintf("子标签 %d", first.ID)); !ok {
		t.Fatal("returning workspace did not expand")
	}
	tt.Advance(400 * time.Millisecond)
	for _, label := range []string{"工作区", "命令面板", "切换侧栏", "切换右侧栏", "调整侧栏宽度", "刷新文件树"} {
		if _, ok := tt.Find(label); ok {
			t.Fatalf("obsolete sidebar UI: %s", label)
		}
	}
	saveInteractionFrame(t, tt, "workbench-subtabs")
}

func TestBrowserMixedWorkspacePersistsAndClosesWhileStarting(t *testing.T) {
	a, first, second := tilingTestApp(t)
	a.work.dock(second.ID, first.ID, splitRight)
	a.browserTarget, a.browserEdge = first.ID, splitBottom
	if err := a.launchBrowser("localhost:3000/demo"); err != nil {
		t.Fatal(err)
	}
	p := a.work.active().Root.Pane
	p.Browser.Title = "Local preview"
	if len(a.work.activeScreen().Root.leaves()) != 3 || p.Dir != first.Dir || a.work.active().GroupID != "alpha" {
		t.Fatal("mixed split lost target directory/group")
	}
	if a.work.active().label() != "Local preview" {
		t.Fatal("page title not reflected in tab")
	}
	state := a.snapshotWorkspace()
	b := &application{config: defaultSettings()}
	if err := b.restoreWorkspace(state); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(state, b.snapshotWorkspace()) {
		t.Fatal("browser mixed workspace did not round-trip")
	}
	bp := b.work.sessionForPane(p.ID).Root.Pane
	if bp.Browser == nil || bp.Browser.URL != "http://localhost:3000/demo" || bp.Term != nil {
		t.Fatal("web pane restored as a terminal")
	}
	p.Starting = true
	a.requestStop(a.work.sessionForPane(p.ID), ui.Rect{})
	if a.stopConfirm || a.work.sessionForPane(p.ID) != nil || a.work.sessionForPane(first.ID) == nil {
		t.Fatal("closing pending browser blocked or affected terminal")
	}
	assertUniquePanes(t, &a.work)
}

func TestBlankBrowserFromSplitPalette(t *testing.T) {
	a, first, _ := tilingTestApp(t)
	tt := ui.NewTester(a.view, 1100, 720)
	a.openSplitPalette(first.ID, splitRight)
	tt.Frame()
	tt.Advance(500 * time.Millisecond)
	tt.Type("new web")
	tt.Key(0, ui.KeyEnter)
	tt.Advance(500 * time.Millisecond)
	if a.work.active().Root.Pane.Browser == nil || len(a.work.activeScreen().Root.leaves()) != 2 {
		t.Fatal("split command did not create blank browser")
	}
	if _, ok := tt.Find("网页地址"); ok {
		t.Fatal("new browser opened a modal")
	}
	tt.Type("localhost:3000")
	tt.Key(0, ui.KeyEnter)
	tt.Advance(500 * time.Millisecond)
	if a.work.active().Root.Pane.Browser.URL != "http://localhost:3000" {
		t.Fatal("address bar did not navigate")
	}

}

func TestExpandedChildrenScrollAndLegacySidebarStateIsIgnored(t *testing.T) {
	a := &application{config: defaultSettings(), collapsed: true}
	first := a.work.add(profile{ID: "shell"}, t.TempDir())
	for i := 0; i < 18; i++ {
		p := a.work.add(profile{ID: "shell"}, t.TempDir())
		a.work.active().Alias = fmt.Sprintf("Pane %d", i)
		a.work.dock(p.ID, first.ID, splitRight)
	}
	a.work.focusPane(a.work.activeScreen().Root.leaves()[18].ID)
	tt := ui.NewTester(a.view, 1100, 720)
	tt.Advance(time.Second)
	s := a.work.activeScreen()
	last, ok := tt.Find(fmt.Sprintf("子标签 %d", s.Focused))
	tools, _ := tt.Find("新建终端")
	if !ok || a.tabScroll.X <= 0 || last.X+last.W > tools.X {
		t.Fatalf("selected child overflowed: %+v, scroll %+v", last, a.tabScroll)
	}
	a.focusPane(first)
	tt.Frame()
	tt.Advance(time.Second)
	child, _ := tt.Find(fmt.Sprintf("子标签 %d", first.ID))
	if child.X < 8 || child.X+child.W > tools.X {
		t.Fatal("focus change did not reveal child")
	}
	tt.Key(ui.Ctrl|ui.Shift, ui.KeyT)
	tt.Advance(time.Second)
	if _, ok := tt.Find("启动 Codex"); !ok {
		t.Fatal("legacy sidebar state hid global launcher")
	}
}

func TestWorkspaceExpansionReversesWithoutJumping(t *testing.T) {
	a, first, second := tilingTestApp(t)
	a.work.dock(second.ID, first.ID, splitRight)
	s := a.work.activeScreen()
	a.work.add(profile{ID: "shell"}, t.TempDir())
	other := a.work.activeScreen()
	a.selectScreen(s)
	tt := ui.NewTester(a.view, 1100, 720)
	tt.Advance(400 * time.Millisecond)
	label := fmt.Sprintf("工作区子标签 %d", s.ID)
	full, _ := tt.Find(label)
	tt.Click(fmt.Sprintf("内容标签 %d", other.ID))
	tt.Advance(80 * time.Millisecond)
	tt.Frame()
	mid, ok := tt.Find(label)
	if !ok || mid.W <= 0 || mid.W >= full.W {
		t.Fatalf("collapse skipped intermediate frames: %+v -> %+v", full, mid)
	}
	a.selectScreen(s)
	tt.Frame()
	reversed, _ := tt.Find(label)
	if reversed.W != mid.W {
		t.Fatalf("reversal jumped: %+v -> %+v", mid, reversed)
	}
	tt.Advance(400 * time.Millisecond)
	settled, _ := tt.Find(label)
	if settled.W != full.W {
		t.Fatalf("expansion did not settle: %+v -> %+v", full, settled)
	}
	saveInteractionFrame(t, tt, "glass-workspace-subtabs")
}

func TestBlankBrowserDoesNotStartProcessAndSurvivesRestart(t *testing.T) {
	a := &application{config: defaultSettings()}
	starts := 0
	a.start = func(*pane) { starts++ }
	a.newBrowser(0, splitRight)
	p := a.work.active().Root.Pane
	if starts != 0 || p.Browser == nil || p.Browser.URL != "" {
		t.Fatal("blank browser started work")
	}
	tt := ui.NewTester(a.view, 1000, 720)
	if !tt.HasText("在地址栏输入网址") {
		t.Fatal("missing browser empty state")
	}
	if _, ok := tt.Find("网页地址"); ok {
		t.Fatal("unexpected browser modal")
	}
	for _, label := range []string{"工作区", "命令面板"} {
		if _, ok := tt.Find(label); ok {
			t.Fatal("unexpected header control", label)
		}
	}
	b := &application{config: defaultSettings()}
	if err := b.restoreWorkspace(a.snapshotWorkspace()); err != nil {
		t.Fatal(err)
	}
	if b.work.active().Root.Pane.Browser == nil {
		t.Fatal("empty website restored as shell")
	}
	tt.Type("localhost:3000")
	tt.Key(0, ui.KeyEnter)
	if p.Browser.URL != "http://localhost:3000" {
		t.Fatal("blank address did not receive focus")
	}
	saveInteractionFrame(t, tt, "glass-browser-empty")
}
