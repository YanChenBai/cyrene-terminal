package app

import (
	"github.com/egoist/mygo/plugins/terminal"
	"io"
	"net/url"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReportedDirectoryAndRestartLocation(t *testing.T) {
	r, w := io.Pipe()
	term, err := terminal.New(terminal.Options{Conn: &graphicsConn{r, w}, Transparent: true})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	original, changed := t.TempDir(), t.TempDir()
	p := &pane{Dir: original, Term: term}
	if p.directory() != original {
		t.Fatal("lost initial directory")
	}
	path := filepath.ToSlash(changed)
	if runtime.GOOS == "windows" {
		path = "/" + path
	}
	uri := (&url.URL{Scheme: "file", Path: path}).String()
	term.Feed([]byte("\x1b]7;" + uri + "\x07"))
	if filepath.Clean(p.directory()) != filepath.Clean(changed) {
		t.Fatal("stale directory", p.directory(), changed)
	}
	tab := &tab{Root: &layout{Pane: p}, Focused: p.ID}
	term.Feed([]byte("\x1b]0;Unrelated agent title\x07"))
	if tab.label() != filepath.Base(changed) {
		t.Fatal("name did not follow directory", tab.label())
	}
	tab.Alias = "Review"
	if tab.label() != "Review" {
		t.Fatal("directory overrode alias")
	}
	tab.Alias = ""
	if tab.label() != filepath.Base(changed) {
		t.Fatal("clearing alias did not restore directory")
	}
	a := application{sessions: map[*terminal.Terminal]bool{term: true}}
	a.restartPane(p)
	if p.Dir != changed || p.Term != nil {
		t.Fatal("restart discarded live directory")
	}
}
func TestPowerShellDirectoryPrompt(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("PowerShell integration")
	}
	shell, err := exec.LookPath("powershell.exe")
	if err != nil {
		t.Skip(err)
	}
	dir := t.TempDir()
	script := `function global:prompt { 'CustomPrompt> ' };` + powerShellDirectoryIntegration + `;Set-Location -LiteralPath '` + strings.ReplaceAll(dir, "'", "''") + `';prompt`
	out, err := exec.Command(shell, "-NoLogo", "-NoProfile", "-Command", script).CombinedOutput()
	if err != nil || !strings.Contains(string(out), "\x1b]7;file:///") || !strings.Contains(string(out), "CustomPrompt>") {
		t.Fatalf("prompt integration: %v %q", err, out)
	}
}
