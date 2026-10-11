package client

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"sync"
	"time"
	"unicode/utf8"

	"borderpass/config"
	"borderpass/internal/clipboard"
	"borderpass/internal/platform"
	"borderpass/internal/telemetry"
	"borderpass/protocol"
)

// RunClient starts the BorderPass client with context support
func RunClient(ctx context.Context, cfg *config.Config) error {
	telemetry.Global.SetRunning(true, config.RoleClient)
	defer telemetry.Global.SetRunning(false, config.RoleClient)

	injector, err := platform.NewInjector(cfg)
	if err != nil {
		telemetry.Global.SetError(err.Error())
		return fmt.Errorf("input injection initialization error: %w", err)
	}
	defer injector.Close()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		telemetry.Global.SetStatus("CONNECTING...", cfg.Address)
		var d net.Dialer
		conn, err := d.DialContext(ctx, "tcp", cfg.Address)
		if err != nil {
			telemetry.Global.SetStatus("DISCONNECTED", cfg.Address)
			telemetry.Global.SetError(err.Error())
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(2 * time.Second):
				continue
			}
		}

		if tcpConn, ok := conn.(*net.TCPConn); ok {
			_ = tcpConn.SetNoDelay(true)
			_ = tcpConn.SetReadBuffer(128 * 1024)
			_ = tcpConn.SetWriteBuffer(128 * 1024)
		}

		telemetry.Global.SetStatus("CONNECTED", conn.RemoteAddr().String())
		telemetry.Global.SetControl("HOST (REMOTE)")

		sessionErr := runClientSession(ctx, conn, cfg, injector)
		_ = conn.Close()

		telemetry.Global.SetStatus("RECONNECTING...", cfg.Address)
		if sessionErr != nil {
			telemetry.Global.SetError(sessionErr.Error())
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(2 * time.Second):
		}
	}
}

func runClientSession(ctx context.Context, conn net.Conn, cfg *config.Config, injector platform.Injector) error {
	var sendMu sync.Mutex
	safeSend := func(pkt protocol.Packet) error {
		sendMu.Lock()
		defer sendMu.Unlock()
		telemetry.Global.IncSent()
		return protocol.SendPacket(conn, pkt)
	}

	reader := bufio.NewReaderSize(conn, 64*1024)

	// Clipboard synchronization initialization
	var clipMgr clipboard.Manager
	if cfg.EnableClipboard {
		clipMgr = clipboard.NewManager()
		defer clipMgr.Close()

		clipMgr.StartWatching(func(text string) {
			telemetry.Global.IncClipboard()
			_ = safeSend(protocol.Packet{
				Type: protocol.TypeClipboard,
				Data: []byte(text),
			})
		})
	}

	stopCh := make(chan struct{})
	defer close(stopCh)

	currentPosition := cfg.Position
	var posMu sync.Mutex
	var lastReleaseTime time.Time

	// Cache screen size once to avoid thousands of Win32 GetSystemMetrics calls
	sw, sh, _ := injector.GetScreenSize()
	if sw <= 0 {
		sw = int32(cfg.ScreenWidth)
	}
	if sh <= 0 {
		sh = int32(cfg.ScreenHeight)
	}

	checkAndRelease := func(x, y int32, screenW, screenH int32, dx, dy int32) bool {
		posMu.Lock()
		pos := currentPosition
		posMu.Unlock()

		shouldRelease := false
		switch pos {
		case config.PosRight: // Second PC is to the RIGHT of Host -> user moves LEFT to return to Host
			if (x <= 15 && dx < 0) || x <= 2 {
				shouldRelease = true
			}
		case config.PosLeft: // Second PC is to the LEFT of Host -> user moves RIGHT to return to Host
			if (x >= screenW-15 && dx > 0) || x >= screenW-2 {
				shouldRelease = true
			}
		case config.PosTop: // Second PC is ABOVE Host -> user moves DOWN to return to Host
			if (y >= screenH-15 && dy > 0) || y >= screenH-2 {
				shouldRelease = true
			}
		case config.PosBottom: // Second PC is BELOW Host -> user moves UP to return to Host
			if (y <= 15 && dy < 0) || y <= 2 {
				shouldRelease = true
			}
		}

		if shouldRelease && time.Since(lastReleaseTime) > 200*time.Millisecond {
			lastReleaseTime = time.Now()
			_ = safeSend(protocol.Packet{
				Type: protocol.TypeRelease,
			})
			return true
		}
		return false
	}

	// Border monitoring goroutine to return control back to host
	go func() {
		ticker := time.NewTicker(12 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-stopCh:
				return
			case <-ticker.C:
				x, y, err := injector.GetCursorPos()
				if err != nil {
					continue
				}
				checkAndRelease(x, y, sw, sh, 0, 0)
			}
		}
	}()

	// Background listener to close connection when context is cancelled
	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	// Main packet receiver and injection loop
	for {
		pkt, err := protocol.ReadPacket(reader)
		if err != nil {
			return err
		}
		telemetry.Global.IncRecv()

		switch pkt.Type {
		case protocol.TypeHandshake:
			if len(pkt.Data) > 0 {
				newPos := string(pkt.Data)
				fmt.Printf("[Client] Server synchronized position layout: %s\n", newPos)
				posMu.Lock()
				currentPosition = newPos
				posMu.Unlock()
			}

		case protocol.TypeMouseMove:
			dx := pkt.Value
			dy := pkt.Extra

			// Coalesce any queued mouse moves already waiting in the TCP read buffer
			for reader.Buffered() >= protocol.HeaderSize {
				peekBytes, err := reader.Peek(protocol.HeaderSize)
				if err != nil || peekBytes[0] != protocol.TypeMouseMove {
					break
				}
				nextPkt, err := protocol.ReadPacket(reader)
				if err != nil {
					break
				}
				telemetry.Global.IncRecv()
				dx += nextPkt.Value
				dy += nextPkt.Extra
			}

			_ = injector.MoveMouse(dx, dy)

		case protocol.TypeMouseBtn:
			_ = injector.MouseButton(pkt.Code, pkt.Value != 0)

		case protocol.TypeMouseWheel:
			_ = injector.MouseWheel(pkt.Value, pkt.Extra)

		case protocol.TypeKey:
			// Emergency release hotkey: Pause/Break (119) or ScrollLock (78)
			if pkt.Code == 119 || pkt.Code == 78 {
				_ = safeSend(protocol.Packet{
					Type: protocol.TypeRelease,
				})
			} else {
				_ = injector.Key(pkt.Code, pkt.Value)
			}

		case protocol.TypeClipboard:
			if clipMgr != nil && len(pkt.Data) > 0 {
				if utf8.Valid(pkt.Data) && bytes.IndexByte(pkt.Data, 0) == -1 && !bytes.HasPrefix(pkt.Data, []byte("\x89PNG")) {
					telemetry.Global.IncClipboard()
					_ = clipMgr.SetText(string(pkt.Data))
				}
			}
		}
	}
}
