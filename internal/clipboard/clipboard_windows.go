//go:build windows

package clipboard

import (
	"sync"
	"syscall"
	"time"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"
)

var (
	user32                 = syscall.NewLazyDLL("user32.dll")
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procOpenClipboard      = user32.NewProc("OpenClipboard")
	procCloseClipboard     = user32.NewProc("CloseClipboard")
	procEmptyClipboard     = user32.NewProc("EmptyClipboard")
	procGetClipboardData   = user32.NewProc("GetClipboardData")
	procSetClipboardData   = user32.NewProc("SetClipboardData")
	procGetClipboardSeqNum = user32.NewProc("GetClipboardSequenceNumber")
	procGlobalAlloc        = kernel32.NewProc("GlobalAlloc")
	procGlobalLock         = kernel32.NewProc("GlobalLock")
	procGlobalUnlock       = kernel32.NewProc("GlobalUnlock")
)

const (
	CF_UNICODETEXT = 13
	GMEM_MOVEABLE  = 0x0002
)

type windowsClipboardManager struct {
	lastText   string
	lastSeqNum uint32
	mu         sync.Mutex
	stopCh     chan struct{}
}

func newPlatformManager() Manager {
	return &windowsClipboardManager{
		stopCh: make(chan struct{}),
	}
}

func (m *windowsClipboardManager) GetText() (string, error) {
	r, _, _ := procOpenClipboard.Call(0)
	if r == 0 {
		return "", nil
	}
	defer procCloseClipboard.Call()

	hMem, _, _ := procGetClipboardData.Call(CF_UNICODETEXT)
	if hMem == 0 {
		return "", nil
	}

	ptr, _, _ := procGlobalLock.Call(hMem)
	if ptr == 0 {
		return "", nil
	}
	defer procGlobalUnlock.Call(hMem)

	// Read null-terminated UTF-16 string
	var utf16Slice []uint16
	for p := (*uint16)(unsafe.Pointer(ptr)); *p != 0; p = (*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + 2)) {
		utf16Slice = append(utf16Slice, *p)
	}

	return string(utf16.Decode(utf16Slice)), nil
}

func (m *windowsClipboardManager) SetText(text string) error {
	if text == "" || !utf8.ValidString(text) {
		return nil
	}

	m.mu.Lock()
	m.lastText = text
	m.mu.Unlock()

	utf16Data := utf16.Encode([]rune(text + "\x00"))
	size := len(utf16Data) * 2

	hMem, _, _ := procGlobalAlloc.Call(GMEM_MOVEABLE, uintptr(size))
	if hMem == 0 {
		return nil
	}

	ptr, _, _ := procGlobalLock.Call(hMem)
	if ptr == 0 {
		return nil
	}

	dest := (*[1 << 28]uint16)(unsafe.Pointer(ptr))[:len(utf16Data):len(utf16Data)]
	copy(dest, utf16Data)
	procGlobalUnlock.Call(hMem)

	r, _, _ := procOpenClipboard.Call(0)
	if r == 0 {
		return nil
	}
	defer procCloseClipboard.Call()

	procEmptyClipboard.Call()
	procSetClipboardData.Call(CF_UNICODETEXT, hMem)
	return nil
}

func (m *windowsClipboardManager) StartWatching(onChange func(text string)) {
	go func() {
		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-m.stopCh:
				return
			case <-ticker.C:
				seqNum, _, _ := procGetClipboardSeqNum.Call()
				curSeq := uint32(seqNum)

				m.mu.Lock()
				if curSeq == m.lastSeqNum {
					m.mu.Unlock()
					continue
				}
				m.lastSeqNum = curSeq
				m.mu.Unlock()

				text, err := m.GetText()
				if err != nil || text == "" || !utf8.ValidString(text) {
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

func (m *windowsClipboardManager) Close() error {
	close(m.stopCh)
	return nil
}
