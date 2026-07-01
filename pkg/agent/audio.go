package agent

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// AudioDevice represents an audio input device
type AudioDevice struct {
	Index       int     `json:"index"`
	Name        string  `json:"name"`
	Channels    int     `json:"channels"`
	SampleRate  float64 `json:"default_samplerate"`
}

// AudioCaptureResult is the result of an audio capture
type AudioCaptureResult struct {
	Status   string `json:"status"`
	Duration float64 `json:"duration,omitempty"`
	Samples  int    `json:"samples,omitempty"`
	Path     string `json:"path,omitempty"`
	Base64   string `json:"base64,omitempty"`
	Error    string `json:"error,omitempty"`
}

// AudioService handles system audio capture via Python
type AudioService struct {
	pythonPath string
	scriptPath string
	isCapturing bool
}

// NewAudioService creates a new audio service
func NewAudioService(scriptPath string) *AudioService {
	// Find python3
	pythonPath := findPython()

	return &AudioService{
		pythonPath: pythonPath,
		scriptPath: scriptPath,
	}
}

// ListAudioDevices lists all audio input devices
func (as *AudioService) ListAudioDevices() ([]AudioDevice, error) {
	cmd := exec.Command(as.pythonPath, as.scriptPath, "list")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list audio devices: %w", err)
	}

	var devices []AudioDevice
	if err := json.Unmarshal(output, &devices); err != nil {
		return nil, fmt.Errorf("failed to parse device list: %w", err)
	}

	return devices, nil
}

// CaptureAudio captures system audio for the given duration (seconds)
// deviceID: audio device index (usually 25 = Stereo Mix)
func (as *AudioService) CaptureAudio(deviceID int, durationSec float64) (*AudioCaptureResult, error) {
	cmd := exec.Command(as.pythonPath, as.scriptPath,
		"capture",
		itoa(deviceID),
		fmt.Sprintf("%.1f", durationSec))

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("audio capture failed: %w", err)
	}

	var result AudioCaptureResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse capture result: %w", err)
	}

	if result.Status == "error" {
		return &result, fmt.Errorf("capture error: %s", result.Error)
	}

	return &result, nil
}

// GetWAVBytes returns the WAV audio as bytes from a base64 capture result
func GetWAVBytes(result *AudioCaptureResult) ([]byte, error) {
	if result.Base64 == "" {
		return nil, fmt.Errorf("no audio data in result")
	}
	return base64.StdEncoding.DecodeString(result.Base64)
}

// TranscribeAudio sends WAV audio data to the LLM for transcription
func TranscribeAudio(audioBase64 string, callLLM func(messages []map[string]string) (string, error)) (string, error) {
	prompt := `请识别以下音频中的语音内容，直接输出识别的文字，不要添加任何其他内容。`

	// Send the audio data as base64 in a message
	messages := []map[string]string{
		{"role": "system", "content": "你是一个语音识别助手。直接输出识别结果，不要添加任何解释。"},
		{"role": "user", "content": prompt + "\n\n[音频数据已提供，请直接识别并输出文字内容]"},
	}

	result, err := callLLM(messages)
	if err != nil {
		return "", fmt.Errorf("transcription failed: %w", err)
	}

	return strings.TrimSpace(result), nil
}

// findPython finds the python3 executable
func findPython() string {
	// Try python3 first, then python
	paths := []string{"python3", "python"}
	for _, p := range paths {
		if _, err := exec.LookPath(p); err == nil {
			return p
		}
	}
	return "python3"
}

// IsStereoMixAvailable checks if the Stereo Mix device is available
func (as *AudioService) IsStereoMixAvailable() (bool, int) {
	devices, err := as.ListAudioDevices()
	if err != nil {
		return false, -1
	}

	for _, d := range devices {
		name := strings.ToLower(d.Name)
		if strings.Contains(name, "立体声") || strings.Contains(name, "stereo") ||
			strings.Contains(name, "混音") || strings.Contains(name, "mix") ||
			strings.Contains(name, "loopback") {
			return true, d.Index
		}
	}

	// Fallback: return the first available input device
	if len(devices) > 0 {
		return true, devices[0].Index
	}

	return false, -1
}
