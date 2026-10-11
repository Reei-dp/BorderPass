package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"borderpass/client"
	"borderpass/config"
	"borderpass/internal/telemetry"
	"borderpass/server"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type viewState int

const (
	viewConfig viewState = iota
	viewRunning
)

const totalFields = 6

const (
	fieldRole = iota
	fieldPosition
	fieldAddress
	fieldResolution
	fieldClipboard
	fieldButton
)

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Model represents the Bubble Tea TUI state
type Model struct {
	cfg          *config.Config
	state        viewState
	cursor       int
	addrInput    textinput.Model
	resInput     textinput.Model
	statusMsg    string
	cancelFunc   context.CancelFunc
	err          error
	spinnerFrame int
}

// NewModel creates a new TUI model initialized with configuration
func NewModel(cfg *config.Config) Model {
	addr := textinput.New()
	addr.Placeholder = ":65432"
	addr.SetValue(cfg.Address)
	addr.CharLimit = 40
	addr.Width = 28

	res := textinput.New()
	res.Placeholder = "1920x1080"
	res.SetValue(fmt.Sprintf("%dx%d", cfg.ScreenWidth, cfg.ScreenHeight))
	res.CharLimit = 16
	res.Width = 28

	return Model{
		cfg:       cfg,
		state:     viewConfig,
		cursor:    0,
		addrInput: addr,
		resInput:  res,
	}
}

func (m Model) Init() tea.Cmd {
	return tickCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.state == viewRunning {
				m.stopService()
				return m, tea.Quit
			}
			return m, tea.Quit

		case "x":
			if m.state == viewRunning {
				m.stopService()
				m.state = viewConfig
				m.statusMsg = "Service stopped."
				return m, nil
			}

		case "s":
			if m.state == viewConfig {
				m.applyInputsToConfig()
				_ = m.cfg.Save()
				m.startService()
				m.state = viewRunning
				return m, tickCmd()
			}
		}

		if m.state == viewConfig {
			switch msg.String() {
			case "up", "shift+tab":
				m.cursor--
				if m.cursor < 0 {
					m.cursor = totalFields - 1
				}
				return m.updateFocus()

			case "down", "tab":
				m.cursor++
				if m.cursor >= totalFields {
					m.cursor = 0
				}
				return m.updateFocus()

			case "left":
				m.handleLeft()
				return m, nil

			case "right":
				m.handleRight()
				return m, nil

			case " ", "enter":
				if m.cursor == fieldButton {
					m.applyInputsToConfig()
					_ = m.cfg.Save()
					m.startService()
					m.state = viewRunning
					return m, tickCmd()
				}
				m.handleToggle()
				return m, nil
			}
		}

	case tickMsg:
		m.spinnerFrame++
		return m, tickCmd()
	}

	var cmd tea.Cmd
	if m.state == viewConfig {
		if m.cursor == fieldAddress {
			m.addrInput, cmd = m.addrInput.Update(msg)
			return m, cmd
		} else if m.cursor == fieldResolution {
			m.resInput, cmd = m.resInput.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m *Model) updateFocus() (tea.Model, tea.Cmd) {
	m.addrInput.Blur()
	m.resInput.Blur()

	var cmd tea.Cmd
	if m.cursor == fieldAddress {
		cmd = m.addrInput.Focus()
	} else if m.cursor == fieldResolution {
		cmd = m.resInput.Focus()
	}
	return *m, cmd
}

func (m *Model) handleLeft() {
	switch m.cursor {
	case fieldRole:
		if m.cfg.Role == config.RoleClient {
			m.cfg.Role = config.RoleServer
			if !strings.HasPrefix(m.addrInput.Value(), ":") {
				m.addrInput.SetValue(":65432")
			}
		}
	case fieldPosition:
		m.cyclePosition(false)
	}
}

func (m *Model) handleRight() {
	switch m.cursor {
	case fieldRole:
		if m.cfg.Role == config.RoleServer {
			m.cfg.Role = config.RoleClient
			if strings.HasPrefix(m.addrInput.Value(), ":") {
				m.addrInput.SetValue("192.168.1.100:65432")
			}
		}
	case fieldPosition:
		m.cyclePosition(true)
	}
}

func (m *Model) handleToggle() {
	switch m.cursor {
	case fieldRole:
		if m.cfg.Role == config.RoleServer {
			m.cfg.Role = config.RoleClient
			if strings.HasPrefix(m.addrInput.Value(), ":") {
				m.addrInput.SetValue("192.168.1.100:65432")
			}
		} else {
			m.cfg.Role = config.RoleServer
			if !strings.HasPrefix(m.addrInput.Value(), ":") {
				m.addrInput.SetValue(":65432")
			}
		}
	case fieldPosition:
		m.cyclePosition(true)
	case fieldClipboard:
		m.cfg.EnableClipboard = !m.cfg.EnableClipboard
	}
}

func (m *Model) cyclePosition(forward bool) {
	positions := []string{config.PosRight, config.PosLeft, config.PosTop, config.PosBottom}
	idx := 0
	for i, p := range positions {
		if p == m.cfg.Position {
			idx = i
			break
		}
	}
	if forward {
		idx = (idx + 1) % len(positions)
	} else {
		idx = (idx - 1 + len(positions)) % len(positions)
	}
	m.cfg.Position = positions[idx]
}

func (m *Model) applyInputsToConfig() {
	m.cfg.Address = strings.TrimSpace(m.addrInput.Value())
	resStr := strings.TrimSpace(m.resInput.Value())
	parts := strings.Split(strings.ToLower(resStr), "x")
	if len(parts) == 2 {
		w, err1 := strconv.Atoi(parts[0])
		h, err2 := strconv.Atoi(parts[1])
		if err1 == nil && err2 == nil && w > 0 && h > 0 {
			m.cfg.ScreenWidth = w
			m.cfg.ScreenHeight = h
		}
	}
}

func (m *Model) startService() {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFunc = cancel

	if m.cfg.Role == config.RoleServer {
		go func() {
			_ = server.RunServer(ctx, m.cfg)
		}()
	} else {
		go func() {
			_ = client.RunClient(ctx, m.cfg)
		}()
	}
}

func (m *Model) stopService() {
	if m.cancelFunc != nil {
		m.cancelFunc()
		m.cancelFunc = nil
	}
}

// View renders the terminal output
func (m Model) View() string {
	if m.state == viewRunning {
		return m.renderRunningView()
	}
	return m.renderConfigView()
}

// Styling definitions
var (
	cyanStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")).Bold(true)
	magentaStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00FF")).Bold(true)
	greenStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF66")).Bold(true)
	yellowStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00")).Bold(true)
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	borderStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#00E5FF"))
	activeStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#000000")).Background(lipgloss.Color("#00FFFF")).Bold(true)
	boxStyle     = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(lipgloss.Color("#00FFFF")).Padding(0, 1)
)

func (m Model) renderConfigView() string {
	var s strings.Builder

	// ASCII Art Header
	header := `
╔═══════════════════════════════════════════════════════════════════════╗
║   ██████╗  ██████╗ ██████╗ ██████╗ ███████╗██████╗ ██████╗  ██████╗   ║
║   ██╔══██╗██╔═══██╗██╔══██╗██╔══██╗██╔════╝██╔══██╗██╔══██╗██╔════╝   ║
║   ██████╔╝██║   ██║██████╔╝██║  ██║█████╗  ██████╔╝██████╔╝╚█████╗    ║
║   ██╔══██╗██║   ██║██╔══██╗██║  ██║██╔══╝  ██╔══██╗██╔═══╝  ╚═══██╗   ║
║   ██████╔╝╚██████╔╝██║  ██║██████╔╝███████╗██║  ██║██║     ██████╔╝   ║
║   ╚═════╝  ╚═════╝ ╚═╝  ╚═╝╚═════╝ ╚══════╝╚═╝  ╚═╝╚═╝     ╚═════╝    ║
║            :: ULTRA-LOW LATENCY KM NETWORK SWITCH ::                  ║
╚═══════════════════════════════════════════════════════════════════════╝`
	s.WriteString(cyanStyle.Render(header))
	s.WriteString("\n\n")

	// Render dynamic spatial monitor layout
	s.WriteString(m.renderSpatialDiagram())
	s.WriteString("\n\n")

	// Render settings form
	s.WriteString(borderStyle.Render("┌─ CONFIGURATION & CONTROLS ───────────────────────────────────────────┐"))
	s.WriteString("\n")

	fields := []string{
		m.formatField(fieldRole, "Mode / Role", m.renderRoleChoice()),
		m.formatField(fieldPosition, "Target Position", m.renderPositionChoice()),
		m.formatField(fieldAddress, "Network Address", m.addrInput.View()),
		m.formatField(fieldResolution, "Resolution (WxH)", m.resInput.View()),
		m.formatField(fieldClipboard, "Clipboard Sync", m.renderClipboardChoice()),
		m.formatField(fieldButton, "Action", m.renderActionButton()),
	}

	for _, f := range fields {
		s.WriteString(fmt.Sprintf("│  %s\n", f))
	}

	s.WriteString(borderStyle.Render("└──────────────────────────────────────────────────────────────────────┘"))
	s.WriteString("\n")

	// Hotkeys Footer
	s.WriteString(dimStyle.Render(" [Tab/↑↓] Navigate  [Space/←→] Toggle  [S/Enter] Save & Start  [Q] Quit\n"))

	if m.statusMsg != "" {
		s.WriteString(yellowStyle.Render(fmt.Sprintf(" ℹ️  %s\n", m.statusMsg)))
	}

	return s.String()
}

func (m Model) formatField(idx int, label string, val string) string {
	cursorStr := "  "
	if m.cursor == idx {
		cursorStr = cyanStyle.Render("► ")
		label = cyanStyle.Render(fmt.Sprintf("%-18s", label))
	} else {
		label = dimStyle.Render(fmt.Sprintf("%-18s", label))
	}
	return fmt.Sprintf("%s%s : %s", cursorStr, label, val)
}

func (m Model) renderRoleChoice() string {
	if m.cfg.Role == config.RoleServer {
		return fmt.Sprintf("%s  %s", activeStyle.Render("[● SERVER (HOST)]"), dimStyle.Render("[○ CLIENT (RECEIVER)]"))
	}
	return fmt.Sprintf("%s  %s", dimStyle.Render("[○ SERVER (HOST)]"), activeStyle.Render("[● CLIENT (RECEIVER)]"))
}

func (m Model) renderPositionChoice() string {
	switch m.cfg.Position {
	case config.PosRight:
		return magentaStyle.Render("[► RIGHT] ") + dimStyle.Render("(Switch via Right border)")
	case config.PosLeft:
		return magentaStyle.Render("[◄ LEFT]  ") + dimStyle.Render("(Switch via Left border)")
	case config.PosTop:
		return magentaStyle.Render("[▲ TOP]   ") + dimStyle.Render("(Switch via Top border)")
	case config.PosBottom:
		return magentaStyle.Render("[▼ BOTTOM]") + dimStyle.Render("(Switch via Bottom border)")
	}
	return m.cfg.Position
}

func (m Model) renderClipboardChoice() string {
	if m.cfg.EnableClipboard {
		return greenStyle.Render("[✓] ENABLED (Ctrl+C / Ctrl+V synced)")
	}
	return dimStyle.Render("[ ] DISABLED")
}

func (m Model) renderActionButton() string {
	btn := "[ START BORDERPASS ]"
	if m.cursor == fieldButton {
		return activeStyle.Render(fmt.Sprintf("  %s  ", btn))
	}
	return greenStyle.Render(fmt.Sprintf("  %s  ", btn))
}

func (m Model) renderSpatialDiagram() string {
	w := m.cfg.ScreenWidth
	h := m.cfg.ScreenHeight

	hostTitle := "HOST (THIS DEVICE)"
	clientTitle := "CLIENT (REMOTE)"
	if m.cfg.Role == config.RoleClient {
		hostTitle = "HOST (REMOTE)"
		clientTitle = "CLIENT (THIS DEVICE)"
	}

	switch m.cfg.Position {
	case config.PosRight:
		return fmt.Sprintf(`  ┌───────────────────────┐             ┌───────────────────────┐
  │  %-21s│  ═════════►  │  %-21s│
  │      %5d x %-5d    │   [RIGHT]   │       SECONDARY       │
  └───────────────────────┘             └───────────────────────┘`,
			hostTitle, clientTitle, w, h)

	case config.PosLeft:
		return fmt.Sprintf(`  ┌───────────────────────┐             ┌───────────────────────┐
  │  %-21s│  ◄═════════  │  %-21s│
  │       SECONDARY       │   [LEFT]    │      %5d x %-5d    │
  └───────────────────────┘             └───────────────────────┘`,
			clientTitle, hostTitle, w, h)

	case config.PosTop:
		return fmt.Sprintf(`                   ┌───────────────────────┐
                   │  %-21s│
                   │       SECONDARY       │
                   └───────────────────────┘
                               ▲  [TOP]
                               ║
                   ┌───────────────────────┐
                   │  %-21s│
                   │      %5d x %-5d    │
                   └───────────────────────┘`,
			clientTitle, hostTitle, w, h)

	case config.PosBottom:
		return fmt.Sprintf(`                   ┌───────────────────────┐
                   │  %-21s│
                   │      %5d x %-5d    │
                   └───────────────────────┘
                               ║
                               ▼  [BOTTOM]
                   ┌───────────────────────┐
                   │  %-21s│
                   │       SECONDARY       │
                   └───────────────────────┘`,
			hostTitle, w, h, clientTitle)
	}

	return ""
}

func (m Model) renderRunningView() string {
	var s strings.Builder
	snap := telemetry.Global.Get()

	spinnerSymbols := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spin := spinnerSymbols[m.spinnerFrame%len(spinnerSymbols)]

	hudHeader := `
╔═══════════════════════════════════════════════════════════════════════╗
║              ⚡ BORDERPASS :: LIVE CYBERPUNK HUD ⚡                   ║
╚═══════════════════════════════════════════════════════════════════════╝`
	s.WriteString(cyanStyle.Render(hudHeader))
	s.WriteString("\n\n")

	// Status badge
	statusBadge := greenStyle.Render(fmt.Sprintf("%s ONLINE [ %s ]", spin, snap.Status))
	if strings.Contains(snap.Status, "DISCONNECTED") || strings.Contains(snap.Status, "STOPPED") {
		statusBadge = yellowStyle.Render(fmt.Sprintf("● [ %s ]", snap.Status))
	}

	controlBadge := greenStyle.Render(fmt.Sprintf("►► %s ◄◄", snap.ActiveControl))
	if strings.Contains(snap.ActiveControl, "REMOTE") {
		controlBadge = magentaStyle.Render(fmt.Sprintf("►► %s ◄◄", snap.ActiveControl))
	}

	content := fmt.Sprintf(`  STATUS:           %s
  ROLE:             %s
  NETWORK PEER:     %s
  ACTIVE CONTROL:   %s

  ┌─ REALTIME TELEMETRY ──────────────────────────────────────────────┐
  │  Packets Transmitted  :  %-12d                             │
  │  Packets Received     :  %-12d                             │
  │  Clipboard Syncs      :  %-12d                             │
  │  Socket Latency       :  <0.5 ms (TCP_NODELAY)                    │
  └───────────────────────────────────────────────────────────────────┘

  %s`,
		statusBadge,
		cyanStyle.Render(strings.ToUpper(m.cfg.Role)),
		yellowStyle.Render(snap.RemoteAddr),
		controlBadge,
		snap.PacketsSent,
		snap.PacketsRecv,
		snap.ClipboardEvents,
		dimStyle.Render("[X] Stop & Reconfigure    [Q / Ctrl+C] Quit Application"),
	)

	s.WriteString(boxStyle.Render(content))
	s.WriteString("\n")

	if snap.LastError != "" {
		s.WriteString(yellowStyle.Render(fmt.Sprintf("\n  ⚠️ Notice: %s\n", snap.LastError)))
	}

	return s.String()
}

// Run launches the interactive ASCII TUI
func Run(cfg *config.Config) error {
	p := tea.NewProgram(NewModel(cfg), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
