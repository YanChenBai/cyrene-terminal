package app

import (
	"io"
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

const testAgentSession = "019a1234-5678-7123-8123-123456789abc"

func TestResumeCommandsAlwaysTargetExactSession(t *testing.T) {
	for _, tc := range []struct{ command, want []string }{
		{[]string{"codex", "--no-alt-screen"}, []string{"codex", "resume", "--no-alt-screen", testAgentSession}},
		{[]string{"claude", "--model", "sonnet"}, []string{"claude", "--model", "sonnet", "--resume", testAgentSession}},
		{[]string{"pi"}, []string{"pi", "--session", testAgentSession}},
		{[]string{"omp"}, []string{"omp", "--resume", testAgentSession}},
		{[]string{"opencode"}, []string{"opencode", "--session", testAgentSession}},
		{[]string{"unknown"}, nil},
		{nil, nil},
	} {
		p := profile{Command: tc.command}
		if got := p.resumeCommand(testAgentSession); !slices.Equal(got, tc.want) {
			t.Fatalf("%v: %v != %v", tc.command, got, tc.want)
		}
		if got := p.resumeCommand(""); got != nil {
			t.Fatal("missing ID must never fall back to picker/latest")
		}
	}
	p := profile{ResumeCommand: []string{"wrapper", "--thread={session}"}}
	if got := p.resumeCommand(testAgentSession); !slices.Equal(got, []string{"wrapper", "--thread=" + testAgentSession}) {
		t.Fatal(got)
	}
	if p.ResumeCommand[1] != "--thread={session}" {
		t.Fatal("resume mutated configuration")
	}
}

func TestFreshAgentIDsAreScopedToEachPane(t *testing.T) {
	for _, name := range []string{"claude", "pi"} {
		p, q := &pane{Profile: profile{Command: []string{name}}}, &pane{Profile: profile{Command: []string{name}}}
		prepareAgentSession(p)
		prepareAgentSession(q)
		if !agentSessionID.MatchString(p.AgentSession) || p.AgentSession == q.AgentSession {
			t.Fatal("fresh panes need distinct IDs")
		}
		if !slices.Equal(p.Command, []string{name, "--session-id", p.AgentSession}) || len(p.Profile.Command) != 1 {
			t.Fatal("new session command mutated profile")
		}
	}
}

func TestLauncherOnlyCreatesNewSessions(t *testing.T) {
	a := &application{config: defaultSettings()}
	a.openLauncher("")
	tt := ui.NewTester(a.view, 1000, 720)
	tt.Advance(time.Second)
	if tt.HasText("恢复") || tt.HasText("恢复会话") {
		t.Fatal("plus launcher must only create")
	}
	if _, ok := tt.Find("恢复 Codex 会话"); ok {
		t.Fatal("global restore entry must not exist")
	}
}

func TestExitedAgentResumesInsideOriginalTab(t *testing.T) {
	r, w := io.Pipe()
	term, err := terminal.New(terminal.Options{Conn: &graphicsConn{r, w}})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	a := &application{config: defaultSettings()}
	p := a.work.add(profile{ID: "codex", Command: []string{"codex"}}, t.TempDir())
	tab := a.work.active()
	tab.Alias, tab.GroupID = "Review", "project"
	p.Term, p.AgentSession = term, testAgentSession
	starts := 0
	a.start = func(got *pane) {
		starts++
		if got != p || !slices.Equal(got.Command, []string{"codex", "resume", testAgentSession}) {
			t.Fatal("restore replaced pane or launched a fresh conversation")
		}
	}
	a.resumePane(p)
	if starts != 0 {
		t.Fatal("resume terminated a running agent")
	}
	_ = w.Close()
	select {
	case <-term.Done():
	case <-time.After(time.Second):
		t.Fatal("test terminal did not exit")
	}
	tt := ui.NewTester(a.view, 1000, 720)
	if err := tt.Click("恢复会话"); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		tt.Frame()
	}
	if starts != 1 || len(a.work.Tabs) != 1 || a.work.active() != tab || tab.Alias != "Review" || tab.GroupID != "project" {
		t.Fatal("resume lost tab identity")
	}
	if !slices.Equal(p.Profile.Command, []string{"codex"}) {
		t.Fatal("restore corrupted original launch profile")
	}
}

func TestMissingAgentIDUsesNativePickerWithoutGuessing(t *testing.T) {
	r, w := io.Pipe()
	term, err := terminal.New(terminal.Options{Conn: &graphicsConn{r, w}})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	a := &application{config: defaultSettings()}
	p := a.work.add(profile{ID: "codex", Command: []string{"codex"}}, t.TempDir())
	p.Term = term
	_ = w.Close()
	select {
	case <-term.Done():
	case <-time.After(time.Second):
		t.Fatal("test terminal did not exit")
	}
	starts := 0
	a.start = func(*pane) { starts++ }
	tt := ui.NewTester(a.view, 1000, 720)
	if err := tt.Click("恢复会话"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	if a.resumeDialog || starts != 1 || p.AgentSession != "" || !slices.Equal(p.Command, []string{"codex", "resume"}) {
		t.Fatal("missing association must open native history, without copying IDs or guessing")
	}
}
