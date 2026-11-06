package app

import (
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

type pane struct {
	Entered      bool
	ShowActions  bool
	Actions      ui.Handle
	ID           int
	Profile      profile
	Command      []string
	AgentSession string
	AgentToken   string
	AgentRecord  string
	NeedsResume  bool
	Browser      *browserState
	ResumeHandle ui.Handle
	Surface      ui.Handle
	Header       ui.Handle
	PaneClose    ui.Handle
	ChildTab     ui.Handle
	ChildClose   ui.Handle
	DropEdge     splitEdge
	Dir          string
	Term         *terminal.Terminal
	Error        string
	Starting     bool
	Handle       ui.Handle
	clipboard    *clipboardStore
	pasteTail    <-chan struct{}
}

// A leaf is a pane. A branch owns two subtrees and a draggable split ratio.
type layout struct {
	ID       int
	Pane     *pane
	Vertical bool
	Ratio    float32
	First    *layout
	Second   *layout
}

func (n *layout) leaves() []*pane {
	if n == nil {
		return nil
	}
	if n.Pane != nil {
		return []*pane{n.Pane}
	}
	return append(n.First.leaves(), n.Second.leaves()...)
}

func (n *layout) find(id int) *layout {
	if n == nil {
		return nil
	}
	if n.Pane != nil && n.Pane.ID == id {
		return n
	}
	if found := n.First.find(id); found != nil {
		return found
	}
	return n.Second.find(id)
}

func (n *layout) remove(id int) *layout {
	if n == nil {
		return nil
	}
	if n.Pane != nil {
		if n.Pane.ID == id {
			return nil
		}
		return n
	}
	n.First, n.Second = n.First.remove(id), n.Second.remove(id)
	if n.First == nil {
		return n.Second
	}
	if n.Second == nil {
		return n.First
	}
	return n
}

type tab struct {
	ID      int
	Name    string
	Alias   string
	Root    *layout
	Focused int
	GroupID string
	Close   ui.Handle
	Select  ui.Handle
}

type workspace struct {
	nextWorkspace int
	Tabs          []*tab
	Screens       []*screen
	Active        int
	nextID        int
}

func (w *workspace) id() int { w.nextID++; return w.nextID }

func (w *workspace) active() *tab {
	for _, t := range w.Tabs {
		if t.ID == w.Active {
			return t
		}
	}
	return nil
}

func (w *workspace) newPane(p profile, dir string) *pane {
	return &pane{ID: w.id(), Profile: p, Dir: dir}
}

func (w *workspace) add(p profile, dir string) *pane {
	leaf := w.newPane(p, dir)
	t := &tab{ID: w.id(), Name: p.Name, Root: &layout{ID: w.id(), Pane: leaf}, Focused: leaf.ID}
	w.Tabs = append(w.Tabs, t)
	w.Screens = append(w.Screens, &screen{ID: t.ID, Root: &layout{ID: t.Root.ID, Pane: leaf}, Focused: leaf.ID})
	w.Active = t.ID
	return leaf
}

func (w *workspace) removeTab(id int) {
	i := slices.IndexFunc(w.Tabs, func(t *tab) bool { return t.ID == id })
	if i < 0 {
		return
	}
	var remaining *pane
	for _, p := range w.Tabs[i].Root.leaves() {
		if s := w.screenForPane(p.ID); s != nil {
			s.Root = s.Root.remove(p.ID)
			if s.Root != nil {
				remaining = s.Root.leaves()[0]
				if s.Focused == p.ID {
					s.Focused = remaining.ID
				}
			}
		}
	}
	w.pruneScreens()
	w.Tabs = slices.Delete(w.Tabs, i, i+1)
	if w.Active == id {
		w.Active = 0
		if len(w.Tabs) > 0 {
			w.Active = w.Tabs[min(i, len(w.Tabs)-1)].ID
		}
		if remaining != nil {
			if t := w.sessionForPane(remaining.ID); t != nil {
				w.Active = t.ID
			}
		}
	}
}

func (w *workspace) all() []*tab { return slices.Clone(w.Tabs) }

func (t *tab) label() string {
	if t == nil {
		return ""
	}
	if alias := strings.TrimSpace(t.Alias); alias != "" {
		return alias
	}
	if n := t.Root.find(t.Focused); n != nil {
		if n.Pane.Browser != nil {
			return n.Pane.Browser.label()
		}
		if dir := strings.TrimSpace(n.Pane.directory()); dir != "" {
			return filepath.Base(filepath.Clean(dir))
		}
		if n.Pane.Term != nil {
			if title := strings.TrimSpace(n.Pane.Term.Title()); title != "" {
				return displayTitle(title)
			}
		}
		return "终端"
	}
	return "终端"
}

func displayTitle(title string) string {
	title = strings.TrimSpace(title)
	if strings.ContainsAny(title, `/\\`) && strings.HasSuffix(strings.ToLower(title), ".exe") {
		name := filepath.Base(strings.ReplaceAll(title, `\`, "/"))
		name = name[:len(name)-4]
		switch strings.ToLower(name) {
		case "pwsh", "powershell":
			return "PowerShell"
		case "cmd":
			return "Command Prompt"
		}
		return name
	}
	return title
}

func (p *pane) directory() string {
	if p.Term != nil {
		if dir := strings.TrimSpace(p.Term.Dir()); dir != "" {
			if runtime.GOOS == "windows" {
				if len(dir) > 3 && dir[0] == '/' && dir[2] == ':' {
					dir = dir[1:]
				}
				dir = filepath.FromSlash(dir)
			}
			return dir
		}
	}
	return p.Dir
}
