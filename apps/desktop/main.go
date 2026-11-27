package main

import (
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func main() {
	lock, err := acquireSingleInstance()
	if err != nil {
		return
	}
	defer lock.Close()
	inheritLoginShellPath()

	// Cocoa ends the process from its own terminate path, so deferred calls in
	// main never run on Cmd+Q. Every exit path goes through shutdown instead,
	// or plane and agentd outlive the app.
	var cleanups []func()
	var once sync.Once
	shutdown := func() {
		once.Do(func() {
			for i := len(cleanups) - 1; i >= 0; i-- {
				cleanups[i]()
			}
		})
	}
	defer shutdown()
	fatal := func(format string, args ...any) {
		shutdown()
		log.Fatalf(format, args...)
	}

	webURL := os.Getenv("TANDEMESH_WEB_URL")
	var server *desktopServer
	var plane *localPlane
	localToken := ""
	if webURL == "" {
		planeURL := os.Getenv("TANDEMESH_PLANE_URL")
		if planeURL == "" && os.Getenv("TANDEMESH_MANAGE_PLANE") != "0" {
			if binary := findPlaneBinary(); binary != "" {
				var err error
				token, tokenErr := loadOrCreateToken(dataDirPath())
				if tokenErr != nil {
					fatal("load local token: %v", tokenErr)
				}
				localToken = token
				plane, err = startLocalPlane(localToken)
				if err != nil {
					fatal("start local plane: %v", err)
				}
				cleanups = append(cleanups, plane.Stop)
				planeURL = plane.URL
				if os.Getenv("TANDEMESH_LAN") == "1" {
					log.Printf("LAN access enabled. Open http://<this-host>:%d/?token=%s once to set the access cookie.", plane.Port, localToken)
				}
				agentd := startAgentd(planeURL, dataDirPath(), localToken)
				cleanups = append(cleanups, agentd.Stop)
			}
		}
		if planeURL == "" {
			planeURL = "http://127.0.0.1:8080"
		}
		var err error
		server, webURL, err = startDesktopServer(planeURL, desktopWebFS(), localToken)
		if err != nil {
			fatal("start desktop server: %v", err)
		}
		cleanups = append(cleanups, func() { _ = server.server.Close() })
	}

	app := application.New(application.Options{
		Name:        "Tandemesh",
		Description: "Local-first agent execution plane",
		// The app lives in the tray; hiding or closing the window must not
		// quit it. Quit comes from the tray menu or Cmd+Q.
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
		OnShutdown: shutdown,
	})
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signals
		app.Quit()
	}()

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "Tandemesh",
		Width:     1440,
		Height:    900,
		MinWidth:  960,
		MinHeight: 640,
		// Windows draws its own caption buttons in the web topbar; macOS keeps
		// the native traffic lights on the left over a transparent title bar.
		Frameless: runtime.GOOS != "darwin",
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBar{
				AppearsTransparent:   true,
				HideTitle:            true,
				FullSizeContent:      true,
				UseToolbar:           true,
				HideToolbarSeparator: true,
				ToolbarStyle:         application.MacToolbarStyleUnifiedCompact,
			},
		},
		Windows: application.WindowsWindow{
			NonClientRegionSupport: true,
		},
		BackgroundColour: application.NewRGB(6, 7, 15),
		URL:              webURL,
	})
	installTray(app, window)
	go runUpdateChecks(app)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
