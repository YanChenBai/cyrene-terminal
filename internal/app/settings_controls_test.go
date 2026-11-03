package app

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"time"
)

func TestFontChoiceAndFreeformAndColorWell(t *testing.T) {
	a := application{config: defaultSettings(), fontNames: []string{"Cascadia Mono", "Consolas"}, fontsLoaded: true}
	tt := ui.NewTester(a.view, 1000, 720)
	if err := tt.Click("外观设置"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	if err := tt.Click("终端字体"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("Consolas"); err != nil {
		t.Fatal(err, tt.Texts())
	}
	tt.Frame()
	if a.draft.FontFamily != "Consolas" {
		t.Fatal(a.draft.FontFamily)
	}
	if err := tt.Click("终端字体"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Ctrl, ui.KeyA)
	tt.Type("Custom Terminal Font")
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.draft.FontFamily != "Custom Terminal Font" {
		t.Fatal("manual font discarded", a.draft.FontFamily)
	}
	if err := tt.Click("背景颜色选择器"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	saveInteractionFrame(t, tt, "native-color-picker")
	tt.Key(0, ui.KeyEscape)
	tt.Advance(time.Second)
	a.draft.Background = "#"
	tt.Frame() // Incomplete hex input must not panic.
}
