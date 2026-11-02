package app

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

type pane struct {
	ID       int
	Profile  profile
	Dir      string
	Term     *terminal.Terminal
	Error    string
	Starting bool
	Handle   ui.Handle
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
	Root    *layout
	Focused int
	GroupID string
	Close   ui.Handle
	Select  ui.Handle
}

type workspace struct {
	Tabs   []*tab
	Active int
	nextID int
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
	w.Active = t.ID
	return leaf
}

func (w *workspace) removeTab(id int) {
	i := slices.IndexFunc(w.Tabs, func(t *tab) bool { return t.ID == id })
	if i < 0 {
		return
	}
	w.Tabs = slices.Delete(w.Tabs, i, i+1)
	if w.Active == id {
		w.Active = 0
		if len(w.Tabs) > 0 {
			w.Active = w.Tabs[min(i, len(w.Tabs)-1)].ID
		}
	}
}

func (w *workspace) all() []*tab { return slices.Clone(w.Tabs) }

func (t *tab) label() string {
	if t == nil {
		return ""
	}
	if n := t.Root.find(t.Focused); n != nil {
		if n.Pane.Term != nil {
			if title := strings.TrimSpace(n.Pane.Term.Title()); title != "" {
				return title
			}
		}
		return filepath.Base(n.Pane.Dir)
	}
	return "终端"
}
