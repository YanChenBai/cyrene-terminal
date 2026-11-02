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
			ui.Column(c).Gap(10).Width(472).Children(func() {
				ui.Text(c, "终端字体").TextColor(c.Theme().TextMuted)
				ui.TextInput(c.Key("font"), &a.draft.FontFamily).Height(36).Label("终端字体")
				a.slider(c, "字体大小", &a.draft.FontSize, 8, 36)
				a.slider(c, "黑色背景浓度", &a.draft.Opacity, 0.15, 1)
				a.slider(c, "玻璃清晰度", &a.draft.Clarity, 0, 1)
				ui.Text(c, "背景 / 终端文字 / 光标与强调色").FontSize(11).TextColor(c.Theme().TextMuted)
				ui.Row(c).Gap(8).Children(func() {
					ui.TextInput(c.Key("bg"), &a.draft.Background).Height(36).Width(150).Label("背景颜色")
					ui.TextInput(c.Key("fg"), &a.draft.Foreground).Height(36).Width(150).Label("文字颜色")
					ui.TextInput(c.Key("accent"), &a.draft.Accent).Height(36).Width(150).Label("强调色")
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

func (a *application) slider(c *ui.Context, label string, value *float32, lo, hi float64) {
	ui.Text(c, fmt.Sprintf("%s  %.2f", label, *value)).FontSize(11).TextColor(c.Theme().TextMuted)
	v := float64(*value)
	ui.Slider(c.Key(label), &v, lo, hi).Label(label).OnChange(func() { *value = float32(v) })
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
