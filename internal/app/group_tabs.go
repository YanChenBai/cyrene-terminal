package app

import "slices"

type groupDropRequest struct {
	Drag         terminalDrag
	Screen, Pane int
	Before       bool
}

func (s *screen) orderedPanes() []*pane {
	all := s.Root.leaves()
	result := make([]*pane, 0, len(all))
	seen := map[int]bool{}
	for _, id := range s.Order {
		for _, p := range all {
			if p.ID == id && !seen[id] {
				result = append(result, p)
				seen[id] = true
			}
		}
	}
	for _, p := range all {
		if !seen[p.ID] {
			result = append(result, p)
		}
	}
	return result
}

func (s *screen) paneOrder() []int {
	ids := make([]int, 0)
	for _, p := range s.orderedPanes() {
		ids = append(ids, p.ID)
	}
	return ids
}

func (s *screen) placeChildren(ids []int, target int, before bool) {
	order := s.paneOrder()
	order = slices.DeleteFunc(order, func(id int) bool { return slices.Contains(ids, id) })
	index := slices.Index(order, target)
	if index < 0 {
		index = len(order)
	} else if !before {
		index++
	}
	s.Order = slices.Insert(order, index, ids...)
}

// Merge layout ownership without changing session or live process/browser ownership.
func (a *application) dropInGroup(r groupDropRequest) {
	to := a.work.screenByID(r.Screen)
	from := a.work.screenForPane(r.Drag.PaneID)
	if to == nil || from == nil || (r.Pane != 0 && to.Root.find(r.Pane) == nil) {
		return
	}
	if r.Drag.ScreenID != 0 && r.Drag.ScreenID != from.ID {
		return
	}
	if from == to {
		if r.Drag.ScreenID == 0 && r.Pane != r.Drag.PaneID {
			to.placeChildren([]int{r.Drag.PaneID}, r.Pane, r.Before)
			a.selectScreen(to)
		}
		return
	}
	a.work.markWorkspace(to)
	to.Order = to.paneOrder()
	ids := []int{r.Drag.PaneID}
	if r.Drag.ScreenID != 0 {
		ids = from.paneOrder()
		to.Root = &layout{ID: a.work.id(), First: to.Root, Second: from.Root, Ratio: 50}
		from.Root = nil
		a.work.pruneScreens()
		a.work.focusPane(r.Drag.PaneID)
	} else {
		target := r.Pane
		if target == 0 {
			target = to.Focused
		}
		if !a.work.dock(r.Drag.PaneID, target, splitRight) {
			return
		}
	}
	to.placeChildren(ids, r.Pane, r.Before)
	a.selectScreen(to)
}
