//go:build linux

package platform

import (
	"fmt"

	"borderpass/config"
	"borderpass/protocol"

	evdev "github.com/holoplot/go-evdev"
)

type linuxInjector struct {
	dev *evdev.InputDevice
}

// NewInjector creates a virtual /dev/uinput device on Linux
func NewInjector(cfg *config.Config) (Injector, error) {
	capabilities := map[evdev.EvType][]evdev.EvCode{
		evdev.EV_KEY: {
			evdev.BTN_LEFT, evdev.BTN_RIGHT, evdev.BTN_MIDDLE, evdev.BTN_SIDE, evdev.BTN_EXTRA,
		},
		evdev.EV_REL: {
			evdev.REL_X, evdev.REL_Y, evdev.REL_WHEEL, evdev.REL_HWHEEL,
		},
	}

	// Add standard keyboard keys (1..128)
	var keys []evdev.EvCode
	for k := 1; k <= 128; k++ {
		keys = append(keys, evdev.EvCode(k))
	}
	capabilities[evdev.EV_KEY] = append(capabilities[evdev.EV_KEY], keys...)

	dev, err := evdev.CreateDevice("BorderPass Virtual Input", evdev.InputID{
		BusType: evdev.BUS_USB,
		Vendor:  0x1234,
		Product: 0x5678,
		Version: 1,
	}, capabilities)

	if err != nil {
		fmt.Printf("⚠️  Failed to create /dev/uinput device: %v. Linux input injection disabled.\n", err)
		return &linuxInjector{dev: nil}, nil
	}

	return &linuxInjector{dev: dev}, nil
}

func (l *linuxInjector) MoveMouse(dx, dy int32) error {
	if l.dev == nil {
		return nil
	}
	if dx != 0 {
		_ = l.dev.WriteOne(&evdev.InputEvent{Type: evdev.EV_REL, Code: evdev.REL_X, Value: int32(dx)})
	}
	if dy != 0 {
		_ = l.dev.WriteOne(&evdev.InputEvent{Type: evdev.EV_REL, Code: evdev.REL_Y, Value: int32(dy)})
	}
	_ = l.dev.WriteOne(&evdev.InputEvent{Type: evdev.EV_SYN, Code: evdev.SYN_REPORT, Value: 0})
	return nil
}

func (l *linuxInjector) MouseButton(button uint16, pressed bool) error {
	if l.dev == nil {
		return nil
	}
	var evCode evdev.EvCode
	switch button {
	case protocol.BtnLeft:
		evCode = evdev.BTN_LEFT
	case protocol.BtnRight:
		evCode = evdev.BTN_RIGHT
	case protocol.BtnMiddle:
		evCode = evdev.BTN_MIDDLE
	case protocol.BtnSide:
		evCode = evdev.BTN_SIDE
	case protocol.BtnExtra:
		evCode = evdev.BTN_EXTRA
	}

	val := int32(0)
	if pressed {
		val = 1
	}
	_ = l.dev.WriteOne(&evdev.InputEvent{Type: evdev.EV_KEY, Code: evCode, Value: val})
	_ = l.dev.WriteOne(&evdev.InputEvent{Type: evdev.EV_SYN, Code: evdev.SYN_REPORT, Value: 0})
	return nil
}

func (l *linuxInjector) MouseWheel(vWheel, hWheel int32) error {
	if l.dev == nil {
		return nil
	}
	if vWheel != 0 {
		_ = l.dev.WriteOne(&evdev.InputEvent{Type: evdev.EV_REL, Code: evdev.REL_WHEEL, Value: vWheel})
	}
	if hWheel != 0 {
		_ = l.dev.WriteOne(&evdev.InputEvent{Type: evdev.EV_REL, Code: evdev.REL_HWHEEL, Value: hWheel})
	}
	_ = l.dev.WriteOne(&evdev.InputEvent{Type: evdev.EV_SYN, Code: evdev.SYN_REPORT, Value: 0})
	return nil
}

func (l *linuxInjector) Key(code uint16, state int32) error {
	if l.dev == nil {
		return nil
	}
	_ = l.dev.WriteOne(&evdev.InputEvent{Type: evdev.EV_KEY, Code: evdev.EvCode(code), Value: state})
	_ = l.dev.WriteOne(&evdev.InputEvent{Type: evdev.EV_SYN, Code: evdev.SYN_REPORT, Value: 0})
	return nil
}

func (l *linuxInjector) GetCursorPos() (int32, int32, error) {
	return 0, 0, nil
}

func (l *linuxInjector) GetScreenSize() (int32, int32, error) {
	return 1920, 1080, nil
}

func (l *linuxInjector) Close() error {
	if l.dev != nil {
		return l.dev.Close()
	}
	return nil
}
