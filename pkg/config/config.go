package config

import (
	"ai-assistant/pkg/shortcut"
	"encoding/json"
	"math"
	"net"
	"net/url"
	"runtime"
	"strings"
	"unicode/utf8"
)

type Config struct {
	APIKey  string `json:"apiKey,omitempty"`
	BaseURL string `json:"baseURL,omitempty"`
	Model   string `json:"model,omitempty"`

	// 截图专用视觉模型（如 Qwen-VL），与 F7 聊天模型分离
	ScreenshotAPIKey  string `json:"screenshotApiKey,omitempty"`
	ScreenshotBaseURL string `json:"screenshotBaseUrl,omitempty"`
	ScreenshotModel   string `json:"screenshotModel,omitempty"`

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

	AnswerLength            string `json:"answerLength,omitempty"`
	AnswerStyle             string `json:"answerStyle,omitempty"`
	AnswerStructure         string `json:"answerStructure,omitempty"`
	AnswerDuration          string `json:"answerDuration,omitempty"`
	IncludeTechnicalDetails bool   `json:"includeTechnicalDetails,omitempty"`

	KBPath string `json:"kbPath,omitempty"`
}

type PublicConfig struct {
	APIKeyConfigured           bool                           `json:"apiKeyConfigured"`
	BaseURL                    string                         `json:"baseURL,omitempty"`
	Model                      string                         `json:"model,omitempty"`
	ScreenshotAPIKeyConfigured bool                           `json:"screenshotApiKeyConfigured"`
	ScreenshotBaseURL          string                         `json:"screenshotBaseUrl,omitempty"`
	ScreenshotModel            string                         `json:"screenshotModel,omitempty"`
	Prompt                     string                         `json:"prompt,omitempty"`
	DomainId                   string                         `json:"domainId,omitempty"`
	Opacity                    float64                        `json:"opacity,omitempty"`
	NoCompression              bool                           `json:"noCompression,omitempty"`
	CompressionQuality         int                            `json:"compressionQuality,omitempty"`
	Sharpening                 float64                        `json:"sharpening,omitempty"`
	Grayscale                  bool                           `json:"grayscale,omitempty"`
	KeepContext                bool                           `json:"keepContext,omitempty"`
	InterruptThinking          bool                           `json:"interruptThinking,omitempty"`
	ScreenshotMode             string                         `json:"screenshotMode,omitempty"`
	ResumePath                 string                         `json:"resumePath,omitempty"`
	ResumeContent              string                         `json:"resumeContent,omitempty"`
	Shortcuts                  map[string]shortcut.KeyBinding `json:"shortcuts,omitempty"`
	AssistantModel             string                         `json:"assistantModel,omitempty"`
	WindowWidth                int                            `json:"windowWidth,omitempty"`
	WindowHeight               int                            `json:"windowHeight,omitempty"`
	Theme                      string                         `json:"theme,omitempty"`
	STTModel                   string                         `json:"sttModel,omitempty"`
	STTDevice                  string                         `json:"sttDevice,omitempty"`
	STTLanguage                string                         `json:"sttLanguage,omitempty"`
	STTSensitivity             float64                        `json:"sttSensitivity,omitempty"`
	STTService                 string                         `json:"sttService,omitempty"`
	AnswerLength               string                         `json:"answerLength,omitempty"`
	AnswerStyle                string                         `json:"answerStyle,omitempty"`
	AnswerStructure            string                         `json:"answerStructure,omitempty"`
	AnswerDuration             string                         `json:"answerDuration,omitempty"`
	IncludeTechnicalDetails    bool                           `json:"includeTechnicalDetails,omitempty"`
	KBPath                     string                         `json:"kbPath,omitempty"`
}

func (c Config) Public() PublicConfig {
	shortcuts := make(map[string]shortcut.KeyBinding, len(c.Shortcuts))
	for action, binding := range c.Shortcuts {
		shortcuts[action] = binding
	}
	return PublicConfig{
		APIKeyConfigured:           c.APIKey != "",
		BaseURL:                    c.BaseURL,
		Model:                      c.Model,
		ScreenshotAPIKeyConfigured: c.ScreenshotAPIKey != "",
		ScreenshotBaseURL:          c.ScreenshotBaseURL,
		ScreenshotModel:            c.ScreenshotModel,
		Prompt:                     c.Prompt,
		DomainId:                   c.DomainId,
		Opacity:                    c.Opacity,
		NoCompression:              c.NoCompression,
		CompressionQuality:         c.CompressionQuality,
		Sharpening:                 c.Sharpening,
		Grayscale:                  c.Grayscale,
		KeepContext:                c.KeepContext,
		InterruptThinking:          c.InterruptThinking,
		ScreenshotMode:             c.ScreenshotMode,
		ResumePath:                 c.ResumePath,
		ResumeContent:              c.ResumeContent,
		Shortcuts:                  shortcuts,
		AssistantModel:             c.AssistantModel,
		WindowWidth:                c.WindowWidth,
		WindowHeight:               c.WindowHeight,
		Theme:                      c.Theme,
		STTModel:                   c.STTModel,
		STTDevice:                  c.STTDevice,
		STTLanguage:                c.STTLanguage,
		STTSensitivity:             c.STTSensitivity,
		STTService:                 c.STTService,
		AnswerLength:               c.AnswerLength,
		AnswerStyle:                c.AnswerStyle,
		AnswerStructure:            c.AnswerStructure,
		AnswerDuration:             c.AnswerDuration,
		IncludeTechnicalDetails:    c.IncludeTechnicalDetails,
		KBPath:                     c.KBPath,
	}
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

		STTModel:                "qwen-audio-3.0-asr-flash",
		STTDevice:               "auto",
		STTLanguage:             "zh",
		STTSensitivity:          0.5,
		STTService:              "qwen_cloud",
		AnswerLength:            "standard",
		AnswerStyle:             "natural",
		AnswerStructure:         "free",
		AnswerDuration:          "1m",
		IncludeTechnicalDetails: true,

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

func validateServiceURL(field, value string) error {
	if value == "" {
		return nil
	}
	if len(value) > 2048 || !utf8.ValidString(value) {
		return &ValidationError{Field: field, Message: "接口地址无效或过长"}
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return &ValidationError{Field: field, Message: "接口地址无效"}
	}
	if parsed.Scheme == "https" {
		return nil
	}
	host := parsed.Hostname()
	if parsed.Scheme == "http" && (host == "localhost" || net.ParseIP(host).IsLoopback()) {
		return nil
	}
	return &ValidationError{Field: field, Message: "仅允许 HTTPS 或本机 HTTP 地址"}
}

func validateLength(field, value string, max int) error {
	if !utf8.ValidString(value) || len(value) > max {
		return &ValidationError{Field: field, Message: "内容无效或过长"}
	}
	return nil
}

func (c *Config) Validate() error {
	for _, item := range []struct {
		field string
		value string
		max   int
	}{
		{"apiKey", c.APIKey, 4096}, {"screenshotApiKey", c.ScreenshotAPIKey, 4096},
		{"model", c.Model, 256}, {"screenshotModel", c.ScreenshotModel, 256}, {"assistantModel", c.AssistantModel, 256},
		{"prompt", c.Prompt, 64 << 10}, {"resumeContent", c.ResumeContent, 1 << 20},
		{"resumePath", c.ResumePath, 4096}, {"kbPath", c.KBPath, 4096}, {"sttDevice", c.STTDevice, 512},
	} {
		if err := validateLength(item.field, item.value, item.max); err != nil {
			return err
		}
	}
	if err := validateServiceURL("baseURL", strings.TrimSpace(c.BaseURL)); err != nil {
		return err
	}
	if err := validateServiceURL("screenshotBaseUrl", strings.TrimSpace(c.ScreenshotBaseURL)); err != nil {
		return err
	}
	if len(c.Shortcuts) > 64 {
		return &ValidationError{Field: "shortcuts", Message: "快捷键数量过多"}
	}
	for action, binding := range c.Shortcuts {
		if err := validateLength("shortcuts", action+binding.ComboID+binding.KeyName, 512); err != nil {
			return err
		}
	}
	if c.ScreenshotMode != "" && c.ScreenshotMode != "fullscreen" && c.ScreenshotMode != "window" {
		return &ValidationError{Field: "screenshotMode", Message: "截图模式必须是 'fullscreen' 或 'window'"}
	}
	if math.IsNaN(c.Opacity) || math.IsInf(c.Opacity, 0) || c.Opacity < 0 || c.Opacity > 1 {
		return &ValidationError{Field: "opacity", Message: "透明度必须在 0-1 之间"}
	}
	if c.CompressionQuality < 1 || c.CompressionQuality > 100 {
		return &ValidationError{Field: "compressionQuality", Message: "压缩质量必须在 1-100 之间"}
	}
	if math.IsNaN(c.Sharpening) || math.IsInf(c.Sharpening, 0) || c.Sharpening < 0 || c.Sharpening > 10 {
		return &ValidationError{Field: "sharpening", Message: "锐化强度必须在 0-10 之间"}
	}
	if math.IsNaN(c.STTSensitivity) || math.IsInf(c.STTSensitivity, 0) || c.STTSensitivity < 0 || c.STTSensitivity > 1 {
		return &ValidationError{Field: "sttSensitivity", Message: "语音灵敏度必须在 0-1 之间"}
	}
	if c.WindowWidth < 0 || c.WindowWidth > 10000 || c.WindowHeight < 0 || c.WindowHeight > 10000 {
		return &ValidationError{Field: "windowSize", Message: "窗口尺寸无效"}
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
