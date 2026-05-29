package app

import "ai-assistant/pkg/config"

// GetSettings 返回当前配置
func (a *App) GetSettings() config.Config {
	return a.configManager.Get()
}

// UpdateSettings 更新配置（从前端 JSON）
func (a *App) UpdateSettings(configJson string) string {
	if err := a.configManager.UpdateFromJSON(configJson); err != nil {
		return err.Error()
	}
	return ""
}
