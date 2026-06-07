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
	AssistantModel     string                         `json:"assistantModel,omitempty"`
	WindowWidth        int                            `json:"windowWidth,omitempty"`
	WindowHeight       int                            `json:"windowHeight,omitempty"`
	Theme              string                         `json:"theme,omitempty"`
	STTModel           string                         `json:"sttModel,omitempty"`
	STTDevice          string                         `json:"sttDevice,omitempty"`
	STTLanguage        string                         `json:"sttLanguage,omitempty"`
	STTSensitivity     float64                        `json:"sttSensitivity,omitempty"`
	STTService         string                         `json:"sttService,omitempty"`
	KBPath             string                         `json:"kbPath,omitempty"`
}

const DefaultModel = ""

func NewDefaultConfig() Config {
	return Config{
		APIKey:             "",
		BaseURL:            "https://api.openai.com/v1",
		Model:              DefaultModel,
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
		Shortcuts:          getDefaultShortcuts(),
		AssistantModel:     "",
		WindowWidth:        0,
		WindowHeight:       0,
		Theme:              "light",
		STTModel:           "base",
		STTDevice:          "auto",
		STTLanguage:        "zh",
		STTSensitivity:     0.5,
		STTService:         "qwen",
		KBPath:             "",
	}
}

func getDefaultShortcuts() map[string]shortcut.KeyBinding {
	if runtime.GOOS == "darwin" {
		return map[string]shortcut.KeyBinding{
			"solve":        {ComboID: "Cmd+1", KeyName: "Cmd+1"},
			"send":         {ComboID: "Cmd+J", KeyName: "Cmd+J"},
			"delete":       {ComboID: "Cmd+D", KeyName: "Cmd+D"},
			"toggle":       {ComboID: "Cmd+2", KeyName: "Cmd+2"},
			"clickthrough": {ComboID: "Cmd+3", KeyName: "Cmd+3"},
			"move_up":      {ComboID: "Cmd+Option+Up", KeyName: "Cmd+Option+Up"},
			"move_down":    {ComboID: "Cmd+Option+Down", KeyName: "Cmd+Option+Down"},
			"move_left":    {ComboID: "Cmd+Option+Left", KeyName: "Cmd+Option+Left"},
			"move_right":   {ComboID: "Cmd+Option+Right", KeyName: "Cmd+Option+Right"},
			"scroll_up":    {ComboID: "Cmd+Option+Shift+Up", KeyName: "Cmd+Option+Shift+Up"},
			"scroll_down":  {ComboID: "Cmd+Option+Shift+Down", KeyName: "Cmd+Option+Shift+Down"},
		}
	}

	return map[string]shortcut.KeyBinding{
		"screenshot":   {ComboID: "119", KeyName: "F8"},
		"send":         {ComboID: "74+162", KeyName: "Ctrl+J"},
		"delete":       {ComboID: "68+162", KeyName: "Ctrl+D"},
		"toggle":       {ComboID: "120", KeyName: "F9"},
		"clickthrough": {ComboID: "121", KeyName: "F10"},
		"chat":         {ComboID: "118", KeyName: "F7"},
		"move_up":      {ComboID: "38+164", KeyName: "Alt+Up"},
		"move_down":    {ComboID: "40+164", KeyName: "Alt+Down"},
		"move_left":    {ComboID: "37+164", KeyName: "Alt+Left"},
		"move_right":   {ComboID: "39+164", KeyName: "Alt+Right"},
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
		return &ValidationError{Field: "screenshotMode", Message: "screenshot mode must be 'fullscreen' or 'window'"}
	}
	if c.Opacity < 0 || c.Opacity > 1 {
		return &ValidationError{Field: "opacity", Message: "opacity must be between 0 and 1"}
	}
	if c.CompressionQuality < 1 || c.CompressionQuality > 100 {
		return &ValidationError{Field: "compressionQuality", Message: "compression quality must be between 1 and 100"}
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
