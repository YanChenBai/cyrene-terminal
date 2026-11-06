package app

import (
	"fmt"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
)

type splitEdge uint8

const (
	splitRight splitEdge = iota
	splitLeft
	splitBottom
	splitTop
)

func (e splitEdge) label() string {
	switch e {
	case splitLeft:
		return "向左分屏"
	case splitBottom:
		return "向下分屏"
	case splitTop:
		return "向上分屏"
	}
	return "向右分屏"
}

// Screens own layout only. The sidebar's sessions retain process ownership,
// group, alias and directory when moved between or within screens.
type screen struct {
	WorkspaceIndex int
	Order          []int
	ID             int
	Alias          string
	Root           *layout
	Focused        int
	Close          ui.Handle
	Select         ui.Handle
}

type terminalDrag struct{ PaneID, ScreenID int }
type tabDropRequest struct {
	Drag   terminalDrag
	Target int
	Before bool
}
type dockRequest struct {
	Source, Target int
	Edge           splitEdge
}

func (w *workspace) sessionForPane(id int) *tab {
	for _, t := range w.Tabs {
		if t.Root.find(id) != nil {
			return t
		}
	}
	return nil
}

func (w *workspace) screenForPane(id int) *screen {
	for _, s := range w.Screens {
		if s.Root.find(id) != nil {
			return s
		}
	}
	return nil
}

func (w *workspace) activeScreen() *screen {
	if t := w.active(); t != nil {
		return w.screenForPane(t.Focused)
	}
	return nil
}

func (w *workspace) pruneScreens() {
	kept := w.Screens[:0]
	for _, s := range w.Screens {
		if s.Root != nil {
			kept = append(kept, s)
		}
	}
	w.Screens = kept
}

func (w *workspace) focusPane(id int) bool {
	t, s := w.sessionForPane(id), w.screenForPane(id)
	if t == nil || s == nil {
		return false
	}
	w.Active, t.Focused, s.Focused = t.ID, id, id
	return true
}

func (w *workspace) dock(source, target int, edge splitEdge) bool {
	if source == target {
		return false
	}
	from, to := w.screenForPane(source), w.screenForPane(target)
	if from == nil || to == nil {
		return false
	}
	p := from.Root.find(source).Pane
	from.Root = from.Root.remove(source)
	if from.Root != nil && from.Focused == source {
		from.Focused = from.Root.leaves()[0].ID
	}
	// Refetch after removal: collapsing a common ancestor can replace the target.
	n := to.Root.find(target)
	if n == nil {
		return false
	}
	old := &layout{ID: w.id(), Pane: n.Pane}
	added := &layout{ID: w.id(), Pane: p}
	n.ID, n.Pane, n.Vertical, n.Ratio = w.id(), nil, edge == splitTop || edge == splitBottom, 50
	n.First, n.Second = old, added
	if edge == splitLeft || edge == splitTop {
		n.First, n.Second = added, old
	}
	w.markWorkspace(to)
	w.pruneScreens()
	w.focusPane(source)
	return true
}

func (w *workspace) detach(id int) bool {
	s := w.screenForPane(id)
	if s == nil || len(s.Root.leaves()) < 2 {
		return false
	}
	p := s.Root.find(id).Pane
	s.Root = s.Root.remove(id)
	if s.Focused == id {
		s.Focused = s.Root.leaves()[0].ID
	}
	w.Screens = append(w.Screens, &screen{ID: w.id(), Root: &layout{ID: w.id(), Pane: p}, Focused: id})
	w.focusPane(id)
	return true
}

func balanceLayout(n *layout) {
	if n == nil || n.Pane != nil {
		return
	}
	n.Ratio = 50
	balanceLayout(n.First)
	balanceLayout(n.Second)
}

func (a *application) focusPane(p *pane) {
	if a.work.focusPane(p.ID) {
		p.Handle.Focus()
	}
}

func (a *application) selectScreen(s *screen) {
	if n := s.Root.find(s.Focused); n != nil {
		a.focusPane(n.Pane)
	}
}

func (a *application) cyclePane(direction int) {
	if s := a.work.activeScreen(); s != nil {
		panes := s.orderedPanes()
		for i, p := range panes {
			if p.ID == s.Focused {
				a.focusPane(panes[(i+direction+len(panes))%len(panes)])
				return
			}
		}
	}
}

func (a *application) dockPane(source, target int, edge splitEdge) {
	if a.work.dock(source, target, edge) {
		if t := a.work.active(); t != nil {
			if n := t.Root.find(t.Focused); n != nil {
				n.Pane.Handle.Focus()
			}
		}
	}
}

func (a *application) screenLabel(s *screen) string {
	label := "终端"
	if t := a.work.sessionForPane(s.Focused); t != nil {
		label = t.label()
	}
	if s.WorkspaceIndex > 0 {
		label = "Workspace"
		if s.WorkspaceIndex > 1 {
			label = fmt.Sprintf("Workspace%d", s.WorkspaceIndex)
		}
	}
	if alias := strings.TrimSpace(s.Alias); alias != "" {
		label = alias
	}
	if count := len(s.Root.leaves()); count > 1 {
		label += fmt.Sprintf(" · %d", count)
	}
	return label
}

func (w *workspace) screenByID(id int) *screen {
	for _, s := range w.Screens {
		if s.ID == id {
			return s
		}
	}
	return nil
}

func (w *workspace) reorderScreen(id, target int, before bool) bool {
	if id == target {
		return false
	}
	from := slices.IndexFunc(w.Screens, func(s *screen) bool { return s.ID == id })
	if from < 0 || target != 0 && w.screenByID(target) == nil {
		return false
	}
	s := w.Screens[from]
	w.Screens = slices.Delete(w.Screens, from, from+1)
	index := len(w.Screens)
	if target != 0 {
		index = slices.IndexFunc(w.Screens, func(s *screen) bool { return s.ID == target })
		if !before {
			index++
		}
	}
	w.Screens = slices.Insert(w.Screens, index, s)
	return true
}

// Dropping a pane on the strip detaches it; dragging a tab reorders its entire
// screen, including all its panes and alias. Session ownership never changes.
func (a *application) dropOnTabStrip(request tabDropRequest) {
	if request.Target != 0 && a.work.screenByID(request.Target) == nil {
		return
	}
	var s *screen
	if request.Drag.ScreenID != 0 {
		s = a.work.screenByID(request.Drag.ScreenID)
		if s == nil || s.Root.find(request.Drag.PaneID) == nil {
			return
		}
	} else {
		s = a.work.screenForPane(request.Drag.PaneID)
		if s == nil {
			return
		}
		if len(s.Root.leaves()) > 1 {
			a.work.detach(request.Drag.PaneID)
			s = a.work.screenForPane(request.Drag.PaneID)
		}
	}
	if s == nil {
		return
	}
	a.work.reorderScreen(s.ID, request.Target, request.Before)
	a.selectScreen(s)
}

func (a *application) requestStopScreen(s *screen, origin ui.Rect) {
	panes := s.Root.leaves()
	if len(panes) == 1 {
		a.requestStop(a.work.sessionForPane(panes[0].ID), origin)
		return
	}
	hasTerminal := false
	for _, p := range panes {
		hasTerminal = hasTerminal || p.Browser == nil
	}
	if !hasTerminal {
		for _, p := range panes {
			a.stopTab(a.work.sessionForPane(p.ID))
		}
		return
	}
	a.stopScreen, a.stopTarget, a.stopConfirm, a.modalOrigin = s, nil, true, origin
}

func (a *application) stopTabs() []*tab {
	if a.stopScreen != nil {
		var tabs []*tab
		for _, p := range a.stopScreen.Root.leaves() {
			if t := a.work.sessionForPane(p.ID); t != nil {
				tabs = append(tabs, t)
			}
		}
		return tabs
	}
	if a.stopTarget != nil {
		return []*tab{a.stopTarget}
	}
	return nil
}

func (a *application) stopSelection() {
	tabs := a.stopTabs()
	for _, t := range tabs {
		if nStarting(t) {
			return
		}
	}
	for _, t := range tabs {
		a.stopTab(t)
	}
	a.stopScreen, a.stopTarget, a.stopConfirm = nil, nil, false
}

func (a *application) groupLabel(id string) string {
	for _, g := range a.config.Groups {
		if g.ID == id {
			return g.Name
		}
	}
	return ""
}

func (w *workspace) markWorkspace(s *screen) {
	if s == nil || s.WorkspaceIndex > 0 {
		return
	}
	w.nextWorkspace++
	s.WorkspaceIndex = w.nextWorkspace
}
