package app

import (
	"ai-assistant/pkg/logger"
	"ai-assistant/pkg/solution"
	"context"
	"strings"
)

const MaxScreenshots = 3

var screenshotBuffer []string
var pendingUserMessage string

func (a *App) TriggerScreenshot() {
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

	if len(screenshotBuffer) >= MaxScreenshots {
		a.EmitEvent("toast", "最多截图 3 张图片，请先发送或删除")
		return
	}

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

	screenshotBuffer = append(screenshotBuffer, previewResult.Base64)
	a.EmitEvent("screenshot-taken", previewResult.Base64, len(screenshotBuffer))
}

func (a *App) RemoveScreenshot(index int) {
	if index < 0 || index >= len(screenshotBuffer) {
		return
	}
	screenshotBuffer = append(screenshotBuffer[:index], screenshotBuffer[index+1:]...)
	a.EmitEvent("screenshot-removed", index, len(screenshotBuffer))
}

func (a *App) RemoveLastScreenshot() {
	if len(screenshotBuffer) == 0 {
		return
	}
	index := len(screenshotBuffer) - 1
	screenshotBuffer = screenshotBuffer[:index]
	a.EmitEvent("screenshot-removed", index, len(screenshotBuffer))
}

func (a *App) ClearScreenshots() {
	screenshotBuffer = nil
	a.EmitEvent("screenshots-cleared")
}

func (a *App) TriggerSend() {
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

	if len(screenshotBuffer) == 0 {
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
		screenshotBuffer = append(screenshotBuffer, previewResult.Base64)
	}

	if a.taskManager.HasRunningTask() {
		logger.Println("忽略重复触发：当前有任务正在运行")
		a.EmitEvent("toast", "正在处理中，请稍候...")
		return
	}

	screenshots := make([]string, len(screenshotBuffer))
	copy(screenshots, screenshotBuffer)
	screenshotBuffer = nil

	a.EmitEvent("start-solving")
	a.EmitEvent("user-message", screenshots[0])

	ctx, taskID := a.taskManager.StartTask("solve")
	go func() {
		defer a.taskManager.CompleteTask(taskID)
		a.solveInternal(ctx, screenshots)
	}()
}

func (a *App) TriggerSolve() {
	a.TriggerSend()
}

func (a *App) TriggerDeleteScreenshot() {
	a.RemoveLastScreenshot()
}

func (a *App) solveInternal(ctx context.Context, screenshots []string) bool {
	cfg := a.configManager.Get()

	if cfg.APIKey == "" {
		a.EmitEvent("require-api-key")
		return false
	}

	req := solution.Request{
		Config:      cfg,
		Screenshots: screenshots,
		UserMessage: pendingUserMessage,
	}
	pendingUserMessage = ""

	// Auto-inject KB context if sidecar is running and KB is configured
	if cfg.KBPath != "" && a.sidecar != nil && a.sidecar.IsRunning() {
		searchQuery := req.UserMessage
		if searchQuery == "" {
			searchQuery = cfg.DomainId
		}
		if searchQuery != "" {
			searchResult, err := a.sidecar.Client().KBSearch(searchQuery, 5)
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

	cb := solution.Callbacks{
		EmitEvent: a.EmitEvent,
	}

	return a.solver.Solve(ctx, req, cb)
}

func (a *App) SetPendingUserMessage(text string) {
	pendingUserMessage = text
}

func (a *App) CancelRunningTask() bool {
	return a.taskManager.CancelCurrentTask()
}

func (a *App) IsInterruptThinkingEnabled() bool {
	return a.configManager.Get().InterruptThinking
}
