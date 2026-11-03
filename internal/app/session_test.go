package app

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"time"
)

func TestSessionReadableTitleAndAlias(t *testing.T) {
	for title, want := range map[string]string{`C:\Program Files\PowerShell\7\pwsh.exe`: "PowerShell", `C:\Windows\System32\cmd.exe`: "Command Prompt", "Yazi: ~/Pictures": "Yazi: ~/Pictures"} {
		if got := displayTitle(title); got != want {
			t.Fatalf("%s: %s", title, got)
		}
	}
	w := workspace{}
	w.add(profile{Name: "Terminal"}, "project")
	tab := w.active()
	tab.Alias = "  Review  "
	if tab.label() != "Review" {
		t.Fatal(tab.label())
	}
	tab.Alias = " "
	if tab.label() != "project" {
		t.Fatal(tab.label())
	}
}
func TestRestartPreservesSessionIdentity(t *testing.T) {
	a := application{}
	p := a.work.add(profile{Name: "Codex"}, "project")
	tab := a.work.active()
	tab.Alias = "Review"
	tab.GroupID = "g"
	starts := 0
	a.start = func(got *pane) {
		if got != p {
			t.Fatal("replaced session")
		}
		starts++
		got.Starting = true
	}
	a.restartPane(p)
	a.restartPane(p)
	if starts != 1 || tab.Alias != "Review" || tab.GroupID != "g" || p.Dir != "project" {
		t.Fatal("restart lost identity or duplicated launch")
	}
}

func TestAliasEditorSavesAndClears(t *testing.T) {
	a := application{config: defaultSettings()}
	a.work.add(profile{Name: "Terminal"}, "project")
	tab := a.work.active()
	a.aliasTarget = tab
	a.aliasDraft = "  Review  "
	a.aliasEditor = true
	tt := ui.NewTester(a.view, 900, 700)
	tt.Advance(time.Second)
	if _, ok := tt.Find("拖动窗口"); !ok {
		t.Fatal("modal blocks window dragging surface")
	}
	if err := tt.Click("保存"); err != nil {
		t.Fatal(err)
	}
	if tab.label() != "Review" || a.aliasEditor {
		t.Fatal("alias not saved")
	}
	tt.Advance(time.Second)
	a.aliasEditor = true
	a.aliasDraft = ""
	tt.Frame()
	tt.Advance(time.Second)
	if err := tt.Click("保存"); err != nil {
		t.Fatal(err)
	}
	if tab.label() != "project" {
		t.Fatal("automatic title not restored")
	}
}
