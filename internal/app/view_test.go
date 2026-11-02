package app

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestGroupedLauncherAndExtraFrames(t *testing.T) {
	a := &application{config: defaultSettings()}
	starts := 0
	a.start = func(*pane) { starts++ }
	tt := ui.NewTester(a.view, 1000, 720)
	click := func(label string) {
		t.Helper()
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
		tt.Advance(500 * time.Millisecond)
	}
	click("新建文件夹分组")
	dir := t.TempDir()
	a.groupName, a.groupDirectory = "Project", dir
	click("保存")
	if len(a.config.Groups) != 1 || a.groupEditor {
		t.Fatal("group should be created")
	}
	click("在 Project 新建终端")
	button, _ := tt.Find("在 Project 新建终端")
	item, ok := tt.Find("启动 Codex")
	if !ok || item.Y < button.Y+button.H {
		t.Fatalf("launcher must open below its anchor: anchor=%+v item=%+v", button, item)
	}
	click("启动 Codex")
	groupID := a.config.Groups[0].ID
	if a.work.active().GroupID != groupID || a.work.active().Root.Pane.Dir != dir {
		t.Fatal("group directory was not inherited")
	}
	if a.work.active().label() != filepath.Base(dir) {
		t.Fatal("fallback title must use directory, not agent name")
	}
	if tt.HasText("左右分屏") || tt.HasText("上下分屏") || tt.HasText("/ terminal") {
		t.Fatal("old layout should be removed")
	}
	click("新建终端")
	click("启动 Terminal")
	home, _ := os.UserHomeDir()
	if a.work.active().Root.Pane.Dir != home || a.work.active().GroupID != "" {
		t.Fatal("ungrouped terminal must start in user home")
	}
	click("切换侧栏")
	tt.Advance(300 * time.Millisecond)
	click("切换侧栏")
	tt.Advance(300 * time.Millisecond)
	if _, ok := tt.Find("后台会话"); ok {
		t.Fatal("standalone background UI must be removed")
	}
	tt.Key(ui.Ctrl|ui.Shift, ui.KeyW)
	tt.Advance(500 * time.Millisecond)
	if !a.stopConfirm || len(a.work.Tabs) != 2 {
		t.Fatal("close shortcut must ask before stopping the current terminal")
	}
	click("取消")
	if len(a.work.Tabs) != 2 || starts != 2 {
		t.Fatal("canceling close must preserve the sessions")
	}
	for range 3 {
		tt.Frame()
	}
	tt.Key(ui.Ctrl|ui.Shift, ui.KeyT)
	tt.Key(0, ui.KeyEscape)
	if a.launcher {
		t.Fatal("Escape should close the popover")
	}
	tt.Key(ui.Ctrl|ui.Shift, ui.KeyD)
	if a.launcher {
		t.Fatal("split shortcut should be disabled")
	}
	click("外观设置")
	click("取消")
	for range 3 {
		tt.Frame()
	}
}

func TestGroupChevronSpacingAndTreeGuide(t *testing.T) {
	a := &application{config: defaultSettings()}
	dir := filepath.Join(t.TempDir(), "workspace")
	a.config.Groups = []folderGroup{{ID: "alpha", Name: "Alpha", Directory: dir}, {ID: "beta", Name: "Beta", Directory: dir}}
	a.work.add(profile{ID: "shell", Name: "Terminal"}, dir)
	a.work.active().GroupID = "alpha"
	a.work.add(profile{ID: "shell", Name: "Terminal"}, dir)
	a.work.active().GroupID = "beta"
	tt := ui.NewTester(a.view, 1000, 720)
	header, ok := tt.Find("分组 Alpha")
	if !ok {
		t.Fatal("missing group header")
	}
	text, ok := tt.Find("Alpha")
	if !ok || math.Abs(float64(text.X-(header.X+6+11+7))) > .5 {
		t.Fatalf("chevron-label gap must be 7 DIP: header=%+v text=%+v", header, text)
	}
	saveInteractionFrame(t, tt, "group-tree")
	before := text.X
	betaBefore, _ := tt.Find("分组 Beta")
	_ = tt.Click("分组 Alpha")
	tt.Advance(60 * time.Millisecond)
	if a.groupState("alpha").Motion.Value <= 0 || a.groupState("alpha").Motion.Value >= 1 {
		t.Fatal("group must animate through intermediate heights")
	}
	betaMid, _ := tt.Find("分组 Beta")
	if betaMid.Y >= betaBefore.Y || betaMid.Y <= header.Y+30 {
		t.Fatalf("neighbor must move gradually: %v -> %v", betaBefore.Y, betaMid.Y)
	}
	saveInteractionFrame(t, tt, "group-closing")
	tt.Advance(200 * time.Millisecond)
	text, _ = tt.Find("Alpha")
	if text.X != before {
		t.Fatal("fold must not shift group text")
	}
	_ = tt.Click("分组 Alpha")
	tt.Advance(200 * time.Millisecond)
	text, _ = tt.Find("Alpha")
	if text.X != before {
		t.Fatal("unfold must not shift group text")
	}
}

func TestStartupFailureCardRetryAndTitleSpacing(t *testing.T) {
	a := &application{config: defaultSettings()}
	p := a.work.add(profile{ID: "codex", Name: "Codex"}, filepath.Join(t.TempDir(), "project"))
	p.Error = "找不到命令，请检查安装路径。"
	starts := 0
	a.start = func(p *pane) { starts++; p.Error = ""; p.Starting = true }
	tt := ui.NewTester(a.view, 1000, 720)
	button, _ := tt.Find("新建终端")
	title, ok := tt.Find("project")
	if !ok || title.X < button.X+button.W+16 {
		t.Fatalf("title needs breathing room after tools: title=%+v button=%+v", title, button)
	}
	retry, ok := tt.Find("重试")
	if !ok || retry.Y < 200 || retry.Y > 550 {
		t.Fatalf("retry must stay in the centered error card: %+v", retry)
	}
	saveInteractionFrame(t, tt, "startup-error")
	if err := tt.Click("重试"); err != nil {
		t.Fatal(err)
	}
	if starts != 1 || !p.Starting {
		t.Fatal("retry must start a fresh launch")
	}
}

func TestLauncherFitsMinimumWindowAndKeyboardReachesLastProfile(t *testing.T) {
	a := &application{config: defaultSettings()}
	a.openLauncher("")
	tt := ui.NewTester(a.view, 720, 440)
	tt.Advance(500 * time.Millisecond)
	item, ok := tt.Find("启动 Terminal")
	if !ok || item.Y < 0 || item.Y+item.H > 440 {
		t.Fatalf("popover must fit minimum window: %+v", item)
	}
	for range 5 {
		tt.Key(0, ui.KeyTab)
	}
	tt.Key(0, ui.KeyEnter)
	tt.Advance(500 * time.Millisecond)
	if a.work.active() == nil || a.work.active().Root.Pane.Profile.ID != "shell" {
		t.Fatal("last profile must be keyboard accessible")
	}
}

func TestButtonHoverEasesInAndOut(t *testing.T) {
	a := &application{config: defaultSettings()}
	tt := ui.NewTester(a.view, 720, 440)
	r, ok := tt.Find("新建终端")
	if !ok {
		t.Fatal("missing new-terminal button")
	}
	x, y := int(r.X+2), int(r.Y+r.H/2)
	start := tt.Image().RGBAAt(x, y)
	tt.Move(r.X+3, r.Y+r.H/2)
	tt.Advance(50 * time.Millisecond)
	mid := tt.Image().RGBAAt(x, y)
	tt.Advance(180 * time.Millisecond)
	end := tt.Image().RGBAAt(x, y)
	if start == mid || mid == end {
		t.Fatalf("hover should interpolate, got %v %v %v", start, mid, end)
	}
	tt.Move(600, 300)
	tt.Advance(200 * time.Millisecond)
	if tt.Image().RGBAAt(x, y) != start {
		t.Fatal("hover must animate back to idle")
	}
}
