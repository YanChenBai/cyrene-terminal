package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/plugins/terminal"
)

func (a *application) watchAgentSession(p *pane, term *terminal.Terminal) {
	if (p.Profile.agentName() != "codex" && p.Profile.agentName() != "claude") || p.AgentToken == "" {
		return
	}
	token, record, win := p.AgentToken, p.AgentRecord, a.win
	last := p.AgentSession
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		publish := func() {
			if id := readAgentRecord(record); id != "" && id != last {
				last = id
				win.Update(func() {
					if p.Term == term && p.AgentToken == token {
						p.AgentSession = id
					}
				})
			}
		}
		for {
			publish()
			select {
			case <-term.Done():
				publish()
				return
			case <-ticker.C:
			}
		}
	}()
}

func codexEnvironment(env []string, record string) []string {
	if record == "" {
		return env
	}
	out := make([]string, 0, len(env)+1)
	for _, entry := range env {
		key, _, _ := strings.Cut(entry, "=")
		if !strings.EqualFold(key, "CYRENE_AGENT_SESSION_FILE") {
			out = append(out, entry)
		}
	}
	return append(out, "CYRENE_AGENT_SESSION_FILE="+record)
}

func readAgentRecord(path string) string {
	if path == "" {
		return ""
	}
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, 256))
	if err != nil {
		return ""
	}
	id := strings.TrimSpace(string(b))
	if !agentSessionID.MatchString(id) {
		return ""
	}
	return id
}

// A per-launch hook observes only the ID on stdin. It does not alter user
// config files, agent identity, prompts, permissions or conversation contents.
const codexSessionHook = `$ErrorActionPreference='Stop'
$eventData=[Console]::In.ReadToEnd() | ConvertFrom-Json
$session=[string]$eventData.session_id
if($session -match '^[a-zA-Z0-9][a-zA-Z0-9_-]{7,127}$' -and $env:CYRENE_AGENT_SESSION_FILE) {
    [IO.File]::WriteAllText($env:CYRENE_AGENT_SESSION_FILE,$session)
}
`

var sessionHookMu sync.Mutex

func sessionHook(record string) (string, error) {
	sessionHookMu.Lock()
	defer sessionHookMu.Unlock()
	dir := filepath.Dir(record)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	// A stable command definition avoids a new trust review for every pane.
	script := filepath.Join(dir, "session-start.ps1")
	if data, err := os.ReadFile(script); err != nil || string(data) != codexSessionHook {
		if err = os.WriteFile(script, []byte(codexSessionHook), 0600); err != nil {
			return "", err
		}
	}
	hook := `powershell.exe -NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -File "` + script + `"`
	return hook, nil
}

func prepareCodexCommand(command []string, record string) ([]string, error) {
	hook, err := sessionHook(record)
	if err != nil {
		return nil, err
	}
	config := fmt.Sprintf(`hooks.SessionStart=[{hooks=[{type="command",command=%s,timeout=5}]}]`, strconv.Quote(hook))
	return append(append([]string(nil), command...), "-c", config), nil
}

// Merge the per-launch hook with explicit Claude settings instead of replacing them.
func prepareClaudeCommand(command []string, record, dir string) ([]string, error) {
	hook, err := sessionHook(record)
	if err != nil {
		return nil, err
	}
	settings := map[string]any{}
	out := make([]string, 0, len(command)+2)
	for i := 0; i < len(command); i++ {
		arg := command[i]
		if arg != "--settings" && !strings.HasPrefix(arg, "--settings=") {
			out = append(out, arg)
			continue
		}
		source := strings.TrimPrefix(arg, "--settings=")
		if arg == "--settings" {
			if i+1 == len(command) {
				return nil, fmt.Errorf("Claude --settings 缺少参数")
			}
			i++
			source = command[i]
		}
		data := []byte(source)
		if !strings.HasPrefix(strings.TrimSpace(source), "{") {
			if !filepath.IsAbs(source) {
				source = filepath.Join(dir, source)
			}
			data, err = os.ReadFile(source)
			if err != nil {
				return nil, err
			}
		}
		if err = json.Unmarshal(data, &settings); err != nil {
			return nil, err
		}
		if settings == nil {
			return nil, fmt.Errorf("Claude --settings 必须为 JSON 对象")
		}
	}
	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}
	starts, _ := hooks["SessionStart"].([]any)
	hooks["SessionStart"] = append(starts, map[string]any{"hooks": []any{map[string]any{"type": "command", "command": hook, "timeout": 5}}})
	settings["hooks"] = hooks
	data, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	return append(out, "--settings", string(data)), nil
}
