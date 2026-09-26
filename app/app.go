package app

import (
	"ai-assistant/pkg/auth"
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
	"sync"
	"sync/atomic"
	"time"
)

type App struct {
	ctx context.Context

	configManager *config.ConfigManager
	authService   *auth.Service
	stateManager  *state.StateManager
	taskManager   *task.TaskCoordinator

	llmService      *llm.Service
	resumeService   *resume.Service
	shortcutService *shortcut.Service
	screenService   *screen.Service
	solver          *solution.Solver
	sidecar         *sidecar.Manager

	followUpMu     sync.RWMutex
	followUpActive bool // 追问对话框是否打开
	standaloneMode bool // 是否为独立面试窗口模式

	screenshotMu        sync.Mutex
	screenshotCaptureMu sync.Mutex
	screenshotBuffer    []string
	pendingUserMessage  string
	triggerSendMu       sync.Mutex

	sttOperationMu sync.Mutex
	sttMu          sync.Mutex
	sttGeneration  uint64
	sttCancel      context.CancelFunc
	sttActive      bool
	sttWG          sync.WaitGroup
	shuttingDown   atomic.Bool
	sidecarWG      sync.WaitGroup
	startupCancel  context.CancelFunc
}

func NewApp(mode string) *App {
	configManager := config.NewConfigManager()
	authService, err := auth.New()
	if err != nil {
		logger.Printf("初始化认证服务失败: %v", err)
	}

	return &App{
		configManager:  configManager,
		authService:    authService,
		stateManager:   state.NewStateManager(),
		taskManager:    task.NewTaskCoordinator(),
		screenService:  screen.NewService(),
		standaloneMode: mode == "--standalone-interview",
	}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.taskManager = task.NewTaskCoordinatorWithContext(ctx)

	if a.standaloneMode {
		runtime.WindowSetTitle(ctx, "AI 辅助面试")
	}

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

	// Start Python sidecar. Its startup is cancellable and participates in shutdown.
	a.sidecar = sidecar.NewManager(18765)
	startupCtx, startupCancel := context.WithCancel(ctx)
	a.startupCancel = startupCancel
	a.stateManager.UpdateInitStatus(state.StatusLoadingModel)
	a.sidecarWG.Add(1)
	go func() {
		defer a.sidecarWG.Done()
		sttService := cfg.STTService
		if sttService == "" {
			sttService = "qwen_cloud" // fallback
		}
		if err := a.sidecar.StartContext(startupCtx, cfg.STTModel, cfg.STTDevice, cfg.STTLanguage, cfg.STTSensitivity, sttService); err != nil {
			if !a.shuttingDown.Load() {
				logger.Printf("[Sidecar] Start failed: %v", err)
				a.stateManager.UpdateInitStatus(state.StatusError)
			}
			return
		}
		if a.shuttingDown.Load() {
			return
		}
		if cfg.KBPath != "" {
			a.stateManager.UpdateInitStatus(state.StatusLoadingModel)
			loadCtx, cancel := context.WithTimeout(startupCtx, 30*time.Second)
			result, err := a.sidecar.Client().KBLoadContext(loadCtx, cfg.KBPath)
			cancel()
			if err != nil {
				logger.Printf("[Sidecar] KB load failed: %v", err)
				a.stateManager.UpdateInitStatus(state.StatusError)
				return
			}
			logger.Printf("[Sidecar] KB loaded: %d files, %d sections", result.FileCount, result.SectionCount)
		}
		if !a.shuttingDown.Load() {
			a.stateManager.UpdateInitStatus(state.StatusReady)
		}
	}()

	a.configManager.Subscribe(a.onConfigChanged)
}

// IsStandaloneInterview 返回是否独立面试窗口模式
func (a *App) IsStandaloneInterview() bool {
	return a.standaloneMode
}

// OpenStandaloneInterview 打开独立面试窗口（主窗口调用）
func (a *App) OpenStandaloneInterview() {
	logger.Println("[Standalone] 打开 AI 辅助面试面板")
	// 由于系统策略阻止创建新进程，改为直接打开已有的面试面板
	// 同时最大化主窗口使其看起来像独立窗口
	runtime.WindowMaximise(a.ctx)
	a.EmitEvent("open-chat")
	a.EmitEvent("toast", "AI 辅助面试已展开（最大化窗口可作独立窗口使用）")
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
	a.shuttingDown.Store(true)
	if a.startupCancel != nil {
		a.startupCancel()
	}
	a.shutdownSTT()
	if a.taskManager != nil {
		a.taskManager.CancelCurrentTask()
	}
	if a.shortcutService != nil {
		a.shortcutService.Stop()
	}
	a.sidecarWG.Wait()
	if a.authService != nil {
		a.authService.Logout()
	}
	if a.sidecar != nil {
		a.sidecar.Stop()
	}
	if err := a.configManager.Save(); err != nil {
		logger.Printf("保存配置失败: %v", err)
	}
}

func (a *App) EmitEvent(eventName string, data ...interface{}) {
	if a.shuttingDown.Load() || a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, eventName, data...)
}

func (a *App) Show() {
	runtime.WindowShow(a.ctx)
}
