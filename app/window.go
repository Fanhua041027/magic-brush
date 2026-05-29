package app

import "ai-assistant/pkg/platform"

// ToggleVisibility 切换可见性
func (a *App) ToggleVisibility() {
	a.stateManager.ToggleVisibility()
}

// ToggleClickThrough 切换鼠标穿透
func (a *App) ToggleClickThrough() {
	a.stateManager.ToggleClickThrough()
}

// MoveWindow 移动窗口
func (a *App) MoveWindow(dx, dy int) {
	a.stateManager.MoveWindow(dx, dy)
}

// RestoreFocus 恢复焦点
func (a *App) RestoreFocus() {
	a.stateManager.RestoreFocus()
}

// RemoveFocus 移除焦点
func (a *App) RemoveFocus() {
	a.stateManager.RemoveFocus()
}

// SetWindowAlwaysOnTop 设置窗口是否置顶
func (a *App) SetWindowAlwaysOnTop(alwaysOnTop bool) {
	hwnd := a.stateManager.GetHwnd()
	if hwnd == 0 {
		return
	}
	if alwaysOnTop {
		platform.SetWindowLevel(hwnd, platform.WindowLevelFloating)
	} else {
		platform.SetWindowLevel(hwnd, platform.WindowLevelNormal)
	}
}
