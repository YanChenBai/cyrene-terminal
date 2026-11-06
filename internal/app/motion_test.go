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
	if m.Scale < .9 || m.Scale > 1.04 {
		t.Fatalf("expected restrained opening, got %v", m.Scale)
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
	if a.launcherMotion.Scale < .85 || a.launcherMotion.Scale > 1.04 {
		t.Fatal("popover opening must remain restrained")
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
	if peak < 1 || peak > 1.04 {
		t.Fatalf("spring peak: %v", peak)
	}
}
