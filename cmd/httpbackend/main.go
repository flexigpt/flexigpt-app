package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/danielgtaylor/huma/v2/humacli"
	conversationStore "github.com/flexigpt/flexigpt-app/internal/conversation/store"
	"github.com/flexigpt/flexigpt-app/internal/inferencewrapper"
	"github.com/flexigpt/flexigpt-app/internal/logrotate"
	modelpresetStore "github.com/flexigpt/flexigpt-app/internal/modelpreset/store"
	promptStore "github.com/flexigpt/flexigpt-app/internal/prompt/store"
	settingStore "github.com/flexigpt/flexigpt-app/internal/setting/store"
	skillStore "github.com/flexigpt/flexigpt-app/internal/skill/store"
	toolStore "github.com/flexigpt/flexigpt-app/internal/tool/store"
	"github.com/flexigpt/flexigpt-app/internal/toolruntime"

	// Run registry init.
	_ "github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
)

// Options for the server cli.
type Options struct {
	Host                   string `doc:"Hostname to listen on."                  default:"127.0.0.1"`
	Port                   int    `doc:"Port to listen on"                       default:"8888"`
	SettingsDirPath        string `doc:"path to directory of settings file"`
	ConversationsDirPath   string `doc:"path to conversations directory"`
	ModelPresetsDirPath    string `doc:"path to modelPresets data directory"`
	PromptTemplatesDirPath string `doc:"path to prompt templates data directory"`
	ToolsDirPath           string `doc:"path to tools data directory"`
	SkillsDirPath          string `doc:"path to skills data directory"`
	LogsDirPath            string `doc:"path to logs directory"`
	Debug                  bool   `doc:"Enable debug logs"`
}

func main() {
	cli := humacli.New(func(hooks humacli.Hooks, opts *Options) {
		log.Printf("Options are %+v\n", opts)
		writer := initSlog(opts.LogsDirPath, opts.Debug)
		router := http.NewServeMux()
		api := humago.New(router, huma.DefaultConfig("FlexiGPTServer API", "1.0.0"))
		app := NewBackendApp(
			opts.SettingsDirPath,
			opts.ConversationsDirPath,
			opts.ModelPresetsDirPath,
			opts.PromptTemplatesDirPath,
			opts.ToolsDirPath,
			opts.SkillsDirPath,
		)
		settingStore.InitSettingStoreHandlers(api, app.settingStoreAPI)
		conversationStore.InitConversationStoreHandlers(api, app.conversationStoreAPI)
		inferencewrapper.InitProviderSetHandlers(api, app.providerSetAPI)
		modelpresetStore.InitModelPresetStoreHandlers(api, app.modelPresetStoreAPI)
		promptStore.InitPromptTemplateStoreHandlers(api, app.promptTemplateStoreAPI)
		toolStore.InitToolStoreHandlers(api, app.toolStoreAPI)
		toolruntime.InitToolRuntimeHandlers(api, app.toolRuntimeAPI)
		skillStore.InitSkillStoreHandlers(api, app.skillStoreAPI)
		// Create the HTTP server.
		server := http.Server{
			Addr:              fmt.Sprintf("%s:%d", opts.Host, opts.Port),
			Handler:           router,
			ReadHeaderTimeout: 10 * time.Second,
		}
		hooks.OnStart(func() {
			if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Fatalf("listen: %s\n", err)
			}
		})

		hooks.OnStop(func() {
			// Gracefully shutdown your server here.
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			defer writer.Close()
			_ = server.Shutdown(ctx)
		})
	})

	cli.Run()
}

func initSlog(logsDirPath string, debug bool) *logrotate.Writer {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	slogOpts := &slog.HandlerOptions{
		Level: level,
	}

	var stdoutHandler slog.Handler = slog.NewTextHandler(os.Stdout, slogOpts)
	stdoutLogger := slog.New(stdoutHandler)

	// Init logger.
	opts := logrotate.Options{
		Directory:            logsDirPath,
		MaximumFileSize:      10 * 1024 * 1024,
		MaximumLifetime:      24 * time.Hour,
		FileNameFunc:         logrotate.DefaultFilenameFunc,
		FlushAfterEveryWrite: true,
	}
	writer, err := logrotate.New(stdoutLogger, opts)
	if err != nil {
		slog.Error("failed to create log writer", "error", err)
		panic("Init logs failed")
	}

	var handler slog.Handler = slog.NewTextHandler(writer, slogOpts)
	slogger := slog.New(handler)
	slog.SetDefault(slogger)
	return writer
}
