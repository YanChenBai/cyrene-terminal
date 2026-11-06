package app

import (
	"github.com/egoist/mygo/ui"
	"strings"
)

func (a *application) loadFonts() {
	if a.fontsLoaded || a.fontsLoading || a.win == nil {
		return
	}
	a.fontsLoading = true
	win := a.win
	go func() {
		families := ui.FontFamilies()
		names := make([]string, 0, len(families))
		for _, f := range families {
			names = append(names, f.Name)
		}
		win.Update(func() { a.fontNames = names; a.fontsLoading = false; a.fontsLoaded = true })
	}()
}
func (a *application) fontPicker(c *ui.Context) {
	cb := ui.ComboboxBase(c.Key("font"), &a.draft.FontFamily)
	cb.Input.Height(36).Radius(7).Padding(7, 10).Background(ui.RGBA(230, 235, 245, .055)).Border(1, ui.RGBA(230, 235, 245, .18)).Label("终端字体")
	cb.Popup(func(panel ui.Element) {
		panel.Padding(5).Radius(10).Material(frostedOverlay{Clarity: .3}).Border(1, ui.RGBA(255, 255, 255, .12))
		ui.Scroll(c).MaxHeight(220).Children(func() {
			ui.Column(c).Children(func() {
				count := 0
				for _, name := range a.fontNames {
					if cb.Filtering() && !strings.Contains(strings.ToLower(name), strings.ToLower(a.draft.FontFamily)) {
						continue
					}
					item := cb.Item(name).Height(30).Padding(5, 9).Radius(5)
					if item.Highlighted() {
						item.Background(ui.RGBA(230, 235, 245, .12))
					}
					item.Children(func() { ui.Text(c, name).FontSize(12).SingleLine() })
					count++
				}
				if count == 0 {
					message := "可直接输入字体名称"
					if a.fontsLoading {
						message = "正在读取系统字体…"
					}
					ui.Text(c, message).Padding(8).FontSize(11).TextColor(c.Theme().TextMuted)
				}
			})
		})
	})
	if font, chosen := cb.Chosen(); chosen {
		a.draft.FontFamily = font
	}
}
