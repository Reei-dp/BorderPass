//go:build linux

package clipboard

import (
	"bytes"
	"os/exec"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type linuxClipboardManager struct {
	lastText string
	mu       sync.Mutex
	stopCh   chan struct{}
}

func newPlatformManager() Manager {
	return &linuxClipboardManager{
		stopCh: make(chan struct{}),
	}
}

func (m *linuxClipboardManager) GetText() (string, error) {
	// First check clipboard MIME types with wl-paste --list-types
	typeCmd := exec.Command("wl-paste", "--list-types")
	typeOut, err := typeCmd.Output()
	if err == nil {
		typesStr := string(typeOut)
		// If clipboard contains an image (e.g. screenshot), NEVER touch or treat it as text!
		if strings.Contains(typesStr, "image/") {
			return "", nil
		}
	}

	// Explicitly request text/plain
	cmd := exec.Command("wl-paste", "--type", "text/plain", "--no-newline")
	out, err := cmd.Output()
	if err != nil {
		return "", nil
	}

	// Validate that it is genuine UTF-8 text and not binary garbage
	if len(out) == 0 || !utf8.Valid(out) || bytes.IndexByte(out, 0) != -1 {
		return "", nil
	}

	// Double-check: reject if it has binary image signatures
	if bytes.HasPrefix(out, []byte("\x89PNG")) || bytes.HasPrefix(out, []byte("\xff\xd8\xff")) || bytes.HasPrefix(out, []byte("GIF8")) {
		return "", nil
	}

	return string(out), nil
}

func (m *linuxClipboardManager) SetText(text string) error {
	if text == "" || !utf8.ValidString(text) {
		return nil
	}

	// If the user currently has an image/screenshot in the clipboard, DO NOT overwrite it!
	typeCmd := exec.Command("wl-paste", "--list-types")
	typeOut, err := typeCmd.Output()
	if err == nil && strings.Contains(string(typeOut), "image/") {
		return nil
	}

	m.mu.Lock()
	m.lastText = text
	m.mu.Unlock()

	cmd := exec.Command("wl-copy", "--type", "text/plain")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		// Fallback to xclip
		cmdX := exec.Command("xclip", "-selection", "clipboard")
		cmdX.Stdin = strings.NewReader(text)
		return cmdX.Run()
	}
	return nil
}


func (m *linuxClipboardManager) StartWatching(onChange func(text string)) {
	go func() {
		ticker := time.NewTicker(400 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-m.stopCh:
				return
			case <-ticker.C:
				text, err := m.GetText()
				if err != nil || text == "" {
					continue
				}

				m.mu.Lock()
				if text != m.lastText {
					m.lastText = text
					m.mu.Unlock()
					onChange(text)
				} else {
					m.mu.Unlock()
				}
			}
		}
	}()
}

func (m *linuxClipboardManager) Close() error {
	close(m.stopCh)
	return nil
}
