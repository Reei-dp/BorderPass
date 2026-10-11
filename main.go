package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"os"

	"borderpass/client"
	"borderpass/config"
	"borderpass/internal/tui"
	"borderpass/server"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	tuiFlag := flag.Bool("tui", false, "Run in terminal ASCII TUI mode")
	cliFlag := flag.Bool("cli", false, "Run in headless CLI mode without window")
	serverFlag := flag.Bool("server", false, "Force start in SERVER mode")
	clientFlag := flag.Bool("client", false, "Force start in CLIENT mode")
	addrFlag := flag.String("addr", "", "Override address")
	posFlag := flag.String("pos", "", "Override screen position")
	flag.Parse()

	var cfg *config.Config
	if config.Exists() {
		var err error
		cfg, err = config.Load()
		if err != nil {
			c := config.DefaultConfig()
			cfg = &c
		}
	} else {
		c := config.DefaultConfig()
		cfg = &c
	}

	if *serverFlag {
		cfg.Role = config.RoleServer
	} else if *clientFlag {
		cfg.Role = config.RoleClient
	}
	if *addrFlag != "" {
		cfg.Address = *addrFlag
	}
	if *posFlag != "" {
		cfg.Position = *posFlag
	}

	if *cliFlag {
		runHeadless(cfg)
		return
	}

	if *tuiFlag {
		if err := tui.Run(cfg); err != nil {
			fmt.Printf("❌ TUI error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Launch Wails Desktop Application!
	app := NewApp()
	app.cfg = cfg

	err := wails.Run(&options.App{
		Title:         "BorderPass",
		Width:         860,
		Height:        660,
		MinWidth:      800,
		MinHeight:     600,
		DisableResize: false,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 13, G: 17, B: 23, A: 255},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Linux: &linux.Options{
			ProgramName: "borderpass",
		},
	})

	if err != nil {
		fmt.Printf("❌ Wails error: %v\n", err)
		os.Exit(1)
	}
}

func runHeadless(cfg *config.Config) {
	ctx := context.Background()
	fmt.Printf("[BorderPass] Running headless: %s mode on %s (Position: %s)\n",
		cfg.Role, cfg.Address, cfg.Position)

	if cfg.Role == config.RoleServer {
		if err := server.RunServer(ctx, cfg); err != nil {
			fmt.Printf("❌ Server error: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := client.RunClient(ctx, cfg); err != nil {
			fmt.Printf("❌ Client error: %v\n", err)
			os.Exit(1)
		}
	}
}
