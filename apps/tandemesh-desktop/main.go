package main

import (
	"log"
	"os"
	"runtime"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func main() {
	lock, err := acquireSingleInstance()
	if err != nil {
		return
	}
	defer lock.Close()
	inheritLoginShellPath()
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
					log.Fatalf("load local token: %v", tokenErr)
				}
				localToken = token
				plane, err = startLocalPlane(localToken)
				if err != nil {
					log.Fatalf("start local plane: %v", err)
				}
				defer plane.Stop()
				planeURL = plane.URL
				if os.Getenv("TANDEMESH_LAN") == "1" {
					log.Printf("LAN access enabled. Open http://<this-host>:%d/?token=%s once to set the access cookie.", plane.Port, localToken)
				}
				agentd := startAgentd(planeURL, dataDirPath(), localToken)
				defer agentd.Stop()
			}
		}
		if planeURL == "" {
			planeURL = "http://127.0.0.1:8080"
		}
		var err error
		server, webURL, err = startDesktopServer(planeURL, desktopWebFS(), localToken)
		if err != nil {
			log.Fatal(err)
		}
		defer server.server.Close()
	}

	app := application.New(application.Options{
		Name:        "Tandemesh",
		Description: "Local-first agent execution plane",
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

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
	go func() {
		time.Sleep(10 * time.Second)
		checkForUpdates(app)
	}()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
