//go:build e2e

package app

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	_ "github.com/egoist/mygo/plugins/automation"
	"github.com/egoist/mygo/plugins/terminal"
)

// The acceptance build exposes the exact tray menu callbacks to the native
// automation protocol. Production has no application exit menu or shortcut.
func installAcceptanceMenu(menu *mygo.Menu) { mygo.App.SetMenu(menu) }

func acceptanceTerminalDescription(term *terminal.Terminal) string { return term.Text() }

func configureAcceptance() {
	instance := os.Getenv("CYRENE_ACCEPTANCE_INSTANCE")
	if instance == "" {
		instance = os.Getenv("MYGO_AUTOMATION")
	}
	if instance != "" {
		sum := sha256.Sum256([]byte(instance))
		mygo.App.SetName(fmt.Sprintf("cyrene-terminal-acceptance-%x", sum[:8]))
	}
}

func acceptanceWindowOptions(options mygo.WindowOptions) mygo.WindowOptions {
	if os.Getenv("CYRENE_ACCEPTANCE_NO_FOCUS") == "1" {
		options.Hidden = true
	}
	return options
}

func showAcceptanceWindow(win *mygo.Window) {
	if os.Getenv("CYRENE_ACCEPTANCE_NO_FOCUS") == "1" {
		win.SetPosition(-10000, 0)
		win.ShowInactive()
	}
}

// Native window captures omit the embedded page's separate compositor surface.
func captureAcceptanceBrowser(view *mygo.WebView, id int) {
	dir := os.Getenv("CYRENE_ACCEPTANCE_BROWSER_CAPTURE")
	if dir == "" {
		return
	}
	go func() {
		data, err := view.CapturePage()
		if err != nil {
			return
		}
		if os.MkdirAll(dir, 0700) != nil {
			return
		}
		path := filepath.Join(dir, fmt.Sprintf("pane-%d.png", id))
		if os.WriteFile(path+".tmp", data, 0600) == nil {
			_ = os.Rename(path+".tmp", path)
		}
	}()
}
