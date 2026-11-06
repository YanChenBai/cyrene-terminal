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
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
)

type application struct {
	rendered               bool
	inspector              inspectorState
	windowTitle            string
	tabScroll              ui.ScrollState
	revealedScreen         int
	revealedPane           int
	savingWorkspace        bool
	quitSaved              bool
	restoredWorkspace      bool
	fontNames              []string
	fontsLoading           bool
	fontsLoaded            bool
	aliasEditor            bool
	aliasTarget            *tab
	aliasScreen            *screen
	aliasDraft             string
	resumeTarget           *pane
	resumeDraft            string
	resumeDialog           bool
	work                   workspace
	config                 settings
	draft                  settings
	configFile             string
	win                    *mygo.Window
	tray                   *mygo.Tray
	quitting               bool
	closed                 atomic.Bool
	sessionsMu             sync.Mutex
	sessions               map[*terminal.Terminal]bool
	start                  func(*pane)
	collapsed              bool
	sidebarWidth           float32
	sidebarMotion          sidebarTransition
	launcher               bool
	launchGroup            string
	launchWorkspace        bool
	overlaySnapshotsActive bool
	overlaySnapshotEpoch   uint64
	launchTarget           int
	launchEdge             splitEdge
	browserTarget          int
	browserEdge            splitEdge
	addHandle              ui.Handle
	toggleHandle           ui.Handle
	preferencesHandle      ui.Handle
	groupEditorHandle      ui.Handle
	modalOrigin            ui.Rect
	dialogs                map[string]*overlayMotion
	launcherMotion         overlayMotion
	groupUI                map[string]*groupViewState
	groupEditor            bool
	groupEditID            string
	groupName              string
	groupDirectory         string
	groupError             string
	groupPicking           bool
	preferences            bool
	preferencesError       string
	settingsBusy           bool
	stopTarget             *tab
	stopScreen             *screen
	pendingDock            *dockRequest
	pendingTabDrop         *tabDropRequest
	pendingGroupDrop       *groupDropRequest
	newGroupHandle         ui.Handle
	palette                paletteState
	stopConfirm            bool
	status                 string
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
	a.launchTarget = 0
	a.launchWorkspace = false
}

func (a *application) launch(p profile) {
	dir, group := a.directoryForGroup(a.launchGroup), a.launchGroup
	if a.launchTarget != 0 {
		owner := a.work.sessionForPane(a.launchTarget)
		if owner == nil {
			a.status = "目标窗格已关闭"
			return
		}
		dir, group = owner.Root.find(a.launchTarget).Pane.directory(), owner.GroupID
	}
	leaf := a.work.add(p, dir)
	prepareAgentSession(leaf)
	a.work.active().GroupID = group
	if a.launchGroup != "" {
		a.groupState(a.launchGroup).Folded = false
	}
	if a.launchTarget != 0 {
		a.dockPane(leaf.ID, a.launchTarget, a.launchEdge)
	}
	if a.launchWorkspace || (a.launchGroup != "" && a.launchTarget == 0) {
		a.work.markWorkspace(a.work.activeScreen())
	}
	a.launchWorkspace = false
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
					if name != "cmd.exe" {
						return []string{path, "-NoLogo", "-NoExit", "-Command", powerShellDirectoryIntegration}, nil
					}
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
	if p.Browser != nil {
		a.startBrowser(p)
		return
	}
	if p.Starting {
		return
	}
	p.Starting, p.Error = true, ""
	p.clipboard = &clipboardStore{}
	conf, win, dir, command := a.config, a.win, p.Dir, slices.Clone(p.Profile.Command)
	agent := p.Profile.agentName()
	prepareAgentToken(p)
	if (agent == "codex" || agent == "claude") && p.AgentToken != "" && p.AgentRecord == "" {
		p.AgentRecord = filepath.Join(filepath.Dir(a.configFile), "agent-sessions", p.AgentToken+".id")
	}
	agentRecord := p.AgentRecord
	if len(p.Command) > 0 {
		command = slices.Clone(p.Command)
	}
	go func() {
		env := codexEnvironment(terminalEnvironment(), agentRecord)
		var err error
		if agent == "codex" && agentRecord != "" {
			command, err = prepareCodexCommand(command, agentRecord)
		}
		if err == nil && agent == "claude" && agentRecord != "" {
			command, err = prepareClaudeCommand(command, agentRecord, dir)
		}
		var cmd []string
		if err == nil {
			cmd, err = shellCommandInEnvironment(command, env)
		}
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
				OnPaste:       func(d transfer.Data) bool { return a.pasteTerminal(p, d) },
				OnTitle:       func(string) { win.Invalidate() },
				OnExit:        func(int) { win.Invalidate() },
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
			if term != nil {
				p.NeedsResume = false
				a.watchAgentSession(p, term)
				if t := a.work.active(); t != nil && t.Focused == p.ID {
					p.Handle.Focus()
				}
			}
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
		if p.Starting && p.Browser == nil {
			return
		}
	}
	for _, p := range t.Root.leaves() {
		if p.Browser != nil && p.Browser.View != nil {
			p.Browser.View.Destroy()
			p.Browser.View = nil
		}
		if term := p.Term; term != nil {
			a.sessionsMu.Lock()
			delete(a.sessions, term)
			a.sessionsMu.Unlock()
			p.Term = nil
			store := p.clipboard
			go func() { term.Close(); store.close() }()
		}
	}
	a.work.removeTab(t.ID)
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
	for _, t := range a.work.Tabs {
		for _, p := range t.Root.leaves() {
			p.clipboard.close()
		}
	}
}

func (a *application) requestStop(t *tab, origin ui.Rect) {
	if t != nil && t.Root.Pane != nil && t.Root.Pane.Browser != nil {
		a.stopTab(t)
		return
	}
	a.stopScreen = nil
	a.stopTarget, a.stopConfirm, a.modalOrigin = t, true, origin
}

func (a *application) ready() {
	a.win = mygo.NewWindow(acceptanceWindowOptions(mygo.WindowOptions{
		Title: "Cyrene Terminal", Width: 1280, Height: 820, MinWidth: 720, MinHeight: 440,
		StateKey: "main", TitleBarStyle: mygo.TitleBarHidden, TitleBarHeight: workspaceTitleHeight,
		Transparent: true, Vibrancy: mygo.VibrancyAcrylic, BackgroundColor: "#00000000",
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
	if a.restoredWorkspace {
		for _, t := range a.work.Tabs {
			for _, p := range t.Root.leaves() {
				if !p.NeedsResume || len(p.Command) > 0 {
					a.start(p)
				}
			}
		}
		return
	}
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
	if saved, loadErr := readWorkspace(workspacePath(a.configFile)); loadErr == nil {
		if restoreErr := a.restoreWorkspace(saved); restoreErr != nil {
			a.status = "工作区恢复失败，原记录已保留：" + restoreErr.Error()
		}
	} else if !os.IsNotExist(loadErr) {
		a.status = "工作区读取失败，原记录已保留：" + loadErr.Error()
	}
	mygo.Theme.SetSource(mygo.ThemeDark)
	mygo.App.OnSecondInstance(func([]string, string) { a.show() })
	mygo.App.OnActivate(func(bool) { a.show() })
	mygo.App.OnBeforeQuit(a.beforeQuit)
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
