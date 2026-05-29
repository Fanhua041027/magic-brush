package main

import (
	application "ai-assistant/app"
	"ai-assistant/pkg/common"
	"embed"
	"os"
	"runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Windows 专用环境变量
	if runtime.GOOS == "windows" {
		os.Setenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGS", "--disable-gpu")
	}

	app := application.NewApp()
	err := wails.Run(&options.App{
		Title:     "",
		Width:     1024,
		Height:    768,
		MinWidth:  840,
		MinHeight: 700,
		MaxWidth:  1024,
		MaxHeight: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		AlwaysOnTop:      true,
		OnStartup:        app.Startup,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			BackdropType:         windows.None,
			WebviewBrowserPath:   "",
			Theme:                windows.SystemDefault,
		},
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarHidden(),
			WebviewIsTransparent: true,
			WindowIsTranslucent:  false,
			About: &mac.AboutInfo{
				Title:   common.AppName,
				Message: "",
			},
		},
		OnShutdown: app.OnShutdown,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "6c201f7f-17c8-4029-8000-2f9035fea412",
			OnSecondInstanceLaunch: func(secondInstanceData options.SecondInstanceData) {
				app.Show()
			},
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
