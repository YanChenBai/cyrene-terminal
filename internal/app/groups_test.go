package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGroupPersistenceEditingAndRemovalPreserveTerminals(t *testing.T) {
	a := &application{config: defaultSettings()}
	dir := t.TempDir()
	a.editGroup(nil)
	a.groupName, a.groupDirectory = "project", dir
	a.saveGroup()
	if len(a.config.Groups) != 1 {
		t.Fatal(a.groupError)
	}
	g := a.config.Groups[0]
	a.openLauncher(g.ID)
	a.launch(profile{ID: "shell", Name: "Terminal"})
	p := a.work.active().Root.Pane
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := saveSettings(path, a.config); err != nil {
		t.Fatal(err)
	}
	s, err := loadSettings(path)
	if err != nil || len(s.Groups) != 1 || s.Groups[0] != g {
		t.Fatal("group binding should survive app restart", err)
	}
	newDir := t.TempDir()
	a.editGroup(&g)
	a.groupDirectory = newDir
	a.saveGroup()
	if a.directoryForGroup(g.ID) != newDir || p.Dir != dir {
		t.Fatal("editing a group must affect new terminals without changing existing ones")
	}
	a.removeGroup(g.ID)
	if a.work.active().GroupID != "" || a.work.active().Root.Pane != p || p.Dir != dir {
		t.Fatal("removing a group must retain live terminal identity and directory")
	}
	home, _ := os.UserHomeDir()
	if a.directoryForGroup("") != home {
		t.Fatal("ungrouped default must be home")
	}
}

func TestGroupRejectsMissingDirectory(t *testing.T) {
	a := &application{config: defaultSettings()}
	a.groupName, a.groupDirectory = "bad", filepath.Join(t.TempDir(), "missing")
	a.saveGroup()
	if len(a.config.Groups) != 0 || a.groupError == "" {
		t.Fatal("missing directory must be reported without accepting a group")
	}
}
