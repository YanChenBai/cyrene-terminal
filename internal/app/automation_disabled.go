//go:build !e2e

package app

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/terminal"
)

func installAcceptanceMenu(*mygo.Menu)                        {}
func acceptanceTerminalDescription(*terminal.Terminal) string { return "" }

func configureAcceptance()                                                  {}
func acceptanceWindowOptions(options mygo.WindowOptions) mygo.WindowOptions { return options }
func showAcceptanceWindow(*mygo.Window)                                     {}
