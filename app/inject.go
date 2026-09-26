package app

import (
	"ai-assistant/pkg/logger"
	"errors"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32                    = syscall.NewLazyDLL("user32.dll")
	kernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procOpenClipboard         = user32.NewProc("OpenClipboard")
	procCloseClipboard        = user32.NewProc("CloseClipboard")
	procEmptyClipboard        = user32.NewProc("EmptyClipboard")
	procSetClipboardData      = user32.NewProc("SetClipboardData")
	procGetClipboardData      = user32.NewProc("GetClipboardData")
	procCountClipboardFormats = user32.NewProc("CountClipboardFormats")
	procGlobalAlloc           = kernel32.NewProc("GlobalAlloc")
	procGlobalFree            = kernel32.NewProc("GlobalFree")
	procGlobalSize            = kernel32.NewProc("GlobalSize")
	procGlobalLock            = kernel32.NewProc("GlobalLock")
	procGlobalUnlock          = kernel32.NewProc("GlobalUnlock")
	procKeybdEvent            = user32.NewProc("keybd_event")
)

var clipboardInjectMu sync.Mutex

const (
	CF_UNICODETEXT = 13
	GMEM_MOVEABLE  = 0x0002
	VK_CONTROL     = 0x11
	VK_V           = 0x56
)

// injectTextViaClipboard 通过剪贴板注入文字到当前活动窗口
func injectTextViaClipboard(text string) {
	clipboardInjectMu.Lock()
	defer clipboardInjectMu.Unlock()

	oldClipboard, hadText, err := getClipboardText()
	if err != nil {
		logger.Printf("[Inject] Clipboard unavailable: %v", err)
		return
	}
	if err := setClipboardText(text); err != nil {
		logger.Printf("[Inject] Failed to set clipboard: %v", err)
		return
	}
	defer func() {
		if hadText {
			err = setClipboardText(oldClipboard)
		} else {
			err = clearClipboard()
		}
		if err != nil {
			logger.Printf("[Inject] Failed to restore clipboard: %v", err)
		}
	}()

	time.Sleep(50 * time.Millisecond)
	simulateCtrlV()
	time.Sleep(100 * time.Millisecond)
	logger.Printf("[Inject] Text injected: length=%d", len(text))
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
		procGlobalFree.Call(hMem)
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
		procGlobalFree.Call(hMem)
		return syscall.GetLastError()
	}

	return nil
}

func clearClipboard() error {
	r, _, _ := procOpenClipboard.Call(0, 0, 0)
	if r == 0 {
		return syscall.GetLastError()
	}
	defer procCloseClipboard.Call()
	if r, _, _ = procEmptyClipboard.Call(); r == 0 {
		return syscall.GetLastError()
	}
	return nil
}

// getClipboardText returns whether the clipboard contained Unicode text.
func getClipboardText() (string, bool, error) {
	r, _, _ := procOpenClipboard.Call(0, 0, 0)
	if r == 0 {
		return "", false, syscall.GetLastError()
	}
	defer procCloseClipboard.Call()

	h, _, _ := procGetClipboardData.Call(CF_UNICODETEXT)
	if h == 0 {
		count, _, _ := procCountClipboardFormats.Call()
		if count == 0 {
			return "", false, nil
		}
		return "", false, errors.New("剪贴板包含非文本内容")
	}

	p, _, _ := procGlobalLock.Call(h)
	if p == 0 {
		return "", false, syscall.GetLastError()
	}
	defer procGlobalUnlock.Call(h)

	size, _, _ := procGlobalSize.Call(h)
	if size < 2 || size > 2*(1<<20) {
		return "", false, errors.New("剪贴板文本过大或无效")
	}
	charCount := int(size / 2)
	chars := unsafe.Slice((*uint16)(unsafe.Pointer(p)), charCount)
	for i := 0; i < charCount; i++ {
		if chars[i] == 0 {
			return syscall.UTF16ToString(chars[:i]), true, nil
		}
	}
	return "", false, errors.New("剪贴板文本过大")
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
