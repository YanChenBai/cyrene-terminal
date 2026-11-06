package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
)

type savedPane struct {
	Browser                                          bool
	ID                                               int
	Profile                                          profile
	Directory, AgentSession, AgentToken, AgentRecord string
	BrowserURL, BrowserTitle                         string
}

type savedLayout struct {
	ID, PaneID    int
	Vertical      bool
	Ratio         float32
	First, Second *savedLayout
}

type savedSession struct {
	ID, Focused          int
	Name, Alias, GroupID string
	Root                 *savedLayout
}

type savedScreen struct {
	WorkspaceIndex int
	Order          []int
	ID, Focused    int
	Alias          string
	Root           *savedLayout
}

type workspaceState struct {
	NextWorkspace                int
	Version, Active, NextID      int
	Panes                        []savedPane
	Sessions                     []savedSession
	Screens                      []savedScreen
	SidebarWidth, InspectorWidth float32
	Collapsed, InspectorOpen     bool
}

func encodeLayout(n *layout) *savedLayout {
	if n == nil {
		return nil
	}
	r := &savedLayout{ID: n.ID, Vertical: n.Vertical, Ratio: n.Ratio}
	if n.Pane != nil {
		r.PaneID = n.Pane.ID
	} else {
		r.First, r.Second = encodeLayout(n.First), encodeLayout(n.Second)
	}
	return r
}

// Snapshot on the UI thread; file work only sees these independent values.
func (a *application) snapshotWorkspace() workspaceState {
	s := workspaceState{NextWorkspace: a.work.nextWorkspace, Version: 1, Active: a.work.Active, NextID: a.work.nextID, SidebarWidth: a.sidebarWidth, Collapsed: a.collapsed, InspectorOpen: a.inspector.Open, InspectorWidth: a.inspector.Width}
	for _, t := range a.work.Tabs {
		s.Sessions = append(s.Sessions, savedSession{t.ID, t.Focused, t.Name, t.Alias, t.GroupID, encodeLayout(t.Root)})
		for _, p := range t.Root.leaves() {
			id := p.AgentSession
			if p.Term != nil {
				if hint := resumeHint[p.Profile.agentName()]; hint != nil {
					if matches := hint.FindAllStringSubmatch(p.Term.Text(), -1); len(matches) > 0 {
						id = matches[len(matches)-1][1]
					}
				}
			}
			saved := savedPane{ID: p.ID, Profile: p.Profile, Directory: p.directory(), AgentSession: id, AgentToken: p.AgentToken, AgentRecord: p.AgentRecord}
			if p.Browser != nil {
				saved.Browser = true
				saved.BrowserURL = p.Browser.URL
				saved.BrowserTitle = p.Browser.Title
			}
			s.Panes = append(s.Panes, saved)
		}
	}
	for _, screen := range a.work.Screens {
		s.Screens = append(s.Screens, savedScreen{WorkspaceIndex: screen.WorkspaceIndex, ID: screen.ID, Focused: screen.Focused, Alias: screen.Alias, Root: encodeLayout(screen.Root), Order: screen.paneOrder()})
	}
	return s
}

func workspacePath(config string) string {
	return filepath.Join(filepath.Dir(config), "workspace.json")
}

func writeWorkspace(path string, s workspaceState) error {
	for i := range s.Panes {
		p := &s.Panes[i]
		if id := readAgentRecord(p.AgentRecord); id != "" {
			p.AgentSession = id
		}
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), "workspace-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func readWorkspace(path string) (workspaceState, error) {
	var s workspaceState
	f, err := os.Open(path)
	if err != nil {
		return s, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 8<<20+1))
	if err != nil {
		return s, err
	}
	if len(data) > 8<<20 {
		return s, fmt.Errorf("工作区记录过大")
	}
	if err = json.Unmarshal(data, &s); err != nil {
		return s, err
	}
	return s, nil
}

func (a *application) restoreWorkspace(s workspaceState) error {
	if s.Version != 1 || len(s.Panes) > 1024 || len(s.Screens) > 1024 {
		return fmt.Errorf("工作区记录版本或大小无效")
	}
	w := workspace{Active: s.Active, nextID: s.NextID}
	panes := map[int]*pane{}
	for _, saved := range s.Panes {
		if saved.ID <= 0 || panes[saved.ID] != nil {
			return fmt.Errorf("工作区终端标识重复")
		}
		p := &pane{ID: saved.ID, Profile: saved.Profile, Dir: saved.Directory, AgentSession: saved.AgentSession, AgentToken: saved.AgentToken, AgentRecord: saved.AgentRecord}
		if saved.BrowserURL != "" || saved.Browser {
			address := ""
			if saved.BrowserURL != "" {
				var err error
				address, err = normalizeBrowserURL(saved.BrowserURL)
				if err != nil {
					return err
				}
			}
			p.Browser = &browserState{URL: address, Draft: address, Title: saved.BrowserTitle}
		}
		if id := readAgentRecord(p.AgentRecord); id != "" {
			p.AgentSession = id
		}
		if len(p.Profile.Command) > 0 && p.Profile.Command[0] == "" {
			return fmt.Errorf("工作区启动命令无效")
		}
		if p.Profile.supportsResume() {
			p.NeedsResume = true
			p.Command = p.Profile.resumeCommand(p.AgentSession)
			if len(p.Command) == 0 {
				p.Command = p.Profile.resumePickerCommand()
			}
			if len(p.Command) == 0 {
				p.Error = "未取得原 Agent 会话 ID。请绑定原会话后继续。"
			}
		}
		panes[p.ID] = p
		w.nextID = max(w.nextID, p.ID)
	}
	decode := func(root *savedLayout, seen map[int]bool) (*layout, error) {
		var visit func(*savedLayout, int) (*layout, error)
		visit = func(n *savedLayout, depth int) (*layout, error) {
			if n == nil || depth > 64 || n.ID <= 0 {
				return nil, fmt.Errorf("工作区分屏结构无效")
			}
			w.nextID = max(w.nextID, n.ID)
			r := &layout{ID: n.ID, Vertical: n.Vertical, Ratio: n.Ratio}
			if n.PaneID != 0 {
				if panes[n.PaneID] == nil || seen[n.PaneID] || n.First != nil || n.Second != nil {
					return nil, fmt.Errorf("工作区窗格引用无效")
				}
				seen[n.PaneID] = true
				r.Pane = panes[n.PaneID]
				return r, nil
			}
			if n.Ratio <= 0 || n.Ratio >= 100 {
				return nil, fmt.Errorf("工作区分屏比例无效")
			}
			var err error
			if r.First, err = visit(n.First, depth+1); err != nil {
				return nil, err
			}
			if r.Second, err = visit(n.Second, depth+1); err != nil {
				return nil, err
			}
			return r, nil
		}
		return visit(root, 0)
	}
	sessionSeen, screenSeen := map[int]bool{}, map[int]bool{}
	ids := map[int]bool{}
	for _, t := range s.Sessions {
		if t.ID <= 0 || ids[t.ID] {
			return fmt.Errorf("工作区会话标识无效")
		}
		ids[t.ID] = true
		root, err := decode(t.Root, sessionSeen)
		if err != nil {
			return err
		}
		if root.find(t.Focused) == nil {
			return fmt.Errorf("工作区聚焦终端无效")
		}
		w.Tabs = append(w.Tabs, &tab{ID: t.ID, Name: t.Name, Alias: t.Alias, GroupID: t.GroupID, Focused: t.Focused, Root: root})
		w.nextID = max(w.nextID, t.ID)
	}
	ids = map[int]bool{}
	for _, sc := range s.Screens {
		if sc.ID <= 0 || ids[sc.ID] {
			return fmt.Errorf("工作区标签标识无效")
		}
		ids[sc.ID] = true
		root, err := decode(sc.Root, screenSeen)
		if err != nil {
			return err
		}
		if root.find(sc.Focused) == nil {
			return fmt.Errorf("工作区聚焦窗格无效")
		}
		w.nextWorkspace = max(w.nextWorkspace, sc.WorkspaceIndex)
		w.Screens = append(w.Screens, &screen{WorkspaceIndex: sc.WorkspaceIndex, Order: append([]int(nil), sc.Order...), ID: sc.ID, Alias: sc.Alias, Focused: sc.Focused, Root: root})
		w.nextID = max(w.nextID, sc.ID)
	}
	if len(sessionSeen) != len(panes) || len(screenSeen) != len(panes) {
		return fmt.Errorf("工作区包含未归属窗格")
	}
	if len(w.Tabs) > 0 && w.active() == nil {
		w.Active = w.Tabs[0].ID
	}
	w.nextWorkspace = max(w.nextWorkspace, s.NextWorkspace)
	for _, screen := range w.Screens {
		if len(screen.Root.leaves()) > 1 {
			w.markWorkspace(screen)
		}
	}
	a.work = w
	a.collapsed = s.Collapsed
	a.sidebarWidth = s.SidebarWidth
	a.inspector.Open = s.InspectorOpen
	a.inspector.Width = s.InspectorWidth
	a.restoredWorkspace = len(panes) > 0
	return nil
}

func (a *application) beforeQuit(e *mygo.QuitEvent) {
	if a.quitSaved || a.win == nil {
		a.quitting = true
		return
	}
	e.PreventDefault()
	if a.savingWorkspace {
		return
	}
	a.savingWorkspace = true
	snapshot, path, win := a.snapshotWorkspace(), workspacePath(a.configFile), a.win
	go func() {
		err := writeWorkspace(path, snapshot)
		win.Update(func() {
			a.savingWorkspace = false
			if err != nil {
				a.status = "保存工作区失败，应用仍保持运行：" + err.Error()
				return
			}
			a.quitSaved = true
			mygo.App.Quit()
		})
	}()
}
