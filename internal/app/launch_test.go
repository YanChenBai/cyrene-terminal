package app

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestResolveCommandFromSuppliedEnvironment(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows command extensions")
	}
	dir := t.TempDir()
	shim := filepath.Join(dir, "agent.cmd")
	if err := os.WriteFile(shim, []byte("@echo ok\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := resolveExecutable("agent", []string{"Path=" + dir})
	if err != nil || got != shim {
		t.Fatalf("command lookup must use child Path: %s, %v", got, err)
	}
}

func TestInstalledCommandLaunchWithStaleParentPath(t *testing.T) {
	if os.Getenv("CYRENE_LIVE_CLI") != "1" || runtime.GOOS != "windows" {
		t.Skip("opt-in installed CLI verification")
	}
	t.Setenv("PATH", filepath.Join(os.Getenv("SystemRoot"), "System32"))
	env := terminalEnvironment()
	report := map[string]string{}
	for _, name := range []string{"codex", "claude", "pi", "omp", "opencode", "oh-my-posh", "winget"} {
		t.Run(name, func(t *testing.T) {
			args, err := shellCommandInEnvironment([]string{name, "--version"}, env)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, args[0], args[1:]...)
			cmd.Env = append(os.Environ(), env...)
			out, err := cmd.CombinedOutput()
			text := strings.TrimSpace(string(out))
			if err != nil {
				t.Fatalf("launch failed: %v: %s", err, text)
			}
			report[name] = text
			t.Log(text)
		})
	}
	t.Run("PowerShell profile", func(t *testing.T) {
		args, err := shellCommandInEnvironment(nil, env)
		if err != nil || len(args) == 0 {
			t.Fatalf("default shell: %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, args[0], "-NoLogo", "-Command", `if ($Error.Count -gt 0) { exit 9 }; 'CYRENE_PROFILE_OK'`)
		cmd.Env = append(os.Environ(), env...)
		out, err := cmd.CombinedOutput()
		if err != nil || !strings.Contains(string(out), "CYRENE_PROFILE_OK") {
			t.Fatalf("profile startup: %v: %s", err, out)
		}
		report["PowerShell profile"] = "CYRENE_PROFILE_OK"
	})
	_ = os.MkdirAll("test-results", 0755)
	data, _ := json.MarshalIndent(report, "", "  ")
	_ = os.WriteFile("test-results/cli-launch.json", data, 0600)
}
