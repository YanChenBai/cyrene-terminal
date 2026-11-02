package app

import (
	"os"
	"path/filepath"
	"strings"
)

// Identify ourselves honestly; do not advertise Ghostty/Kitty graphics until
// Windows ConPTY passes the protocol end to end. Yazi can use Chafa's ANSI
// fallback when the optional portable helper is installed.
func terminalEnvironment() []string {
	base := os.Environ()
	path := refreshedSearchPath()
	found := false
	for i, entry := range base {
		if key, _, ok := strings.Cut(entry, "="); ok && strings.EqualFold(key, "PATH") {
			base[i] = key + "=" + path
			found = true
			break
		}
	}
	if !found {
		base = append(base, "PATH="+path)
	}
	env := []string{"TERM_PROGRAM=cyrene-terminal", "TERM_PROGRAM_VERSION=0.0.0", prependSearchPath("", base)}
	if executable, err := os.Executable(); err == nil {
		dir := filepath.Join(filepath.Dir(executable), "tools", "chafa")
		if info, err := os.Stat(filepath.Join(dir, "chafa.exe")); err == nil && !info.IsDir() {
			return append(env[:2], prependSearchPath(dir, base))
		}
	}
	if root := os.Getenv("LOCALAPPDATA"); root != "" {
		dir := filepath.Join(root, "cyrene-terminal", "tools", "chafa")
		matches, _ := filepath.Glob(filepath.Join(dir, "*", "Chafa.exe"))
		if len(matches) > 0 {
			env = append(env[:2], prependSearchPath(filepath.Dir(matches[0]), base))
		}
	}
	return env
}

func prependSearchPath(dir string, environ []string) string {
	// MyGo replaces environment entries case-sensitively. Match the existing
	// Windows "Path" spelling to avoid a second PATH that PowerShell 7 ignores.
	for _, entry := range environ {
		key, value, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(key, "PATH") {
			if dir == "" {
				return entry
			}
			return key + "=" + dir + string(os.PathListSeparator) + value
		}
	}
	return "PATH=" + dir
}
