package app

import (
	"ai-assistant/pkg/config"
	"ai-assistant/pkg/logger"
	"time"

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

func (a *App) GetSTTDevices() map[string]interface{} {
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
			"id":                d.ID,
			"name":              d.Name,
			"channels":          d.Channels,
			"default_samplerate": d.DefaultSamplerate,
			"host_api":          d.HostAPI,
			"is_default":        d.IsDefault,
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
	if a.sidecar == nil {
		logger.Println("[STT] StartSTTRecording: sidecar is nil")
		return
	}
	if !a.sidecar.IsRunning() {
		logger.Println("[STT] StartSTTRecording: sidecar not running")
		return
	}
	// 使用流式转写
	if err := a.sidecar.Client().STTStartStreaming(); err != nil {
		logger.Printf("[STT] StartSTTRecording: streaming start failed: %v", err)
		return
	}
	logger.Println("[STT] StartSTTRecording: streaming started")
	a.EmitEvent("stt-recording-started")

	// 启动轮询流式结果的 goroutine
	go a.pollStreamingResults()
}

func (a *App) StopSTTRecording() {
	if a.sidecar == nil {
		logger.Println("[STT] StopSTTRecording: sidecar is nil")
		return
	}
	if !a.sidecar.IsRunning() {
		logger.Println("[STT] StopSTTRecording: sidecar not running")
		return
	}
	logger.Println("[STT] StopSTTRecording: calling STTStop")
	result, err := a.sidecar.Client().STTStop()
	if err != nil {
		logger.Printf("[STT] StopSTTRecording: STTStop failed: %v", err)
		return
	}
	logger.Printf("[STT] StopSTTRecording: text=%q", result.Text)
	if result.Text != "" {
		// 将识别结果发送到 AI 对话输入框
		a.EmitEvent("stt-transcribed", result.Text)
	}
	a.EmitEvent("stt-recording-stopped")
}

// InjectTextToActiveInput 注入文字到当前活动输入框（不抢焦点）
func (a *App) InjectTextToActiveInput(text string) {
	// 通过剪贴板 + 模拟 Ctrl+V 注入文字
	injectTextViaClipboard(text)
}

func (a *App) pollStreamingResults() {
	for {
		// 检查是否还在录音
		status, err := a.sidecar.Client().STTStatus()
		if err != nil || !status.Recording {
			return
		}

		// 获取流式转写结果
		results, err := a.sidecar.Client().STTStreamingResults()
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// 发送流式转写结果到前端
		for _, text := range results {
			if text != "" {
				a.EmitEvent("stt-streaming-text", text)
			}
		}

		time.Sleep(100 * time.Millisecond)
	}
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
