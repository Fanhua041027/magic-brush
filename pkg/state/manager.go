package state

import (
	"ai-assistant/pkg/logger"
	"ai-assistant/pkg/platform"
	"context"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// InitStatus 初始化状态
type InitStatus string

const (
	StatusInitializing InitStatus = "initializing"
	StatusLoadingModel InitStatus = "loading-model"
	StatusReady        InitStatus = "ready"
	StatusError        InitStatus = "error"
)

// WindowState 窗口状态
type WindowState struct {
	Visible      bool
	ClickThrough bool
	StealthMode  bool
}

// StateManager 管理应用全局状态
type StateManager struct {
	ctx context.Context

	// 窗口状态
	windowState WindowState
	windowMu    sync.RWMutex
	hwnd        platform.WindowHandle

	// 初始化状态
	initStatus   InitStatus
	initStatusMu sync.RWMutex

	// 事件发送器
	emitEvent func(string, ...interface{})
}

// NewStateManager 创建状态管理器
func NewStateManager() *StateManager {
	return &StateManager{
		windowState: WindowState{
			Visible:      true,
			ClickThrough: false,
			StealthMode:  true,
		},
		initStatus: StatusInitializing,
	}
}

// Startup 启动状态管理器
func (sm *StateManager) Startup(ctx context.Context, emitEvent func(string, ...interface{})) {
	sm.ctx = ctx
	sm.emitEvent = emitEvent

	// 异步初始化窗口句柄
	sm.initWindowHandle()
}

// initWindowHandle 初始化窗口句柄
func (sm *StateManager) initWindowHandle() {
	const maxRetries = 10
	const retryInterval = 500 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		hwnd, err := platform.GetWindowHandle()
		if err == nil && hwnd != 0 {
			sm.windowMu.Lock()
			sm.hwnd = hwnd
			sm.windowMu.Unlock()

			if err := platform.ApplyGhostMode(hwnd); err != nil {
				logger.Printf("应用幽灵模式失败: %v\n", err)
			} else {
				logger.Println("幽灵模式已激活 (无边框/隐身/防抢焦/隐藏任务栏)")
			}
			return
		}
		time.Sleep(retryInterval)
	}
	logger.Println("无法找到主窗口以启用隐身模式")
}

// GetHwnd 获取窗口句柄
func (sm *StateManager) GetHwnd() platform.WindowHandle {
	sm.windowMu.RLock()
	defer sm.windowMu.RUnlock()
	return sm.hwnd
}

// GetInitStatus 获取初始化状态
func (sm *StateManager) GetInitStatus() InitStatus {
	sm.initStatusMu.RLock()
	defer sm.initStatusMu.RUnlock()
	return sm.initStatus
}

// GetInitStatusString 获取初始化状态字符串
func (sm *StateManager) GetInitStatusString() string {
	return string(sm.GetInitStatus())
}

// UpdateInitStatus 更新初始化状态
func (sm *StateManager) UpdateInitStatus(status InitStatus) {
	sm.initStatusMu.Lock()
	sm.initStatus = status
	sm.initStatusMu.Unlock()

	if sm.emitEvent != nil {
		sm.emitEvent("init-status", string(status))
	}
}

// IsReady 检查是否已就绪
func (sm *StateManager) IsReady() bool {
	return sm.GetInitStatus() == StatusReady
}

// GetWindowState 获取窗口状态
func (sm *StateManager) GetWindowState() WindowState {
	sm.windowMu.RLock()
	defer sm.windowMu.RUnlock()
	return sm.windowState
}

// IsVisible 检查是否可见
func (sm *StateManager) IsVisible() bool {
	sm.windowMu.RLock()
	defer sm.windowMu.RUnlock()
	return sm.windowState.Visible
}

// IsClickThrough 检查是否鼠标穿透
func (sm *StateManager) IsClickThrough() bool {
	sm.windowMu.RLock()
	defer sm.windowMu.RUnlock()
	return sm.windowState.ClickThrough
}

// ToggleVisibility 切换可见性（隐身模式）
func (sm *StateManager) ToggleVisibility() bool {
	sm.windowMu.Lock()
	defer sm.windowMu.Unlock()

	if sm.hwnd == 0 {
		logger.Println("无法切换可见性：窗口句柄未初始化")
		return sm.windowState.Visible
	}

	if sm.windowState.Visible {
		// 禁用隐身模式，可被录屏检测
		err := platform.SetDisplayAffinity(sm.hwnd, false)
		if err != nil {
			logger.Printf("设置显示亲和性失败: %v\n", err)
		} else {
			logger.Println("隐身模式已禁用，现在可被录屏程序检测到")
		}
	} else {
		// 启用隐身模式
		err := platform.SetDisplayAffinity(sm.hwnd, true)
		if err != nil {
			logger.Printf("设置显示亲和性失败: %v\n", err)
		} else {
			logger.Println("隐身模式已启用，录屏程序现在无法检测")
		}
	}

	sm.windowState.Visible = !sm.windowState.Visible

	if sm.emitEvent != nil {
		sm.emitEvent("toggle-visibility", sm.windowState.Visible)
	}

	return sm.windowState.Visible
}

// ToggleClickThrough 切换鼠标穿透
func (sm *StateManager) ToggleClickThrough() bool {
	sm.windowMu.Lock()
	defer sm.windowMu.Unlock()

	if sm.hwnd == 0 {
		logger.Println("无法切换鼠标穿透：窗口句柄未初始化")
		return sm.windowState.ClickThrough
	}

	newState := !sm.windowState.ClickThrough
	err := platform.SetClickThrough(sm.hwnd, newState)
	if err != nil {
		logger.Printf("切换鼠标穿透失败: %v\n", err)
		return sm.windowState.ClickThrough
	}

	sm.windowState.ClickThrough = newState

	if newState {
		logger.Println("鼠标穿透已启用")
	} else {
		logger.Println("鼠标穿透已禁用")
	}

	if sm.emitEvent != nil {
		sm.emitEvent("click-through-state", newState)
	}

	return newState
}

// RestoreFocus 恢复焦点
func (sm *StateManager) RestoreFocus() {
	sm.windowMu.RLock()
	hwnd := sm.hwnd
	sm.windowMu.RUnlock()

	if hwnd != 0 {
		platform.RestoreFocus(hwnd)
	}
}

// RemoveFocus 移除焦点
func (sm *StateManager) RemoveFocus() {
	sm.windowMu.RLock()
	hwnd := sm.hwnd
	sm.windowMu.RUnlock()

	if hwnd != 0 {
		platform.RemoveFocus(hwnd)
	}
}

// MoveWindow 移动窗口
func (sm *StateManager) MoveWindow(dx, dy int) {
	if sm.ctx == nil {
		return
	}
	x, y := runtime.WindowGetPosition(sm.ctx)
	runtime.WindowSetPosition(sm.ctx, x+dx, y+dy)
}
