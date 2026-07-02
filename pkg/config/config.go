package config

import (
	"ai-assistant/pkg/shortcut"
	"encoding/json"
	"runtime"
)

type Config struct {
	APIKey             string                         `json:"apiKey,omitempty"`
	BaseURL            string                         `json:"baseURL,omitempty"`
	Model              string                         `json:"model,omitempty"`

	// 截图专用视觉模型（如 Qwen-VL），与 F7 聊天模型分离
	ScreenshotAPIKey   string                         `json:"screenshotApiKey,omitempty"`
	ScreenshotBaseURL  string                         `json:"screenshotBaseUrl,omitempty"`
	ScreenshotModel    string                         `json:"screenshotModel,omitempty"`

	Prompt             string                         `json:"prompt,omitempty"`
	DomainId           string                         `json:"domainId,omitempty"`
	Opacity            float64                        `json:"opacity,omitempty"`
	NoCompression      bool                           `json:"noCompression,omitempty"`
	CompressionQuality int                            `json:"compressionQuality,omitempty"`
	Sharpening         float64                        `json:"sharpening,omitempty"`
	Grayscale          bool                           `json:"grayscale,omitempty"`
	KeepContext        bool                           `json:"keepContext,omitempty"`
	InterruptThinking  bool                           `json:"interruptThinking,omitempty"`
	ScreenshotMode     string                         `json:"screenshotMode,omitempty"`
	ResumePath         string                         `json:"resumePath,omitempty"`
	ResumeContent      string                         `json:"resumeContent,omitempty"`
	Shortcuts          map[string]shortcut.KeyBinding `json:"shortcuts,omitempty"`

	AssistantModel string `json:"assistantModel,omitempty"`

	WindowWidth  int `json:"windowWidth,omitempty"`
	WindowHeight int `json:"windowHeight,omitempty"`

	Theme string `json:"theme,omitempty"`

	STTModel       string  `json:"sttModel,omitempty"`
	STTDevice      string  `json:"sttDevice,omitempty"`
	STTLanguage    string  `json:"sttLanguage,omitempty"`
	STTSensitivity float64 `json:"sttSensitivity,omitempty"`
	STTService     string  `json:"sttService,omitempty"`

	KBPath string `json:"kbPath,omitempty"`
}

const DefaultModel = ""

const (
	DefaultScreenshotModel   = "qwen3.6-flash"
	DefaultScreenshotBaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
)

func NewDefaultConfig() Config {
	return Config{
		APIKey:             "",
		BaseURL:            "https://api.openai.com/v1",
		Model:              DefaultModel,
		ScreenshotModel:    DefaultScreenshotModel,
		ScreenshotBaseURL:  DefaultScreenshotBaseURL,
		ResumePath:         "",
		Prompt:             "",
		DomainId:           "general-assistant",
		Opacity:            1.0,
		KeepContext:        false,
		InterruptThinking:  false,
		ScreenshotMode:     "fullscreen",
		NoCompression:      false,
		CompressionQuality: 92,
		Sharpening:         0.3,
		Grayscale:          false,
		ResumeContent:      "",

		Shortcuts: getDefaultShortcuts(),

		AssistantModel: "",

		WindowWidth:  0,
		WindowHeight: 0,

		Theme: "light",

		STTModel:       "base",
		STTDevice:      "auto",
		STTLanguage:    "zh",
		STTSensitivity: 0.5,
		STTService:     "qwen_local",

		KBPath: "",
	}
}

func getDefaultShortcuts() map[string]shortcut.KeyBinding {
	if runtime.GOOS == "darwin" {
		return map[string]shortcut.KeyBinding{
			"solve":        {ComboID: "Cmd+1", KeyName: "⌘1"},
			"send":         {ComboID: "Cmd+J", KeyName: "⌘J"},
			"delete":       {ComboID: "Cmd+D", KeyName: "⌘D"},
			"toggle":       {ComboID: "Cmd+2", KeyName: "⌘2"},
			"clickthrough": {ComboID: "Cmd+3", KeyName: "⌘3"},
			"move_up":      {ComboID: "Cmd+Option+Up", KeyName: "⌘⌥↑"},
			"move_down":    {ComboID: "Cmd+Option+Down", KeyName: "⌘⌥↓"},
			"move_left":    {ComboID: "Cmd+Option+Left", KeyName: "⌘⌥←"},
			"move_right":   {ComboID: "Cmd+Option+Right", KeyName: "⌘⌥→"},
			"scroll_up":    {ComboID: "Cmd+Option+Shift+Up", KeyName: "⌘⌥⇧↑"},
			"scroll_down":  {ComboID: "Cmd+Option+Shift+Down", KeyName: "⌘⌥⇧↓"},
		}
	}
	return map[string]shortcut.KeyBinding{
		"screenshot":   {ComboID: "119", KeyName: "F8"},
		"send":         {ComboID: "74+162", KeyName: "Ctrl+J"},
		"delete":       {ComboID: "68+162", KeyName: "Ctrl+D"},
		"toggle":       {ComboID: "120", KeyName: "F9"},
		"clickthrough": {ComboID: "121", KeyName: "F10"},
		"chat":         {ComboID: "118", KeyName: "F7"},
		"move_up":      {ComboID: "38+164", KeyName: "Alt+↑"},
		"move_down":    {ComboID: "40+164", KeyName: "Alt+↓"},
		"move_left":    {ComboID: "37+164", KeyName: "Alt+←"},
		"move_right":   {ComboID: "39+164", KeyName: "Alt+→"},
		"scroll_up":    {ComboID: "33+164", KeyName: "Alt+PgUp"},
		"scroll_down":  {ComboID: "34+164", KeyName: "Alt+PgDn"},
		"standalone":   {ComboID: "112", KeyName: "F1"},
		"standalone2":  {ComboID: "90+162+164", KeyName: "Ctrl+Alt+Z"},
	}
}

func (c *Config) ToJSON() string {
	data, _ := json.MarshalIndent(c, "", "  ")
	return string(data)
}

func (c *Config) Validate() error {
	if c.ScreenshotMode != "" && c.ScreenshotMode != "fullscreen" && c.ScreenshotMode != "window" {
		return &ValidationError{Field: "screenshotMode", Message: "截图模式必须是 'fullscreen' 或 'window'"}
	}
	if c.Opacity < 0 || c.Opacity > 1 {
		return &ValidationError{Field: "opacity", Message: "透明度必须在 0-1 之间"}
	}
	if c.CompressionQuality < 1 || c.CompressionQuality > 100 {
		return &ValidationError{Field: "compressionQuality", Message: "压缩质量必须在 1-100 之间"}
	}
	return nil
}

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
