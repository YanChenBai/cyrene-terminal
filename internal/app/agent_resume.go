package app

import (
	"crypto/rand"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
)

func (p profile) agentName() string {
	if len(p.Command) == 0 {
		return ""
	}
	name := strings.ToLower(filepath.Base(p.Command[0]))
	return strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(name, ".exe"), ".cmd"), ".ps1"), ".bat")
}

func (p profile) supportsResume() bool {
	if len(p.ResumeCommand) > 0 {
		return true
	}
	switch p.agentName() {
	case "codex", "claude", "pi", "omp", "opencode":
		return true
	}
	return false
}

var agentSessionID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{7,127}$`)

// A restore always targets this pane's exact agent session, never --last or a
// global picker. A custom command can position its ID using {session}.
func (p profile) resumeCommand(session string) []string {
	if !agentSessionID.MatchString(session) {
		return nil
	}
	if len(p.ResumeCommand) > 0 {
		command, replaced := slices.Clone(p.ResumeCommand), false
		for i, arg := range command {
			if strings.Contains(arg, "{session}") {
				command[i], replaced = strings.ReplaceAll(arg, "{session}", session), true
			}
		}
		if !replaced {
			command = append(command, session)
		}
		return command
	}
	if len(p.Command) == 0 {
		return nil
	}
	command := []string{p.Command[0]}
	for i := 1; i < len(p.Command); i++ {
		arg := p.Command[i]
		if arg == "--resume" || arg == "-r" || arg == "--session" || arg == "--session-id" || (p.agentName() == "codex" && arg == "resume") {
			if i+1 < len(p.Command) && agentSessionID.MatchString(p.Command[i+1]) {
				i++
			}
			continue
		}
		if arg == "--continue" || arg == "-c" && p.agentName() == "claude" || arg == "--last" || arg == "--fork-session" {
			continue
		}
		command = append(command, arg)
	}
	switch p.agentName() {
	case "codex":
		// A subcommand precedes any existing launch options.
		return append(append([]string{command[0], "resume"}, command[1:]...), session)
	case "claude", "omp":
		return append(command, "--resume", session)
	case "pi":
		return append(command, "--session", session)
	case "opencode":
		return append(command, "--session", session)
	}
	return nil
}

// Claude and Pi accept an explicit ID on a new session. Keep it per pane so
// simultaneous sessions in the same directory cannot restore one another.
func prepareAgentSession(p *pane) {
	name := p.Profile.agentName()
	prepareAgentToken(p)
	for i, arg := range p.Profile.Command {
		if i+1 < len(p.Profile.Command) && (arg == "--session-id" || arg == "--session" || arg == "--resume" || arg == "-r" || (name == "codex" && arg == "resume")) && agentSessionID.MatchString(p.Profile.Command[i+1]) {
			p.AgentSession = p.Profile.Command[i+1]
			return
		}
	}
	if name == "codex" {
		return
	}
	if name != "claude" && name != "pi" {
		return
	}
	for _, arg := range p.Profile.Command[1:] {
		if strings.HasPrefix(arg, "--session") || arg == "--resume" || arg == "-r" || arg == "--continue" || arg == "-c" {
			return
		}
	}
	var id [16]byte
	_, _ = rand.Read(id[:])
	id[6], id[8] = id[6]&0x0f|0x40, id[8]&0x3f|0x80
	p.AgentSession = fmt.Sprintf("%x-%x-%x-%x-%x", id[:4], id[4:6], id[6:8], id[8:10], id[10:])
	p.Command = append(slices.Clone(p.Profile.Command), "--session-id", p.AgentSession)
}

func prepareAgentToken(p *pane) {
	if p.AgentToken != "" || (p.Profile.agentName() != "codex" && p.Profile.agentName() != "claude") {
		return
	}
	var token [16]byte
	_, _ = rand.Read(token[:])
	p.AgentToken = fmt.Sprintf("cyrene_%x", token[:])
}

// Old panes without an association use the agent's own history picker, never --last.
func (p profile) resumePickerCommand() []string {
	if len(p.ResumeCommand) > 0 || (p.agentName() != "codex" && p.agentName() != "claude") {
		return nil
	}
	command := p.resumeCommand("cyrene_picker_placeholder")
	return command[:len(command)-1]
}

func (p *pane) exited() bool {
	if p == nil || p.Starting {
		return false
	}
	if p.Term == nil {
		return p.NeedsResume
	}
	select {
	case <-p.Term.Done():
		return true
	default:
		return false
	}
}

func (a *application) resumePane(p *pane) {
	if !p.exited() {
		return
	}
	command := p.Profile.resumeCommand(p.AgentSession)
	if len(command) == 0 {
		return
	}
	a.restartPaneWithCommand(p, command)
}

var resumeHint = map[string]*regexp.Regexp{
	"codex":  regexp.MustCompile(`(?i)To\s+continue\s+this\s+session,?\s+run\s+codex\s+resume\s+([a-z0-9][a-z0-9_-]{7,127})`),
	"claude": regexp.MustCompile(`(?i)Resume\s+this\s+session\s+with:?\s+claude\s+--resume\s+([a-z0-9][a-z0-9_-]{7,127})`),
}

func (a *application) requestAgentResume(p *pane, origin ui.Rect) {
	if !p.exited() {
		return
	}
	if hint := resumeHint[p.Profile.agentName()]; hint != nil && p.Term != nil {
		if matches := hint.FindAllStringSubmatch(p.Term.Text(), -1); len(matches) > 0 {
			p.AgentSession = matches[len(matches)-1][1]
		}
	}
	if p.AgentSession != "" {
		a.resumePane(p)
		return
	}
	if command := p.Profile.resumePickerCommand(); len(command) > 0 {
		a.restartPaneWithCommand(p, command)
		return
	}
	// Custom agents may not expose a history picker. Let the user
	// bind an ID if an agent did not expose its own session in the exit output.
	a.resumeTarget, a.resumeDraft, a.resumeDialog, a.modalOrigin = p, "", true, origin
}

func (a *application) agentResumeView(c *ui.Context) {
	a.dialog(c, "agent-resume", &a.resumeDialog, 440, func() {
		ui.Text(c, "恢复此会话").FontSize(20).Bold()
		ui.Text(c, "未能自动取得此会话的 Agent Session ID。填写原会话 ID 后恢复。").Wrap().TextColor(c.Theme().TextMuted)
		ui.TextInput(c.Key("agent-session-id"), &a.resumeDraft).Label("Agent Session ID").Height(36).AutoFocus()
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			dialogButton(c, "取消", false).OnClick(func() { a.resumeDialog = false })
			dialogButton(c, "确认恢复", true).Disabled(!agentSessionID.MatchString(strings.TrimSpace(a.resumeDraft))).OnClick(func() {
				if p := a.resumeTarget; p != nil && p.exited() {
					p.AgentSession = strings.TrimSpace(a.resumeDraft)
					a.resumePane(p)
				}
				a.resumeDialog = false
			})
		})
	})
}
