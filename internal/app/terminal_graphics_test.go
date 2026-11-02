package app

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/color"
	"io"
	"testing"

	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

type graphicsConn struct {
	r *io.PipeReader
	w *io.PipeWriter
}

func (c *graphicsConn) Read(p []byte) (int, error)  { return c.r.Read(p) }
func (c *graphicsConn) Write(p []byte) (int, error) { return len(p), nil }
func (c *graphicsConn) Close() error                { c.w.Close(); return c.r.Close() }

func TestKittyGraphicsDrawDeleteAndTabSwitch(t *testing.T) {
	r, w := io.Pipe()
	term, err := terminal.New(terminal.Options{Conn: &graphicsConn{r, w}, Transparent: true})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	a := &application{config: defaultSettings()}
	p := a.work.add(profile{ID: "shell"}, t.TempDir())
	p.Term = term
	tabID := a.work.Active
	tt := ui.NewTester(a.view, 800, 500)
	ink := color.RGBA{R: 231, G: 43, B: 137, A: 255}
	count := func() int {
		n := 0
		img := tt.Image()
		for y := 32; y < img.Bounds().Dy(); y++ {
			for x := 244; x < img.Bounds().Dx(); x++ {
				if img.RGBAAt(x, y) == ink {
					n++
				}
			}
		}
		return n
	}
	pixels := bytes.Repeat([]byte{ink.R, ink.G, ink.B, 255}, 8*8)
	sequence := fmt.Sprintf("\x1b[3;3H\x1b_Ga=T,f=32,s=8,v=8,c=10,r=5,i=37,q=2;%s\x1b\\", base64.StdEncoding.EncodeToString(pixels))
	term.Feed([]byte(sequence))
	tt.Frame()
	if count() < 1000 {
		t.Fatal("Kitty image did not render in the native terminal")
	}
	saveInteractionFrame(t, tt, "kitty-graphics")
	a.work.add(profile{ID: "shell"}, t.TempDir())
	tt.Frame()
	if count() != 0 {
		t.Fatal("image leaked into another tab")
	}
	for _, tab := range a.work.Tabs {
		if tab.ID == tabID {
			a.selectTab(tab)
		}
	}
	tt.Frame()
	if count() < 1000 {
		t.Fatal("image disappeared when returning to the original terminal")
	}
	term.Feed([]byte("\x1b_Ga=d,d=I,i=37,q=2\x1b\\"))
	tt.Frame()
	if count() != 0 {
		t.Fatal("Kitty delete did not remove the image")
	}
}
