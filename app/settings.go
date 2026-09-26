package app

import (
	"ai-assistant/pkg/config"
	"unicode/utf8"
)

const maxSettingsJSONBytes = 1 << 20

// GetSettings returns a public configuration view. Credentials are write-only.
func (a *App) GetSettings() config.PublicConfig {
	return a.configManager.Get().Public()
}

// UpdateSettings updates configuration from frontend JSON.
func (a *App) UpdateSettings(configJSON string) string {
	if len(configJSON) > maxSettingsJSONBytes || !utf8.ValidString(configJSON) {
		return "配置内容无效或过大"
	}
	if err := a.configManager.UpdateFromJSON(configJSON); err != nil {
		return err.Error()
	}
	return ""
}
