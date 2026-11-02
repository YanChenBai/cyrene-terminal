package app

import (
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func saveInteractionFrame(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	if os.Getenv("CYRENE_CAPTURE") != "1" {
		return
	}
	if err := os.MkdirAll("test-results", 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join("test-results", name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err = png.Encode(f, tt.Image()); err != nil {
		t.Fatal(err)
	}
}

func TestDialogOriginOvershootDismissalAndFocus(t *testing.T) {
	a := &application{config: defaultSettings()}
	tt := ui.NewTester(a.view, 1000, 720)
	r, ok := tt.Find("外观设置")
	if !ok {
		t.Fatal("missing settings")
	}
	if err := tt.Click("外观设置"); err != nil {
		t.Fatal(err)
	}
	m := a.dialogs["preferences"]
	if m.Source != r || m.Position != 0 || m.Scale > 0.13 {
		t.Fatalf("dialog must start at its clicked control: %+v", m)
	}
	saveInteractionFrame(t, tt, "dialog-origin")
	tt.Advance(50 * time.Millisecond)
	if m.Position <= 0 || m.Position >= 1 || m.Scale <= 0.12 || m.Scale >= 1.1 {
		t.Fatal("missing intermediate opening pose")
	}
	saveInteractionFrame(t, tt, "dialog-expanding")
	tt.Advance(130 * time.Millisecond)
	if math.Abs(float64(m.Scale-1.1)) > 0.015 {
		t.Fatalf("expected 1.1 overshoot, got %v", m.Scale)
	}
	saveInteractionFrame(t, tt, "dialog-overshoot")
	tt.Advance(650 * time.Millisecond)
	if m.Scale != 1 || m.Position != 1 {
		t.Fatal("dialog must settle at natural size")
	}
	saveInteractionFrame(t, tt, "dialog-settled")
	tt.Key(0, ui.KeyEscape)
	tt.Advance(80 * time.Millisecond)
	if !m.Visible || m.Opacity <= 0 || m.Opacity >= 1 {
		t.Fatal("dismissal must remain mounted while fading out")
	}
	saveInteractionFrame(t, tt, "dialog-closing")
	tt.Advance(180 * time.Millisecond)
	if m.Visible || a.preferences {
		t.Fatal("dialog must unmount after exit")
	}
	if err := tt.Click("新建终端"); err != nil {
		t.Fatal("focus/input must return after dismiss", err)
	}
	tt.Advance(500 * time.Millisecond)
	if _, ok := tt.Find("启动 Codex"); !ok {
		t.Fatal("closed modal must not block the next interaction")
	}
}

func TestPopoverBounceAndInterruptedExit(t *testing.T) {
	a := &application{config: defaultSettings()}
	tt := ui.NewTester(a.view, 800, 500)
	if err := tt.Click("新建终端"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(90 * time.Millisecond)
	if math.Abs(float64(a.launcherMotion.Scale-1.1)) > 0.035 {
		t.Fatal("popover must reach 1.1")
	}
	saveInteractionFrame(t, tt, "menu-overshoot")
	tt.Key(0, ui.KeyEscape)
	tt.Advance(60 * time.Millisecond)
	if !a.launcherMotion.Visible {
		t.Fatal("popover disappeared without an exit animation")
	}
	previous := a.launcherMotion.Scale
	a.launcher = true
	tt.Frame()
	if math.Abs(float64(a.launcherMotion.Scale-previous)) > 0.01 {
		t.Fatal("reopening during exit must retain the current pose")
	}
	tt.Advance(800 * time.Millisecond)
	if a.launcherMotion.Scale != 1 || !a.launcher {
		t.Fatal("reopened popover should settle")
	}
	tt.ClickAt(750, 400)
	tt.Advance(420 * time.Millisecond)
	if a.launcher || a.launcherMotion.Visible {
		t.Fatal("outside click must dismiss and finish animating")
	}
}

func TestSidebarFullyHiddenAndTitleControlsCompact(t *testing.T) {
	a := &application{config: defaultSettings()}
	a.work.add(profile{ID: "shell", Name: "Terminal"}, t.TempDir())
	tt := ui.NewTester(a.view, 800, 500)
	row, ok := tt.Find("关闭终端 2")
	if !ok || row.Y > 80 {
		t.Fatalf("sidebar tabs must align to the top: %+v", row)
	}
	if err := tt.Click("切换侧栏"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(160 * time.Millisecond)
	if a.sidebarMotion.Value <= 0 || a.sidebarMotion.Value >= 1 {
		t.Fatal("sidebar must have visible intermediate closing frames")
	}
	saveInteractionFrame(t, tt, "sidebar-closing")
	tt.Advance(420 * time.Millisecond)
	if !a.collapsed {
		t.Fatal("expected collapsed sidebar")
	}
	if _, ok := tt.Find("关闭终端 2"); ok {
		t.Fatal("collapsed sidebar must not leave a compact rail")
	}
	r, ok := tt.Find("切换侧栏")
	if !ok || r.X > 12 || r.Y > 5 || r.H > 26 {
		t.Fatalf("toggle must remain on the left of the compact title bar: %+v", r)
	}
	if _, ok := tt.Find("后台会话"); ok {
		t.Fatal("standalone background feature must be absent")
	}
	for _, label := range []string{"新建终端", "新建文件夹分组"} {
		if _, ok := tt.Find(label); ok {
			t.Fatalf("collapsed toolbar must hide %s", label)
		}
	}
	saveInteractionFrame(t, tt, "sidebar-hidden")
	if err := tt.Click("切换侧栏"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(160 * time.Millisecond)
	if a.sidebarMotion.Value <= 0 || a.sidebarMotion.Value >= 1 {
		t.Fatal("sidebar must have visible intermediate opening frames")
	}
	saveInteractionFrame(t, tt, "sidebar-opening")
	tt.Advance(420 * time.Millisecond)
	if _, ok := tt.Find("关闭终端 2"); !ok {
		t.Fatal("sidebar should return with the original tab")
	}
}

func TestSidebarResizeAndCollapsedShortcut(t *testing.T) {
	a := &application{config: defaultSettings()}
	a.work.add(profile{ID: "codex", Name: "Codex"}, t.TempDir())
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Press(244, 320)
	tt.Move(330, 320)
	tt.Release(330, 320)
	if a.sidebarWidth < 320 || a.sidebarWidth > 340 {
		t.Fatalf("drag did not resize sidebar: %v", a.sidebarWidth)
	}
	saveInteractionFrame(t, tt, "sidebar-resized")
	_ = tt.Click("切换侧栏")
	tt.Advance(420 * time.Millisecond)
	tt.Key(ui.Ctrl|ui.Shift, ui.KeyT)
	tt.Advance(800 * time.Millisecond)
	if _, ok := tt.Find("启动 Claude"); !ok {
		t.Fatal("collapsed launcher shortcut must anchor to toggle")
	}
	saveInteractionFrame(t, tt, "agent-logos")
	tt.Key(0, ui.KeyEscape)
	tt.Advance(200 * time.Millisecond)
	_ = tt.Click("切换侧栏")
	tt.Advance(420 * time.Millisecond)
	if a.sidebarWidth < 320 {
		t.Fatal("collapse discarded chosen width")
	}
}

func TestSpringPositionAndScaleMoveTogether(t *testing.T) {
	m := &overlayMotion{}
	now := time.Unix(100, 0)
	m.step(now, true, ui.Rect{}, .12)
	peak := float32(0)
	for ms := 10; ms <= 510; ms += 10 {
		m.step(now.Add(time.Duration(ms)*time.Millisecond), true, ui.Rect{}, .12)
		peak = max(peak, m.Scale)
		if math.Abs(float64((m.Scale-.12)/.88-m.Position)) > .0001 {
			t.Fatal("scale and travel must follow the same spring")
		}
	}
	if peak < 1.08 || peak > 1.12 {
		t.Fatalf("spring peak: %v", peak)
	}
}
