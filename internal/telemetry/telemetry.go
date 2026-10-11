package telemetry

import (
	"sync"
	"sync/atomic"
)

// Snapshot contains an instantaneous copy of current metrics
type Snapshot struct {
	IsRunning       bool
	Role            string
	Status          string
	RemoteAddr      string
	ActiveControl   string
	PacketsSent     uint64
	PacketsRecv     uint64
	ClipboardEvents uint64
	LastError       string
}

// Store provides thread-safe runtime telemetry
type Store struct {
	mu              sync.RWMutex
	isRunning       bool
	role            string
	status          string
	remoteAddr      string
	activeControl   string
	lastError       string

	packetsSent     atomic.Uint64
	packetsRecv     atomic.Uint64
	clipboardEvents atomic.Uint64
}

// Global telemetry store instance
var Global = &Store{
	status:        "OFFLINE",
	activeControl: "HOST (LOCAL)",
}

// Get returns the current telemetry snapshot
func (s *Store) Get() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Snapshot{
		IsRunning:       s.isRunning,
		Role:            s.role,
		Status:          s.status,
		RemoteAddr:      s.remoteAddr,
		ActiveControl:   s.activeControl,
		PacketsSent:     s.packetsSent.Load(),
		PacketsRecv:     s.packetsRecv.Load(),
		ClipboardEvents: s.clipboardEvents.Load(),
		LastError:       s.lastError,
	}
}

// SetRunning updates the running state and role
func (s *Store) SetRunning(running bool, role string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isRunning = running
	s.role = role
	if !running {
		s.status = "STOPPED"
		s.activeControl = "HOST (LOCAL)"
	}
}

// SetStatus updates connection status and peer address
func (s *Store) SetStatus(status, remoteAddr string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = status
	s.remoteAddr = remoteAddr
}

// SetControl updates which machine currently owns input focus
func (s *Store) SetControl(control string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeControl = control
}

// IncSent increments the transmitted packet counter locklessly
func (s *Store) IncSent() {
	s.packetsSent.Add(1)
}

// IncRecv increments the received packet counter locklessly
func (s *Store) IncRecv() {
	s.packetsRecv.Add(1)
}

// IncClipboard increments the clipboard sync counter locklessly
func (s *Store) IncClipboard() {
	s.clipboardEvents.Add(1)
}

// SetError records the last runtime error
func (s *Store) SetError(err string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastError = err
}

