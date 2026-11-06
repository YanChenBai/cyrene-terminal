package app

import (
	"time"

	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
)

const workspaceTitleHeight = 50

// Desktop blur belongs to the native window, keeping pane alpha independent.
type frostedSurface struct{ Clarity, Light float32 }

func (g frostedSurface) PaintMaterial(p *ui.Painter, box ui.Rect, radii [4]float32) {
	light := g.Light * (1 - .35*g.Clarity)
	p.Fill(box, ui.RGBA(230, 232, 240, light), radii[0])
}

func plainSurface(e ui.Element, clarity, light float32) ui.Element {
	return e.Material(frostedSurface{Clarity: clarity, Light: light}).Border(.5, ui.Transparent)
}

// Popups frost actual in-window content, without refraction or a lit rim.
type frostedOverlay struct{ Clarity float32 }

func (g frostedOverlay) PaintMaterial(p *ui.Painter, box ui.Rect, radii [4]float32) {
	glass.Blur{Radius: 22 + 10*(1-g.Clarity), Replace: true}.PaintMaterial(p, box, radii)
	p.Fill(box, ui.RGBA(26, 27, 31, .32-.08*g.Clarity), radii[0])
}

// Raised edges are optical shading only; desktop translucency stays unchanged.
func beveledSurface(e ui.Element, clarity, light, strength float32) ui.Element {
	return plainSurface(e, clarity, light).BorderLinearGradient(ui.LinearGradient{Angle: 180, Stops: []ui.GradientStop{
		{Color: ui.RGBA(244, 240, 255, strength), At: 0},
		{Color: ui.RGBA(215, 222, 242, strength*.25), At: .4},
		{Color: ui.RGBA(250, 236, 255, strength*.8), At: 1},
	}}).InsetShadow(0, 1, 3, 0, ui.RGBA(255, 245, 255, strength*.35)).InsetShadow(0, -4, 6, 0, ui.RGBA(0, 0, 0, strength*.8)).InsetShadow(0, -1, 2, 0, ui.RGBA(240, 230, 255, strength*.65)).Shadow(0, 2, 3, 0, ui.RGBA(0, 0, 0, .18))
}

type paneVisualID int

func paneTransition(id int) ui.ElementTransition {
	return ui.ElementTransition{Duration: 220 * time.Millisecond, Position: true, Size: true, Match: paneVisualID(id), Enter: &ui.Motion{Opacity: 0, Scale: .985}, Exit: &ui.Motion{Opacity: 0, Scale: .985}}
}

func raisedOverlay(e ui.Element, clarity float32) ui.Element {
	return e.Material(frostedOverlay{Clarity: clarity}).Border(.75, ui.Transparent).
		BorderLinearGradient(ui.LinearGradient{Angle: 180, Stops: []ui.GradientStop{{Color: ui.RGBA(248, 245, 255, .30), At: 0}, {Color: ui.RGBA(230, 235, 250, .07), At: .5}, {Color: ui.RGBA(245, 236, 255, .18), At: 1}}}).
		InsetShadow(0, 1, 2, 0, ui.RGBA(255, 255, 255, .08)).InsetShadow(0, -1, 2, 0, ui.RGBA(230, 235, 255, .06)).Shadow(0, 12, 28, 0, ui.RGBA(0, 0, 0, .28))
}
