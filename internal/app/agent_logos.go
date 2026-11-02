package app

import (
	"strings"

	"github.com/YanChenBai/cyrene-terminal/assets"

	"github.com/egoist/mygo/ui"
)

var agentLogos = func() map[string]*ui.SVG {
	logos := map[string]*ui.SVG{}
	for _, id := range []string{"codex", "claude", "pi", "omp", "opencode"} {
		data, err := assets.FS.ReadFile("agents/" + id + ".svg")
		if err != nil {
			panic(err)
		}
		// Resolve the website's CSS mark to currentColor for native SVG.
		if id == "pi" {
			data = []byte(strings.ReplaceAll(string(data), `class="mark"`, `fill="currentColor"`))
		}
		logos[id] = ui.MustParseSVG(data)
	}
	return logos
}()

func agentLogo(c *ui.Context, id string, size float32) {
	// Equal slots keep the labels aligned; optical sizes compensate for the
	// marks' different silhouettes and internal SVG padding.
	ui.Box(c).Size(size, size).Center().Children(func() {
		optical := size
		switch id {
		case "codex", "claude":
			optical = size * 0.88
		case "pi":
			optical = size * 0.86
		case "omp":
			optical = size * 1.10
		case "opencode":
			optical = size * 1.04
		default:
			optical = size * 1.12
		}
		drawAgentLogo(c, id, optical)
	})
}

func drawAgentLogo(c *ui.Context, id string, size float32) {
	if logo := agentLogos[id]; logo != nil {
		color := ui.Hex("#f5f5f5")
		if id == "pi" {
			color = ui.Hex("#bca3f5")
		}
		if id == "opencode" {
			color = ui.Hex("#75c8c0")
		}
		// Image preserves multicolor SVG fills; Icon would flatten the logo.
		ui.Image(c, logo).Size(size, size).Fit(ui.Contain).TextColor(color)
		return
	}
	ui.Icon(c, terminalLogo).Size(size, size).TextColor(ui.Hex("#9aa7bd"))
}

var terminalLogo = svgIcon(`<path d="m5 7 5 5-5 5M13 17h6"/>`)
