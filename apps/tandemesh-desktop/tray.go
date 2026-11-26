package main

import (
	_ "embed"
	"log"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// The brand fin. macOS takes a black template image and tints it for the
// menu bar; other platforms show the full-colour app tile.
var (
	//go:embed trayicons/tray-template.png
	trayTemplateIcon []byte
	//go:embed trayicons/tray.png
	trayIcon []byte
)

func installTray(app *application.App, window application.Window) {
	tray := app.SystemTray.New()
	if runtime.GOOS == "darwin" {
		tray.SetTemplateIcon(trayTemplateIcon)
	} else {
		tray.SetIcon(trayIcon)
	}
	tray.SetTooltip("Tandemesh")

	show := func() {
		window.Show()
		window.Focus()
	}
	// Closing the window keeps the app running in the tray.
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})
	// Clicking the Dock icon brings a hidden window back.
	app.Event.OnApplicationEvent(events.Mac.ApplicationShouldHandleReopen, func(*application.ApplicationEvent) {
		show()
	})

	menu := app.NewMenu()
	menu.Add("Show Tandemesh").OnClick(func(*application.Context) { show() })
	menu.Add("Check for updates").OnClick(func(*application.Context) {
		checkForUpdates(app)
	})
	menu.AddCheckbox("Start at login", isAutostartEnabled()).OnClick(func(ctx *application.Context) {
		enabled := ctx.ClickedMenuItem().Checked()
		if err := setAutostart(enabled); err != nil {
			log.Printf("autostart: %v", err)
			ctx.ClickedMenuItem().SetChecked(!enabled)
		}
	})
	menu.AddSeparator()
	menu.Add("Quit").OnClick(func(*application.Context) {
		app.Quit()
	})
	// No attached window: that is meant for small tray popovers and would
	// move the main window under the menu bar. A click opens the menu.
	tray.SetMenu(menu)
}
