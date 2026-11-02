package app

import "testing"

func TestRemoveActiveTabSelectsNeighbor(t *testing.T) {
	var w workspace
	p := profile{ID: "shell", Name: "Terminal"}
	w.add(p, "a")
	first := w.Active
	w.add(p, "b")
	second := w.Active
	w.add(p, "c")
	w.Active = second
	w.removeTab(second)
	if w.Active == first || w.active() == nil {
		t.Fatal("expected right neighbor")
	}
	w.removeTab(w.Active)
	if w.Active != first {
		t.Fatal("expected remaining left neighbor")
	}
	w.removeTab(999)
	if len(w.Tabs) != 1 {
		t.Fatal("unknown identities should not mutate workspace")
	}
}
