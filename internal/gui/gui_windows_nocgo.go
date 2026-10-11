//go:build windows && !cgo

package gui

import (
	"borderpass/config"
	"borderpass/internal/tui"
)

// Start launches the TUI interface on Windows when built without CGO
func Start(cfg *config.Config) error {
	return tui.Run(cfg)
}
