package main

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"

	"borderpass/client"
	"borderpass/config"
	"borderpass/internal/telemetry"
	"borderpass/server"
)

// App struct
type App struct {
	ctx        context.Context
	cfg        *config.Config
	cancelFunc context.CancelFunc
	doneChan   chan struct{}
	isRunning  bool
	gen        int
	mu         sync.Mutex
}

// NewApp creates a new App application struct
func NewApp() *App {
	cfg, err := config.Load()
	if err != nil {
		def := config.DefaultConfig()
		cfg = &def
	}
	return &App{
		cfg: cfg,
	}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Float and size window in Hyprland
	go func() {
		for i := 0; i < 8; i++ {
			time.Sleep(250 * time.Millisecond)
			luaFloat := `for _, w in ipairs(hl.get_windows()) do if w.class == "borderpass" and not w.floating then hl.dispatch(hl.dsp.window.float()) hl.dispatch(hl.dsp.window.resize({ x = 860, y = 660, window = w })) hl.dispatch(hl.dsp.window.center(w)) end end`
			_ = exec.Command("hyprctl", "eval", luaFloat).Run()
		}
	}()
}

// GetConfig returns current config
func (a *App) GetConfig() config.Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	return *a.cfg
}

// SaveConfig updates and saves config
func (a *App) SaveConfig(role, position, address string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if role != "" {
		a.cfg.Role = role
	}
	if position != "" {
		a.cfg.Position = position
	}
	if address != "" {
		a.cfg.Address = address
	}
	return a.cfg.Save()
}

// GetLocalIP returns host IPv4 address
func (a *App) GetLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		localAddr := conn.LocalAddr().(*net.UDPAddr)
		return localAddr.IP.String()
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}
	return "127.0.0.1"
}

// StartService starts server or client engine
func (a *App) StartService(role, address, position string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cancelFunc != nil {
		a.cancelFunc()
		a.cancelFunc = nil
		if a.doneChan != nil {
			done := a.doneChan
			a.mu.Unlock()
			select {
			case <-done:
			case <-time.After(600 * time.Millisecond):
			}
			a.mu.Lock()
		}
	}

	a.cfg.Role = role
	a.cfg.Address = strings.TrimSpace(address)
	a.cfg.Position = position
	_ = a.cfg.Save()

	a.gen++
	currentGen := a.gen
	done := make(chan struct{})
	a.doneChan = done

	ctx, cancel := context.WithCancel(context.Background())
	a.cancelFunc = cancel
	a.isRunning = true
	telemetry.Global.SetRunning(true, a.cfg.Role)

	if a.cfg.Role == config.RoleServer {
		go func() {
			defer close(done)
			if err := server.RunServer(ctx, a.cfg); err != nil {
				fmt.Printf("❌ Server error: %v\n", err)
				telemetry.Global.SetError(err.Error())
			}
			a.mu.Lock()
			if a.gen == currentGen {
				a.isRunning = false
				telemetry.Global.SetRunning(false, a.cfg.Role)
			}
			a.mu.Unlock()
		}()
	} else {
		go func() {
			defer close(done)
			if err := client.RunClient(ctx, a.cfg); err != nil {
				fmt.Printf("❌ Client error: %v\n", err)
				telemetry.Global.SetError(err.Error())
			}
			a.mu.Lock()
			if a.gen == currentGen {
				a.isRunning = false
				telemetry.Global.SetRunning(false, a.cfg.Role)
			}
			a.mu.Unlock()
		}()
	}

	return fmt.Sprintf("Started %s on %s", strings.ToUpper(a.cfg.Role), a.cfg.Address), nil
}

// StopService terminates running KM switch engine
func (a *App) StopService() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cancelFunc != nil {
		a.cancelFunc()
		a.cancelFunc = nil
		if a.doneChan != nil {
			done := a.doneChan
			a.mu.Unlock()
			select {
			case <-done:
			case <-time.After(600 * time.Millisecond):
			}
			a.mu.Lock()
		}
	}
	a.gen++
	a.isRunning = false
	telemetry.Global.SetRunning(false, a.cfg.Role)
	return nil
}

// TelemetryData contains metrics passed to frontend
type TelemetryData struct {
	IsRunning       bool   `json:"isRunning"`
	Role            string `json:"role"`
	Status          string `json:"status"`
	RemoteAddr      string `json:"remoteAddr"`
	ActiveControl   string `json:"activeControl"`
	PacketsSent     uint64 `json:"packetsSent"`
	PacketsRecv     uint64 `json:"packetsRecv"`
	ClipboardEvents uint64 `json:"clipboardEvents"`
	LastError       string `json:"lastError"`
}

// GetTelemetry returns current metrics snapshot
func (a *App) GetTelemetry() TelemetryData {
	snap := telemetry.Global.Get()
	return TelemetryData{
		IsRunning:       snap.IsRunning,
		Role:            snap.Role,
		Status:          snap.Status,
		RemoteAddr:      snap.RemoteAddr,
		ActiveControl:   snap.ActiveControl,
		PacketsSent:     snap.PacketsSent,
		PacketsRecv:     snap.PacketsRecv,
		ClipboardEvents: snap.ClipboardEvents,
		LastError:       snap.LastError,
	}
}
