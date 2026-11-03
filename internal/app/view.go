package app

import (
	"fmt"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
	"path/filepath"
)

func (a *application) settingsView(c *ui.Context) {
	a.dialog(c, "preferences", &a.preferences, 520, func() {
		_, viewportHeight := c.Size()
		ui.Text(c, "外观设置").FontSize(22).Bold()
		ui.Scroll(c).MaxHeight(min(430, max(100, viewportHeight-220))).Children(func() {
			ui.Column(c).Gap(8).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, "字体").TextColor(c.Theme().TextMuted)
				a.fontPicker(c)
				a.numberSetting(c, "字体大小", &a.draft.FontSize, 8, 36)
				a.numberSetting(c, "黑色背景浓度", &a.draft.Opacity, 0.15, 1)
				a.numberSetting(c, "玻璃清晰度", &a.draft.Clarity, 0, 1)
				ui.Text(c, "颜色").FontSize(11).TextColor(c.Theme().TextMuted)
				ui.Row(c).Gap(8).Children(func() {
					a.colorSetting(c.Key("bg"), "背景", &a.draft.Background)
					a.colorSetting(c.Key("fg"), "终端文字", &a.draft.Foreground)
					a.colorSetting(c.Key("accent"), "光标与强调色", &a.draft.Accent)
				})
				ui.Text(c, "启动命令可在 settings.json 的 profiles 中定制。").FontSize(11).TextColor(c.Theme().TextMuted)
			})
		})
		if a.preferencesError != "" {
			ui.Text(c, a.preferencesError).Wrap().TextColor(c.Theme().Danger)
		}
		ui.Row(c).Gap(8).Children(func() {
			textButton(c, "打开配置文件夹").OnClick(func() { go mygo.Shell.OpenPath(filepath.Dir(a.configFile)) })
			ui.Spacer(c)
			dialogButton(c, "取消", false).Disabled(a.settingsBusy).OnClick(func() { a.preferences = false })
			dialogButton(c, "保存", true).Disabled(a.settingsBusy).OnClick(a.applySettings)
		})
	})
}

func (a *application) numberSetting(c *ui.Context, label string, value *float32, lo, hi float64) {
	factor, unit := float64(1), "px"
	if hi <= 1 {
		factor, unit = 100, "%"
	}
	v := float64(*value) * factor
	ui.Row(c.Key(label + "-row")).Height(44).AlignItems(ui.Center).Gap(12).Children(func() {
		ui.Text(c, label).FontSize(12).Grow(1).TextColor(c.Theme().TextMuted)
		ui.Row(c.Key(label+"-stepper")).Height(34).Radius(7).Clip().Border(1, ui.RGBA(230, 235, 245, .14)).Background(ui.RGBA(230, 235, 245, .04)).AlignItems(ui.Stretch).Children(func() {
			segmentButton(c.Key(label+"-decrease"), "−").Label("减小" + label).Width(34).Height(34).Radius(0).Disabled(v <= lo*factor).OnClick(func() { *value = float32(max(lo, (float64(*value)*factor-1)/factor)) })
			ui.Box(c).Width(1).Background(ui.RGBA(230, 235, 245, .10))
			ui.Row(c).Width(76).Height(34).Center().Gap(4).Children(func() {
				ui.Text(c, fmt.Sprintf("%.0f", v)).FontSize(12)
				ui.Text(c, unit).FontSize(10).TextColor(c.Theme().TextMuted)
			})
			ui.Box(c).Width(1).Background(ui.RGBA(230, 235, 245, .10))
			segmentButton(c.Key(label+"-increase"), "+").Label("增大" + label).Width(34).Height(34).Radius(0).Disabled(v >= hi*factor).OnClick(func() { *value = float32(min(hi, (float64(*value)*factor+1)/factor)) })
		})
	})
}

func (a *application) applySettings() {
	s := a.draft
	s.Groups = a.config.Groups
	a.persistConfig(s, func() {
		a.preferences, a.preferencesError = false, ""
		for _, t := range a.work.all() {
			for _, p := range t.Root.leaves() {
				if p.Term != nil {
					p.Term.SetFont(terminal.Font{Family: s.FontFamily, Size: s.FontSize})
					p.Term.SetTheme(s.terminalTheme(), nil)
				}
			}
		}
	}, func(err error) { a.preferencesError = err.Error() })
}

func (a *application) confirmations(c *ui.Context) {
	a.dialog(c, "stop", &a.stopConfirm, 440, func() {
		ui.Text(c, "关闭这个终端？").FontSize(20).Bold()
		ui.Text(c, "此终端中的进程会结束；关闭应用窗口则只收起到托盘。").Wrap().TextColor(c.Theme().TextMuted)
		starting := false
		if a.stopTarget != nil {
			for _, p := range a.stopTarget.Root.leaves() {
				starting = starting || p.Starting
			}
		}
		if starting {
			ui.Text(c, "请等待终端启动完成。").TextColor(c.Theme().TextMuted)
		}
		ui.Row(c).Gap(8).Children(func() {
			textButton(c, "取消").OnClick(func() { a.stopConfirm = false })
			textButton(c, "终止进程").Disabled(starting).TextColor(c.Theme().Danger).OnClick(func() { a.stopTab(a.stopTarget) })
		})
	})
}

// Segment backgrounds stay flush with their shared frame while pressed.
func segmentButton(c *ui.Context, text string) ui.Element {
	b := ui.ButtonBase(c).Label(text).FocusRing(false).Center()
	alpha := float32(0)
	if b.Hovered() || b.FocusVisible() {
		alpha = .075
	}
	if b.Pressed() {
		alpha = .16
	}
	b.Background(ui.RGBA(230, 235, 245, alpha)).Children(func() {
		ink := ui.Text(c, text).FontSize(12)
		if b.Pressed() {
			ink.Opacity(.7)
		}
	})
	return b
}

func (a *application) colorSetting(c *ui.Context, label string, value *string) {
	color := ui.Hex("#808080")
	if hexColor.MatchString(*value) {
		color = ui.Hex(*value)
	}
	ui.Column(c).Grow(1).MinWidth(0).Gap(6).Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(c.Theme().TextMuted)
		ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
			ui.ColorWell(c.Key("well"), &color).Width(44).Height(34).Padding(4).Radius(7).Clip().Label(label + "颜色选择器").OnChange(func() { *value = fmt.Sprintf("#%02x%02x%02x", color.R, color.G, color.B) })
			ui.TextInput(c.Key("hex"), value).Grow(1).MinWidth(0).Height(34).Radius(7).Padding(5, 6).FontSize(11).Label(label + "颜色")
		})
	})
}
