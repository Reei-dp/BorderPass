package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Roles
const (
	RoleServer = "server"
	RoleClient = "client"
)

// Position of the secondary screen relative to this computer
const (
	PosRight  = "right"
	PosLeft   = "left"
	PosTop    = "top"
	PosBottom = "bottom"
)

// DefaultConfigFileName is the standard configuration file name
const DefaultConfigFileName = "borderpass.json"

// Config represents application settings
type Config struct {
	Role            string `json:"role"`             // "server" or "client"
	Position        string `json:"position"`         // "right", "left", "top", "bottom"
	Address         string `json:"address"`          // Port for server (e.g. ":65432") or IP:port for client
	ScreenWidth     int    `json:"screen_width"`     // Screen width in pixels
	ScreenHeight    int    `json:"screen_height"`    // Screen height in pixels
	EdgeThreshold   int    `json:"edge_threshold"`   // Edge resistance threshold in pixels
	MouseDevice     string `json:"mouse_device"`     // Mouse device path (Linux) or "auto"
	KeyboardDevice  string `json:"keyboard_device"`  // Keyboard device path (Linux) or "auto"
	EnableClipboard bool   `json:"enable_clipboard"` // Enable clipboard synchronization
}

// DefaultConfig returns reasonable default configuration values
func DefaultConfig() Config {
	return Config{
		Role:            RoleServer,
		Position:        PosRight,
		Address:         ":65432",
		ScreenWidth:     1920,
		ScreenHeight:    1080,
		EdgeThreshold:   8,
		MouseDevice:     "auto",
		KeyboardDevice:  "auto",
		EnableClipboard: true,
	}
}

// ConfigFilePath returns the path to the configuration file (executable dir or working dir)
func ConfigFilePath() string {
	exePath, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exePath)
		candidate := filepath.Join(dir, DefaultConfigFileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return DefaultConfigFileName
}

// Exists checks if a configuration file already exists
func Exists() bool {
	_, err := os.Stat(ConfigFilePath())
	return err == nil
}

// Load reads and parses the configuration file
func Load() (*Config, error) {
	data, err := os.ReadFile(ConfigFilePath())
	if err != nil {
		return nil, err
	}
	cfg := DefaultConfig()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save writes the configuration to disk
func (c *Config) Save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigFilePath(), data, 0644)
}

// InteractiveSetup launches a console interactive configuration wizard
func InteractiveSetup(current *Config) (*Config, error) {
	reader := bufio.NewReader(os.Stdin)
	cfg := DefaultConfig()
	if current != nil {
		cfg = *current
	}

	fmt.Println("\n=======================================================")
	fmt.Println("         🚀 BorderPass — Quick Setup Wizard            ")
	fmt.Println("=======================================================")

	// 1. Role selection
	fmt.Println("\n[1] Select operation mode for this computer:")
	fmt.Println("  1) SERVER (Host)   — share keyboard & mouse from this device")
	fmt.Println("  2) CLIENT (Client) — receive control on this device")
	defaultRoleIdx := "1"
	if cfg.Role == RoleClient {
		defaultRoleIdx = "2"
	}
	fmt.Printf("Select option [default %s]: ", defaultRoleIdx)
	roleInput := readLine(reader)
	if roleInput == "" {
		roleInput = defaultRoleIdx
	}
	if roleInput == "2" {
		cfg.Role = RoleClient
	} else {
		cfg.Role = RoleServer
	}

	// 2. Relative position of the secondary display
	fmt.Println("\n[2] Where is the secondary computer physically positioned?")
	fmt.Println("  1) RIGHT  — switch when cursor moves past the right screen border")
	fmt.Println("  2) LEFT   — switch when cursor moves past the left screen border")
	fmt.Println("  3) TOP    — switch when cursor moves past the top screen border")
	fmt.Println("  4) BOTTOM — switch when cursor moves past the bottom screen border")
	defaultPosIdx := "1"
	switch cfg.Position {
	case PosLeft:
		defaultPosIdx = "2"
	case PosTop:
		defaultPosIdx = "3"
	case PosBottom:
		defaultPosIdx = "4"
	}
	fmt.Printf("Select option [default %s]: ", defaultPosIdx)
	posInput := readLine(reader)
	if posInput == "" {
		posInput = defaultPosIdx
	}
	switch posInput {
	case "2":
		cfg.Position = PosLeft
	case "3":
		cfg.Position = PosTop
	case "4":
		cfg.Position = PosBottom
	default:
		cfg.Position = PosRight
	}

	// 3. Network configuration
	if cfg.Role == RoleServer {
		defaultPort := "65432"
		if strings.HasPrefix(cfg.Address, ":") {
			defaultPort = strings.TrimPrefix(cfg.Address, ":")
		}
		fmt.Printf("\n[3] Port to listen on [default %s]: ", defaultPort)
		portInput := readLine(reader)
		if portInput == "" {
			portInput = defaultPort
		}
		if !strings.HasPrefix(portInput, ":") {
			portInput = ":" + portInput
		}
		cfg.Address = portInput
	} else {
		defaultAddr := "192.168.1.100:65432"
		if cfg.Address != "" && !strings.HasPrefix(cfg.Address, ":") {
			defaultAddr = cfg.Address
		}
		fmt.Printf("\n[3] Server IP address and port [default %s]: ", defaultAddr)
		addrInput := readLine(reader)
		if addrInput == "" {
			addrInput = defaultAddr
		}
		if !strings.Contains(addrInput, ":") {
			addrInput = addrInput + ":65432"
		}
		cfg.Address = addrInput
	}

	// 4. Screen resolution
	fmt.Printf("\n[4] Screen resolution for this display (Width x Height) [default %dx%d]: ", cfg.ScreenWidth, cfg.ScreenHeight)
	resInput := readLine(reader)
	if resInput != "" {
		resInput = strings.ReplaceAll(resInput, " ", "")
		parts := strings.Split(strings.ToLower(resInput), "x")
		if len(parts) == 2 {
			w, err1 := strconv.Atoi(parts[0])
			h, err2 := strconv.Atoi(parts[1])
			if err1 == nil && err2 == nil && w > 0 && h > 0 {
				cfg.ScreenWidth = w
				cfg.ScreenHeight = h
			}
		}
	}

	// 5. Shared clipboard
	fmt.Printf("\n[5] Enable shared clipboard (Ctrl+C / Ctrl+V)? (Y/n) [default Y]: ")
	clipInput := strings.ToLower(readLine(reader))
	if clipInput == "n" || clipInput == "no" {
		cfg.EnableClipboard = false
	} else {
		cfg.EnableClipboard = true
	}

	// Save configuration
	if err := cfg.Save(); err != nil {
		fmt.Printf("⚠️  Failed to save config: %v\n", err)
	} else {
		fmt.Printf("✅ Configuration saved to %s\n", ConfigFilePath())
	}

	return &cfg, nil
}

func readLine(r *bufio.Reader) string {
	line, _ := r.ReadString('\n')
	return strings.TrimSpace(line)
}
