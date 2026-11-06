package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestGlassCenterPreservesWindowTransparency(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		c.Root().Background(ui.RGBA(8, 9, 11, .4))
		plainSurface(ui.Box(c).Fill().Radius(12), .55, .018)
	}, 200, 150)
	center := tt.Image().RGBAAt(100, 75)
	if center.A < 100 || center.A > 110 {
		t.Fatalf("glass duplicated backdrop alpha: %v", center)
	}
	top := tt.Image().RGBAAt(100, 5)
	if top != center {
		t.Fatal("flat surface still has an edge highlight")
	}
	saveInteractionFrame(t, tt, "frosted-alpha-flat")
}

func TestWorkspaceOwnsAddButtonWithTopBreathingRoom(t *testing.T) {
	a, first, second := tilingTestApp(t)
	a.work.dock(second.ID, first.ID, splitRight)
	s := a.work.activeScreen()
	a.work.add(profile{ID: "shell"}, t.TempDir())
	other := a.work.activeScreen()
	a.selectScreen(s)
	tt := ui.NewTester(a.view, 1100, 720)
	tt.Advance(time.Second)
	add, _ := tt.Find("新建终端")
	children, _ := tt.Find(fmt.Sprintf("工作区子标签 %d", s.ID))
	main, _ := tt.Find(fmt.Sprintf("内容标签 %d", s.ID))
	next, _ := tt.Find(fmt.Sprintf("内容标签 %d", other.ID))
	if main.Y < 5 || main.Y+main.H > workspaceTitleHeight-5 {
		t.Fatalf("tabs touch titlebar edges: %+v", main)
	}
	if add.X < children.X+children.W || add.X+add.W > next.X {
		t.Fatalf("add is outside selected workspace: %+v %+v %+v", add, children, next)
	}
	if err := tt.Click("新建终端"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	if !a.launcher {
		t.Fatal("workspace add did not open launcher")
	}
	if _, ok := tt.Find("启动分组"); ok {
		t.Fatal("workspace launcher exposed group selector")
	}
	header, _ := tt.Find(fmt.Sprintf("窗格 %d", first.ID))
	if header.Y-main.Y-main.H > 10 {
		t.Fatalf("excessive tab/content gap: %+v %+v", main, header)
	}
	saveInteractionFrame(t, tt, "glass-inset-tabs-launcher")
	target := s.Focused
	dir := a.work.sessionForPane(target).Root.Pane.directory()
	starts := 0
	a.start = func(*pane) { starts++ }
	// Scope belongs to the opener, even if focus changes while the menu is open.
	a.selectScreen(other)
	tt.Frame()
	if err := tt.Click("启动 Terminal"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	if a.work.activeScreen() != s || len(s.Root.leaves()) != 3 || len(other.Root.leaves()) != 1 || starts != 1 || a.work.active().Root.Pane.directory() != dir {
		t.Fatal("launcher lost its clicked workspace or directory")
	}
	tt.Click("新建终端")
	tt.Advance(time.Second)
	tt.Click("新建网页")
	tt.Advance(time.Second)
	if a.work.activeScreen() != s || len(s.Root.leaves()) != 4 || a.work.active().Root.Pane.Browser == nil || starts != 1 {
		t.Fatal("website was not added to the clicked workspace")
	}
}

// The headless host only records the native view; it never calls its platform API.
type portalTestView struct{ ui.NativeWebView }

func TestBrowserCornersMaskPageWithoutMaskingPane(t *testing.T) {
	a := &application{config: defaultSettings()}
	var handle ui.Handle
	tt := ui.NewTester(func(c *ui.Context) {
		c.Root().Background(ui.RGBA(8, 9, 11, .4))
		ui.Column(c).Fill().Children(func() { a.browserViewport(c, &portalTestView{}, &handle) })
	}, 200, 150)
	img := tt.Image()
	if img.RGBAAt(100, 75).A != 0 {
		t.Fatal("native page portal was not cleared")
	}
	if img.RGBAAt(4, 0).A > 110 {
		t.Fatal("corner backing is no longer transparent")
	}
	if img.RGBAAt(1, 75).A > 110 {
		t.Fatal("webview backing made the surrounding pane opaque")
	}
	if tt.WebViewAt(4, 0) != nil || tt.WebViewAt(100, 75) == nil {
		t.Fatal("native hit testing ignores rounded portal")
	}
	saveInteractionFrame(t, tt, "webview-corner-mask")
}

func TestApplicationGlassKeepsConfiguredAlpha(t *testing.T) {
	a, _, _ := tilingTestApp(t)
	a.config.Opacity = .4
	tt := ui.NewTester(a.view, 1000, 700)
	tt.SetVibrancy(true)
	tt.Frame()
	alpha := tt.Image().RGBAAt(150, 400).A
	if alpha < 100 || alpha > 110 {
		t.Fatalf("application pane opacity drifted: %d", alpha)
	}
}

func TestRaisedPaneKeepsCenterTransparentAndLightsLowerEdge(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		c.Root().Background(ui.RGBA(8, 9, 11, .4))
		beveledSurface(ui.Box(c).Fill().Radius(12), .55, .032, .44)
	}, 200, 150)
	center := tt.Image().RGBAAt(100, 75)
	lower := tt.Image().RGBAAt(100, 149)
	if center.A < 100 || center.A > 110 {
		t.Fatalf("bevel increased background opacity: %v", center)
	}
	if int(lower.R)-int(center.R) < 12 {
		t.Fatalf("bottom edge has no return highlight: %v %v", center, lower)
	}
	saveInteractionFrame(t, tt, "frosted-raised-edge")
}

func TestRoundedWebViewHitGeometryIncludesPaddingAndAllCorners(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		ui.WebView(c, &portalTestView{}).Size(100, 80).Radius(24).Border(4, ui.Transparent).Padding(2)
	}, 100, 80)
	for _, p := range [][2]float32{{6, 6}, {93, 6}, {93, 73}, {6, 73}} {
		if tt.WebViewAt(p[0], p[1]) != nil {
			t.Fatalf("rounded corner takes native input: %v", p)
		}
	}
	if tt.WebViewAt(25, 7) == nil || tt.WebViewAt(50, 40) == nil {
		t.Fatal("rounded hit clip rejected page interior")
	}
}
