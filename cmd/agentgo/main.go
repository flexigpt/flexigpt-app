package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	assets "github.com/flexigpt/flexigpt-app/frontend"
	"github.com/flexigpt/flexigpt-app/internal/logrotate"

	// Run registry init.
	_ "github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
)

var Version string

func main() {
	appDisplayTitle := AppTitle + " - " + Version
	aboutMessage := "An AI app platform.\n\nCopyright © 2023 - Present"

	// Allocate stable wrapper instances and preserve the existing data paths.
	app := newApp()

	slogOpts := &slog.HandlerOptions{
		Level: &appSlogLevelVar,
	}

	stdoutLogger := slog.New(slog.NewTextHandler(os.Stdout, slogOpts))

	writer, err := logrotate.New(stdoutLogger, logrotate.Options{
		Directory:            filepath.Join(app.dataBasePath, "logs"),
		MaximumFileSize:      10 * 1024 * 1024,
		MaximumLifetime:      24 * time.Hour,
		FileNameFunc:         logrotate.DefaultFilenameFunc,
		FlushAfterEveryWrite: true,
	})
	if err != nil {
		slog.Error("failed to create log writer", "error", err)
		panic("init failed")
	}
	defer writer.Close()

	slogger := slog.New(slog.NewTextHandler(writer, slogOpts))
	slog.SetDefault(slogger)
	logService := NewLogService(slogger)

	// Also handles initialization panics and an unsuccessful desktop.Run().
	// Registered after writer.Close so managers close before the logger.
	defer app.shutdown(context.Background())

	// Keep hydration, baseline provisioning, and runtime composition ahead
	// of frontend access to the registered services.
	app.initManagers()

	desktop := application.New(application.Options{
		Name:        AppTitle,
		Description: aboutMessage,
		Icon:        assets.Icon,
		Logger:      slogger,
		Services: []application.Service{
			application.NewService(app),
			application.NewService(logService),
			application.NewService(app.settingStoreAPI),
			application.NewService(app.sessionStoreAPI),
			application.NewService(app.modelStoreAPI),
			application.NewService(app.modelAggregateAPI),
			application.NewService(app.toolStoreAPI),
			application.NewService(app.toolRuntimeAPI),
			application.NewService(app.skillStoreAPI),
			application.NewService(app.skillAggregateAPI),
			application.NewService(app.skillRuntimeAPI),
			application.NewService(app.workspaceRuntimeAPI),
			application.NewService(app.workspaceStoreAPI),
			application.NewService(app.mcpStoreAPI),
			application.NewService(app.mcpRuntimeAPI),
			application.NewService(app.mcpAggregateAPI),
			application.NewService(app.completionAPI),
			application.NewService(app.agentStoreAPI),
			application.NewService(app.textStoreAPI),
			application.NewService(app.sessionRuntimeAPI),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets.Assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})
	app.desktop = desktop

	backgroundColour := application.NewRGB(255, 255, 255)
	if runtime.GOOS == "darwin" {
		// Allow the translucent macOS backdrop to show through the webview.
		backgroundColour = application.NewRGBA(255, 255, 255, 0)
	}

	app.window = desktop.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:               "main",
		Title:              appDisplayTitle,
		URL:                "/",
		Width:              1024,
		Height:             768,
		MinWidth:           1024,
		MinHeight:          768,
		Frameless:          true,
		BackgroundColour:   backgroundColour,
		ZoomControlEnabled: true,
		EnableFileDrop:     true,
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBar{
				AppearsTransparent: true,
				Hide:               true,
			},
			Backdrop: application.MacBackdropTranslucent,
		},
		Linux: application.LinuxWindow{
			Icon:             assets.Icon,
			WebviewGpuPolicy: application.WebviewGpuPolicyOnDemand,
		},
	})

	if runtime.GOOS == "darwin" {
		menu := desktop.NewMenu()
		menu.AddRole(application.AppMenu)
		menu.AddRole(application.EditMenu)

		helpMenu := menu.AddSubmenu("Help")
		helpMenu.Add("About " + AppTitle).OnClick(func(_ *application.Context) {
			desktop.Dialog.Info().
				SetTitle(appDisplayTitle).
				SetMessage(aboutMessage).
				Show()
		})

		desktop.Menu.Set(menu)
	}

	slog.Info("initialized", "app", appDisplayTitle)

	if err := desktop.Run(); err != nil {
		panic(err)
	}
}
