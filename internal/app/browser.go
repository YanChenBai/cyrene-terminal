package app

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type browserState struct {
	OverlaySnapshot          *ui.Bitmap
	OverlayCaptured          bool
	URL, Draft, Title, Error string
	View                     *mygo.WebView
	Address                  ui.Handle
	Editing                  bool
}

func normalizeBrowserURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("请输入网址")
	}
	if !strings.Contains(raw, "://") {
		if strings.ContainsAny(raw, " \t\r\n") {
			return "", fmt.Errorf("请输入有效网址")
		}
		host := strings.Split(raw, "/")[0]
		name := host
		if h, _, err := net.SplitHostPort(host); err == nil {
			name = h
		}
		if strings.EqualFold(name, "localhost") || net.ParseIP(strings.Trim(name, "[]")) != nil {
			raw = "http://" + raw
		} else {
			raw = "https://" + raw
		}
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
		return "", fmt.Errorf("请使用 http 或 https 网址")
	}
	return u.String(), nil
}

func (b *browserState) label() string {
	if strings.TrimSpace(b.Title) != "" {
		return b.Title
	}
	if u, err := url.Parse(b.URL); err == nil && u.Hostname() != "" {
		return u.Hostname()
	}
	return "网页"
}

func (a *application) newBrowser(target int, edge splitEdge) {
	a.browserTarget, a.browserEdge = target, edge
	if err := a.launchBrowser(""); err != nil {
		a.status = err.Error()
	}
}

func (a *application) launchBrowser(raw string) error {
	address := ""
	if strings.TrimSpace(raw) != "" {
		var err error
		address, err = normalizeBrowserURL(raw)
		if err != nil {
			return err
		}
	}
	dir, group := a.directoryForGroup(""), ""
	if t := a.work.active(); t != nil {
		group = t.GroupID
		if n := t.Root.find(t.Focused); n != nil {
			dir = n.Pane.directory()
		}
	}
	if a.browserTarget != 0 {
		t := a.work.sessionForPane(a.browserTarget)
		if t == nil {
			return fmt.Errorf("目标窗格已关闭")
		}
		dir, group = t.Root.find(a.browserTarget).Pane.directory(), t.GroupID
	}
	p := a.work.add(profile{ID: "web", Name: "网页"}, dir)
	p.Browser = &browserState{URL: address, Draft: address}
	a.work.active().GroupID = group
	if a.browserTarget != 0 {
		a.dockPane(p.ID, a.browserTarget, a.browserEdge)
	}
	if address == "" {
		p.Browser.Address.Focus()
	} else if a.start != nil {
		a.start(p)
	}
	return nil
}

func (a *application) startBrowser(p *pane) {
	if p.Starting || p.Browser == nil || p.Browser.URL == "" || a.win == nil {
		return
	}
	b, win := p.Browser, a.win
	p.Starting = true
	b.Error = ""
	address := b.URL
	go func() {
		view, err := win.NewWebView(mygo.WebViewOptions{BackgroundColor: "#17151a", Page: mygo.PageOptions{Untrusted: true, PinchZoom: true, SwipeNavigation: true}})
		if err == nil {
			page := view.Page()
			page.OnWillNavigate(func(e *mygo.NavigateEvent) {
				if _, err := normalizeBrowserURL(e.URL); err != nil {
					e.PreventDefault()
				}
			})
			page.SetWindowOpenHandler(func(req mygo.WindowOpenRequest) *mygo.WindowOptions {
				if req.UserInitiated {
					if target, err := normalizeBrowserURL(req.URL); err == nil {
						_ = page.LoadURL(target)
					}
				}
				return nil
			})
			update := func(fn func()) {
				win.Update(func() {
					if p.Browser == b && a.work.sessionForPane(p.ID) != nil {
						fn()
					}
				})
			}
			page.OnDidNavigate(func(address string) {
				update(func() {
					previous := b.URL
					b.URL = address
					if !b.Editing || b.Draft == previous {
						b.Draft = address
					}
					b.Error = ""
				})
			})
			page.OnPageTitleUpdated(func(e *mygo.TitleEvent) { title := e.Title; update(func() { b.Title = title }) })
			page.OnDidFinishLoad(func() { win.Invalidate(); captureAcceptanceBrowser(view, p.ID) })
			page.OnDidFailLoad(func(err *mygo.LoadError) { message := err.Description; update(func() { b.Error = message }) })
			err = page.LoadURL(address)
		}
		win.Update(func() {
			if a.closed.Load() || p.Browser != b || a.work.sessionForPane(p.ID) == nil {
				if view != nil {
					view.Destroy()
				}
				return
			}
			p.Starting = false
			if err != nil {
				if view != nil {
					view.Destroy()
				}
				b.Error = err.Error()
				return
			}
			b.View = view
		})
	}()
}

func (a *application) browserView(c *ui.Context, p *pane) {
	b := p.Browser
	b.Editing = b.Address.Focused(c)
	c.OnShortcut(ui.Ctrl, ui.KeyL, func() { b.Address.Focus() })
	alpha := float32(0)
	if p.ShowActions {
		alpha = 1
	}
	toolbar := ui.Row(c.Key("browser-toolbar")).Bind(&p.Actions)
	alpha = toolbar.Animate("actions-visible", alpha, 170*time.Millisecond)
	toolbar.Height(44).Shrink(0).Padding(7, 10, 5, 10).Gap(4).AlignItems(ui.Center).Children(func() {
		ui.ButtonBase(c.Key("browser-drag")).Bind(&p.Header).Label(fmt.Sprintf("窗格 %d", p.ID)).Drag(terminalDrag{PaneID: p.ID}).Size(26, 26).Shrink(0).Padding(5).Center().Children(func() { ui.Icon(c, webIcon).Size(16, 16) }).OnClick(func() { a.focusPane(p) })
		back, forward := false, false
		if b.View != nil {
			back, forward = b.View.Page().CanGoBack(), b.View.Page().CanGoForward()
		}
		ui.Row(c.Key("browser-nav")).Width(86 * alpha).Height(26).Shrink(0).Gap(4).AlignItems(ui.Center).Clip().Opacity(alpha).Disabled(!p.ShowActions).Children(func() {
			softButton(c.Key("browser-back"), "网页后退").Size(26, 26).Shrink(0).Padding(5).Center().Disabled(!back).Children(func() { ui.Icon(c, browserBackIcon).Size(16, 16) }).OnClick(func() { b.View.Page().GoBack() })
			softButton(c.Key("browser-forward"), "网页前进").Size(26, 26).Shrink(0).Padding(5).Center().Disabled(!forward).Children(func() { ui.Icon(c, browserForwardIcon).Size(16, 16) }).OnClick(func() { b.View.Page().GoForward() })
			softButton(c.Key("browser-reload"), "刷新网页").Size(26, 26).Shrink(0).Padding(5).Center().Disabled(b.View == nil).Children(func() { ui.Icon(c, browserReloadIcon).Size(16, 16) }).OnClick(func() { b.View.Page().Reload() })
		})
		ui.TextInput(c.Key("browser-address"), &b.Draft).Bind(&b.Address).Label("当前网页地址").Placeholder("输入网址或 localhost:3000").Height(28).Border(0, ui.Transparent).Background(ui.RGBA(230, 235, 245, .035)).Grow(1).MinWidth(0).Radius(7).OnSubmit(func() {
			address, err := normalizeBrowserURL(b.Draft)
			if err != nil {
				b.Error = err.Error()
				return
			}
			b.URL, b.Draft, b.Error = address, address, ""
			if b.View != nil {
				if err := b.View.Page().LoadURL(address); err != nil {
					b.Error = err.Error()
					return
				}
				p.Handle.Focus()
			} else {
				a.startBrowser(p)
			}
		})
		ui.Box(c.Key("browser-close-slot")).Width(26 * alpha).Shrink(0).Clip().Opacity(alpha).Disabled(!p.ShowActions).Children(func() {
			iconButton(c.Key("browser-close"), closeIcon, fmt.Sprintf("关闭窗格 %d", p.ID)).Size(26, 26).OnClick(func() { a.requestStop(a.work.sessionForPane(p.ID), p.Header.Bounds(c)) })
		})
	})
	if b.Error != "" {
		ui.Text(c, b.Error).FontSize(11).TextColor(c.Theme().Danger).Padding(5, 10).Wrap()
	}
	if b.View != nil {
		var snapshot *ui.Bitmap
		if a.overlaySnapshotsActive {
			snapshot = b.OverlaySnapshot
		}
		a.browserViewport(c, b.View, &p.Handle, snapshot)
	} else {
		ui.Column(c).Grow(1).Center().Gap(10).Children(func() {
			if b.URL == "" {
				ui.Icon(c, webIcon).Size(32, 32).Opacity(.3)
				ui.Text(c, "在地址栏输入网址").FontSize(15).TextColor(c.Theme().TextMuted)
				ui.Text(c, "支持网站和 localhost 本地预览").FontSize(11).TextColor(c.Theme().TextMuted)
			} else if p.Starting {
				ui.Text(c, "正在打开网页…").TextColor(c.Theme().TextMuted)
			} else {
				textButton(c, "重试打开网页").OnClick(func() { a.startBrowser(p) })
			}
		})
	}
}

func (a *application) browserViewport(c *ui.Context, view ui.NativeWebView, handle *ui.Handle, snapshot ...*ui.Bitmap) {
	// The native composition layer shares this rounded portal; no solid corner patches.

	ui.Box(c.Key("browser-viewport")).Label("网页圆角视口").Grow(1).MinHeight(0).MinWidth(0).Margin(0, 4, 4, 4).Clip().Children(func() {
		if len(snapshot) > 0 && snapshot[0] != nil {
			ui.Image(c.Key("browser-snapshot"), snapshot[0]).Grow(1).MinHeight(0).MinWidth(0).Fit(ui.Cover).Radius(10)
			return
		}
		ui.WebView(c.Key("browser-content"), view).Bind(handle).Grow(1).MinHeight(0).MinWidth(0).Radius(10)
	})
}

var browserBackIcon = svgIcon(`<path d="m14 7-5 5 5 5"/>`)
var browserForwardIcon = svgIcon(`<path d="m10 7 5 5-5 5"/>`)
var browserReloadIcon = svgIcon(`<path d="M19 8a8 8 0 1 0 1 6M19 3v5h-5"/>`)
