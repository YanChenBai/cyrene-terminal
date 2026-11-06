package app

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestChildDragReordersAndPersistsWithoutChangingLayout(t *testing.T) {
	a, first, second := tilingTestApp(t)
	a.work.dock(second.ID, first.ID, splitRight)
	s := a.work.activeScreen()
	root := s.Root
	starts := 0
	a.start = func(*pane) { starts++ }
	tt := ui.NewTester(a.view, 1100, 720)
	tt.Advance(time.Second)
	from, _ := tt.Find(fmt.Sprintf("子标签 %d", second.ID))
	to, _ := tt.Find(fmt.Sprintf("子标签 %d", first.ID))
	tt.Press(from.X+10, from.Y+12)
	tt.Move(to.X+3, to.Y+12)
	tt.Advance(150 * time.Millisecond)
	tt.Release(to.X+3, to.Y+12)
	tt.Advance(300 * time.Millisecond)
	if !slices.Equal(s.paneOrder(), []int{second.ID, first.ID}) || s.Root != root || starts != 0 {
		t.Fatalf("child drop changed layout/ownership or failed to reorder: %v", s.paneOrder())
	}
	b := &application{config: a.config}
	if err := b.restoreWorkspace(a.snapshotWorkspace()); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(b.work.activeScreen().paneOrder(), s.paneOrder()) {
		t.Fatal("child order was not persisted")
	}
	saveInteractionFrame(t, tt, "frosted-child-reorder")
}

func TestExternalWorkspaceDropJoinsWholeLayout(t *testing.T) {
	a, first, second := tilingTestApp(t)
	a.work.dock(second.ID, first.ID, splitBottom)
	from := a.work.activeScreen()
	root := from.Root
	third := a.work.add(profile{ID: "shell"}, t.TempDir())
	to := a.work.activeScreen()
	to.Alias = "Destination"
	starts := 0
	a.start = func(*pane) { starts++ }
	tt := ui.NewTester(a.view, 1100, 720)
	tt.Advance(time.Second)
	source, _ := tt.Find(fmt.Sprintf("内容标签 %d", from.ID))
	target, _ := tt.Find(fmt.Sprintf("内容标签 %d", to.ID))
	tt.Press(source.X+15, source.Y+15)
	tt.Move(target.X+target.W/2, target.Y+15)
	tt.Advance(150 * time.Millisecond)
	tt.Release(target.X+target.W/2, target.Y+15)
	tt.Advance(300 * time.Millisecond)
	if len(a.work.Screens) != 1 || to.Root.Second != root || len(to.Root.leaves()) != 3 || to.Alias != "Destination" || starts != 0 {
		t.Fatal("group drop did not preserve and join the source layout")
	}
	if a.work.sessionForPane(second.ID).GroupID != "beta" || a.work.sessionForPane(third.ID) == nil {
		t.Fatal("group drop changed session ownership")
	}
	assertUniquePanes(t, &a.work)
}

func TestPaneActionsAnimateOnFocusAndHover(t *testing.T) {
	a, first, second := tilingTestApp(t)
	a.work.dock(second.ID, first.ID, splitRight)
	a.work.focusPane(first.ID)
	var firstWidth, secondWidth float32
	observe := func(c *ui.Context) {
		a.view(c)
		firstWidth = first.Actions.Bounds(c).W
		secondWidth = second.Actions.Bounds(c).W
	}
	tt := ui.NewTester(observe, 1100, 720)
	tt.Move(0, 500)
	tt.Advance(time.Second)
	if firstWidth < 70 || secondWidth > 1 {
		t.Fatalf("focus visibility wrong: %v %v", firstWidth, secondWidth)
	}
	other, _ := tt.Find(fmt.Sprintf("窗格 %d", second.ID))
	tt.Move(other.X+20, other.Y+14)
	tt.Advance(60 * time.Millisecond)
	if secondWidth <= 0 || secondWidth >= 78 {
		t.Fatalf("hover must interpolate: %v", secondWidth)
	}
	tt.Advance(250 * time.Millisecond)
	if secondWidth < 70 || a.work.activeScreen().Focused != first.ID {
		t.Fatal("hover should reveal buttons without changing focus")
	}
	tt.Move(0, 500)
	tt.Advance(250 * time.Millisecond)
	if secondWidth > 1 {
		t.Fatal("unfocused pane controls did not hide")
	}
	if err := tt.Click(fmt.Sprintf("子标签 %d", second.ID)); err != nil {
		t.Fatal(err)
	}
	tt.Move(0, 500)
	tt.Advance(250 * time.Millisecond)
	if firstWidth > 1 || secondWidth < 70 {
		t.Fatal("child selection did not move focused controls")
	}
	if err := tt.Click(fmt.Sprintf("关闭子标签 %d", first.ID)); err != nil {
		t.Fatal(err)
	}
	tt.Advance(300 * time.Millisecond)
	if !a.stopConfirm || a.stopTarget != a.work.sessionForPane(first.ID) {
		t.Fatal("child close targeted another session")
	}
	saveInteractionFrame(t, tt, "frosted-focused-controls")
}

func TestClaudeHookPreservesSettings(t *testing.T) {
	dir := t.TempDir()
	settings := `{"model":"sonnet","hooks":{"SessionStart":[{"hooks":[{"type":"command","command":"echo existing"}]}]}}`
	if err := os.WriteFile(filepath.Join(dir, "claude.json"), []byte(settings), 0600); err != nil {
		t.Fatal(err)
	}
	command, err := prepareClaudeCommand([]string{"claude", "--resume", "--settings", "claude.json"}, filepath.Join(dir, "sessions", "pane.id"), dir)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(command[:3], []string{"claude", "--resume", "--settings"}) {
		t.Fatalf("changed resume command: %v", command)
	}
	var decoded map[string]any
	if err = json.Unmarshal([]byte(command[3]), &decoded); err != nil {
		t.Fatal(err)
	}
	hooks := decoded["hooks"].(map[string]any)["SessionStart"].([]any)
	if decoded["model"] != "sonnet" || len(hooks) != 2 {
		t.Fatal("per-launch hook replaced existing settings/hooks")
	}
	if !slices.Equal((profile{Command: []string{"claude", "--model", "sonnet"}}).resumePickerCommand(), []string{"claude", "--model", "sonnet", "--resume"}) {
		t.Fatal("Claude picker changed model options")
	}
}

func TestPaneLayoutInterpolatesOnAddAndRemove(t *testing.T) {
	a, first, second := tilingTestApp(t)
	a.work.focusPane(first.ID)
	var width float32
	tt := ui.NewTester(func(c *ui.Context) { a.view(c); width = first.Surface.Bounds(c).W }, 1100, 720)
	tt.Advance(time.Second)
	tt.Frame()
	wide := width
	a.work.dock(second.ID, first.ID, splitRight)
	tt.Frame()
	tt.Advance(60 * time.Millisecond)
	tt.Frame()
	during := width
	tt.Advance(300 * time.Millisecond)
	tt.Frame()
	narrow := width
	if !(wide > during && during > narrow) {
		t.Fatalf("addition jumped instead of interpolating: %v %v %v", wide, during, narrow)
	}
	a.work.removeTab(a.work.sessionForPane(second.ID).ID)
	tt.Frame()
	tt.Advance(60 * time.Millisecond)
	tt.Frame()
	during = width
	tt.Advance(300 * time.Millisecond)
	tt.Frame()
	if !(narrow < during && during < width) {
		t.Fatalf("removal jumped instead of interpolating: %v %v %v", narrow, during, width)
	}
}

func TestSessionStartHookCapturesIDWithoutModelRequest(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows per-launch hook")
	}
	dir := t.TempDir()
	record := filepath.Join(dir, "pane.id")
	if _, err := sessionHook(record); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("powershell.exe", "-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", filepath.Join(dir, "session-start.ps1"))
	cmd.Env = codexEnvironment(os.Environ(), record)
	cmd.Stdin = strings.NewReader(`{"session_id":"` + testAgentSession + `"}`)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("hook: %v %s", err, out)
	}
	if readAgentRecord(record) != testAgentSession {
		t.Fatal("hook failed to bind its exact pane session")
	}
}
