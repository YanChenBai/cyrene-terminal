//go:build e2e

package app

import (
	"crypto/sha256"
	"fmt"
	"os"

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
