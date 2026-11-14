//go:build wails

// Command tandemesh-desktop is the Wails v3 shell.
//
// It is excluded from the default build because Wails v3 is still alpha and
// pulls a large dependency tree. To build it:
//
//	go get github.com/wailsapp/wails/v3@latest
//	mkdir -p cmd/tandemesh-desktop/frontend
//	cp -r web/dist cmd/tandemesh-desktop/frontend/dist
//	go build -tags wails -o tandemesh-desktop ./cmd/tandemesh-desktop
package main

import (
	"context"
	"embed"
	"log/slog"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/OpenSlash/tandemesh/internal/app"
	"github.com/OpenSlash/tandemesh/internal/config"
	"github.com/OpenSlash/tandemesh/internal/desktop"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	core, err := app.New(ctx, config.Load(), log)
	if err != nil {
		log.Error("start core", "err", err)
		os.Exit(1)
	}
	defer core.Close()

	service := desktop.New(core.Store, core.Registry, "0.1.0-dev")
	wailsApp := application.New(application.Options{
		Name: "Orca",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Services: []application.Service{
			application.NewService(service),
		},
	})
	wailsApp.NewWebviewWindow()
	wailsApp.Run()
}
