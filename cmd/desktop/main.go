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
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: tutor.FrontendDist,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
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
