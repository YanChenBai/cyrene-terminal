package app

import (
	"fmt"
	"image/color"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestContentTabsReuseSessionsAndConfirmClosing(t *testing.T) {
	a := &application{config: defaultSettings()}
	a.work.add(profile{ID: "shell"}, t.TempDir())
	first := a.work.active()
	a.work.add(profile{ID: "shell"}, t.TempDir())
	second := a.work.active()
	starts := 0
	a.start = func(*pane) { starts++ }
	tt := ui.NewTester(a.view, 1000, 720)
	if err := tt.Click(fmt.Sprintf("内容标签 %d", first.ID)); err != nil {
		t.Fatal(err)
	}
	if a.work.active() != first || starts != 0 {
		t.Fatal("switching content started a new terminal")
	}
	if err := tt.Click(fmt.Sprintf("关闭内容标签 %d", second.ID)); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	if !a.stopConfirm || a.stopTarget != second || len(a.work.Tabs) != 2 {
		t.Fatal("close must confirm the clicked tab, including an inactive tab")
	}
	if err := tt.Click("取消"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	if _, ok := tt.Find("添加内容标签"); ok {
		t.Fatal("tab strip must not expose an add button")
	}
	item, ok := tt.Find(fmt.Sprintf("内容标签 %d", first.ID))
	if !ok || item.Y+item.H > workspaceTitleHeight {
		t.Fatalf("tabs must be in title bar: %+v", item)
	}
	for range 3 {
		tt.Frame()
	}
}

func TestTitlebarTabHoverCoversCloseRegion(t *testing.T) {
	a := &application{config: defaultSettings()}
	a.work.add(profile{ID: "shell"}, t.TempDir())
	tt := ui.NewTester(a.view, 1000, 720)
	tt.Move(900, 400)
	tt.Advance(300 * time.Millisecond)
	selectRect, _ := tt.Find(fmt.Sprintf("内容标签 %d", a.work.activeScreen().ID))
	closeRect, _ := tt.Find(fmt.Sprintf("关闭内容标签 %d", a.work.activeScreen().ID))
	x, y := int(closeRect.X+2), int(closeRect.Y+closeRect.H/2)
	idle := tt.Image().RGBAAt(x, y)
	tt.Move(closeRect.X+closeRect.W/2, closeRect.Y+closeRect.H/2)
	tt.Advance(250 * time.Millisecond)
	hover := tt.Image().RGBAAt(x, y)
	labelBackground := tt.Image().RGBAAt(int(selectRect.X+3), y)
	if idle == hover || absColorDifference(hover, labelBackground) > 3 {
		t.Fatalf("tab must share one hover surface through close icon: idle=%v close=%v label=%v", idle, hover, labelBackground)
	}
	saveInteractionFrame(t, tt, "titlebar-tab-hover")
}

func absColorDifference(a, b color.RGBA) int {
	d := 0
	for _, v := range []int{int(a.R) - int(b.R), int(a.G) - int(b.G), int(a.B) - int(b.B), int(a.A) - int(b.A)} {
		if v < 0 {
			v = -v
		}
		d = max(d, v)
	}
	return d
}
