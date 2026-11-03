package app

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"os"
	"path/filepath"
	"testing"
	"time"
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
func TestReadOnlyFileTreeCollapseAndSelection(t *testing.T) {
	root := t.TempDir()
	node := func(name string) *fileNode { return &fileNode{Path: filepath.Join(root, name), Name: name} }
	src := node("src")
	src.Directory = true
	src.Loaded = true
	src.Open = true
	src.Children = []*fileNode{{Path: filepath.Join(src.Path, "view.go"), Name: "view.go"}, {Path: filepath.Join(src.Path, "view_test.go"), Name: "view_test.go"}}
	scripts := node("scripts")
	scripts.Directory = true
	scripts.Loaded = true
	scripts.Open = true
	scripts.Children = []*fileNode{{Path: filepath.Join(scripts.Path, "build.ps1"), Name: "build.ps1"}, {Path: filepath.Join(scripts.Path, "native-smoke.py"), Name: "native-smoke.py"}}
	modules := node("node_modules")
	modules.Directory = true
	modules.Loaded = true
	a := application{config: defaultSettings()}
	a.work.add(profile{ID: "shell"}, root)
	a.inspector = inspectorState{Directory: root, Root: &fileNode{Path: root, Name: "cyrene-terminal", Directory: true, Open: true, Loaded: true, Children: []*fileNode{src, modules, scripts, node(".gitignore"), node("go.mod"), node("LICENSE"), node("mygo.json"), node("README.md")}}}
	tt := ui.NewTester(a.view, 1200, 760)
	if err := tt.Click("切换右侧栏"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(350 * time.Millisecond)
	if err := tt.Click("文件 view.go"); err != nil {
		t.Fatal(err)
	}
	if a.inspector.Selected != filepath.Join(src.Path, "view.go") {
		t.Fatal("file selection failed")
	}
	saveInteractionFrame(t, tt, "file-tree-panel")
	if err := tt.Click("目录 src"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(350 * time.Millisecond)
	if _, ok := tt.Find("文件 view.go"); ok {
		t.Fatal("closed folder still exposes descendants")
	}
	if err := tt.Click("目录 src"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(350 * time.Millisecond)
	if _, ok := tt.Find("文件 view.go"); !ok {
		t.Fatal("reopened folder lost descendants")
	}
	if err := tt.Click("切换右侧栏"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(350 * time.Millisecond)
	if _, ok := tt.Find("刷新文件树"); ok {
		t.Fatal("right sidebar did not hide")
	}
	if _, err := os.Stat(filepath.Join(root, "view.go")); !os.IsNotExist(err) {
		t.Fatal("view-only file selection touched the filesystem")
	}
}

func TestFileTreeScrollsLongDirectory(t *testing.T) {
	dir := t.TempDir()
	root := &fileNode{Path: dir, Name: "project", Directory: true, Open: true, Loaded: true}
	for i := 0; i < 120; i++ {
		name := fmt.Sprintf("file-%03d.go", i)
		root.Children = append(root.Children, &fileNode{Path: filepath.Join(dir, name), Name: name})
	}
	a := application{config: defaultSettings()}
	a.work.add(profile{ID: "shell"}, dir)
	a.inspector = inspectorState{Open: true, Directory: dir, Root: root}
	tt := ui.NewTester(a.view, 1100, 620)
	before, ok := tt.Find("文件 file-000.go")
	if !ok {
		t.Fatal("missing first row")
	}
	tt.Wheel(1000, 400, 0, 500)
	tt.Advance(time.Second)
	after, _ := tt.Find("文件 file-000.go")
	if after.H > 0 && after.Y >= before.Y-30 {
		t.Fatalf("wheel did not scroll: before=%+v after=%+v", before, after)
	}
	tt.Scroll(1000, 400, 0, 10000)
	tt.Advance(time.Second)
	last, ok := tt.Find("文件 file-119.go")
	if !ok || last.Y < 32 || last.Y+last.H > 620 {
		t.Fatalf("last item cannot be reached: %+v", last)
	}
	saveInteractionFrame(t, tt, "file-tree-scrolled")
}

func TestRightSidebarResizeRetainsWidth(t *testing.T) {
	a := application{config: defaultSettings()}
	a.inspector.Open = true
	tt := ui.NewTester(a.view, 1200, 700)
	grip, ok := tt.Find("调整右侧栏宽度")
	if !ok {
		t.Fatal("missing resize grip")
	}
	original := a.inspector.Width
	tt.Press(grip.X+3, 320)
	tt.Move(grip.X-77, 320)
	tt.Release(grip.X-77, 320)
	if a.inspector.Width < original+70 || a.inspector.Width > original+90 {
		t.Fatalf("drag width=%v original=%v", a.inspector.Width, original)
	}
	resized := a.inspector.Width
	if err := tt.Click("切换右侧栏"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(350 * time.Millisecond)
	if _, ok := tt.Find("调整右侧栏宽度"); ok {
		t.Fatal("collapsed sidebar retains resize grip")
	}
	if err := tt.Click("切换右侧栏"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(350 * time.Millisecond)
	if a.inspector.Width != resized {
		t.Fatalf("lost width after toggle: %v", a.inspector.Width)
	}
	saveInteractionFrame(t, tt, "file-tree-resized")
}
