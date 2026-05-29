package app

import (
	"ai-assistant/pkg/config"
	"ai-assistant/pkg/logger"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ── STT (Speech-to-Text) bindings ─────────────────────────────────────

func (a *App) GetSTTStatus() map[string]bool {
	if a.sidecar == nil || !a.sidecar.IsRunning() {
		return map[string]bool{"recording": false, "ready": false}
	}
	status, err := a.sidecar.Client().STTStatus()
	if err != nil {
		return map[string]bool{"recording": false, "ready": false}
	}
	return map[string]bool{"recording": status.Recording, "ready": true}
}

func (a *App) STTStart() map[string]string {
	if a.sidecar == nil || !a.sidecar.IsRunning() {
		return map[string]string{"error": "Sidecar not running"}
	}
	if err := a.sidecar.Client().STTStart(); err != nil {
		return map[string]string{"error": err.Error()}
	}
	return map[string]string{"status": "recording"}
}

func (a *App) STTStop() map[string]string {
	if a.sidecar == nil || !a.sidecar.IsRunning() {
		return map[string]string{"error": "Sidecar not running"}
	}
	result, err := a.sidecar.Client().STTStop()
	if err != nil {
		return map[string]string{"error": err.Error()}
	}
	return map[string]string{"text": result.Text, "status": "transcribed"}
}

func (a *App) ToggleSTT() map[string]string {
	status, err := a.sidecar.Client().STTStatus()
	if err != nil {
		return map[string]string{"error": err.Error()}
	}
	if status.Recording {
		return a.STTStop()
	}
	return a.STTStart()
}

func (a *App) StartSTTRecording() {
	if a.sidecar == nil || !a.sidecar.IsRunning() {
		return
	}
	a.sidecar.Client().STTStart()
	a.EmitEvent("stt-recording-started")
}

func (a *App) StopSTTRecording() {
	if a.sidecar == nil || !a.sidecar.IsRunning() {
		return
	}
	result, err := a.sidecar.Client().STTStop()
	if err != nil {
		return
	}
	if result.Text != "" {
		// Set the transcribed text as pending user message for next solve
		a.SetPendingUserMessage(result.Text)
		a.EmitEvent("stt-transcribed", result.Text)
	}
	a.EmitEvent("stt-recording-stopped")
}

// ── KB (Knowledge Base) bindings ──────────────────────────────────────

func (a *App) LoadKB(path string) map[string]interface{} {
	if a.sidecar == nil || !a.sidecar.IsRunning() {
		return map[string]interface{}{"error": "Sidecar not running"}
	}
	result, err := a.sidecar.Client().KBLoad(path)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{
		"status":        result.Status,
		"file_count":    result.FileCount,
		"section_count": result.SectionCount,
	}
}

func (a *App) SearchKB(query string) map[string]interface{} {
	if a.sidecar == nil || !a.sidecar.IsRunning() {
		return map[string]interface{}{"error": "Sidecar not running"}
	}
	result, err := a.sidecar.Client().KBSearch(query, 5)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	items := make([]map[string]interface{}, 0, len(result.Results))
	for _, r := range result.Results {
		items = append(items, map[string]interface{}{
			"source":  r.Source,
			"header":  r.Header,
			"content": r.Content,
			"score":   r.Score,
		})
	}
	return map[string]interface{}{"results": items}
}

func (a *App) GetKBStatus() map[string]interface{} {
	if a.sidecar == nil || !a.sidecar.IsRunning() {
		return map[string]interface{}{"ready": false}
	}

	cfg := a.configManager.Get()
	result := map[string]interface{}{
		"ready":   cfg.KBPath != "",
		"kb_path": cfg.KBPath,
	}

	// Query actual section/file counts from sidecar
	info, err := a.sidecar.Client().KBInfo()
	if err == nil {
		result["file_count"] = info.FileCount
		result["section_count"] = info.SectionCount
	}
	return result
}

func (a *App) SelectKBDirectory() string {
	if a.ctx == nil {
		return ""
	}
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择知识库目录（含 .md 文件）",
	})
	if err != nil || dir == "" {
		return ""
	}
	// Save path in config
	a.configManager.Patch(func(cfg *config.Config) {
		cfg.KBPath = dir
	})
	// Load into sidecar
	if a.sidecar != nil && a.sidecar.IsRunning() {
		result, err := a.sidecar.Client().KBLoad(dir)
		if err != nil {
			logger.Printf("[KB] Load failed: %v", err)
		} else {
			logger.Printf("[KB] Loaded: %d files, %d sections", result.FileCount, result.SectionCount)
		}
	}
	return dir
}

func (a *App) ClearKB() {
	a.configManager.Patch(func(cfg *config.Config) {
		cfg.KBPath = ""
	})
	a.EmitEvent("kb-cleared")
	logger.Println("[KB] Cleared")
}
