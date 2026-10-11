//go:build windows

package platform

import (
	"sync"
	"syscall"

	"borderpass/config"
)

var (
	user32 = syscall.NewLazyDLL("user32.dll")
)

type windowsCapturer struct {
	mu        sync.Mutex
	isGrabbed bool
}

// NewCapturer creates a Capturer for Windows
func NewCapturer(cfg *config.Config) (Capturer, error) {
	return &windowsCapturer{}, nil
}

func (c *windowsCapturer) Start(onEvent func(ev InputEvent)) error {
	// Base implementation for Windows host mode
	return nil
}

func (c *windowsCapturer) Grab() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isGrabbed = true
	return nil
}

func (c *windowsCapturer) Ungrab() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isGrabbed = false
	return nil
}

func (c *windowsCapturer) IsGrabbed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isGrabbed
}

func (c *windowsCapturer) Close() error {
	return nil
}
