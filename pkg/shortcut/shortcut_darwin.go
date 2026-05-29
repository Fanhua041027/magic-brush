//go:build darwin

package shortcut

import (
	"ai-assistant/pkg/logger"

	"golang.design/x/hotkey"
)

type Manager struct {
	hotkeys   map[string]*hotkey.Hotkey
	Shortcuts map[string]KeyBinding
	stopChan  chan struct{}
	running   bool

	// Callbacks
	OnTrigger           func(action string)
	OnRecord            func(action string, keyName string, comboID string)
	OnRecordingComplete func(action string, keyName string, comboID string)
	OnError             func(msg string)
}

// macOS 默认快捷键映射 - 使用 Command + 数字键
var macDefaultShortcuts = map[string]struct {
	mods []hotkey.Modifier
	key  hotkey.Key
}{
	"screenshot":   {[]hotkey.Modifier{hotkey.ModCmd}, hotkey.Key1},
	"send":         {[]hotkey.Modifier{hotkey.ModCmd}, hotkey.KeyJ},
	"delete":       {[]hotkey.Modifier{hotkey.ModCmd}, hotkey.KeyD},
	"toggle":       {[]hotkey.Modifier{hotkey.ModCmd}, hotkey.Key2},
	"clickthrough": {[]hotkey.Modifier{hotkey.ModCmd}, hotkey.Key3},
	// 方向键快捷键使用 Command + Option + 方向键
	"move_up":    {[]hotkey.Modifier{hotkey.ModCmd, hotkey.ModOption}, hotkey.KeyUp},
	"move_down":  {[]hotkey.Modifier{hotkey.ModCmd, hotkey.ModOption}, hotkey.KeyDown},
	"move_left":  {[]hotkey.Modifier{hotkey.ModCmd, hotkey.ModOption}, hotkey.KeyLeft},
	"move_right": {[]hotkey.Modifier{hotkey.ModCmd, hotkey.ModOption}, hotkey.KeyRight},
	// 滚动使用 Command + Option + Shift + 方向键
	"scroll_up":   {[]hotkey.Modifier{hotkey.ModCmd, hotkey.ModOption, hotkey.ModShift}, hotkey.KeyUp},
	"scroll_down": {[]hotkey.Modifier{hotkey.ModCmd, hotkey.ModOption, hotkey.ModShift}, hotkey.KeyDown},
}

func NewManager() *Manager {
	return &Manager{
		hotkeys:   make(map[string]*hotkey.Hotkey),
		Shortcuts: make(map[string]KeyBinding),
		stopChan:  make(chan struct{}),
		running:   false,
	}
}

func (m *Manager) Start() {
	if m.running {
		return
	}
	m.running = true

	// 在后台 goroutine 中注册热键
	go m.registerHotkeys()
}

func (m *Manager) registerHotkeys() {
	for action, binding := range macDefaultShortcuts {
		hk := hotkey.New(binding.mods, binding.key)
		if err := hk.Register(); err != nil {
			logger.Printf("[macOS] 注册热键失败 %s: %v (可能需要辅助功能权限)", action, err)
			continue
		}
		m.hotkeys[action] = hk

		// 启动监听协程
		go m.listenHotkey(action, hk)
	}
	logger.Println("[macOS] 全局热键注册完成")
}

func (m *Manager) listenHotkey(action string, hk *hotkey.Hotkey) {
	for {
		select {
		case <-m.stopChan:
			return
		case <-hk.Keydown():
			if m.OnTrigger != nil {
				m.OnTrigger(action)
			}
		}
	}
}

func (m *Manager) Stop() {
	if !m.running {
		return
	}

	// 发送停止信号
	close(m.stopChan)

	for action, hk := range m.hotkeys {
		if err := hk.Unregister(); err != nil {
			logger.Printf("[macOS] 注销热键失败 %s: %v", action, err)
		} else {
			logger.Printf("[macOS] 注销热键: %s", action)
		}
	}
	m.hotkeys = make(map[string]*hotkey.Hotkey)
	m.stopChan = make(chan struct{})
	m.running = false
}

// macOS 不支持热键录制，这些方法为空实现
func (m *Manager) StartRecording(action string) {
	logger.Println("[macOS] 热键录制不支持")
	// 通过回调通知前端显示 toast
	if m.OnError != nil {
		m.OnError("macOS 不支持自定义快捷键，请使用预设快捷键")
	}
}

func (m *Manager) StopRecording() {
	// 空实现
}
