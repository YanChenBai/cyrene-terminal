package app

import (
	"bytes"
	"errors"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/egoist/mygo/transfer"
)

// Files remain available for repeated agent reads until this terminal closes.
// The lock also handles a terminal closing while an image is being saved.
type clipboardStore struct {
	mu     sync.Mutex
	dir    string
	closed bool
}

func (s *clipboardStore) savePNG(data []byte) (string, error) {
	if len(data) > 64<<20 {
		return "", errors.New("剪贴板图片超过 64 MiB")
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > 64<<20 {
		return "", errors.New("剪贴板 PNG 无效或尺寸过大")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return "", errors.New("终端已关闭")
	}
	if s.dir == "" {
		s.dir, err = os.MkdirTemp("", "cyrene-clipboard-")
		if err != nil {
			return "", err
		}
	}
	f, err := os.CreateTemp(s.dir, "image-*.png")
	if err != nil {
		return "", err
	}
	_, err = f.Write(data)
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func (s *clipboardStore) close() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	if s.dir != "" {
		_ = os.RemoveAll(s.dir)
		s.dir = ""
	}
}

// Match file drops: offer both a file list and shell-quoted path text. This
// works for agents launched directly and those started later inside a shell.
func filePasteData(paths []string, command []string) (transfer.Data, error) {
	d, err := transfer.FileData(paths...)
	if err != nil {
		return transfer.Data{}, err
	}
	quoted := make([]string, len(paths))
	cmd := ""
	if len(command) > 0 {
		cmd = strings.TrimSuffix(strings.ToLower(filepath.Base(command[0])), ".exe")
	}
	for i, path := range paths {
		if strings.ContainsAny(path, "\x00\r\n") {
			return transfer.Data{}, errors.New("文件路径含有控制字符")
		}
		if runtime.GOOS == "windows" && cmd == "cmd" {
			quoted[i] = `"` + strings.ReplaceAll(path, `"`, `""`) + `"`
		} else if runtime.GOOS == "windows" && cmd != "bash" && cmd != "sh" && cmd != "zsh" && cmd != "fish" {
			// Keep ordinary paths byte-for-byte identical to MyGo file drops;
			// agents commonly recognize these as image attachments.
			if path != "" && !strings.ContainsFunc(path, func(r rune) bool {
				return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune(`_-.:\/`, r))
			}) {
				quoted[i] = path
				continue
			}
			// PowerShell treats typographic quotes as quotation marks too.
			quoted[i] = "'" + strings.NewReplacer("'", "''", "‘", "‘‘", "’", "’’", "‚", "‚‚", "‛", "‛‛").Replace(path) + "'"
		} else {
			quoted[i] = "'" + strings.ReplaceAll(path, "'", "'\"'\"'") + "'"
		}
	}
	items := d.Items()
	if len(items) == 0 {
		return transfer.Data{}, errors.New("剪贴板没有本地文件")
	}
	files, err := items[0].Read(transfer.FileList)
	if err != nil {
		return transfer.Data{}, err
	}
	items[0] = transfer.NewItem(transfer.Bytes(transfer.FileList, files), transfer.Bytes(transfer.URIList, files), transfer.Bytes(transfer.Text, []byte(strings.Join(quoted, " "))))
	return transfer.New(items...), nil
}

func (a *application) pasteTerminal(p *pane, d transfer.Data) bool {
	_, image := d.Preferred(transfer.PNG)
	paths, filesErr := d.Files()
	if !image && (filesErr != nil || len(paths) == 0) {
		return false
	}
	term, win, store := p.Term, a.win, p.clipboard
	if term == nil || win == nil || store == nil {
		return false
	}
	command := append([]string(nil), p.Profile.Command...)
	previous, done := p.pasteTail, make(chan struct{})
	p.pasteTail = done
	go func() {
		if previous != nil {
			<-previous
		}
		defer close(done)
		var err error
		if len(paths) == 0 {
			var data []byte
			data, err = d.Read(transfer.PNG)
			if err == nil {
				var path string
				path, err = store.savePNG(data)
				paths = []string{path}
			}
		}
		var paste transfer.Data
		if err == nil {
			paste, err = filePasteData(paths, command)
		}
		if a.closed.Load() {
			return
		}
		win.Update(func() {
			if a.closed.Load() || p.Term != term {
				return
			}
			if err != nil {
				a.status = fmt.Sprintf("粘贴图片或文件失败：%v", err)
				return
			}
			term.PasteData(paste)
		})
	}()
	return true
}
