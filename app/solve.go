package app

import (
	"ai-assistant/pkg/llm"
	"ai-assistant/pkg/logger"
	"ai-assistant/pkg/solution"
	"context"
	"strings"
)

const MaxScreenshots = 3

// SetFollowUpActive 设置追问对话框状态（前端调用）
func (a *App) SetFollowUpActive(active bool) {
	a.followUpMu.Lock()
	a.followUpActive = active
	a.followUpMu.Unlock()
}

func (a *App) isFollowUpActive() bool {
	a.followUpMu.RLock()
	defer a.followUpMu.RUnlock()
	return a.followUpActive
}

func (a *App) TriggerScreenshot() {
	// 追问对话框打开时，跳过全局截图（由追问对话框自己处理 F8）
	if a.isFollowUpActive() {
		return
	}

	cfg := a.configManager.Get()

	if cfg.APIKey == "" {
		a.EmitEvent("require-api-key")
		return
	}

	if cfg.Model == "" {
		a.EmitEvent("toast", "请先选择模型")
		a.EmitEvent("open-settings", "model")
		return
	}

	if a.taskManager.HasRunningTask() {
		logger.Println("忽略截图：当前有任务正在运行")
		a.EmitEvent("toast", "正在处理中，请稍候...")
		return
	}

	a.screenshotCaptureMu.Lock()
	a.screenshotMu.Lock()
	if len(a.screenshotBuffer) >= MaxScreenshots {
		a.screenshotMu.Unlock()
		a.screenshotCaptureMu.Unlock()
		a.EmitEvent("toast", "最多截图 3 张图片，请先发送或删除")
		return
	}
	a.screenshotMu.Unlock()

	previewResult, err := a.GetScreenshotPreview(
		cfg.CompressionQuality,
		cfg.Sharpening,
		cfg.Grayscale,
		cfg.NoCompression,
		cfg.ScreenshotMode,
	)
	if err != nil {
		logger.Printf("截图失败: %v\n", err)
		a.screenshotCaptureMu.Unlock()
		a.EmitEvent("toast", "截图失败: "+err.Error())
		return
	}

	a.screenshotMu.Lock()
	a.screenshotBuffer = append(a.screenshotBuffer, previewResult.Base64)
	count := len(a.screenshotBuffer)
	a.screenshotMu.Unlock()
	a.screenshotCaptureMu.Unlock()
	a.EmitEvent("screenshot-taken", previewResult.Base64, count)
}

func (a *App) RemoveScreenshot(index int) {
	a.screenshotMu.Lock()
	if index < 0 || index >= len(a.screenshotBuffer) {
		a.screenshotMu.Unlock()
		return
	}
	a.screenshotBuffer = append(a.screenshotBuffer[:index], a.screenshotBuffer[index+1:]...)
	count := len(a.screenshotBuffer)
	a.screenshotMu.Unlock()
	a.EmitEvent("screenshot-removed", index, count)
}

func (a *App) RemoveLastScreenshot() {
	a.screenshotMu.Lock()
	if len(a.screenshotBuffer) == 0 {
		a.screenshotMu.Unlock()
		return
	}
	index := len(a.screenshotBuffer) - 1
	a.screenshotBuffer = a.screenshotBuffer[:index]
	count := len(a.screenshotBuffer)
	a.screenshotMu.Unlock()
	a.EmitEvent("screenshot-removed", index, count)
}

func (a *App) ClearScreenshots() {
	a.screenshotMu.Lock()
	a.screenshotBuffer = nil
	a.screenshotMu.Unlock()
	a.EmitEvent("screenshots-cleared")
}

// TriggerFollowUpScreenshot 追问截图（F6）
func (a *App) TriggerFollowUpScreenshot() string {
	cfg := a.configManager.Get()

	previewResult, err := a.GetScreenshotPreview(
		cfg.CompressionQuality,
		cfg.Sharpening,
		cfg.Grayscale,
		cfg.NoCompression,
		cfg.ScreenshotMode,
	)
	if err != nil {
		logger.Printf("追问截图失败: %v\n", err)
		a.EmitEvent("toast", "截图失败: "+err.Error())
		return ""
	}

	a.EmitEvent("followup-screenshot-taken", previewResult.Base64)
	return previewResult.Base64
}

// StopThinking 停止当前思考/生成
func (a *App) StopThinking() {
	if a.taskManager.HasRunningTask() {
		a.taskManager.CancelCurrentTask()
		a.EmitEvent("toast", "已停止思考")
		a.EmitEvent("thinking-stopped")
	}
}

func (a *App) TriggerSend() {
	a.triggerSendMu.Lock()
	defer a.triggerSendMu.Unlock()
	a.screenshotCaptureMu.Lock()
	defer a.screenshotCaptureMu.Unlock()
	cfg := a.configManager.Get()

	if cfg.APIKey == "" {
		a.EmitEvent("require-api-key")
		return
	}

	if cfg.Model == "" {
		a.EmitEvent("toast", "请先选择模型")
		a.EmitEvent("open-settings", "model")
		return
	}

	a.screenshotMu.Lock()
	if len(a.screenshotBuffer) == 0 {
		a.screenshotMu.Unlock()
		previewResult, err := a.GetScreenshotPreview(
			cfg.CompressionQuality,
			cfg.Sharpening,
			cfg.Grayscale,
			cfg.NoCompression,
			cfg.ScreenshotMode,
		)
		if err != nil {
			logger.Printf("截图失败: %v\n", err)
			a.EmitEvent("toast", "截图失败: "+err.Error())
			return
		}
		a.screenshotMu.Lock()
		a.screenshotBuffer = append(a.screenshotBuffer, previewResult.Base64)
	}

	if a.taskManager.HasRunningTask() {
		a.screenshotMu.Unlock()
		logger.Println("忽略重复触发：当前有任务正在运行")
		a.EmitEvent("toast", "正在处理中，请稍候...")
		return
	}

	screenshots := make([]string, len(a.screenshotBuffer))
	copy(screenshots, a.screenshotBuffer)
	userMsg := a.pendingUserMessage
	a.pendingUserMessage = ""
	a.screenshotBuffer = nil
	a.screenshotMu.Unlock()

	requestID := newRequestID()
	a.EmitEvent("start-solving", map[string]any{"requestId": requestID})
	a.EmitEvent("user-message", map[string]any{"requestId": requestID, "screenshot": screenshots[0]})

	ctx, taskID := a.taskManager.StartRequest("solve", requestID)
	go func() {
		defer a.taskManager.CompleteTask(taskID)
		a.solveInternal(ctx, taskID, requestID, screenshots, userMsg)
	}()
}

func (a *App) TriggerSolve() {
	a.TriggerSend()
}

func (a *App) TriggerDeleteScreenshot() {
	a.RemoveLastScreenshot()
}

func (a *App) solveInternal(ctx context.Context, taskID int64, requestID string, screenshots []string, userMsg string) bool {
	cfg := a.configManager.Get()

	if cfg.APIKey == "" {
		a.EmitEvent("require-api-key")
		return false
	}

	req := solution.Request{
		Config:      cfg,
		Screenshots: screenshots,
		UserMessage: userMsg,
	}

	// Auto-inject KB context if sidecar is running and KB is configured
	if cfg.KBPath != "" && a.sidecar != nil && a.sidecar.IsRunning() {
		searchQuery := req.UserMessage
		if searchQuery == "" {
			searchQuery = cfg.DomainId
		}
		if searchQuery != "" {
			searchResult, err := a.sidecar.Client().KBSearchContext(ctx, searchQuery, 5)
			if err == nil && len(searchResult.Results) > 0 {
				var kbCtx strings.Builder
				for _, item := range searchResult.Results {
					kbCtx.WriteString("\n---\n")
					kbCtx.WriteString("来源: ")
					kbCtx.WriteString(item.Source)
					kbCtx.WriteString("\n")
					kbCtx.WriteString(item.Content)
				}
				req.KBContext = kbCtx.String()
				logger.Printf("[Solve] Injected %d KB sections", len(searchResult.Results))
			}
		}
	}

	// 有截图时自动切换到视觉模型（Qwen），F7 仍使用原模型（DeepSeek）
	if len(screenshots) > 0 && cfg.ScreenshotAPIKey != "" {
		visionCfg := cfg
		visionCfg.APIKey = cfg.ScreenshotAPIKey
		visionCfg.BaseURL = cfg.ScreenshotBaseURL
		visionCfg.Model = cfg.ScreenshotModel
		visionProvider := llm.NewOpenAIAdapter(&visionCfg)
		req.Provider = visionProvider
		logger.Printf("[Solve] 使用视觉模型: %s", cfg.ScreenshotModel)
	}

	cb := solution.Callbacks{
		EmitEvent: func(event string, data ...interface{}) {
			field := ""
			switch event {
			case "solution-stream-thinking":
				field = "thinking"
			case "solution-stream-chunk":
				field = "chunk"
			case "solution-error":
				field = "error"
			case "solution":
				field = "content"
			}
			var value any
			if len(data) > 0 {
				value = data[0]
			}
			a.emitRequestEvent(ctx, taskID, event, requestID, field, value)
		},
	}

	return a.solver.Solve(ctx, req, cb)
}

func (a *App) SetPendingUserMessage(text string) {
	a.screenshotMu.Lock()
	a.pendingUserMessage = text
	a.screenshotMu.Unlock()
}

func (a *App) CancelRequest(requestID string) bool {
	if !validRequestID(requestID) {
		return false
	}
	return a.taskManager.CancelRequest(requestID)
}

func (a *App) CancelRunningTask() bool {
	return a.taskManager.CancelCurrentTask()
}

func (a *App) IsInterruptThinkingEnabled() bool {
	return a.configManager.Get().InterruptThinking
}
