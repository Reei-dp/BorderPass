//go:build linux

package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"borderpass/config"
	"borderpass/protocol"

	evdev "github.com/holoplot/go-evdev"
)

type linuxCapturer struct {
	mouseDev  *evdev.InputDevice
	kbdDevs   []*evdev.InputDevice
	isGrabbed bool
	mu        sync.Mutex
	stopCh    chan struct{}
}

// NewCapturer creates a Linux Capturer using evdev and EVIOCGRAB
func NewCapturer(cfg *config.Config) (Capturer, error) {
	mousePath := cfg.MouseDevice
	kbdPath := cfg.KeyboardDevice

	if mousePath == "" || mousePath == "auto" {
		detectedMouse, err := autoDetectDevice("mouse")
		if err != nil {
			return nil, fmt.Errorf("auto-detect mouse: %w", err)
		}
		mousePath = detectedMouse
	}

	fmt.Printf("[Capturer] Opening mouse device: %s\n", mousePath)
	mDev, err := evdev.Open(mousePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open mouse (%s): %w (ensure user has read access)", mousePath, err)
	}

	var kDevs []*evdev.InputDevice
	if kbdPath != "" && kbdPath != "auto" {
		dev, err := evdev.Open(kbdPath)
		if err != nil {
			fmt.Printf("⚠️  Failed to open keyboard (%s): %v\n", kbdPath, err)
		} else {
			fmt.Printf("[Capturer] Opening keyboard device: %s (%s)\n", kbdPath, dev.Name)
			kDevs = append(kDevs, dev)
		}
	} else {
		detected := autoDetectKeyboards()
		for _, kp := range detected {
			dev, err := evdev.Open(kp)
			if err != nil {
				fmt.Printf("⚠️  Failed to open keyboard %s: %v\n", kp, err)
				continue
			}
			fmt.Printf("[Capturer] Opening keyboard device: %s (%s)\n", kp, dev.Name)
			kDevs = append(kDevs, dev)
		}
	}

	return &linuxCapturer{
		mouseDev: mDev,
		kbdDevs:  kDevs,
		stopCh:   make(chan struct{}),
	}, nil
}

func (c *linuxCapturer) Start(onEvent func(ev InputEvent)) error {
	// Mouse event reader goroutine
	go func() {
		var curDX, curDY int32
		flushMove := func() {
			if curDX != 0 || curDY != 0 {
				onEvent(InputEvent{Type: EventMouseMove, Value: curDX, Extra: curDY})
				curDX = 0
				curDY = 0
			}
		}

		for {
			select {
			case <-c.stopCh:
				return
			default:
				events, err := c.mouseDev.ReadSlice(64)
				if err != nil {
					continue
				}

				for i := range events {
					event := &events[i]
					switch event.Type {
					case evdev.EV_REL:
						if event.Code == evdev.REL_X {
							curDX += int32(event.Value)
						} else if event.Code == evdev.REL_Y {
							curDY += int32(event.Value)
						} else if event.Code == evdev.REL_WHEEL {
							flushMove()
							onEvent(InputEvent{Type: EventMouseWheel, Value: int32(event.Value), Extra: 0})
						} else if event.Code == evdev.REL_HWHEEL {
							flushMove()
							onEvent(InputEvent{Type: EventMouseWheel, Value: 0, Extra: int32(event.Value)})
						}

					case evdev.EV_SYN:
						if event.Code == evdev.SYN_REPORT {
							flushMove()
						}

					case evdev.EV_KEY:
						flushMove()
						var btn uint16
						switch event.Code {
						case evdev.BTN_LEFT:
							btn = protocol.BtnLeft
						case evdev.BTN_RIGHT:
							btn = protocol.BtnRight
						case evdev.BTN_MIDDLE:
							btn = protocol.BtnMiddle
						case evdev.BTN_SIDE:
							btn = protocol.BtnSide
						case evdev.BTN_EXTRA:
							btn = protocol.BtnExtra
						}

						if btn != 0 {
							onEvent(InputEvent{Type: EventMouseButton, Code: btn, Value: int32(event.Value)})
						}
					}
				}
				flushMove()
			}
		}
	}()

	// Keyboard event reader goroutines for all connected keyboards
	for _, kd := range c.kbdDevs {
		kDev := kd
		go func() {
			for {
				select {
				case <-c.stopCh:
					return
				default:
					events, err := kDev.ReadSlice(32)
					if err != nil {
						return
					}
					for i := range events {
						event := &events[i]
						if event.Type == evdev.EV_KEY {
							onEvent(InputEvent{
								Type:  EventKey,
								Code:  uint16(event.Code),
								Value: int32(event.Value),
							})
						}
					}
				}
			}
		}()
	}

	return nil
}

func (c *linuxCapturer) Grab() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.isGrabbed {
		return nil
	}

	if err := c.mouseDev.Grab(); err != nil {
		return fmt.Errorf("mouse grab failed: %w", err)
	}

	for _, kDev := range c.kbdDevs {
		if err := kDev.Grab(); err != nil {
			fmt.Printf("⚠️  Failed to grab keyboard %s (%s): %v\n", kDev.Path, kDev.Name, err)
		}
	}

	c.isGrabbed = true
	return nil
}

func (c *linuxCapturer) Ungrab() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.isGrabbed {
		return nil
	}

	_ = c.mouseDev.Ungrab()
	for _, kDev := range c.kbdDevs {
		_ = kDev.Ungrab()
	}

	c.isGrabbed = false
	return nil
}

func (c *linuxCapturer) IsGrabbed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isGrabbed
}

func (c *linuxCapturer) Close() error {
	close(c.stopCh)
	_ = c.Ungrab()
	if c.mouseDev != nil {
		_ = c.mouseDev.Close()
	}
	for _, kDev := range c.kbdDevs {
		_ = kDev.Close()
	}
	return nil
}

// autoDetectKeyboards enumerates and returns paths for all keyboards (built-in and USB)
func autoDetectKeyboards() []string {
	var kbdPaths []string
	seen := make(map[string]bool)

	// Enumerate /dev/input/event* devices
	devPaths, err := evdev.ListDevicePaths()
	if err == nil {
		for _, p := range devPaths {
			name := strings.ToLower(p.Name)
			// Avoid pure power buttons or HDMI switches; match real keyboards
			if (strings.Contains(name, "keyboard") || strings.Contains(name, "kbd")) && !strings.Contains(name, "power") && !strings.Contains(name, "video") {
				if !seen[p.Path] {
					seen[p.Path] = true
					kbdPaths = append(kbdPaths, p.Path)
				}
			}
		}
	}

	// Also check /dev/input/by-id
	byIDPath := "/dev/input/by-id"
	entries, err := os.ReadDir(byIDPath)
	if err == nil {
		for _, e := range entries {
			name := strings.ToLower(e.Name())
			if strings.Contains(name, "-event-kbd") || strings.Contains(name, "-kbd") {
				target, err := filepath.EvalSymlinks(filepath.Join(byIDPath, e.Name()))
				if err == nil && !seen[target] {
					seen[target] = true
					kbdPaths = append(kbdPaths, target)
				}
			}
		}
	}

	return kbdPaths
}

// autoDetectDevice scans /dev/input/by-id or /dev/input for mouse devices
func autoDetectDevice(devType string) (string, error) {
	byIDPath := "/dev/input/by-id"
	entries, err := os.ReadDir(byIDPath)
	if err == nil {
		for _, e := range entries {
			name := strings.ToLower(e.Name())
			if devType == "mouse" && strings.Contains(name, "-event-mouse") {
				target, err := filepath.EvalSymlinks(filepath.Join(byIDPath, e.Name()))
				if err == nil {
					return target, nil
				}
			}
		}
	}

	// Fallback: enumerate /dev/input/event* devices
	devPaths, err := evdev.ListDevicePaths()
	if err != nil {
		return "", fmt.Errorf("failed to list evdev devices: %w", err)
	}

	for _, p := range devPaths {
		name := strings.ToLower(p.Name)
		if devType == "mouse" && (strings.Contains(name, "mouse") || strings.Contains(name, "wireless receiver") || strings.Contains(name, "touchpad") || strings.Contains(name, "trackpad")) {
			return p.Path, nil
		}
	}

	return "", fmt.Errorf("device of type '%s' not found", devType)
}
