package server

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"unicode/utf8"

	"borderpass/config"
	"borderpass/internal/clipboard"
	"borderpass/internal/platform"
	"borderpass/internal/telemetry"
	"borderpass/protocol"
)

// RunServer starts the BorderPass server with context support
func RunServer(ctx context.Context, cfg *config.Config) error {
	telemetry.Global.SetRunning(true, config.RoleServer)
	defer telemetry.Global.SetRunning(false, config.RoleServer)

	capturer, err := platform.NewCapturer(cfg)
	if err != nil {
		telemetry.Global.SetError(err.Error())
		return fmt.Errorf("device capture initialization error: %w", err)
	}
	defer capturer.Close()

	listenAddr := cfg.Address
	if strings.HasPrefix(listenAddr, ":") {
		listenAddr = "0.0.0.0" + listenAddr
	} else if !strings.Contains(listenAddr, ":") {
		listenAddr = "0.0.0.0:" + listenAddr
	} else {
		_, port, err := net.SplitHostPort(listenAddr)
		if err == nil {
			listenAddr = "0.0.0.0:" + port
		} else {
			listenAddr = "0.0.0.0:65432"
		}
	}

	var lc net.ListenConfig
	listener, err := lc.Listen(ctx, "tcp", listenAddr)
	if err != nil {
		telemetry.Global.SetError(err.Error())
		return fmt.Errorf("failed to start server on %s: %w", listenAddr, err)
	}
	defer listener.Close()

	telemetry.Global.SetStatus("LISTENING", cfg.Address)
	telemetry.Global.SetControl("HOST (LOCAL)")

	// Goroutine to close listener when context is cancelled
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				telemetry.Global.SetError(err.Error())
				continue
			}
		}

		handleClient(ctx, conn, cfg, capturer)
	}
}

func handleClient(ctx context.Context, conn net.Conn, cfg *config.Config, capturer platform.Capturer) {
	defer conn.Close()
	defer capturer.Ungrab()
	defer telemetry.Global.SetStatus("LISTENING", cfg.Address)
	defer telemetry.Global.SetControl("HOST (LOCAL)")

	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(true)
		_ = tcpConn.SetWriteBuffer(128 * 1024)
		_ = tcpConn.SetReadBuffer(128 * 1024)
	}

	remoteAddr := conn.RemoteAddr().String()
	telemetry.Global.SetStatus("CONNECTED", remoteAddr)

	var sendMu sync.Mutex
	safeSend := func(pkt protocol.Packet) error {
		sendMu.Lock()
		defer sendMu.Unlock()
		telemetry.Global.IncSent()
		return protocol.SendPacket(conn, pkt)
	}

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

	// Send initial layout handshake to client so it knows return edge
	_ = safeSend(protocol.Packet{
		Type: protocol.TypeHandshake,
		Data: []byte(cfg.Position),
	})

	releaseChan := make(chan struct{}, 10)
	clientDone := make(chan struct{})

	// Background reader for packets from Client
	go func() {
		defer close(clientDone)
		for {
			pkt, err := protocol.ReadPacket(conn)
			if err != nil {
				return
			}
			telemetry.Global.IncRecv()

			switch pkt.Type {
			case protocol.TypeRelease:
				select {
				case releaseChan <- struct{}{}:
				default:
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
	}()

	virtualX := cfg.ScreenWidth / 2
	virtualY := cfg.ScreenHeight / 2
	edgeAccumulator := 0
	const threshold = 15

	eventChan := make(chan platform.InputEvent, 1024)
	if err := capturer.Start(func(ev platform.InputEvent) {
		select {
		case eventChan <- ev:
		default:
		}
	}); err != nil {
		telemetry.Global.SetError(err.Error())
		return
	}

	for {
		select {
		case <-ctx.Done():
			_ = capturer.Ungrab()
			telemetry.Global.SetControl("HOST (LOCAL)")
			return

		case <-clientDone:
			_ = capturer.Ungrab()
			telemetry.Global.SetStatus("LISTENING (CLIENT DISCONNECTED)", "")
			telemetry.Global.SetControl("HOST (LOCAL)")
			return

		case <-releaseChan:
			if capturer.IsGrabbed() {
				_ = capturer.Ungrab()
				telemetry.Global.SetControl("HOST (LOCAL)")
				switch cfg.Position {
				case config.PosRight:
					virtualX = cfg.ScreenWidth - 80
				case config.PosLeft:
					virtualX = 80
				case config.PosTop:
					virtualY = 80
				case config.PosBottom:
					virtualY = cfg.ScreenHeight - 80
				}
				edgeAccumulator = 0
			}

		case ev := <-eventChan:
			isGrabbed := capturer.IsGrabbed()

			switch ev.Type {
			case platform.EventMouseMove:
				dx := ev.Value
				dy := ev.Extra

				if !isGrabbed {
					virtualX += int(dx)
					virtualY += int(dy)

					if virtualX < 0 {
						virtualX = 0
					}
					if virtualX > cfg.ScreenWidth {
						virtualX = cfg.ScreenWidth
					}
					if virtualY < 0 {
						virtualY = 0
					}
					if virtualY > cfg.ScreenHeight {
						virtualY = cfg.ScreenHeight
					}

					shouldGrab := false
					switch cfg.Position {
					case config.PosRight:
						if virtualX >= cfg.ScreenWidth && dx > 0 {
							edgeAccumulator += int(dx)
							if edgeAccumulator >= threshold {
								shouldGrab = true
							}
						} else {
							edgeAccumulator = 0
						}

					case config.PosLeft:
						if virtualX <= 0 && dx < 0 {
							edgeAccumulator += int(-dx)
							if edgeAccumulator >= threshold {
								shouldGrab = true
							}
						} else {
							edgeAccumulator = 0
						}

					case config.PosTop:
						if virtualY <= 0 && dy < 0 {
							edgeAccumulator += int(-dy)
							if edgeAccumulator >= threshold {
								shouldGrab = true
							}
						} else {
							edgeAccumulator = 0
						}

					case config.PosBottom:
						if virtualY >= cfg.ScreenHeight && dy > 0 {
							edgeAccumulator += int(dy)
							if edgeAccumulator >= threshold {
								shouldGrab = true
							}
						} else {
							edgeAccumulator = 0
						}
					}

					if shouldGrab {
						if err := capturer.Grab(); err == nil {
							edgeAccumulator = 0
							telemetry.Global.SetControl("CLIENT (REMOTE)")
						}
					}
				} else {
					_ = safeSend(protocol.Packet{
						Type:  protocol.TypeMouseMove,
						Value: dx,
						Extra: dy,
					})
				}

			case platform.EventMouseButton:
				if isGrabbed {
					_ = safeSend(protocol.Packet{
						Type:  protocol.TypeMouseBtn,
						Code:  ev.Code,
						Value: ev.Value,
					})
				}

			case platform.EventMouseWheel:
				if isGrabbed {
					_ = safeSend(protocol.Packet{
						Type:  protocol.TypeMouseWheel,
						Value: ev.Value,
						Extra: ev.Extra,
					})
				}

			case platform.EventKey:
				if isGrabbed {
					_ = safeSend(protocol.Packet{
						Type:  protocol.TypeKey,
						Code:  ev.Code,
						Value: ev.Value,
					})
				}
			}
		}
	}
}
