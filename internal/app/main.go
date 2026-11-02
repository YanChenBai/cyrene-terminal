package app

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

type application struct {
	work              workspace
	config            settings
	draft             settings
	configFile        string
	win               *mygo.Window
	tray              *mygo.Tray
	quitting          bool
	closed            atomic.Bool
	sessionsMu        sync.Mutex
	sessions          map[*terminal.Terminal]bool
	start             func(*pane)
	collapsed         bool
	sidebarWidth      float32
	sidebarMotion     sidebarTransition
	launcher          bool
	launchGroup       string
	addHandle         ui.Handle
	toggleHandle      ui.Handle
	preferencesHandle ui.Handle
	groupEditorHandle ui.Handle
	modalOrigin       ui.Rect
	dialogs           map[string]*overlayMotion
	launcherMotion    overlayMotion
	groupUI           map[string]*groupViewState
	groupEditor       bool
	groupEditID       string
	groupName         string
	groupDirectory    string
	groupError        string
	groupPicking      bool
	preferences       bool
	preferencesError  string
	settingsBusy      bool
	stopTarget        *tab
	stopConfirm       bool
	status            string
}

func (a *application) show() {
	if a.win == nil {
		return
	}
	if a.win.IsMinimized() {
		a.win.Restore()
	}
	a.win.Show()
	a.win.Focus()
}

func (a *application) openLauncher(groupID string) {
	a.launchGroup, a.launcher = groupID, !a.launcher || a.launchGroup != groupID
}

func (a *application) launch(p profile) {
	leaf := a.work.add(p, a.directoryForGroup(a.launchGroup))
	a.work.active().GroupID = a.launchGroup
	if a.launchGroup != "" {
		a.groupState(a.launchGroup).Folded = false
	}
	a.launcher = false
	if leaf != nil && a.start != nil {
		leaf.Handle.Focus()
		a.start(leaf)
	}
}

func shellCommand(command []string) ([]string, error) {
	return shellCommandInEnvironment(command, terminalEnvironment())
}

func resolveExecutable(name string, env []string) (string, error) {
	if runtime.GOOS != "windows" || strings.ContainsAny(name, `/\`) {
		return exec.LookPath(name)
	}
	path := ""
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(key, "PATH") {
			path = value
		}
	}
	extensions := []string{""}
	if filepath.Ext(name) == "" {
		extensions = []string{".exe", ".com", ".cmd", ".bat", ".ps1"}
	}
	for _, dir := range filepath.SplitList(path) {
		if dir == "" {
			continue
		}
		for _, extension := range extensions {
			candidate := filepath.Join(dir, name+extension)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate, nil
			}
		}
	}
	return "", fmt.Errorf("%s: %w", name, exec.ErrNotFound)
}

func shellCommandInEnvironment(command []string, env []string) ([]string, error) {
	if len(command) == 0 {
		if runtime.GOOS == "windows" {
			for _, name := range []string{"pwsh.exe", "powershell.exe", "cmd.exe"} {
				if path, err := resolveExecutable(name, env); err == nil {
					return []string{path}, nil
				}
			}
		}
		return nil, nil
	}
	path, err := resolveExecutable(command[0], env)
	if err != nil {
		return nil, fmt.Errorf("找不到 %s，请先安装并添加到 PATH：%w", command[0], err)
	}
	ext := strings.ToLower(filepath.Ext(path))
	if runtime.GOOS != "windows" || (ext != ".cmd" && ext != ".bat" && ext != ".ps1") {
		return append([]string{path}, command[1:]...), nil
	}
	// Agent launchers are often npm .cmd/.ps1 shims; ConPTY needs an executable.
	shell, err := resolveExecutable("pwsh.exe", env)
	if err != nil {
		shell, err = resolveExecutable("powershell.exe", env)
	}
	if err != nil {
		return nil, fmt.Errorf("找不到 PowerShell：%w", err)
	}
	args := append([]string{path}, command[1:]...)
	for i, arg := range args {
		args[i] = "'" + strings.ReplaceAll(arg, "'", "''") + "'"
	}
	return []string{shell, "-NoLogo", "-NoProfile", "-Command", "& " + strings.Join(args, " ")}, nil
}

func (a *application) startPane(p *pane) {
	p.Starting, p.Error = true, ""
	conf, win, dir, command := a.config, a.win, p.Dir, slices.Clone(p.Profile.Command)
	go func() {
		env := terminalEnvironment()
		cmd, err := shellCommandInEnvironment(command, env)
		if err == nil && dir != "" {
			var stat os.FileInfo
			stat, err = os.Stat(dir)
			if err == nil && !stat.IsDir() {
				err = fmt.Errorf("工作目录不是文件夹：%s", dir)
			}
		}
		var term *terminal.Terminal
		if err == nil {
			term, err = terminal.New(terminal.Options{
				Command: cmd, Dir: dir, Transparent: true,
				Env:   env,
				Font:  terminal.Font{Family: conf.FontFamily, Size: conf.FontSize},
				Theme: conf.terminalTheme(), Padding: terminal.Padding{X: 16, Y: 12},
				ClipboardRead: terminal.ClipboardAsk,
				OnTitle: func(title string) {
					win.Update(func() {
						if t := a.work.active(); t != nil && t.Focused == p.ID {
							win.SetTitle(title)
						}
					})
				},
				OnExit: func(int) { win.Invalidate() },
			})
		}
		if term != nil {
			a.sessionsMu.Lock()
			if a.closed.Load() {
				a.sessionsMu.Unlock()
				term.Close()
				return
			}
			a.sessions[term] = true
			a.sessionsMu.Unlock()
		}
		win.Update(func() {
			p.Starting = false
			p.Term = term
			if err != nil {
				p.Error = err.Error()
			}
		})
	}()
}

func (a *application) stopTab(t *tab) {
	if t == nil {
		return
	}
	// Do not terminate a pending start. The confirmation is disabled until ready.
	for _, p := range t.Root.leaves() {
		if p.Starting {
			return
		}
	}
	for _, p := range t.Root.leaves() {
		if term := p.Term; term != nil {
			a.sessionsMu.Lock()
			delete(a.sessions, term)
			a.sessionsMu.Unlock()
			go term.Close()
		}
	}
	a.work.removeTab(t.ID)
	if active := a.work.active(); active != nil && a.win != nil {
		a.win.SetTitle(active.label())
	}
	a.stopTarget, a.stopConfirm = nil, false
}

func (a *application) closeAll() {
	a.sessionsMu.Lock()
	a.closed.Store(true)
	terms := a.sessions
	a.sessions = nil
	a.sessionsMu.Unlock()
	for term := range terms {
		term.Close()
	}
}

func (a *application) requestStop(t *tab, origin ui.Rect) {
	a.stopTarget, a.stopConfirm, a.modalOrigin = t, true, origin
}

func (a *application) ready() {
	a.win = mygo.NewWindow(acceptanceWindowOptions(mygo.WindowOptions{
		Title: "Cyrene Terminal", Width: 1280, Height: 820, MinWidth: 720, MinHeight: 440,
		StateKey: "main", TitleBarStyle: mygo.TitleBarHidden, TitleBarHeight: 32,
		Transparent: true, Vibrancy: mygo.VibrancyGlass, BackgroundColor: "#00000000",
		Content: a.content(),
	}))
	showAcceptanceWindow(a.win)
	a.win.OnClose(func(e *mygo.CloseEvent) {
		if !a.quitting {
			e.PreventDefault()
			a.win.Hide()
		}
	})
	menu := mygo.NewMenu([]*mygo.MenuItem{
		{Label: "打开 Cyrene Terminal", Click: func(*mygo.MenuItem, *mygo.Window) { a.show() }},
		{Label: "新建终端…", Click: func(*mygo.MenuItem, *mygo.Window) {
			a.win.Update(func() { a.openLauncher("") })
			a.show()
		}},
		mygo.Separator(),
		{Label: "退出应用", Click: func(*mygo.MenuItem, *mygo.Window) { mygo.App.Quit() }},
	})
	installAcceptanceMenu(menu)
	var err error
	a.tray, err = mygo.NewTray(mygo.TrayOptions{Icon: trayIcon(), ToolTip: "Cyrene Terminal", Menu: menu})
	if err != nil {
		// Never hide the only route to running sessions if the system tray failed.
		a.status = "托盘不可用；关闭按钮将最小化窗口。"
		a.win.OnClose(func(e *mygo.CloseEvent) {
			if !a.quitting {
				e.PreventDefault()
				a.win.Show()
				a.win.Minimize()
			}
		})
	} else {
		a.tray.OnClick(a.show)
	}
	a.start = a.startPane
	// Open a shell without launching any agent automatically.
	for _, p := range a.config.Profiles {
		if p.ID == "shell" {
			a.launch(p)
			return
		}
	}
	a.launch(profile{ID: "shell", Name: "Terminal"})
}

// Run starts the native application and blocks until it exits.
func Run() {
	a := &application{configFile: configPath(), sessions: map[*terminal.Terminal]bool{}}
	var err error
	a.config, err = loadSettings(a.configFile)
	if err != nil {
		a.status = err.Error()
	}
	mygo.App.SetName("cyrene-terminal")
	mygo.App.SetMenu(mygo.NewMenu(nil))
	configureAcceptance()
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}
	mygo.Theme.SetSource(mygo.ThemeDark)
	mygo.App.OnSecondInstance(func([]string, string) { a.show() })
	mygo.App.OnActivate(func(bool) { a.show() })
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { a.quitting = true })
	mygo.App.OnWillQuit(func(*mygo.QuitEvent) {
		a.closeAll()
		if a.tray != nil {
			a.tray.Destroy()
		}
	})
	mygo.App.WhenReady(a.ready)
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
