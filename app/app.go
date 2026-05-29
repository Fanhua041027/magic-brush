package app

import (
	"ai-assistant/pkg/config"
	"ai-assistant/pkg/llm"
	"ai-assistant/pkg/logger"
	"ai-assistant/pkg/resume"
	"ai-assistant/pkg/screen"
	"ai-assistant/pkg/shortcut"
	"ai-assistant/pkg/sidecar"
	"ai-assistant/pkg/solution"
	"ai-assistant/pkg/state"
	"ai-assistant/pkg/task"
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context

	configManager *config.ConfigManager
	stateManager  *state.StateManager
	taskManager   *task.TaskCoordinator

	llmService      *llm.Service
	resumeService   *resume.Service
	shortcutService *shortcut.Service
	screenService   *screen.Service
	solver          *solution.Solver
	sidecar         *sidecar.Manager
}

func NewApp() *App {
	configManager := config.NewConfigManager()

	return &App{
		configManager: configManager,
		stateManager:  state.NewStateManager(),
		taskManager:   task.NewTaskCoordinator(),
		screenService: screen.NewService(),
	}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	if err := a.configManager.Load(); err != nil {
		logger.Printf("加载配置失败: %v", err)
	}

	cfg := a.configManager.Get()
	if cfg.WindowWidth > 0 && cfg.WindowHeight > 0 {
		runtime.WindowSetSize(ctx, cfg.WindowWidth, cfg.WindowHeight)
		logger.Printf("应用保存的窗口尺寸: %dx%d", cfg.WindowWidth, cfg.WindowHeight)
	}

	a.stateManager.Startup(ctx, a.EmitEvent)
	a.screenService.Startup(ctx)

	a.llmService = llm.NewService(a.configManager.Get(), a.configManager)
	a.solver = solution.NewSolver(a.llmService.GetProvider())
	a.resumeService = resume.NewService(a.configManager.Get(), a.configManager)
	a.resumeService.SetProvider(a.llmService.GetProvider())

	a.shortcutService = shortcut.NewService(a, a.configManager.Get().Shortcuts, func(callback func(map[string]shortcut.KeyBinding)) {
		a.configManager.Subscribe(func(newConfig config.Config, oldConfig config.Config) {
			callback(newConfig.Shortcuts)
		})
	})
	a.shortcutService.Start()

	// Start Python sidecar
	a.sidecar = sidecar.NewManager(18765)
	go func() {
		if err := a.sidecar.Start(cfg.STTModel, cfg.STTDevice); err != nil {
			logger.Printf("[Sidecar] Start failed: %v", err)
		} else if cfg.KBPath != "" {
			result, err := a.sidecar.Client().KBLoad(cfg.KBPath)
			if err != nil {
				logger.Printf("[Sidecar] KB load failed: %v", err)
			} else {
				logger.Printf("[Sidecar] KB loaded: %d files, %d sections", result.FileCount, result.SectionCount)
			}
		}
	}()

	a.configManager.Subscribe(a.onConfigChanged)
	a.stateManager.UpdateInitStatus(state.StatusReady)
}

func (a *App) onConfigChanged(newConfig config.Config, oldConfig config.Config) {
	if a.solver != nil {
		a.solver.SetProvider(a.llmService.GetProvider())
	}
	if a.resumeService != nil {
		a.resumeService.SetProvider(a.llmService.GetProvider())
	}

	if !newConfig.KeepContext && a.solver != nil {
		a.solver.ClearHistory()
	}

	// Reload KB if path changed
	if newConfig.KBPath != oldConfig.KBPath && a.sidecar != nil && a.sidecar.IsRunning() {
		if newConfig.KBPath != "" {
			result, err := a.sidecar.Client().KBLoad(newConfig.KBPath)
			if err != nil {
				logger.Printf("[Sidecar] KB reload failed: %v", err)
			} else {
				logger.Printf("[Sidecar] KB reloaded: %d files, %d sections", result.FileCount, result.SectionCount)
			}
		}
	}

	logger.Println("配置已更新并应用")
}

func (a *App) OnShutdown(ctx context.Context) {
	if a.shortcutService != nil {
		a.shortcutService.Stop()
	}
	if a.sidecar != nil {
		a.sidecar.Stop()
	}
	if err := a.configManager.Save(); err != nil {
		logger.Printf("保存配置失败: %v", err)
	}
}

func (a *App) EmitEvent(eventName string, data ...interface{}) {
	runtime.EventsEmit(a.ctx, eventName, data...)
}

func (a *App) Show() {
	runtime.WindowShow(a.ctx)
}
