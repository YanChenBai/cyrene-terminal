//go:build windows

package app

import (
	"encoding/binary"
	"os"
	"strings"
	"syscall"
)

func refreshedSearchPath() string {
	parts := []string{os.Getenv("PATH"), registryPath(syscall.HKEY_CURRENT_USER, `Environment`), registryPath(syscall.HKEY_LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`)}
	seen := map[string]bool{}
	var result []string
	for _, part := range parts {
		for _, dir := range strings.Split(part, ";") {
			dir = strings.Trim(strings.TrimSpace(dir), `"`)
			for start, expansions := strings.Index(dir, "%"), 0; start >= 0 && expansions < 16; expansions++ {
				end := strings.Index(dir[start+1:], "%")
				if end < 0 {
					break
				}
				end += start + 1
				value := os.Getenv(dir[start+1 : end])
				if value == "" {
					break
				}
				dir = dir[:start] + value + dir[end+1:]
				start = strings.Index(dir, "%")
			}
			key := strings.ToLower(strings.TrimRight(dir, `\/`))
			if dir != "" && !seen[key] {
				seen[key] = true
				result = append(result, dir)
			}
		}
	}
	return strings.Join(result, ";")
}

func registryPath(root syscall.Handle, subkey string) string {
	keyName, _ := syscall.UTF16PtrFromString(subkey)
	var key syscall.Handle
	if syscall.RegOpenKeyEx(root, keyName, 0, syscall.KEY_READ, &key) != nil {
		return ""
	}
	defer syscall.RegCloseKey(key)
	name, _ := syscall.UTF16PtrFromString("Path")
	var length, kind uint32
	if syscall.RegQueryValueEx(key, name, nil, &kind, nil, &length) != nil || length < 2 || length > 1024*1024 {
		return ""
	}
	data := make([]byte, length)
	if syscall.RegQueryValueEx(key, name, nil, &kind, &data[0], &length) != nil {
		return ""
	}
	if kind != syscall.REG_SZ && kind != syscall.REG_EXPAND_SZ {
		return ""
	}
	units := make([]uint16, length/2)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(data[i*2:])
	}
	return syscall.UTF16ToString(units)
}
