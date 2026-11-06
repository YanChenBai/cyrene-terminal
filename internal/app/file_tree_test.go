package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileNodesSortAndIcons(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, "src"), 0700)
	os.Mkdir(filepath.Join(root, "node_modules"), 0700)
	for _, name := range []string{"view.go", "go.mod", ".gitignore", "README.md"} {
		if err := os.WriteFile(filepath.Join(root, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	nodes, truncated, err := readFileNodes(root)
	if err != nil || truncated || len(nodes) != 6 {
		t.Fatal(err, len(nodes))
	}
	if nodes[0].Name != "node_modules" || nodes[1].Name != "src" || nodes[2].Directory {
		t.Fatal("directories must sort first")
	}
	for _, name := range []string{"view.go", "go.mod", ".gitignore", "README.md", "build.ps1", "main.py", "data.json", "notes.txt", "unknown.abc"} {
		if fileIcons[fileIconName(name, false, false)] == nil {
			t.Fatal("missing file icon", name)
		}
	}
	for _, name := range []string{"src", "node_modules", "scripts", "tests", "resources", ".git", "ordinary"} {
		for _, open := range []bool{false, true} {
			if fileIcons[fileIconName(name, true, open)] == nil {
				t.Fatal("missing folder icon", name, open)
			}
		}
	}
}
