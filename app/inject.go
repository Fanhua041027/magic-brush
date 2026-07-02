package app

import (
	"ai-assistant/pkg/logger"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procOpenClipboard  = user32.NewProc("OpenClipboard")
	procCloseClipboard = user32.NewProc("CloseClipboard")
	procEmptyClipboard = user32.NewProc("EmptyClipboard")
	procSetClipboardData = user32.NewProc("SetClipboardData")
	procGetClipboardData = user32.NewProc("GetClipboardData")
	procGlobalAlloc    = kernel32.NewProc("GlobalAlloc")
	procGlobalLock     = kernel32.NewProc("GlobalLock")
	procGlobalUnlock   = kernel32.NewProc("GlobalUnlock")
	procKeybdEvent     = user32.NewProc("keybd_event")
)

const (
	CF_UNICODETEXT = 13
	GMEM_MOVEABLE  = 0x0002
	VK_CONTROL     = 0x11
	VK_V           = 0x56
)

// injectTextViaClipboard 通过剪贴板注入文字到当前活动窗口
func injectTextViaClipboard(text string) {
	// 保存当前剪贴板内容
	oldClipboard := getClipboardText()

	// 设置新文字到剪贴板
	if err := setClipboardText(text); err != nil {
		logger.Printf("[Inject] Failed to set clipboard: %v", err)
		return
	}

	// 等待一下让剪贴板生效
	time.Sleep(50 * time.Millisecond)

	// 模拟 Ctrl+V 粘贴
	simulateCtrlV()

	// 等待粘贴完成
	time.Sleep(100 * time.Millisecond)

	// 恢复原剪贴板内容
	if oldClipboard != "" {
		setClipboardText(oldClipboard)
	}

	logger.Printf("[Inject] Text injected: %s", text)
}

// setClipboardText 设置剪贴板文字
func setClipboardText(text string) error {
	r, _, _ := procOpenClipboard.Call(0, 0, 0)
	if r == 0 {
		return syscall.GetLastError()
	}
	defer procCloseClipboard.Call()

	procEmptyClipboard.Call()

	// 分配内存
	textBytes := syscall.StringToUTF16(text)
	size := len(textBytes) * 2
	hMem, _, _ := procGlobalAlloc.Call(GMEM_MOVEABLE, uintptr(size))
	if hMem == 0 {
		return syscall.GetLastError()
	}

	// 锁定内存并复制文字
	pMem, _, _ := procGlobalLock.Call(hMem)
	if pMem == 0 {
		return syscall.GetLastError()
	}
	// Safe: Convert global lock pointer to slice for copying
	p := unsafe.Pointer(pMem)
	s := unsafe.Slice((*uint16)(p), len(textBytes))
	copy(s, textBytes)
	procGlobalUnlock.Call(hMem)

	// 设置剪贴板数据
	r, _, _ = procSetClipboardData.Call(CF_UNICODETEXT, hMem)
	if r == 0 {
		return syscall.GetLastError()
	}

	return nil
}

// getClipboardText 获取剪贴板文字
func getClipboardText() string {
	r, _, _ := procOpenClipboard.Call(0, 0, 0)
	if r == 0 {
		return ""
	}
	defer procCloseClipboard.Call()

	h, _, _ := procGetClipboardData.Call(CF_UNICODETEXT)
	if h == 0 {
		return ""
	}

	p, _, _ := procGlobalLock.Call(h)
	if p == 0 {
		return ""
	}
	defer procGlobalUnlock.Call(h)

	// 读取 UTF-16 字符为 Go string
	// Safe: Convert to slice to avoid pointer arithmetic
	const maxClipboardChars = 1 << 20
	p2 := unsafe.Pointer(p)
	chars := unsafe.Slice((*uint16)(p2), maxClipboardChars)
	for i := 0; i < maxClipboardChars; i++ {
		if chars[i] == 0 {
			return syscall.UTF16ToString(chars[:i])
		}
	}
	return ""
}

// simulateCtrlV 模拟 Ctrl+V 粘贴
func simulateCtrlV() {
	// 按下 Ctrl
	procKeybdEvent.Call(VK_CONTROL, 0, 0, 0)
	// 按下 V
	procKeybdEvent.Call(VK_V, 0, 0, 0)
	// 松开 V
	procKeybdEvent.Call(VK_V, 0, 2, 0)
	// 松开 Ctrl
	procKeybdEvent.Call(VK_CONTROL, 0, 2, 0)
}
