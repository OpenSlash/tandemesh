package main

import (
	"log"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/icons"
)

func installTray(app *application.App, window application.Window) {
	tray := app.SystemTray.New()
	tray.SetLabel("Tandemesh")
	if runtime.GOOS == "darwin" {
		tray.SetTemplateIcon(icons.SystrayMacTemplate)
	}
	tray.SetTooltip("Tandemesh")

	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})

	menu := app.NewMenu()
	menu.Add("Check for updates").OnClick(func(*application.Context) {
		checkForUpdates(app)
	})
	menu.Add("Show Tandemesh").OnClick(func(*application.Context) {
		window.Show()
		window.Focus()
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
	tray.SetMenu(menu)
	tray.AttachWindow(window).WindowOffset(2)
}
