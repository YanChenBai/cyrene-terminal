// Package assets contains the application branding and agent marks.
package assets

import "embed"

// FS holds the original agent marks. Attribution is in agents/README.md.
//go:embed agents/*.svg
var FS embed.FS

// TrayIcon is generated from cyrene.svg by go run ./scripts/render-icon.
//go:embed cyrene-tray.png
var TrayIcon []byte
