package app

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
)

var (
	rightPanelIcon = svgIcon(`<rect x="4" y="4" width="16" height="16" rx="2"/><path d="M14 4v16"/>`)
	refreshIcon    = svgIcon(`<path d="M20 10a8 8 0 1 0-2 8M20 4v6h-6"/>`)
)

type fileNode struct {
	Path, Name, Error                                 string
	Directory, Link, Open, Loaded, Loading, Truncated bool
	Children                                          []*fileNode
}
type inspectorState struct {
	Open                bool
	Width               float32
	Motion              sidebarTransition
	Directory, Selected string
	Root                *fileNode
	Generation          uint64
}

func readFileNodes(path string) ([]*fileNode, bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, false, err
	}
	nodes := make([]*fileNode, 0, len(entries))
	for _, entry := range entries {
		nodes = append(nodes, &fileNode{Path: filepath.Join(path, entry.Name()), Name: entry.Name(), Directory: entry.IsDir(), Link: entry.Type()&os.ModeSymlink != 0})
	}
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Directory != nodes[j].Directory {
			return nodes[i].Directory
		}
		return strings.ToLower(nodes[i].Name) < strings.ToLower(nodes[j].Name)
	})
	truncated := len(nodes) > 2000
	if truncated {
		nodes = nodes[:2000]
	}
	return nodes, truncated, nil
}
func (a *application) inspectorDirectory() string {
	if t := a.work.active(); t != nil {
		if n := t.Root.find(t.Focused); n != nil {
			return n.Pane.directory()
		}
	}
	return ""
}
func (a *application) refreshInspector(dir string) {
	s := &a.inspector
	if dir == "" {
		return
	}
	s.Generation++
	s.Directory = dir
	s.Selected = ""
	s.Root = &fileNode{Path: dir, Name: filepath.Base(dir), Directory: true, Open: true}
	a.loadFileNode(s.Root)
}
func (a *application) loadFileNode(node *fileNode) {
	if node.Loading || node.Loaded || a.win == nil {
		return
	}
	node.Loading = true
	generation := a.inspector.Generation
	win := a.win
	path := node.Path
	go func() {
		children, truncated, err := readFileNodes(path)
		win.Update(func() {
			if a.closed.Load() || a.inspector.Generation != generation {
				return
			}
			node.Children = children
			node.Truncated = truncated
			node.Loading = false
			node.Loaded = true
			if err != nil {
				node.Error = err.Error()
			}
		})
	}()
}
func (a *application) inspectorWidth(c *ui.Context) float32 {
	w, _ := c.Size()
	if a.inspector.Width <= 0 {
		a.inspector.Width = min(float32(350), max(float32(250), w*.3))
	}
	limit := max(float32(220), min(float32(520), w-a.sidebarWidth*a.sidebarMotion.Value-240))
	return min(max(a.inspector.Width, 220), limit)
}
func (a *application) prepareInspector(c *ui.Context) float32 {
	target := float32(0)
	if a.inspector.Open {
		target = 1
	}
	if a.inspector.Motion.step(c.Now(), target) {
		c.AnimationFrame()
	}
	return a.inspectorWidth(c) * a.inspector.Motion.Value
}
func (a *application) inspectorView(c *ui.Context) {
	s := &a.inspector
	fullWidth := a.inspectorWidth(c)
	width := fullWidth * s.Motion.Value
	if width < .1 {
		return
	}
	dir := a.inspectorDirectory()
	if s.Open && dir != "" && (s.Directory != dir || s.Root == nil) {
		a.refreshInspector(dir)
	}
	ui.Box(c.Key("right-sidebar")).Width(width).FillHeight().MinHeight(0).Shrink(0).Clip().Children(func() {
		ui.Column(c).Width(fullWidth).FillHeight().MinHeight(0).Shrink(0).Disabled(!s.Open).Children(func() {
			ui.Row(c).Height(36).Padding(0, 12).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "文件").FontSize(12).Grow(1).TextColor(c.Theme().TextMuted)
				iconButton(c.Key("refresh-files"), refreshIcon, "刷新文件树").Disabled(dir == "" || s.Root != nil && s.Root.Loading).OnClick(func() { a.refreshInspector(dir) })
			})
			if dir == "" {
				ui.Box(c).Grow(1).Center().Padding(16).Children(func() { ui.Text(c, "打开终端以查看目录结构").FontSize(12).TextColor(c.Theme().TextMuted) })
				return
			}
			ui.Text(c, shortDirectory(dir)).FontSize(10).Padding(5, 12, 10, 12).SingleLine().Clip().Tooltip(dir).TextColor(c.Theme().TextMuted)
			ui.Scroll(c.Key("file-tree-scroll")).Grow(1).MinHeight(0).Children(func() {
				ui.Column(c).Padding(0, 6, 12, 6).Children(func() {
					if s.Root != nil {
						a.fileNodeView(c, s.Root)
					}
				})
			})
		})
	})
}
func (a *application) fileNodeView(c *ui.Context, n *fileNode) {
	ui.Column(c.Key(n.Path)).Children(func() {
		if !n.Directory {
			row := ui.ButtonBase(c.Key("file")).Height(26).Radius(5).Padding(4, 6, 4, 23).Gap(6).AlignItems(ui.Center).FocusRing(false).Label("文件 " + n.Name).Tooltip(n.Path)
			alpha := float32(0)
			if a.inspector.Selected == n.Path {
				alpha = .1
			}
			if row.Hovered() {
				alpha += .045
			}
			row.Background(ui.RGBA(230, 235, 245, alpha)).Children(func() {
				fileTypeIcon(c, n.Name, false, false)
				ui.Text(c, n.Name).FontSize(12).Grow(1).MinWidth(0).SingleLine().Clip()
			}).OnClick(func() { a.inspector.Selected = n.Path })
			return
		}
		if n.Open && !n.Loaded && !n.Loading {
			a.loadFileNode(n)
		}
		parts := ui.CollapsibleBase(c, &n.Open)
		progress := parts.Progress()
		b := parts.Trigger.Height(28).Radius(5).Padding(4, 6).Gap(6).AlignItems(ui.Center).Label("目录 " + n.Name).Tooltip(n.Path)
		alpha := float32(0)
		if b.Hovered() {
			alpha = .07
		}
		b.Background(ui.RGBA(230, 235, 245, alpha)).FocusRing(false)
		b.OnShortcut(0, ui.KeyRight, func() { n.Open = true })
		b.OnShortcut(0, ui.KeyLeft, func() { n.Open = false })
		b.Children(func() {
			ui.Icon(c, chevronIcon).Size(11, 11).Rotate(90 * progress)
			fileTypeIcon(c, n.Name, true, n.Open)
			ui.Text(c, n.Name).FontSize(12).Grow(1).MinWidth(0).SingleLine().Clip()
		})
		parts.Panel(func() {
			ui.Column(c).Padding(0, 0, 0, 12).Children(func() {
				if n.Loading {
					ui.Text(c, "正在读取…").Padding(4, 6, 4, 23).FontSize(11).TextColor(c.Theme().TextMuted)
				}
				if n.Error != "" {
					ui.Text(c, n.Error).Wrap().Padding(4, 6).FontSize(11).TextColor(c.Theme().Danger)
				}
				if n.Loaded && len(n.Children) == 0 && n.Error == "" {
					ui.Text(c, "空文件夹").Padding(4, 6, 4, 23).FontSize(11).TextColor(c.Theme().TextMuted)
				}
				for _, child := range n.Children {
					a.fileNodeView(c, child)
				}
				if n.Truncated {
					ui.Text(c, "仅显示前 2000 项").Padding(4, 6).FontSize(11).TextColor(c.Theme().TextMuted)
				}
			})
		})
	})
}
