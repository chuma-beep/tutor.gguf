//go:build desktop

package main

import (
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	tutor "github.com/chuma-beep/tutor.gguf"
	"github.com/chuma-beep/tutor.gguf/internal/desktop"
)

func main() {
	app := desktop.NewApp()

	err := wails.Run(&options.App{
		Title:  "Tutor.gguf — On-device Math Tutor",
		Width:  1120,
		Height: 780,
		MinWidth:  960,
		MinHeight: 640,
		AssetServer: &assetserver.Options{
			Assets: tutor.FrontendDist,
		},
		BackgroundColour: &options.RGBA{R: 0x1C, G: 0x22, B: 0x2E, A: 1}, // #1C222E chalkboard slate
		OnStartup:        app.Startup,
		OnShutdown:       app.Shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
