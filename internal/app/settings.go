package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

type profile struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Command []string `json:"command,omitempty"`
}

type folderGroup struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Directory string `json:"directory"`
}

type settings struct {
	FontFamily string        `json:"fontFamily"`
	FontSize   float32       `json:"fontSize"`
	Background string        `json:"background"`
	Foreground string        `json:"foreground"`
	Accent     string        `json:"accent"`
	Opacity    float32       `json:"backgroundOpacity"`
	Clarity    float32       `json:"glassClarity"`
	Directory  string        `json:"directory"`
	Profiles   []profile     `json:"profiles"`
	Groups     []folderGroup `json:"groups,omitempty"`
}

func defaultSettings() settings {
	dir, _ := os.UserHomeDir()
	return settings{
		FontFamily: "Cascadia Mono", FontSize: 14,
		Background: "#08090b", Foreground: "#e4e4e7", Accent: "#c5c9d3",
		Opacity: 0.65, Clarity: 0.55, Directory: dir,
		Profiles: []profile{
			{ID: "codex", Name: "Codex", Command: []string{"codex"}},
			{ID: "claude", Name: "Claude", Command: []string{"claude"}},
			{ID: "pi", Name: "Pi", Command: []string{"pi"}},
			{ID: "omp", Name: "OMP", Command: []string{"omp"}},
			{ID: "opencode", Name: "OpenCode", Command: []string{"opencode"}},
			{ID: "shell", Name: "Terminal"},
		},
	}
}

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func (s settings) validate() error {
	for _, c := range []string{s.Background, s.Foreground, s.Accent} {
		if !hexColor.MatchString(c) {
			return fmt.Errorf("颜色格式应为 #rrggbb，例如 #08090b")
		}
	}
	if s.FontSize < 8 || s.FontSize > 36 || s.Opacity < 0.15 || s.Opacity > 1 || s.Clarity < 0 || s.Clarity > 1 {
		return fmt.Errorf("字体应为 8–36，背景浓度为 0.15–1，玻璃清晰度为 0–1")
	}
	if len(s.Profiles) == 0 {
		return fmt.Errorf("至少保留一个启动配置")
	}
	ids := map[string]bool{}
	for _, g := range s.Groups {
		if g.ID == "" || g.Name == "" || g.Directory == "" || ids[g.ID] {
			return fmt.Errorf("文件夹分组需要唯一 id、名称和绑定目录")
		}
		ids[g.ID] = true
	}
	ids = map[string]bool{}
	for _, p := range s.Profiles {
		if p.ID == "" || p.Name == "" || ids[p.ID] {
			return fmt.Errorf("启动配置需要唯一 id 和非空 name")
		}
		if len(p.Command) > 0 && p.Command[0] == "" {
			return fmt.Errorf("启动命令不能为空")
		}
		ids[p.ID] = true
	}
	return nil
}

func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "cyrene-terminal", "settings.json")
}

func loadSettings(path string) (settings, error) {
	s := defaultSettings()
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if err = json.Unmarshal(b, &s); err == nil {
		err = s.validate()
	}
	if err != nil {
		return defaultSettings(), fmt.Errorf("读取配置失败（原文件已保留）：%w", err)
	}
	return s, nil
}

func saveSettings(path string, s settings) error {
	if err := s.validate(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), "settings-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(append(b, '\n')); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func (s settings) theme() *ui.Theme {
	t := ui.DarkTheme()
	t.Background = ui.Hex(s.Background)
	t.Surface = ui.RGBA(255, 255, 255, 0.045)
	t.SurfaceHover = ui.RGBA(255, 255, 255, 0.085)
	t.SurfacePressed = ui.RGBA(255, 255, 255, 0.13)
	t.Border = ui.RGBA(255, 255, 255, 0.09)
	t.Text, t.TextMuted = ui.Hex(s.Foreground), ui.Hex("#969aa5")
	t.Accent = ui.Hex(s.Accent)
	t.AccentText = ui.Hex("#08090b")
	t.AccentHover, t.AccentPressed = ui.Hex("#e4e4e7"), ui.Hex("#a1a1aa")
	t.Focus = ui.RGBA(197, 201, 211, 0.45)
	t.Radius, t.FontSize, t.Spacing = 6, 13, 3
	return t
}

func (s settings) terminalTheme() *terminal.Theme {
	t := terminal.DarkTheme()
	t.Background, t.Foreground = ui.Hex(s.Background), ui.Hex(s.Foreground)
	t.Cursor = ui.Hex(s.Accent)
	t.Selection = ui.RGBA(170, 180, 200, 0.25)
	return t
}
