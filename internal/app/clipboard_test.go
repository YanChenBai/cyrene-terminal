package app

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/egoist/mygo/transfer"
)

func clipboardTestPNG(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestClipboardImageRemainsReadableUntilTerminalCloses(t *testing.T) {
	s := &clipboardStore{}
	defer s.close()
	data := clipboardTestPNG(t)
	path, err := s.savePNG(data)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(path) {
		t.Fatal("agent needs an absolute path")
	}
	read, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(read, data) {
		t.Fatal("saved image differs from clipboard")
	}
	paste, err := filePasteData([]string{path}, []string{"codex"})
	if err != nil {
		t.Fatal(err)
	}
	paths, err := paste.Files()
	if err != nil || len(paths) != 1 || paths[0] != path {
		t.Fatalf("missing file representation: %v, %v", paths, err)
	}
	text, err := paste.Read(transfer.Text)
	if err != nil || !strings.Contains(string(text), path) {
		t.Fatal("missing path text for agents without clipboard protocol")
	}
	s.close()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("temporary image survived terminal close")
	}
	if _, err := s.savePNG(data); err == nil {
		t.Fatal("late paste recreated closed storage")
	}
}

func TestClipboardCloseRacingImageSave(t *testing.T) {
	data := clipboardTestPNG(t)
	for range 10 {
		s := &clipboardStore{}
		var path string
		var wg sync.WaitGroup
		wg.Add(1)
		go func() { defer wg.Done(); path, _ = s.savePNG(data) }()
		s.close()
		wg.Wait()
		if path != "" {
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("close/save race leaked a file")
			}
		}
	}
}

func TestClipboardRejectsInvalidPNGAndKeepsOrdinaryText(t *testing.T) {
	s := &clipboardStore{}
	defer s.close()
	if _, err := s.savePNG([]byte("not PNG")); err == nil || s.dir != "" {
		t.Fatal("invalid PNG created storage")
	}
	a := &application{}
	if a.pasteTerminal(&pane{}, transfer.TextData("ordinary text")) {
		t.Fatal("ordinary text must retain default terminal behavior")
	}
}
