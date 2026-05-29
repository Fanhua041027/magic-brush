package app

import (
	"ai-assistant/pkg/platform"
	"ai-assistant/pkg/screen"
)

// GetScreenshotPreview 获取截图预览
func (a *App) GetScreenshotPreview(quality int, sharpen float64, grayscale bool, noCompression bool, screenshotMode string) (screen.PreviewResult, error) {
	mode := screenshotMode
	if mode == "" {
		mode = a.configManager.Get().ScreenshotMode
	}
	return a.screenService.CapturePreview(quality, sharpen, grayscale, noCompression, mode)
}

// CheckScreenCapturePermission 检查截图权限 (macOS)
func (a *App) CheckScreenCapturePermission() bool {
	return platform.CheckScreenCaptureAccess()
}

// RequestScreenCapturePermission 请求截图权限 (macOS)
func (a *App) RequestScreenCapturePermission() bool {
	return platform.RequestScreenCaptureAccess()
}

// OpenScreenCaptureSettings 打开系统设置的屏幕录制权限页面 (macOS)
func (a *App) OpenScreenCaptureSettings() {
	platform.OpenScreenCaptureSettings()
}

// CheckMicrophoneAccess 检查麦克风权限状态 (macOS)
// 返回: 0=未决定, 1=已授权, 2=已拒绝
func (a *App) CheckMicrophoneAccess() int {
	return platform.CheckMicrophoneAccess()
}

// RequestMicrophoneAccess 请求麦克风权限 (macOS)
func (a *App) RequestMicrophoneAccess() {
	platform.RequestMicrophoneAccess()
}

// OpenMicrophoneSettings 打开系统设置的麦克风权限页面 (macOS)
func (a *App) OpenMicrophoneSettings() {
	platform.OpenMicrophoneSettings()
}
