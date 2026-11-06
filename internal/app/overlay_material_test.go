package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestOverlayActuallyBlursTexture(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		c.Root().Background(ui.RGBA(8, 9, 11, .4))
		ui.Box(c).Fill().Stripes(ui.RGBA(255, 255, 255, .9), 2, 2, 0)
		ui.Box(c).Absolute().Left(30).Top(30).Width(160).Height(100).Radius(12).Material(frostedOverlay{Clarity: .25})
	}, 220, 160)
	lo, hi := 255, 0
	for x := 80; x < 120; x++ {
		v := int(tt.Image().RGBAAt(x, 80).R)
		lo = min(lo, v)
		hi = max(hi, v)
	}
	if hi-lo > 2 {
		t.Fatalf("sharp texture leaked through the backdrop: contrast %d", hi-lo)
	}
	if a := tt.Image().RGBAAt(100, 80).A; a > 245 {
		t.Fatalf("frosted layer became opaque: %d", a)
	}
	saveInteractionFrame(t, tt, "overlay-texture")
}
