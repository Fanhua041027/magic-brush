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
	Prompt             string                         `json:"prompt,omitempty"`
	DomainId           string                         `json:"domainId,omitempty"` // 行业/岗位选择ID
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

	// 辅助模型（用于总结对话生成问题导图）
	AssistantModel string `json:"assistantModel,omitempty"`

	// 窗口尺寸
	WindowWidth  int `json:"windowWidth,omitempty"`
	WindowHeight int `json:"windowHeight,omitempty"`

	// 主题（light / dark）
	Theme string `json:"theme,omitempty"`

	// STT 语音转写
	STTModel  string `json:"sttModel,omitempty"`
	STTDevice string `json:"sttDevice,omitempty"`

	// 知识库
	KBPath string `json:"kbPath,omitempty"`
}

const DefaultModel = ""

func NewDefaultConfig() Config {
	return Config{
		APIKey:             "",
		BaseURL:            "https://api.openai.com/v1",
		Model:              DefaultModel,
		ResumePath:         "",
		Prompt:             "",
		DomainId:           "general-assistant", // 默认选择通用的
		Opacity:            1.0,
		KeepContext:        false,
		InterruptThinking:  false,
		ScreenshotMode:     "fullscreen", // 默认全屏截图，确保捕获完整内容
		NoCompression:      false,        // 保持压缩以减小文件大小
		CompressionQuality: 92,           // 高质量压缩，确保 AI 清晰识别文字
		Sharpening:         0.3,          // 适度锐化，增强文字边缘清晰度
		Grayscale:          false,        // 保持彩色，某些场景颜色有意义
		ResumeContent:      "",

		Shortcuts: getDefaultShortcuts(),

		// 辅助模型
		AssistantModel: "",

		// 窗口尺寸默认值
		WindowWidth:  0,
		WindowHeight: 0,

		// 主题默认值
		Theme: "light",

		// STT
		STTModel:  "base",
		STTDevice: "auto",

		// 知识库
		KBPath: "",
	}
}

// getDefaultShortcuts 根据平台返回默认快捷键配置
func getDefaultShortcuts() map[string]shortcut.KeyBinding {
	if runtime.GOOS == "darwin" {
		// macOS 使用简化的快捷键（不依赖 Windows VK 码）
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
	// Windows 默认快捷键
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
