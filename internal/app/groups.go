package app

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type groupViewState struct {
	Folded bool
	Motion sidebarTransition
	Add    ui.Handle
	Header ui.Handle
}

func (a *application) groupState(id string) *groupViewState {
	if a.groupUI == nil {
		a.groupUI = map[string]*groupViewState{}
	}
	if a.groupUI[id] == nil {
		a.groupUI[id] = &groupViewState{}
	}
	return a.groupUI[id]
}

func (a *application) directoryForGroup(id string) string {
	for _, g := range a.config.Groups {
		if g.ID == id {
			return g.Directory
		}
	}
	home, _ := os.UserHomeDir()
	return home
}

func (a *application) editGroup(g *folderGroup) {
	a.groupEditor, a.groupEditID, a.groupName, a.groupDirectory, a.groupError = true, "", "", "", ""
	if g != nil {
		a.groupEditID, a.groupName, a.groupDirectory = g.ID, g.Name, g.Directory
	}
}

func (a *application) chooseGroupDirectory() {
	if a.groupPicking || a.win == nil {
		return
	}
	a.groupPicking = true
	win, path := a.win, a.groupDirectory
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent: win, Title: "绑定工作目录", Directory: true, DefaultPath: path,
		})
		win.Update(func() {
			a.groupPicking = false
			if err != nil {
				a.groupError = err.Error()
				return
			}
			if len(paths) > 0 {
				a.groupDirectory = paths[0]
				if strings.TrimSpace(a.groupName) == "" {
					a.groupName = filepath.Base(paths[0])
				}
			}
		})
	}()
}

// All configuration writes share one UI-thread busy gate. A snapshot is
// published only after its atomic file replacement succeeds.
func (a *application) persistConfig(s settings, done func(), fail func(error)) {
	if a.settingsBusy {
		return
	}
	if err := s.validate(); err != nil {
		fail(err)
		return
	}
	publish := func(err error) {
		a.settingsBusy = false
		if err != nil {
			fail(err)
			return
		}
		a.config = s
		if done != nil {
			done()
		}
	}
	if a.configFile == "" {
		publish(nil)
		return
	}
	a.settingsBusy = true
	win, path := a.win, a.configFile
	go func() {
		err := saveSettings(path, s)
		win.Update(func() { publish(err) })
	}()
}

func (a *application) saveGroup() {
	if a.settingsBusy || a.groupPicking {
		return
	}
	name, dir := strings.TrimSpace(a.groupName), strings.TrimSpace(a.groupDirectory)
	if name == "" || dir == "" {
		a.groupError = "请填写名称并绑定一个目录"
		return
	}
	home, _ := os.UserHomeDir()
	if dir == "~" {
		dir = home
	} else if strings.HasPrefix(dir, "~/") || strings.HasPrefix(dir, `~\`) {
		dir = filepath.Join(home, dir[2:])
	}
	path, err := filepath.Abs(dir)
	if err != nil {
		a.groupError = err.Error()
		return
	}
	id := a.groupEditID
	if id == "" {
		id = rand.Text()
	}
	next := a.config
	next.Groups = slices.Clone(next.Groups)
	g := folderGroup{ID: id, Name: name, Directory: path}
	i := slices.IndexFunc(next.Groups, func(v folderGroup) bool { return v.ID == id })
	if i >= 0 {
		next.Groups[i] = g
	} else {
		next.Groups = append(next.Groups, g)
	}
	// Validate the directory on a worker before accepting a persistent group.
	a.settingsBusy = true
	win := a.win
	check := func() error {
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("目录不可用：%w", err)
		}
		if !info.IsDir() {
			return fmt.Errorf("请选择一个文件夹")
		}
		return nil
	}
	finish := func(err error) {
		a.settingsBusy = false
		if err != nil {
			a.groupError = err.Error()
			return
		}
		a.persistConfig(next, func() { a.groupEditor = false; a.groupError = "" }, func(err error) { a.groupError = err.Error() })
	}
	if win == nil {
		finish(check())
		return
	}
	go func() { err := check(); win.Update(func() { finish(err) }) }()
}

func (a *application) removeGroup(id string) {
	next := a.config
	next.Groups = slices.DeleteFunc(slices.Clone(next.Groups), func(g folderGroup) bool { return g.ID == id })
	a.persistConfig(next, func() {
		for _, t := range a.work.all() {
			if t.GroupID == id {
				t.GroupID = ""
			}
		}
	}, func(err error) { a.status = err.Error() })
}

func (a *application) groupEditorView(c *ui.Context) {
	a.dialog(c, "group-editor", &a.groupEditor, 480, func() {
		title := "新建文件夹分组"
		if a.groupEditID != "" {
			title = "编辑文件夹分组"
		}
		ui.Text(c, title).FontSize(20).Bold()
		ui.TextInput(c.Key("group-name"), &a.groupName).Height(36).Width(432).Label("分组名称").Placeholder("名称").AutoFocus()
		ui.Row(c).Gap(8).Children(func() {
			ui.TextInput(c.Key("group-directory"), &a.groupDirectory).Height(36).Grow(1).MinWidth(0).Label("绑定目录").Placeholder("绑定目录")
			textButton(c, "选择…").Label("选择绑定目录").Disabled(a.groupPicking || a.settingsBusy).OnClick(a.chooseGroupDirectory)
		})
		ui.Text(c, "在此分组下新建的终端会自动进入该目录。").FontSize(12).TextColor(c.Theme().TextMuted)
		if a.groupError != "" {
			ui.Text(c, a.groupError).Wrap().TextColor(c.Theme().Danger)
		}
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			dialogButton(c, "取消", false).Disabled(a.settingsBusy).OnClick(func() { a.groupEditor = false })
			dialogButton(c, "保存", true).Disabled(a.settingsBusy || a.groupPicking).OnClick(a.saveGroup)
		})
	})
}
