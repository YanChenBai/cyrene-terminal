package app

import (
	"os"
	"testing"
)

func TestSearchPathPreservesWindowsKeySpelling(t *testing.T) {
	got := prependSearchPath(`C:\cyrene\tools\chafa`, []string{"TEMP=C:\\Temp", `Path=C:\Windows;C:\Program Files\PowerShell\7`})
	want := `Path=C:\cyrene\tools\chafa` + string(os.PathListSeparator) + `C:\Windows;C:\Program Files\PowerShell\7`
	if got != want {
		t.Fatalf("must replace existing Path rather than append duplicate PATH: %q", got)
	}
}
