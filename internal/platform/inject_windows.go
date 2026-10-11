//go:build windows

package platform

import (
	"syscall"
	"unsafe"

	"borderpass/config"
	"borderpass/protocol"
)

var (
	winmm                = syscall.NewLazyDLL("winmm.dll")
	procTimeBeginPeriod  = winmm.NewProc("timeBeginPeriod")
	procTimeEndPeriod    = winmm.NewProc("timeEndPeriod")
	procSendInput        = user32.NewProc("SendInput")
	procGetCursorPos     = user32.NewProc("GetCursorPos")
	procSetCursorPos     = user32.NewProc("SetCursorPos")
	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")
	procMapVirtualKey    = user32.NewProc("MapVirtualKeyW")
)

type POINT struct {
	X, Y int32
}

const (
	INPUT_MOUSE    = 0
	INPUT_KEYBOARD = 1
)

const (
	MOUSEEVENTF_MOVE       = 0x0001
	MOUSEEVENTF_LEFTDOWN   = 0x0002
	MOUSEEVENTF_LEFTUP     = 0x0004
	MOUSEEVENTF_RIGHTDOWN  = 0x0008
	MOUSEEVENTF_RIGHTUP    = 0x0010
	MOUSEEVENTF_MIDDLEDOWN = 0x0020
	MOUSEEVENTF_MIDDLEUP   = 0x0040
	MOUSEEVENTF_XDOWN      = 0x0080
	MOUSEEVENTF_XUP        = 0x0100
	MOUSEEVENTF_WHEEL      = 0x0800
	MOUSEEVENTF_HWHEEL     = 0x1000

	XBUTTON1 = 0x0001
	XBUTTON2 = 0x0002

	KEYEVENTF_EXTENDEDKEY = 0x0001
	KEYEVENTF_KEYUP       = 0x0002
	KEYEVENTF_SCANCODE    = 0x0008
)

// MOUSEINPUT describes mouse event data for SendInput
type MOUSEINPUT struct {
	Dx          int32
	Dy          int32
	MouseData   uint32
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

// KEYBDINPUT describes keyboard event data for SendInput
type KEYBDINPUT struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

// Windows 64-bit INPUT structure (40 bytes with alignment)
type INPUT struct {
	Type uint32
	_    uint32 // Padding for 8-byte alignment
	Data [32]byte
}

type windowsInjector struct{}

// NewInjector creates an Injector for Windows
func NewInjector(cfg *config.Config) (Injector, error) {
	// Request 1ms timer resolution for ultra-low latency input processing
	procTimeBeginPeriod.Call(1)
	return &windowsInjector{}, nil
}

func (w *windowsInjector) MoveMouse(dx, dy int32) error {
	var in INPUT
	in.Type = INPUT_MOUSE
	mi := (*MOUSEINPUT)(unsafe.Pointer(&in.Data[0]))
	mi.Dx = dx
	mi.Dy = dy
	mi.DwFlags = MOUSEEVENTF_MOVE
	procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
	return nil
}

func (w *windowsInjector) MouseButton(button uint16, pressed bool) error {
	var in INPUT
	in.Type = INPUT_MOUSE
	mi := (*MOUSEINPUT)(unsafe.Pointer(&in.Data[0]))

	switch button {
	case protocol.BtnLeft:
		if pressed {
			mi.DwFlags = MOUSEEVENTF_LEFTDOWN
		} else {
			mi.DwFlags = MOUSEEVENTF_LEFTUP
		}
	case protocol.BtnRight:
		if pressed {
			mi.DwFlags = MOUSEEVENTF_RIGHTDOWN
		} else {
			mi.DwFlags = MOUSEEVENTF_RIGHTUP
		}
	case protocol.BtnMiddle:
		if pressed {
			mi.DwFlags = MOUSEEVENTF_MIDDLEDOWN
		} else {
			mi.DwFlags = MOUSEEVENTF_MIDDLEUP
		}
	case protocol.BtnSide:
		mi.MouseData = XBUTTON1
		if pressed {
			mi.DwFlags = MOUSEEVENTF_XDOWN
		} else {
			mi.DwFlags = MOUSEEVENTF_XUP
		}
	case protocol.BtnExtra:
		mi.MouseData = XBUTTON2
		if pressed {
			mi.DwFlags = MOUSEEVENTF_XDOWN
		} else {
			mi.DwFlags = MOUSEEVENTF_XUP
		}
	}

	if mi.DwFlags != 0 {
		procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
	}
	return nil
}

func (w *windowsInjector) MouseWheel(vWheel, hWheel int32) error {
	if vWheel != 0 {
		var in INPUT
		in.Type = INPUT_MOUSE
		mi := (*MOUSEINPUT)(unsafe.Pointer(&in.Data[0]))
		mi.DwFlags = MOUSEEVENTF_WHEEL
		mi.MouseData = uint32(vWheel * 120) // 120 units per notch on Windows
		procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
	}
	if hWheel != 0 {
		var in INPUT
		in.Type = INPUT_MOUSE
		mi := (*MOUSEINPUT)(unsafe.Pointer(&in.Data[0]))
		mi.DwFlags = MOUSEEVENTF_HWHEEL
		mi.MouseData = uint32(hWheel * 120)
		procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
	}
	return nil
}

func (w *windowsInjector) Key(code uint16, state int32) error {
	scan, ext := linuxToWindowsScancode(code)
	if scan == 0 {
		return nil
	}

	var in INPUT
	in.Type = INPUT_KEYBOARD
	ki := (*KEYBDINPUT)(unsafe.Pointer(&in.Data[0]))
	ki.WScan = scan
	ki.DwFlags = KEYEVENTF_SCANCODE

	vk, _, _ := procMapVirtualKey.Call(uintptr(scan), 1) // MAPVK_VSC_TO_VK
	if vk > 0 {
		ki.WVk = uint16(vk)
	}

	if ext {
		ki.DwFlags |= KEYEVENTF_EXTENDEDKEY
	}
	if state == 0 {
		ki.DwFlags |= KEYEVENTF_KEYUP
	}

	procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
	return nil
}

func (w *windowsInjector) GetCursorPos() (int32, int32, error) {
	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	return pt.X, pt.Y, nil
}

func (w *windowsInjector) GetScreenSize() (int32, int32, error) {
	r1, _, _ := procGetSystemMetrics.Call(0) // SM_CXSCREEN
	r2, _, _ := procGetSystemMetrics.Call(1) // SM_CYSCREEN
	if r1 > 0 && r2 > 0 {
		return int32(r1), int32(r2), nil
	}
	return 1920, 1080, nil
}

func (w *windowsInjector) Close() error {
	procTimeEndPeriod.Call(1)
	return nil
}

// linuxToWindowsScancode translates a Linux evdev keycode into a Windows hardware scancode and extended flag
func linuxToWindowsScancode(linuxCode uint16) (uint16, bool) {
	// Standard IBM Set 1 keys (1 - 88)
	if linuxCode >= 1 && linuxCode <= 88 {
		return linuxCode, false
	}

	// Extended keys
	switch linuxCode {
	case 96: // KEY_KPENTER
		return 0x1C, true
	case 97: // KEY_RIGHTCTRL
		return 0x1D, true
	case 98: // KEY_KPSLASH
		return 0x35, true
	case 99: // KEY_SYSRQ (PrintScreen)
		return 0x37, true
	case 100: // KEY_RIGHTALT
		return 0x38, true
	case 102: // KEY_HOME
		return 0x47, true
	case 103: // KEY_UP
		return 0x48, true
	case 104: // KEY_PAGEUP
		return 0x49, true
	case 105: // KEY_LEFT
		return 0x4B, true
	case 106: // KEY_RIGHT
		return 0x4D, true
	case 107: // KEY_END
		return 0x4F, true
	case 108: // KEY_DOWN
		return 0x50, true
	case 109: // KEY_PAGEDOWN
		return 0x51, true
	case 110: // KEY_INSERT
		return 0x52, true
	case 111: // KEY_DELETE
		return 0x53, true
	case 125: // KEY_LEFTMETA (Win)
		return 0x5B, true
	case 126: // KEY_RIGHTMETA (Win)
		return 0x5C, true
	}

	return 0, false
}
