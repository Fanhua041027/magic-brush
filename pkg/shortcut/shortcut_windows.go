//go:build windows

package shortcut

import (
	"ai-assistant/pkg/logger"
	"ai-assistant/pkg/platform"
	"syscall"
	"unsafe"
)

type Manager struct {
	hHook           uintptr
	hMouseHook      uintptr
	recordingKeyFor string
	maxComboKeys    map[uint32]bool
	heldKeys        map[uint32]bool

	// Callbacks
	OnTrigger           func(action string)
	OnRecord            func(action string, keyName string, comboID string)
	OnRecordingComplete func(action string, keyName string, comboID string)
	OnError             func(msg string)

	// Configuration
	Shortcuts map[string]KeyBinding
}

var globalManager *Manager

func NewManager() *Manager {
	return &Manager{
		heldKeys:     make(map[uint32]bool),
		maxComboKeys: make(map[uint32]bool),
		Shortcuts:    make(map[string]KeyBinding),
	}
}

func (m *Manager) Start() {
	globalManager = m
	go m.installHooks()
}

func (m *Manager) Stop() {
	if m.hHook != 0 {
		if platform.UnhookWindowsHookEx(m.hHook) {
			logger.Println("卸载键盘Hook成功")
		} else {
			logger.Println("卸载键盘Hook失败")
		}
		m.hHook = 0
	}
	if m.hMouseHook != 0 {
		if platform.UnhookWindowsHookEx(m.hMouseHook) {
			logger.Println("卸载鼠标Hook成功")
		} else {
			logger.Println("卸载鼠标Hook失败")
		}
		m.hMouseHook = 0
	}
	globalManager = nil
}

func (m *Manager) StartRecording(action string) {
	m.recordingKeyFor = action
	m.maxComboKeys = make(map[uint32]bool)
	logger.Printf("开始录制快捷键: %s\n", action)
}

func (m *Manager) StopRecording() {
	m.recordingKeyFor = ""
	logger.Println("停止录制快捷键")
}

func (m *Manager) installHooks() {
	// 获取模块句柄
	hMod := platform.GetModuleHandle("")

	// 创建键盘回调
	kbdCallback := syscall.NewCallback(keyboardHookProc)
	// 安装键盘钩子
	m.hHook = platform.SetWindowsHookEx(platform.WH_KEYBOARD_LL, kbdCallback, hMod, 0)
	if m.hHook == 0 {
		logger.Println("安装键盘钩子失败")
	}

	// 创建鼠标回调
	mouseCallback := syscall.NewCallback(mouseHookProc)
	// 安装鼠标钩子
	m.hMouseHook = platform.SetWindowsHookEx(platform.WH_MOUSE_LL, mouseCallback, hMod, 0)
	if m.hMouseHook == 0 {
		logger.Println("安装鼠标钩子失败")
	}

	if m.hHook == 0 && m.hMouseHook == 0 {
		return
	}

	// 消息循环
	var msg platform.MSG
	for platform.GetMessage(&msg, 0, 0, 0) > 0 {
		// 保持线程活跃以处理钩子消息
	}
}

// 这里解释了为什么只能吞掉第二个键，所以导致丢失焦点的问题：（其实是因为alt键的问题）
// 第一个键（Alt）按下：
// 记录：heldKeys = {Alt}
// 判断：有快捷键是只按 Alt 的吗？ -> 没有。
// 结果：放行（Chrome 收到 Alt）。
// 第二个键（~）按下：
// 记录：heldKeys = {Alt, ~}
// 判断：有快捷键是 Alt + ~ 的吗？ -> 有！
// 结果：拦截（return 1，Chrome 收不到 ~）。
func keyboardHookProc(nCode int, wParam uintptr, lParam uintptr) uintptr {
	if globalManager == nil {
		return 0
	}
	// 只有当 nCode >= 0 时才处理消息，否则直接放行
	if nCode >= 0 {
		// 将 lParam 指针转换为键盘钩子结构体
		kbd := (*platform.KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))
		// 监听按下事件 (WM_KEYDOWN) 或 系统按键按下 (WM_SYSKEYDOWN，比如按住 Alt 时)
		if wParam == platform.WM_KEYDOWN || wParam == platform.WM_SYSKEYDOWN {
			globalManager.heldKeys[kbd.VkCode] = true
			if onKeysChanged() {
				return 1
			}
		}
		// 处理松开事件
		if wParam == platform.WM_KEYUP || wParam == platform.WM_SYSKEYUP {
			// 1. 从 map 中移除该键
			delete(globalManager.heldKeys, kbd.VkCode)

			// 录制模式下，松开按键也要检查是否结束录制
			if globalManager.recordingKeyFor != "" {
				if len(globalManager.heldKeys) == 0 {
					finishRecording()
				}
				return 1 // 录制期间吞掉所有按键
			}
		}
	}

	// 如果不是我们要拦截的键，或者 nCode < 0，必须调用 CallNextHookEx
	// 否则会导致系统键盘卡死或其他人无法使用键盘
	return platform.CallNextHookEx(globalManager.hHook, nCode, wParam, lParam)
}

func mouseHookProc(nCode int, wParam uintptr, lParam uintptr) uintptr {
	if globalManager == nil {
		return 0
	}
	if nCode >= 0 {
		mouseStruct := (*platform.MSLLHOOKSTRUCT)(unsafe.Pointer(lParam))
		var vkCode uint32
		isDown := false
		isUp := false

		switch wParam {
		case platform.WM_XBUTTONDOWN:
			isDown = true
			xButton := (mouseStruct.MouseData >> 16) & 0xFFFF
			switch xButton {
			case 1:
				vkCode = platform.VK_XBUTTON1
			case 2:
				vkCode = platform.VK_XBUTTON2
			}
		case platform.WM_XBUTTONUP:
			isUp = true
			xButton := (mouseStruct.MouseData >> 16) & 0xFFFF
			switch xButton {
			case 1:
				vkCode = platform.VK_XBUTTON1
			case 2:
				vkCode = platform.VK_XBUTTON2
			}
		}

		if vkCode != 0 {
			if isDown {
				globalManager.heldKeys[vkCode] = true
				if onKeysChanged() {
					return 1
				}
			} else if isUp {
				delete(globalManager.heldKeys, vkCode)
				// 录制模式下，松开按键也要检查是否结束录制
				if globalManager.recordingKeyFor != "" {
					if len(globalManager.heldKeys) == 0 {
						finishRecording()
					}
					return 1
				}
			}
		}
	}
	return platform.CallNextHookEx(globalManager.hMouseHook, nCode, wParam, lParam)
}

func onKeysChanged() bool {
	if globalManager == nil {
		return false
	}

	// --- 录制模式 ---
	if globalManager.recordingKeyFor != "" {
		// 更新最大按键组合
		if len(globalManager.heldKeys) >= len(globalManager.maxComboKeys) {
			globalManager.maxComboKeys = make(map[uint32]bool)
			for k, v := range globalManager.heldKeys {
				globalManager.maxComboKeys[k] = v
			}
		}

		// 实时发给前端显示
		readableName := GetReadableName(globalManager.maxComboKeys)
		if globalManager.OnRecord != nil {
			globalManager.OnRecord(globalManager.recordingKeyFor, readableName, GetComboID(globalManager.maxComboKeys))
		}
		return true // 吞掉按键
	}

	// --- 正常模式 ---
	// 将当前按下的所有键生成 ID，去配置里查
	currentComboID := GetComboID(globalManager.heldKeys)
	for action, savedComboID := range globalManager.Shortcuts {
		if savedComboID.ComboID == currentComboID {
			// 检查是否包含 Alt 键 (VK_MENU=18, VK_LMENU=164, VK_RMENU=165)
			// 或者 Win 键 (VK_LWIN=91, VK_RWIN=92)
			// 如果包含这些键，且其他键被我们吞掉了，Windows 会认为用户只按了 Alt/Win，从而激活菜单栏或开始菜单
			hasAlt := globalManager.heldKeys[18] || globalManager.heldKeys[164] || globalManager.heldKeys[165]
			hasWin := globalManager.heldKeys[91] || globalManager.heldKeys[92]

			if hasAlt || hasWin {
				// 模拟按下并松开 Ctrl 键，防止 Windows 激活菜单栏/开始菜单
				// VK_CONTROL = 0x11, KEYEVENTF_KEYUP = 0x0002
				platform.KeybdEvent(platform.VK_CONTROL, 0, 0, 0)
				platform.KeybdEvent(platform.VK_CONTROL, 0, 2, 0)
			}

			if globalManager.OnTrigger != nil {
				globalManager.OnTrigger(action)
			}
			return true // 吞掉按键
		}
	}
	return false
}

func finishRecording() {
	if globalManager == nil || globalManager.recordingKeyFor == "" {
		return
	}

	// 如果没有按任何键（比如直接点击录制然后点别的），忽略
	if len(globalManager.maxComboKeys) == 0 {
		globalManager.recordingKeyFor = ""
		return
	}

	comboID := GetComboID(globalManager.maxComboKeys)
	readableName := GetReadableName(globalManager.maxComboKeys)
	action := globalManager.recordingKeyFor

	// 退出录制模式
	globalManager.recordingKeyFor = ""
	globalManager.maxComboKeys = nil

	// 异步调用回调，避免阻塞 Hook 线程
	go func() {
		if globalManager.OnRecordingComplete != nil {
			globalManager.OnRecordingComplete(action, readableName, comboID)
		}
	}()
}
