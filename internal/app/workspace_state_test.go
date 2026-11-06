package app

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestWorkspaceRestartRestoresLayoutAndExactAgent(t *testing.T) {
	a, first, second := tilingTestApp(t)
	first.Profile = profile{ID: "claude", Command: []string{"claude", "--model", "sonnet"}}
	first.AgentSession = testAgentSession
	a.work.dock(second.ID, first.ID, splitRight)
	a.work.activeScreen().Alias = "Development"
	a.work.activeScreen().Root.Ratio = 37
	a.collapsed = true
	a.sidebarWidth = 275
	a.inspector.Width = 310
	before := a.snapshotWorkspace()
	path := filepath.Join(t.TempDir(), "workspace.json")
	if err := writeWorkspace(path, before); err != nil {
		t.Fatal(err)
	}
	saved, err := readWorkspace(path)
	if err != nil {
		t.Fatal(err)
	}
	b := &application{config: defaultSettings()}
	if err = b.restoreWorkspace(saved); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, b.snapshotWorkspace()) {
		t.Fatal("restart lost workspace identity, alias, group, directory, split ratio or sidebar state")
	}
	p := b.work.sessionForPane(first.ID).Root.find(first.ID).Pane
	if !p.NeedsResume || !slices.Equal(p.Command, []string{"claude", "--model", "sonnet", "--resume", testAgentSession}) {
		t.Fatalf("restored a fresh agent: %+v", p.Command)
	}
	assertUniquePanes(t, &b.work)
	// Replacing an existing snapshot must work on Windows as well.
	if err = writeWorkspace(path, b.snapshotWorkspace()); err != nil {
		t.Fatal(err)
	}
}

func TestUnknownAgentRestoreOpensNativeHistory(t *testing.T) {
	a := &application{config: defaultSettings()}
	a.work.add(profile{ID: "codex", Name: "Codex", Command: []string{"codex"}}, t.TempDir())
	b := &application{config: defaultSettings()}
	if err := b.restoreWorkspace(a.snapshotWorkspace()); err != nil {
		t.Fatal(err)
	}
	p := b.work.active().Root.Pane
	if !p.NeedsResume || p.Error != "" || !slices.Equal(p.Command, []string{"codex", "resume"}) {
		t.Fatal("missing association must use native history, never a fresh or latest session")
	}

}

func TestWorkspaceRejectsDuplicateOrMissingPane(t *testing.T) {
	a, _, _ := tilingTestApp(t)
	s := a.snapshotWorkspace()
	s.Screens[1].Root.PaneID = s.Screens[0].Root.PaneID
	b := &application{}
	if b.restoreWorkspace(s) == nil || len(b.work.Tabs) != 0 {
		t.Fatal("invalid ownership was accepted or partially applied")
	}
}

func TestAgentRecordValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.id")
	for _, id := range []string{"", "not-an-id; echo injected", testAgentSession} {
		if err := os.WriteFile(path, []byte(id), 0600); err != nil {
			t.Fatal(err)
		}
		got := readAgentRecord(path)
		if id == testAgentSession && got != id || id != testAgentSession && got != "" {
			t.Fatalf("invalid ID binding %q", got)
		}
	}
}

func TestManyTabsScrollWithinCenterAndRevealSelection(t *testing.T) {
	a := &application{config: defaultSettings()}
	for range 35 {
		a.work.add(profile{ID: "shell", Name: "Terminal"}, t.TempDir())
	}
	tt := ui.NewTester(a.view, 1200, 720)
	tt.Advance(time.Second)
	if a.tabScroll.MaxX <= 0 || a.tabScroll.X <= 0 {
		t.Fatalf("tab strip grew instead of scrolling: %+v", a.tabScroll)
	}
	tools, _ := tt.Find("外观设置")
	last, _ := tt.Find("内容标签 104")
	if tools.X < 1100 || last.X+last.W > tools.X {
		t.Fatalf("tabs invaded window tools: tab=%+v tools=%+v", last, tools)
	}
	tt.Scroll(400, 16, -100000, 0)
	tt.Advance(time.Second)
	if a.tabScroll.X != 0 {
		t.Fatal("horizontal scroll did not reach the first tab")
	}
	tt.Wheel(400, 16, 0, 120)
	tt.Advance(time.Second)
	if a.tabScroll.X <= 0 {
		t.Fatal("ordinary mouse wheel did not scroll tabs horizontally")
	}
	a.selectScreen(a.work.Screens[len(a.work.Screens)-1])
	a.revealedScreen = 0
	tt.Frame()
	tt.Advance(time.Second)
	if a.tabScroll.X <= 0 {
		t.Fatal("selected tab was not revealed")
	}
	saveInteractionFrame(t, tt, "tabs-overflow-scroll")
}
