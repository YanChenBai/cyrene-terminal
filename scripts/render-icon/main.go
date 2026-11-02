// Render the editable SVG through the same native SVG renderer as the app.
package main

import (
	"image/png"
	"os"
	"path/filepath"

	"github.com/egoist/mygo/ui"
)

func main() {
	data, err := os.ReadFile("assets/cyrene.svg")
	if err != nil {
		panic(err)
	}
	svg := ui.MustParseSVG(data)
	for _, size := range []int{1024, 32} {
		tt := ui.NewTester(func(c *ui.Context) {
			c.Root().Background(ui.RGBA(0, 0, 0, 0))
			ui.Image(c, svg).Size(float32(size), float32(size))
		}, size, size)
		img := tt.Image()
		if img.RGBAAt(0, 0).A != 0 {
			panic("icon must have transparent background")
		}
		path := "resources/icon.png"
		if size == 32 {
			path = "assets/cyrene-tray.png"
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			panic(err)
		}
		f, err := os.Create(path)
		if err != nil {
			panic(err)
		}
		if err := png.Encode(f, img); err != nil {
			panic(err)
		}
		if err := f.Close(); err != nil {
			panic(err)
		}
	}
}
