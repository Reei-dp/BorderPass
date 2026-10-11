package platform

// EventType represents the type of input event
type EventType int

const (
	EventMouseMove EventType = iota
	EventMouseButton
	EventMouseWheel
	EventKey
)

// InputEvent represents an abstracted input event
type InputEvent struct {
	Type  EventType
	Code  uint16
	Value int32
	Extra int32
}

// Capturer defines the interface for capturing host input events (Server side)
type Capturer interface {
	Start(onEvent func(ev InputEvent)) error
	Grab() error
	Ungrab() error
	IsGrabbed() bool
	Close() error
}

// Injector defines the interface for injecting synthetic input events (Client side)
type Injector interface {
	MoveMouse(dx, dy int32) error
	MouseButton(button uint16, pressed bool) error
	MouseWheel(vWheel, hWheel int32) error
	Key(code uint16, state int32) error
	GetCursorPos() (x, y int32, err error)
	GetScreenSize() (width, height int32, err error)
	Close() error
}

// Factory functions NewCapturer and NewInjector are implemented in OS-specific files:
// capture_linux.go / capture_windows.go
// inject_linux.go / inject_windows.go
