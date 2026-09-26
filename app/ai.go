package app

import (
	"ai-assistant/pkg/config"
	"ai-assistant/pkg/logger"
	"context"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const maxInjectedTextBytes = 1 << 20

// ── STT (Speech-to-Text) bindings ─────────────────────────────────────

func (a *App) GetAudioLevel() map[string]float64 {
	if err := a.requireAuthenticated(); err != nil {
		return map[string]float64{"level": 0}
	}
	if a.sidecar == nil || !a.sidecar.IsRunning() {
		return map[string]float64{"level": 0}
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := a.sidecar.Client().AudioLevelContext(ctx)
	if err != nil {
		return map[string]float64{"level": 0}
	}
	return map[string]float64{"level": result.Level}
}

func (a *App) GetSTTStatus() map[string]bool {
	if err := a.requireAuthenticated(); err != nil {
		return map[string]bool{"recording": false, "ready": false}
	}
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
	a.sttOperationMu.Lock()
	defer a.sttOperationMu.Unlock()
	return a.sttStartLocked()
}

func (a *App) sttStartLocked() map[string]string {
	if a.authService == nil || a.authService.Current() == nil {
		return map[string]string{"error": "请先登录后使用语音识别"}
	}
	if a.sidecar == nil || !a.sidecar.IsRunning() {
		return map[string]string{"error": "Sidecar not running"}
	}
	a.stopSTTPoller()
	if err := a.sidecar.Client().STTStart(); err != nil {
		return map[string]string{"error": err.Error()}
	}
	return map[string]string{"status": "recording"}
}

func (a *App) STTStop() map[string]string {
	a.sttOperationMu.Lock()
	defer a.sttOperationMu.Unlock()
	if err := a.requireAuthenticated(); err != nil {
		return map[string]string{"error": err.Error()}
	}
	return a.sttStopLocked()
}

func (a *App) sttStopLocked() map[string]string {
	a.stopSTTPoller()
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
	a.sttOperationMu.Lock()
	defer a.sttOperationMu.Unlock()
	if err := a.requireAuthenticated(); err != nil {
		return map[string]string{"error": err.Error()}
	}
	if a.sidecar == nil || !a.sidecar.IsRunning() {
		return map[string]string{"error": "Sidecar not running"}
	}
	status, err := a.sidecar.Client().STTStatus()
	if err != nil {
		return map[string]string{"error": err.Error()}
	}
	if status.Recording {
		return a.sttStopLocked()
	}
	return a.sttStartLocked()
}

func (a *App) GetSTTDevices() map[string]interface{} {
	if err := a.requireAuthenticated(); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	if a.sidecar == nil || !a.sidecar.IsRunning() {
		return map[string]interface{}{"error": "Sidecar not running"}
	}
	result, err := a.sidecar.Client().STTDevices()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	devices := make([]map[string]interface{}, 0, len(result.Devices))
	for _, d := range result.Devices {
		devices = append(devices, map[string]interface{}{
			"id":                 d.ID,
			"name":               d.Name,
			"channels":           d.Channels,
			"default_samplerate": d.DefaultSamplerate,
			"host_api":           d.HostAPI,
			"is_default":         d.IsDefault,
		})
	}
	out := map[string]interface{}{
		"devices":             devices,
		"current_sample_rate": result.CurrentSampleRate,
	}
	if result.CurrentDeviceID != nil {
		out["current_device_id"] = *result.CurrentDeviceID
	}
	return out
}

func (a *App) SetSTTDevice(deviceID int) map[string]interface{} {
	if err := a.requireAuthenticated(); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	if a.sidecar == nil || !a.sidecar.IsRunning() {
		return map[string]interface{}{"error": "Sidecar not running"}
	}
	result, err := a.sidecar.Client().STTSetDevice(deviceID)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	out := map[string]interface{}{
		"status":      result.Status,
		"sample_rate": result.SampleRate,
	}
	if result.DeviceID != nil {
		out["device_id"] = *result.DeviceID
	}
	return out
}

// SetSTTDeviceByName 按设备名称设置音频输入设备
func (a *App) SetSTTDeviceByName(deviceName string) map[string]interface{} {
	if err := a.requireAuthenticated(); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	if a.sidecar == nil || !a.sidecar.IsRunning() {
		return map[string]interface{}{"error": "Sidecar not running"}
	}
	// 通过 HTTP 调用 sidecar 的设备切换接口
	// 这里复用 STTSetDevice，但传递设备名称
	result, err := a.sidecar.Client().STTSetDeviceByName(deviceName)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{
		"status": result.Status,
	}
}

func (a *App) StartSTTRecording() {
	a.sttOperationMu.Lock()
	defer a.sttOperationMu.Unlock()

	if a.authService == nil || a.authService.Current() == nil {
		logger.Println("[STT] rejected: authentication required")
		return
	}
	if a.sidecar == nil || !a.sidecar.IsRunning() {
		logger.Println("[STT] StartSTTRecording: sidecar not running")
		a.stopSTTPoller()
		a.EmitEvent("stt-recording-stopped")
		return
	}

	wasActive := a.stopSTTPoller()
	baseCtx := a.ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	requestCtx, requestCancel := context.WithTimeout(baseCtx, 10*time.Second)
	if wasActive {
		_, _ = a.sidecar.Client().STTStopContext(requestCtx)
		a.EmitEvent("stt-recording-stopped")
	}
	if err := a.sidecar.Client().STTStartStreamingContext(requestCtx); err != nil {
		requestCancel()
		logger.Printf("[STT] StartSTTRecording: streaming start failed")
		a.EmitEvent("stt-recording-stopped")
		return
	}
	requestCancel()

	pollCtx, cancel := context.WithCancel(baseCtx)
	a.sttMu.Lock()
	a.sttGeneration++
	generation := a.sttGeneration
	a.sttCancel = cancel
	a.sttActive = true
	a.sttWG.Add(1)
	a.sttMu.Unlock()

	logger.Println("[STT] StartSTTRecording: streaming started")
	a.EmitEvent("stt-recording-started")
	go a.pollStreamingResults(pollCtx, generation)
}

func (a *App) StopSTTRecording() {
	a.sttOperationMu.Lock()
	defer a.sttOperationMu.Unlock()

	a.stopSTTPoller()
	if a.sidecar == nil || !a.sidecar.IsRunning() {
		logger.Println("[STT] StopSTTRecording: sidecar not running")
		a.EmitEvent("stt-recording-stopped")
		return
	}

	baseCtx := a.ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	ctx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
	defer cancel()
	result, err := a.sidecar.Client().STTStopContext(ctx)
	if err != nil {
		logger.Printf("[STT] StopSTTRecording: STTStop failed")
		a.EmitEvent("stt-recording-stopped")
		return
	}
	logger.Printf("[STT] StopSTTRecording: text_length=%d", len(result.Text))
	if result.Text != "" {
		a.EmitEvent("stt-transcribed", result.Text)
	}
	a.EmitEvent("stt-recording-stopped")
}

func (a *App) shutdownSTT() {
	a.sttOperationMu.Lock()
	defer a.sttOperationMu.Unlock()
	a.stopSTTPoller()
	if a.sidecar == nil || !a.sidecar.IsRunning() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = a.sidecar.Client().STTStopContext(ctx)
}

// InjectTextToActiveInput 注入文字到当前活动输入框（不抢焦点）
func (a *App) InjectTextToActiveInput(text string) {
	if len(text) == 0 || len(text) > maxInjectedTextBytes {
		return
	}
	// 通过剪贴板 + 模拟 Ctrl+V 注入文字
	injectTextViaClipboard(text)
}

func (a *App) stopSTTPoller() bool {
	a.sttMu.Lock()
	cancel := a.sttCancel
	wasActive := a.sttActive
	a.sttGeneration++
	a.sttCancel = nil
	a.sttActive = false
	a.sttMu.Unlock()
	if cancel != nil {
		cancel()
	}
	a.sttWG.Wait()
	return wasActive
}

func (a *App) isCurrentSTTSession(generation uint64) bool {
	a.sttMu.Lock()
	defer a.sttMu.Unlock()
	return a.sttActive && a.sttGeneration == generation
}

func (a *App) pollStreamingResults(ctx context.Context, generation uint64) {
	defer a.sttWG.Done()
	defer func() {
		a.sttMu.Lock()
		endedCurrent := a.sttActive && a.sttGeneration == generation
		if endedCurrent {
			a.sttActive = false
			a.sttCancel = nil
		}
		a.sttMu.Unlock()
		if endedCurrent {
			a.EmitEvent("stt-recording-stopped")
		}
	}()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if !a.isCurrentSTTSession(generation) || a.sidecar == nil || !a.sidecar.IsRunning() {
			return
		}

		status, err := a.sidecar.Client().STTStatusContext(ctx)
		if err != nil || !status.Recording {
			return
		}
		results, err := a.sidecar.Client().STTStreamingResultsContext(ctx)
		if err != nil {
			continue
		}
		for _, text := range results {
			if text != "" && a.isCurrentSTTSession(generation) {
				a.EmitEvent("stt-streaming-text", text)
			}
		}
	}
}

// ── KB (Knowledge Base) bindings ──────────────────────────────────────

func (a *App) LoadKB(path string) map[string]interface{} {
	if err := a.requireAuthenticated(); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
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
	if err := a.requireAuthenticated(); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
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
	if err := a.requireAuthenticated(); err != nil {
		return map[string]interface{}{"ready": false, "error": err.Error()}
	}
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
	if err := a.requireAuthenticated(); err != nil {
		return ""
	}
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
	if err := a.requireAuthenticated(); err != nil {
		logger.Printf("[KB] clear rejected: %v", err)
		return
	}
	a.configManager.Patch(func(cfg *config.Config) {
		cfg.KBPath = ""
	})
	a.EmitEvent("kb-cleared")
	logger.Println("[KB] Cleared")
}
