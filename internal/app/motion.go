package app

import (
	"math"
	"time"

	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
)

// Keep an overlay mounted through its exit. An interrupted animation starts
// from its current pose and velocity, with an restrained opening overshoot.
type overlayMotion struct {
	Visible, Target                         bool
	Started                                 time.Time
	FromScale, FromOpacity, FromPosition    float32
	Scale, Opacity, Position                float32
	Source                                  ui.Rect
	FromScaleVelocity, FromPositionVelocity float32
	ScaleVelocity, PositionVelocity         float32
}

func easeOut(t float32) float32   { return 1 - (1-t)*(1-t)*(1-t) }
func mix(a, b, t float32) float32 { return a + (b-a)*t }

type sidebarTransition struct {
	Initialized         bool
	From, Value, Target float32
	Started             time.Time
}

func (m *sidebarTransition) step(now time.Time, target float32) bool {
	return m.stepDuration(now, target, 0.22)
}

func (m *sidebarTransition) stepDuration(now time.Time, target, duration float32) bool {
	if !m.Initialized {
		m.Initialized, m.Value, m.Target = true, target, target
		return false
	}
	if m.Target != target {
		m.From, m.Target, m.Started = m.Value, target, now
	}
	p := min(1, max(0, float32(now.Sub(m.Started).Seconds())/duration))
	m.Value = mix(m.From, m.Target, easeOut(p))
	if p >= 1 {
		m.Value = m.Target
	}
	return m.Value != m.Target
}

// Analytic damped spring: position and scale move continuously together,
// retaining velocity when a gesture reverses instead of chaining two eases.
func springPose(from, velocity, target, seconds float32) (float32, float32) {
	const damping, frequency = 0.76, 19.0
	b := damping * frequency
	w := frequency * math.Sqrt(1-damping*damping)
	x := float64(from - target)
	c := (float64(velocity) + b*x) / w
	t := float64(seconds)
	exp, sn, cs := math.Exp(-b*t), math.Sin(w*t), math.Cos(w*t)
	d := x*cs + c*sn
	return target + float32(exp*d), float32(exp * (-b*d - x*w*sn + c*w*cs))
}

func (m *overlayMotion) step(now time.Time, open bool, source ui.Rect, initial float32) bool {
	if open != m.Target {
		if !m.Visible {
			m.Scale, m.Opacity, m.Position = initial, 0, 0
			m.ScaleVelocity, m.PositionVelocity = max(0, initial-0.12)*2.5, 0
		}
		m.FromScale, m.FromOpacity, m.FromPosition = m.Scale, m.Opacity, m.Position
		m.FromScaleVelocity, m.FromPositionVelocity = m.ScaleVelocity, m.PositionVelocity
		m.Started, m.Target, m.Visible = now, open, true
		if open {
			m.Source = source
		}
	}
	if !m.Visible {
		return false
	}
	t := float32(now.Sub(m.Started).Seconds())
	if m.Target {
		m.Scale, m.ScaleVelocity = springPose(m.FromScale, m.FromScaleVelocity, 1, t)
		m.Position, m.PositionVelocity = springPose(m.FromPosition, m.FromPositionVelocity, 1, t)
		m.Opacity = mix(m.FromOpacity, 1, easeOut(min(1, t/0.22)))
		if t >= 0.74 {
			m.Scale, m.Position, m.ScaleVelocity, m.PositionVelocity = 1, 1, 0, 0
			return false
		}
		return true
	}
	p := easeOut(min(1, t/0.22))
	m.Scale, m.ScaleVelocity = springPose(m.FromScale, m.FromScaleVelocity, initial, t)
	m.Position, m.PositionVelocity = springPose(m.FromPosition, m.FromPositionVelocity, 0, t)
	m.Opacity = mix(m.FromOpacity, 0, p)
	if t >= 0.22 {
		m.Visible = false
	}
	return m.Visible
}

func (a *application) dialog(c *ui.Context, key string, open *bool, width float32, build func()) {
	if a.dialogs == nil {
		a.dialogs = map[string]*overlayMotion{}
	}
	if a.dialogs[key] == nil {
		a.dialogs[key] = &overlayMotion{}
	}
	m := a.dialogs[key]
	if m.step(c.Now(), *open, a.modalOrigin, 0.12) {
		c.AnimationFrame()
	}
	if !m.Visible {
		return
	}
	w, h := c.Size()
	dx, dy := float32(0), float32(0)
	if m.Source.W > 0 {
		dx, dy = (m.Source.X+m.Source.W/2-w/2)*(1-m.Position), (m.Source.Y+m.Source.H/2-h/2)*(1-m.Position)
	}
	ui.Overlay(c, func() {
		back := ui.Box(c.Key(key + "-backdrop")).Absolute().Left(0).Top(0).Right(0).Bottom(0).Center().Modal().
			Background(ui.RGBA(0, 0, 0, 0.35*m.Opacity)).OnClick(func() { *open = false })
		if back.OverlayShortcut(0, ui.KeyEscape) {
			*open = false
		}
		back.Children(func() {
			// Keep window movement inside the modal input scope. Native caption buttons
			// remain owned by the OS on the right side of the custom title bar.
			// Do not add OnClick/Focusable here: MyGo only starts native dragging
			// when DragWindow is the target's sole interactive flag.
			bar := c.TitleBar()
			ui.Box(c.Key(key + "-window-drag")).Absolute().Left(0).Top(0).Right(bar.Right).Height(max(32, bar.Height)).DragWindow().Label("拖动窗口")

			ui.Column(c.Key(key+"-panel")).Role(ui.RoleDialog).Width(width).MaxWidth(w-32).Padding(20).Radius(14).Gap(12).
				Material(glass.Glass{Clarity: 0.25}).Border(1, ui.RGBA(255, 255, 255, 0.1)).
				Opacity(m.Opacity).TransformOrigin(0.5, 0.5).
				Transform(ui.Scaling(m.Scale, m.Scale).Then(ui.Translation(dx, dy))).
				Disabled(!m.Target).OnClick(func() {}).Children(build)
		})
	})
}
