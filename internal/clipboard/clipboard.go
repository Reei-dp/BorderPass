package clipboard

// Manager defines the interface for cross-platform clipboard synchronization
type Manager interface {
	GetText() (string, error)
	SetText(text string) error
	StartWatching(onChange func(text string))
	Close() error
}

// NewManager returns a platform-specific clipboard manager implementation
func NewManager() Manager {
	return newPlatformManager()
}
