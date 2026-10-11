//go:build !windows || (windows && cgo)

package gui

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"borderpass/client"
	"borderpass/config"
	"borderpass/internal/telemetry"
	"borderpass/server"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// fixedSizeLayout ensures a container maintains a fixed minimum size without collapsing
type fixedSizeLayout struct {
	size fyne.Size
}

func (l *fixedSizeLayout) MinSize(_ []fyne.CanvasObject) fyne.Size {
	return l.size
}

func (l *fixedSizeLayout) Layout(_ []fyne.CanvasObject, _ fyne.Size) {
	// Objects are manually positioned using Move()
}

// padCenter centers string s inside width runes
func padCenter(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		return string(runes[:width])
	}
	total := width - len(runes)
	left := total / 2
	right := total - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

// renderBox returns the pure 8-line ASCII monitor box with ║ (CP437 186 / \u2551)
func renderBox(title, line1, line2, status string) string {
	l1 := "╔═════════════════════════╗"
	l2 := "║" + padCenter(title, 25) + "║"
	l3 := "║                         ║"
	l4 := "║" + padCenter(line1, 25) + "║"
	l5 := "║" + padCenter(line2, 25) + "║"
	l6 := "║                         ║"
	l7 := "║" + padCenter(status, 25) + "║"
	l8 := "╚═════════════════════════╝"
	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s", l1, l2, l3, l4, l5, l6, l7, l8)
}

// ClickableBox is an interactive monospace ASCII widget that responds to clicks
type ClickableBox struct {
	widget.BaseWidget
	label    *widget.Label
	onTapped func()
}

func newClickableBox(text string, onTapped func()) *ClickableBox {
	b := &ClickableBox{
		label:    widget.NewLabel(text),
		onTapped: onTapped,
	}
	b.label.TextStyle = fyne.TextStyle{Monospace: true, Bold: true}
	b.label.Alignment = fyne.TextAlignCenter
	b.ExtendBaseWidget(b)
	return b
}

func (b *ClickableBox) SetText(text string) {
	b.label.SetText(text)
	b.Refresh()
}

func (b *ClickableBox) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(b.label)
}

func (b *ClickableBox) Tapped(_ *fyne.PointEvent) {
	if b.onTapped != nil {
		b.onTapped()
	}
}

func (b *ClickableBox) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

// DraggableMonitorBox represents a movable display box (PC 1 or PC 2) in the spatial canvas
type DraggableMonitorBox struct {
	widget.BaseWidget
	label       *widget.Label
	name        string
	status      string
	currentSnap string
	onPosition  func(pos string)
	isDragging  bool
}

func newDraggableMonitorBox(name, status, initialSnap string, onPosition func(pos string)) *DraggableMonitorBox {
	d := &DraggableMonitorBox{
		name:        name,
		status:      status,
		currentSnap: initialSnap,
		onPosition:  onPosition,
	}
	d.label = widget.NewLabel(d.renderText())
	d.label.TextStyle = fyne.TextStyle{Monospace: true, Bold: true}
	d.label.Alignment = fyne.TextAlignCenter
	d.ExtendBaseWidget(d)
	return d
}

func (d *DraggableMonitorBox) renderText() string {
	l1 := "╔═══════════════╗"
	l2 := "║" + padCenter(d.name, 15) + "║"
	l3 := "║" + padCenter(d.status, 15) + "║"
	l4 := "╚═══════════════╝"
	return fmt.Sprintf("%s\n%s\n%s\n%s", l1, l2, l3, l4)
}

func (d *DraggableMonitorBox) SetStatus(status string) {
	d.status = status
	d.label.SetText(d.renderText())
	d.Refresh()
}

func (d *DraggableMonitorBox) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(d.label)
}

func (d *DraggableMonitorBox) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

func (d *DraggableMonitorBox) Tapped(_ *fyne.PointEvent) {}

func (d *DraggableMonitorBox) Dragged(e *fyne.DragEvent) {
	d.isDragging = true
	cur := d.Position()
	newX := cur.X + e.Dragged.DX
	newY := cur.Y + e.Dragged.DY

	if newX < 5 {
		newX = 5
	}
	if newX > 370 {
		newX = 370
	}
	if newY < 5 {
		newY = 5
	}
	if newY > 140 {
		newY = 140
	}
	d.Move(fyne.NewPos(newX, newY))
}

func (d *DraggableMonitorBox) DragEnd() {
	d.isDragging = false

	// Host center is at (185 + 75, 70 + 35) = (260, 105)
	hostCenterX := float32(260)
	hostCenterY := float32(105)

	clientCurPos := d.Position()
	clientCenterX := clientCurPos.X + 75
	clientCenterY := clientCurPos.Y + 35

	diffX := clientCenterX - hostCenterX
	diffY := clientCenterY - hostCenterY

	targetPos := config.PosLeft
	absX := diffX
	if absX < 0 {
		absX = -absX
	}
	absY := diffY
	if absY < 0 {
		absY = -absY
	}

	if absX > absY {
		if diffX >= 0 {
			targetPos = config.PosRight
		} else {
			targetPos = config.PosLeft
		}
	} else {
		if diffY >= 0 {
			targetPos = config.PosBottom
		} else {
			targetPos = config.PosTop
		}
	}

	d.snapTo(targetPos)
	if d.onPosition != nil {
		d.onPosition(targetPos)
	}
}

func (d *DraggableMonitorBox) snapTo(pos string) {
	d.currentSnap = pos
	switch pos {
	case config.PosRight:
		d.Move(fyne.NewPos(355, 70))
	case config.PosLeft:
		d.Move(fyne.NewPos(15, 70))
	case config.PosTop:
		d.Move(fyne.NewPos(185, 5))
	case config.PosBottom:
		d.Move(fyne.NewPos(185, 135))
	}
}

// renderHostBox returns the Host monitor representation for the spatial canvas
func renderHostBox(lanIP string) string {
	l1 := "╔═══════════════╗"
	l2 := "║    HOST PC    ║"
	l3 := "║" + padCenter(lanIP, 15) + "║"
	l4 := "╚═══════════════╝"
	return fmt.Sprintf("%s\n%s\n%s\n%s", l1, l2, l3, l4)
}

// getLocalIP finds the primary LAN IPv4 address to show helper hints
func getLocalIP() string {
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

type retroTheme struct {
	fyne.Theme
}

func (m *retroTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNameText {
		return 11
	}
	return m.Theme.Size(name)
}

// Start launches the Fyne application with the 3-step wizard flow
func Start(cfg *config.Config) error {
	a := app.NewWithID("com.borderpass.switch")
	a.Settings().SetTheme(&retroTheme{Theme: theme.DarkTheme()})

	w := a.NewWindow("BorderPass")
	w.Resize(fyne.NewSize(880, 680))

	// Automatically float and center window in Hyprland
	go func() {
		time.Sleep(150 * time.Millisecond)
		luaRule := `hl.window_rule({ match = { class = "com.borderpass.switch" }, float = true, size = "880 680", center = true })`
		_ = exec.Command("hyprctl", "eval", luaRule).Run()
		time.Sleep(100 * time.Millisecond)
		luaFloat := `for _, w in ipairs(hl.get_windows()) do if w.class == "com.borderpass.switch" then hl.dispatch(hl.dsp.window.float(w)) hl.dispatch(hl.dsp.window.resize({ x = 880, y = 680, window = w })) hl.dispatch(hl.dsp.window.center(w)) end end`
		_ = exec.Command("hyprctl", "eval", luaFloat).Run()
	}()

	var cancelFunc context.CancelFunc
	isRunning := false

	stopService := func() {
		if cancelFunc != nil {
			cancelFunc()
			cancelFunc = nil
		}
		isRunning = false
		telemetry.Global.SetRunning(false, cfg.Role)
	}

	var showScreen1 func()
	var showScreen2 func()
	var showScreen3 func()

	// -----------------------------------------------------------------------------------------
	// SCREEN 1: BORDERPASS ASCII title + Continue button matching exact width, NOTHING ELSE!
	// -----------------------------------------------------------------------------------------
	showScreen1 = func() {
		asciiLogo := "╔═══════════════════════════════════════════════════╗\n" +
			"║ █▀▀▄ █▀▀█ █▀▀█ █▀▀▄ █▀▀▀ █▀▀█ █▀▀█ █▀▀█ █▀▀▀ █▀▀▀ ║\n" +
			"║ █▀▀▄ █░░█ █▄▄▀ █░░█ █▀▀░ █▄▄▀ █▄▄█ █▄▄█ ▀▀▀█ ▀▀▀█ ║\n" +
			"║ ▀▀▀░ ▀▀▀▀ ▀░▀▀ ▀▀▀░ ▀▀▀▀ ▀░▀▀ ▀░░░ ▀░░▀ ▀▀▀░ ▀▀▀░ ║\n" +
			"╚═══════════════════════════════════════════════════╝"

		titleLabel := widget.NewLabel(asciiLogo)
		titleLabel.TextStyle = fyne.TextStyle{Monospace: true, Bold: true}
		titleLabel.Alignment = fyne.TextAlignCenter

		asciiContinueText := "╔═══════════════════════════════════════════════════╗\n" +
			"║" + padCenter("[ CONTINUE ]", 51) + "║\n" +
			"╚═══════════════════════════════════════════════════╝"

		continueBtn := newClickableBox(asciiContinueText, func() {
			showScreen2()
		})

		content := container.NewCenter(
			container.NewVBox(
				titleLabel,
				continueBtn,
			),
		)
		w.SetContent(content)
	}

	// -----------------------------------------------------------------------------------------
	// SCREEN 2: Just the two monitor boxes (HOST PC vs SECOND PC) with double-line ║ frames
	// -----------------------------------------------------------------------------------------
	showScreen2 = func() {
		hostBoxText := renderBox(
			"HOST PC",
			"Shares Mouse & Keyboard",
			"Primary Workstation",
			"[ CHOOSE HOST ]",
		)
		secondBoxText := renderBox(
			"SECOND PC",
			"Receives Remote Control",
			"Secondary Computer",
			"[ CHOOSE SECOND PC ]",
		)

		hostBox := newClickableBox(hostBoxText, func() {
			cfg.Role = config.RoleServer
			if cfg.Address == "" || strings.Contains(cfg.Address, "192.168") {
				cfg.Address = ":65432"
			}
			_ = cfg.Save()
			showScreen3()
		})

		secondBox := newClickableBox(secondBoxText, func() {
			cfg.Role = config.RoleClient
			if cfg.Address == ":65432" || cfg.Address == "" {
				cfg.Address = "192.168.1.100:65432"
			}
			_ = cfg.Save()
			showScreen3()
		})

		content := container.NewCenter(
			container.NewHBox(
				hostBox,
				layout.NewSpacer(),
				secondBox,
			),
		)
		w.SetContent(content)
	}

	// -----------------------------------------------------------------------------------------
	// SCREEN 3: Layout matching user's sketch:
	// - Top: Large box with Monitor Arrangement (Host, PC 1, PC 2)
	// - Bottom-Left: Network box with IP and Port fields
	// - Bottom-Right: Client status box (PC 1 idle, PC 2 idle)
	// - Bottom: Start/Stop ASCII button
	// -----------------------------------------------------------------------------------------
	var tickerStop chan struct{}

	showScreen3 = func() {
		stopService()
		if tickerStop != nil {
			close(tickerStop)
			tickerStop = nil
		}

		lanIP := getLocalIP()

		// Top Header with Back button and Role description
		roleTitle := "HOST PC (Shares Input)"
		if cfg.Role == config.RoleClient {
			roleTitle = "SECOND PC (Remote Input)"
		}
		roleLabel := widget.NewLabel(roleTitle)
		roleLabel.TextStyle = fyne.TextStyle{Monospace: true, Bold: true}

		backBtn := newClickableBox("╔══════════╗\n║ [◄ BACK] ║\n╚══════════╝", func() {
			stopService()
			if tickerStop != nil {
				close(tickerStop)
				tickerStop = nil
			}
			showScreen2()
		})

		headerBar := container.NewHBox(
			backBtn,
			layout.NewSpacer(),
			roleLabel,
			layout.NewSpacer(),
		)

		// -----------------------------------------------------------------------------
		// 1. TOP BOX: Monitors Arrangement Canvas (Host, PC 1, PC 2)
		// -----------------------------------------------------------------------------
		transitionLabel := widget.NewLabel("")
		transitionLabel.TextStyle = fyne.TextStyle{Monospace: true, Bold: true}
		transitionLabel.Alignment = fyne.TextAlignCenter

		updateTransition := func() {
			arrow := "◄══"
			switch cfg.Position {
			case config.PosRight:
				arrow = "══►"
			case config.PosTop:
				arrow = "▲"
			case config.PosBottom:
				arrow = "▼"
			}
			transitionLabel.SetText(fmt.Sprintf("%s PC 1: %s   │   ▲ PC 2: TOP", arrow, strings.ToUpper(cfg.Position)))
		}

		// Anchor Host Box in Center
		hostAnchorLabel := widget.NewLabel(renderHostBox(lanIP))
		hostAnchorLabel.TextStyle = fyne.TextStyle{Monospace: true, Bold: true}
		hostAnchorLabel.Alignment = fyne.TextAlignCenter
		hostAnchorLabel.Move(fyne.NewPos(185, 70))
		hostAnchorLabel.Resize(fyne.NewSize(150, 70))

		// PC 1 Box (Draggable, default Left)
		var draggablePC1 *DraggableMonitorBox
		var draggablePC2 *DraggableMonitorBox

		draggablePC1 = newDraggableMonitorBox("PC 1", "● idle", cfg.Position, func(pos string) {
			cfg.Position = pos
			_ = cfg.Save()
			// If PC 1 moved to PC 2's snap, swap PC 2 to avoid collision
			if draggablePC2 != nil && draggablePC2.currentSnap == pos {
				otherSnap := config.PosTop
				if pos == config.PosTop {
					otherSnap = config.PosRight
				}
				draggablePC2.snapTo(otherSnap)
			}
			updateTransition()
		})
		draggablePC1.Resize(fyne.NewSize(150, 70))
		draggablePC1.snapTo(cfg.Position)

		// PC 2 Box (Draggable, default Top)
		draggablePC2 = newDraggableMonitorBox("PC 2", "● idle", config.PosTop, func(pos string) {
			// If PC 2 moved to PC 1's snap, swap PC 1 to avoid collision
			if draggablePC1 != nil && draggablePC1.currentSnap == pos {
				otherSnap := config.PosLeft
				if pos == config.PosLeft {
					otherSnap = config.PosRight
				}
				cfg.Position = otherSnap
				_ = cfg.Save()
				draggablePC1.snapTo(otherSnap)
			}
			updateTransition()
		})
		draggablePC2.Resize(fyne.NewSize(150, 70))
		draggablePC2.snapTo(config.PosTop)

		updateTransition()

		canvasArea := container.New(
			&fixedSizeLayout{size: fyne.NewSize(520, 210)},
			hostAnchorLabel,
			draggablePC1,
			draggablePC2,
		)

		newBorderLabel := func(text string) *widget.Label {
			l := widget.NewLabel(text)
			l.TextStyle = fyne.TextStyle{Monospace: true, Bold: true}
			return l
		}

		topBox := container.NewVBox(
			newBorderLabel("╔═ MONITORS ARRANGEMENT ══════════════════════════╗"),
			container.NewCenter(canvasArea),
			container.NewCenter(transitionLabel),
			newBorderLabel("╚═════════════════════════════════════════════════╝"),
		)

		// -----------------------------------------------------------------------------
		// 2. BOTTOM-LEFT BOX: Network Config (IP and Port)
		// -----------------------------------------------------------------------------
		defaultIP := "0.0.0.0"
		defaultPort := "65432"

		if cfg.Role == config.RoleServer {
			defaultIP = "0.0.0.0"
			if strings.HasPrefix(cfg.Address, ":") {
				defaultPort = strings.TrimPrefix(cfg.Address, ":")
			} else if strings.Contains(cfg.Address, ":") {
				h, p, _ := net.SplitHostPort(cfg.Address)
				if h != "" {
					defaultIP = h
				}
				if p != "" {
					defaultPort = p
				}
			}
		} else {
			defaultIP = "192.168.1.100"
			if strings.Contains(cfg.Address, ":") && !strings.HasPrefix(cfg.Address, ":") {
				h, p, _ := net.SplitHostPort(cfg.Address)
				if h != "" {
					defaultIP = h
				}
				if p != "" {
					defaultPort = p
				}
			}
		}

		ipLabel := widget.NewLabel("IP:  ")
		ipLabel.TextStyle = fyne.TextStyle{Monospace: true, Bold: true}
		ipEntry := widget.NewEntry()
		ipEntry.SetText(defaultIP)
		ipEntry.TextStyle = fyne.TextStyle{Monospace: true}

		portLabel := widget.NewLabel("Port:")
		portLabel.TextStyle = fyne.TextStyle{Monospace: true, Bold: true}
		portEntry := widget.NewEntry()
		portEntry.SetText(defaultPort)
		portEntry.TextStyle = fyne.TextStyle{Monospace: true}

		var helpMsg string
		if cfg.Role == config.RoleServer {
			helpMsg = fmt.Sprintf("LAN: %s", lanIP)
		} else {
			helpMsg = "Enter Host LAN IP"
		}
		helpLabel := widget.NewLabel(helpMsg)
		helpLabel.TextStyle = fyne.TextStyle{Monospace: true}

		ipRow := container.NewBorder(nil, nil, ipLabel, nil, ipEntry)
		portRow := container.NewBorder(nil, nil, portLabel, nil, portEntry)

		leftBox := container.NewVBox(
			newBorderLabel("╔═ NETWORK ═══════════════╗"),
			ipRow,
			portRow,
			helpLabel,
			newBorderLabel("╚═════════════════════════╝"),
		)

		// -----------------------------------------------------------------------------
		// 3. BOTTOM-RIGHT BOX: Client Status (PC 1 idle, PC 2 idle)
		// -----------------------------------------------------------------------------
		statusContent := widget.NewLabel(
			"  PC 1     ● idle\n" +
				"  PC 2     ● idle\n" +
				"  Control: HOST\n" +
				"  Packets: 0 sent",
		)
		statusContent.TextStyle = fyne.TextStyle{Monospace: true}

		rightBox := container.NewVBox(
			newBorderLabel("╔═ CLIENT STATUS ═════════╗"),
			statusContent,
			newBorderLabel("╚═════════════════════════╝"),
		)

		bottomGrid := container.NewGridWithColumns(2, leftBox, rightBox)

		// -----------------------------------------------------------------------------
		// 4. ACTION BUTTON: Start / Stop BorderPass
		// -----------------------------------------------------------------------------
		renderStartBtnText := func(running bool) string {
			txt := "[ ▶ START BORDERPASS ]"
			if running {
				txt = "[ ⏹ STOP BORDERPASS ]"
			}
			return "╔═════════════════════════════════════════════════╗\n" +
				"║" + padCenter(txt, 49) + "║\n" +
				"╚═════════════════════════════════════════════════╝"
		}

		var startBtn *ClickableBox
		startBtn = newClickableBox(renderStartBtnText(false), func() {
			if isRunning {
				stopService()
				startBtn.SetText(renderStartBtnText(false))
				return
			}

			ipVal := strings.TrimSpace(ipEntry.Text)
			portVal := strings.TrimSpace(portEntry.Text)
			if portVal == "" {
				portVal = "65432"
			}

			if cfg.Role == config.RoleServer {
				if ipVal == "" || ipVal == "0.0.0.0" {
					cfg.Address = ":" + portVal
				} else {
					cfg.Address = ipVal + ":" + portVal
				}
			} else {
				if ipVal == "" {
					ipVal = "192.168.1.100"
				}
				cfg.Address = ipVal + ":" + portVal
			}
			_ = cfg.Save()

			ctx, cancel := context.WithCancel(context.Background())
			cancelFunc = cancel
			isRunning = true
			startBtn.SetText(renderStartBtnText(true))
			telemetry.Global.SetRunning(true, cfg.Role)

			if cfg.Role == config.RoleServer {
				go func() {
					_ = server.RunServer(ctx, cfg)
					fyne.Do(func() {
						if !isRunning {
							return
						}
						stopService()
						startBtn.SetText(renderStartBtnText(false))
					})
				}()
			} else {
				go func() {
					_ = client.RunClient(ctx, cfg)
					fyne.Do(func() {
						if !isRunning {
							return
						}
						stopService()
						startBtn.SetText(renderStartBtnText(false))
					})
				}()
			}
		})

		// Real-time telemetry status polling ticker
		tickerStop = make(chan struct{})
		go func(stopChan chan struct{}) {
			ticker := time.NewTicker(250 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-stopChan:
					return
				case <-ticker.C:
					snap := telemetry.Global.Get()
					fyne.Do(func() {
						var pc1Stat, pc2Stat string
						if snap.IsRunning {
							if snap.Status == "CONNECTED" {
								pc1Stat = "● connected"
							} else {
								pc1Stat = "● running"
							}
							pc2Stat = "● idle"
						} else {
							pc1Stat = "● idle"
							pc2Stat = "● idle"
						}

						ctrl := snap.ActiveControl
						if ctrl == "" {
							ctrl = "HOST"
						}

						newStatusText := fmt.Sprintf(
							"  PC 1     %s\n"+
								"  PC 2     %s\n"+
								"  Control: %s\n"+
								"  Packets: %d sent",
							pc1Stat, pc2Stat, ctrl, snap.PacketsSent,
						)
						statusContent.SetText(newStatusText)

						if snap.IsRunning {
							if snap.Status == "CONNECTED" {
								draggablePC1.SetStatus("● connected")
							} else {
								draggablePC1.SetStatus("● active")
							}
						} else {
							draggablePC1.SetStatus("● idle")
						}
					})
				}
			}
		}(tickerStop)

		mainBox := container.NewVBox(
			headerBar,
			topBox,
			bottomGrid,
			container.NewCenter(startBtn),
		)

		w.SetContent(container.NewCenter(mainBox))
	}

	// Initial screen selection (defaults to Screen 1)
	switch os.Getenv("BORDERPASS_SCREEN") {
	case "2":
		showScreen2()
	case "3":
		showScreen3()
	default:
		showScreen1()
	}

	w.SetOnClosed(func() {
		stopService()
		if tickerStop != nil {
			close(tickerStop)
			tickerStop = nil
		}
	})

	w.ShowAndRun()
	return nil
}
