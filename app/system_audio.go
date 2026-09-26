package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// AudioDevice represents an audio input device
type AudioDevice struct {
	Index       int    `json:"index"`
	Name        string `json:"name"`
	Type        string `json:"type"` // "system_audio", "stereo_mix", "cable", or "mic"
	Channels    int    `json:"channels"`
	SampleRate  int    `json:"default_samplerate"`
	HostAPI     string `json:"host_api"`
	IsDefault   bool   `json:"is_default"`
	Recommended bool   `json:"recommended,omitempty"`
	Backend     string `json:"backend,omitempty"`
	CaptureMode string `json:"capture_mode,omitempty"`
	IsLoopback  bool   `json:"is_loopback,omitempty"`
	SpeakerID   string `json:"speaker_id,omitempty"`
}

// AudioCaptureResult is the result of an audio capture
type AudioCaptureResult struct {
	Status     string  `json:"status"`
	Duration   float64 `json:"duration,omitempty"`
	Samples    int     `json:"samples,omitempty"`
	SampleRate int     `json:"sample_rate,omitempty"`
	Path       string  `json:"path,omitempty"`
	Base64     string  `json:"base64,omitempty"`
	Error      string  `json:"error,omitempty"`
}

// AudioCaptureService manages system audio capture
type AudioCaptureService struct {
	pythonPath string
	scriptPath string
}

// NewAudioCaptureService creates a new audio capture service
func NewAudioCaptureService() *AudioCaptureService {
	return &AudioCaptureService{
		pythonPath: findPythonExec(),
		scriptPath: findAudioCaptureScript(),
	}
}

// AudioListDevices lists audio input devices via Python script
func (a *App) AudioListDevices() string {
	if err := a.requireAuthenticated(); err != nil {
		return fmt.Sprintf(`{"error":%q}`, err.Error())
	}
	scriptPath := a.findAudioScript()
	if scriptPath == "" {
		return "[]"
	}

	cmd := exec.Command("python3", scriptPath, "list")
	output, err := cmd.Output()
	if err != nil {
		// Try python instead
		cmd = exec.Command("python", scriptPath, "list")
		output, err = cmd.Output()
		if err != nil {
			return "[]"
		}
	}
	return string(output)
}

// AudioCapture captures system audio for the given duration
func (a *App) AudioCapture(deviceID int, durationSec float64) string {
	if err := a.requireAuthenticated(); err != nil {
		return fmt.Sprintf(`{"error":%q}`, err.Error())
	}
	if math.IsNaN(durationSec) || math.IsInf(durationSec, 0) || durationSec < 1 || durationSec > 120 {
		return `{"error":"duration must be between 1 and 120 seconds"}`
	}
	scriptPath := a.findAudioScript()
	if scriptPath == "" {
		return `{"error":"Audio capture script not found"}`
	}

	args := []string{scriptPath, "capture", fmt.Sprintf("%d", deviceID)}
	args = append(args, fmt.Sprintf("%.1f", durationSec))

	cmd := exec.Command("python3", args...)
	output, err := cmd.Output()
	if err != nil {
		cmd = exec.Command("python", args...)
		output, err = cmd.Output()
		if err != nil {
			return fmt.Sprintf(`{"error":"%s"}`, err.Error())
		}
	}
	return string(output)
}

// AudioCaptureWithEnhancement 捕获系统音频并增强
func (a *App) AudioCaptureWithEnhancement(deviceID int, durationSec float64) string {
	return a.AudioCapture(deviceID, durationSec)
}

// AudioTranscribe sends captured audio for transcription.
// Prioritizes the sidecar STT pipeline (same as Left Alt), falls back to API call.
func (a *App) AudioTranscribe(base64Data string) string {
	if err := a.requireAuthenticated(); err != nil {
		return fmt.Sprintf(`{"error":%q}`, err.Error())
	}
	if len(base64Data) == 0 || len(base64Data) > maxAudioBase64Bytes {
		return `{"error":"音频内容无效或过大"}`
	}
	// 优先走 sidecar STT 管线（与左 Alt 按键同一套，准确率更高）
	if a.sidecar != nil && a.sidecar.IsRunning() {
		result, err := a.sidecar.Client().STTTranscribe(base64Data, 16000)
		if err == nil && result.Text != "" {
			data, _ := json.Marshal(map[string]string{"text": result.Text})
			return string(data)
		}
	}

	// fallback: 通过 API 转写（OpenAI Whisper 等）
	cfg := a.configManager.Get()
	apiKey := cfg.ScreenshotAPIKey
	baseURL := cfg.ScreenshotBaseURL
	model := "paraformer-realtime-v2"
	if apiKey == "" {
		apiKey = cfg.APIKey
		baseURL = cfg.BaseURL
		model = "whisper-1"
	}
	if apiKey == "" {
		return `{"error":"API Key not configured"}`
	}

	baseURL = strings.TrimRight(baseURL, "/")
	if !strings.Contains(baseURL, "/v1") {
		baseURL += "/v1"
	}

	wavData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil || len(wavData) > maxDecodedAudioBytes {
		return `{"error":"音频内容无效或过大"}`
	}

	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	part, _ := writer.CreateFormFile("file", "capture.wav")
	part.Write(wavData)
	writer.WriteField("model", model)
	writer.WriteField("language", "zh")
	writer.Close()

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/audio/transcriptions", &b)
	if err != nil {
		return `{"error":"无法创建转写请求"}`
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return `{"error":"音频转写服务请求失败"}`
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil || len(body) > 1<<20 {
		return `{"error":"音频转写服务响应无效或过大"}`
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf(`{"error":"音频转写服务请求失败 (%d)"}`, resp.StatusCode)
	}

	var result struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &result); err != nil || len(result.Text) > maxChatContextBytes {
		return `{"error":"音频转写服务响应无效"}`
	}
	data, _ := json.Marshal(map[string]string{"text": result.Text})
	return string(data)
}

// AudioIsAvailable checks if Stereo Mix or loopback is available
func (a *App) AudioIsAvailable() string {
	if err := a.requireAuthenticated(); err != nil {
		return fmt.Sprintf(`{"error":%q}`, err.Error())
	}
	scriptPath := a.findAudioScript()
	if scriptPath == "" {
		return `{"available":false}`
	}

	cmd := exec.Command("python3", scriptPath, "list")
	output, err := cmd.Output()
	if err != nil {
		cmd = exec.Command("python", scriptPath, "list")
		output, err = cmd.Output()
		if err != nil {
			return `{"available":false}`
		}
	}

	var devices []AudioDevice
	if err := json.Unmarshal(output, &devices); err != nil {
		return `{"available":false}`
	}

	// Prefer true WASAPI loopback. It mirrors the playback endpoint without
	// changing the default output route, so speakers/headphones keep playing.
	for _, d := range devices {
		if d.Type == "system_audio" && d.IsLoopback {
			data, _ := json.Marshal(map[string]interface{}{
				"available":   true,
				"deviceId":    d.Index,
				"deviceName":  d.Name,
				"deviceType":  d.Type,
				"sampleRate":  d.SampleRate,
				"recommended": true,
				"backend":     d.Backend,
				"captureMode": d.CaptureMode,
			})
			return string(data)
		}
	}

	// Legacy fallback: Stereo Mix.
	for _, d := range devices {
		if d.Type == "stereo_mix" {
			data, _ := json.Marshal(map[string]interface{}{
				"available":   true,
				"deviceId":    d.Index,
				"deviceName":  d.Name,
				"deviceType":  d.Type,
				"sampleRate":  d.SampleRate,
				"recommended": true,
			})
			return string(data)
		}
	}

	// 其次默认设备
	for _, d := range devices {
		if d.IsDefault {
			data, _ := json.Marshal(map[string]interface{}{
				"available":  true,
				"deviceId":   d.Index,
				"deviceName": d.Name,
				"deviceType": d.Type,
				"sampleRate": d.SampleRate,
			})
			return string(data)
		}
	}

	// 最后第一个可用设备
	if len(devices) > 0 {
		data, _ := json.Marshal(map[string]interface{}{
			"available":  true,
			"deviceId":   devices[0].Index,
			"deviceName": devices[0].Name,
			"deviceType": devices[0].Type,
		})
		return string(data)
	}
	return `{"available":false}`
}

// AudioGetBestDevice 获取最佳音频输入设备
func (a *App) AudioGetBestDevice() string {
	return a.AudioIsAvailable()
}

// GenerateInterviewAnswer generates an interview answer using existing ChatWithDeepSeek
func (a *App) GenerateInterviewAnswer(query string) string {
	return a.ChatWithDeepSeek(query)
}

func (a *App) findAudioScript() string {
	// 当前工作目录可能不同，尝试多个路径
	paths := []string{
		"audio_capture.py",
		"../audio_capture.py",
		"./audio_capture.py",
	}
	// 检查文件是否真的存在
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	// 也检查项目根目录
	if _, err := os.Stat("audio_capture.py"); err == nil {
		return "audio_capture.py"
	}
	return ""
}

// findPythonExec 查找可用的 Python 解释器
func findPythonExec() string {
	paths := []string{"python3", "python"}
	for _, p := range paths {
		if _, err := exec.LookPath(p); err == nil {
			return p
		}
	}
	return "python3"
}

// findAudioCaptureScript 查找音频捕获脚本
func findAudioCaptureScript() string {
	paths := []string{
		"audio_capture.py",
		"../audio_capture.py",
		"./audio_capture.py",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
