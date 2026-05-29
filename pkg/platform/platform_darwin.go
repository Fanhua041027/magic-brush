//go:build darwin

package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework AppKit -framework CoreGraphics -framework AVFoundation

#import <Cocoa/Cocoa.h>
#import <CoreGraphics/CoreGraphics.h>
#import <AVFoundation/AVFoundation.h>

// 检查截图权限 (macOS 10.15+)
bool CheckScreenCaptureAccessC() {
    if (@available(macOS 10.15, *)) {
        return CGPreflightScreenCaptureAccess();
    }
    return true; // macOS 10.15 以下不需要权限
}

// 请求截图权限 (macOS 10.15+)
bool RequestScreenCaptureAccessC() {
    if (@available(macOS 10.15, *)) {
        return CGRequestScreenCaptureAccess();
    }
    return true; // macOS 10.15 以下不需要权限
}

// 获取应用主窗口
void* GetMainWindowC() {
    NSApplication* app = [NSApplication sharedApplication];
    NSWindow* window = [app mainWindow];
    if (window == nil && app.windows.count > 0) {
        window = app.windows[0];
    }
    return (__bridge void*)window;
}

// 设置窗口忽略鼠标事件（鼠标穿透）
void SetWindowIgnoresMouseEventsC(void* nsWindow, bool ignores) {
    if (nsWindow == NULL) return;
    NSWindow* window = (__bridge NSWindow*)nsWindow;
    dispatch_async(dispatch_get_main_queue(), ^{
        [window setIgnoresMouseEvents:ignores];
    });
}

// 设置窗口级别（置顶）
void SetWindowLevelC(void* nsWindow, int level) {
    if (nsWindow == NULL) return;
    NSWindow* window = (__bridge NSWindow*)nsWindow;
    dispatch_async(dispatch_get_main_queue(), ^{
        [window setLevel:level];
    });
}

// 设置窗口样式（无边框 + 圆角）
void SetWindowStyleMaskBorderlessC(void* nsWindow) {
    if (nsWindow == NULL) return;
    NSWindow* window = (__bridge NSWindow*)nsWindow;
    dispatch_async(dispatch_get_main_queue(), ^{
        // 保留 titled 样式以维持输入法上下文（IMK），同时隐藏标题栏
        [window setStyleMask:(NSWindowStyleMaskTitled |
                              NSWindowStyleMaskFullSizeContentView)];
        [window setTitlebarAppearsTransparent:YES];
        [window setTitleVisibility:NSWindowTitleHidden];
        // 隐藏标题栏按钮
        [[window standardWindowButton:NSWindowCloseButton] setHidden:YES];
        [[window standardWindowButton:NSWindowMiniaturizeButton] setHidden:YES];
        [[window standardWindowButton:NSWindowZoomButton] setHidden:YES];
        // 透明背景
        [window setBackgroundColor:[NSColor clearColor]];
        [window setOpaque:NO];
        // 允许透明
        [window setHasShadow:YES];
    });
}

// 设置窗口 sharingType（防录屏，macOS 14+）
void SetWindowSharingTypeC(void* nsWindow, int sharingType) {
    if (nsWindow == NULL) return;
    NSWindow* window = (__bridge NSWindow*)nsWindow;
    dispatch_async(dispatch_get_main_queue(), ^{
        if (@available(macOS 14.0, *)) {
            window.sharingType = sharingType;
        }
    });
}

// 设置窗口不激活
void SetWindowNotActivatingC(void* nsWindow, bool noActivate) {
    if (nsWindow == NULL) return;
    NSWindow* window = (__bridge NSWindow*)nsWindow;
    dispatch_async(dispatch_get_main_queue(), ^{
        if (noActivate) {
            // 只设置 CanJoinAllSpaces，不设置 Stationary/IgnoresCycle
            // Stationary 会导致窗口点击后无法成为 key window，从而无法拖动
            [window setCollectionBehavior:NSWindowCollectionBehaviorCanJoinAllSpaces];
        } else {
            [window setCollectionBehavior:NSWindowCollectionBehaviorDefault];
        }
    });
}

// 让窗口可以接收键盘焦点
void SetWindowCanBecomeKeyC(void* nsWindow) {
    if (nsWindow == NULL) return;
    NSWindow* window = (__bridge NSWindow*)nsWindow;
    dispatch_async(dispatch_get_main_queue(), ^{
        [window makeKeyAndOrderFront:nil];
    });
}

// 设置窗口接受鼠标事件时自动成为 key window
void SetWindowAcceptsMouseMovedC(void* nsWindow) {
    if (nsWindow == NULL) return;
    NSWindow* window = (__bridge NSWindow*)nsWindow;
    dispatch_async(dispatch_get_main_queue(), ^{
        [window setAcceptsMouseMovedEvents:YES];
    });
}

// 设置 contentView 圆角
void SetWindowCornerRadiusC(void* nsWindow, float radius) {
    if (nsWindow == NULL) return;
    NSWindow* window = (__bridge NSWindow*)nsWindow;
    dispatch_async(dispatch_get_main_queue(), ^{
        // 设置 contentView 层的圆角
        NSView* contentView = [window contentView];
        if (contentView) {
            [contentView setWantsLayer:YES];
            contentView.layer.cornerRadius = radius;
            contentView.layer.masksToBounds = YES;
        }
    });
}

// 设置窗口透明度 (0.0 - 1.0)
void SetWindowAlphaC(void* nsWindow, float alpha) {
    if (nsWindow == NULL) return;
    NSWindow* window = (__bridge NSWindow*)nsWindow;
    dispatch_async(dispatch_get_main_queue(), ^{
        [window setAlphaValue:alpha];
    });
}

// 打开系统偏好设置的屏幕录制权限页面
void OpenScreenCaptureSettingsC() {
    dispatch_async(dispatch_get_main_queue(), ^{
        NSURL *url = [NSURL URLWithString:@"x-apple.systempreferences:com.apple.preference.security?Privacy_ScreenCapture"];
        [[NSWorkspace sharedWorkspace] openURL:url];
    });
}

// 打开系统偏好设置的麦克风权限页面
void OpenMicrophoneSettingsC() {
    dispatch_async(dispatch_get_main_queue(), ^{
        NSURL *url = [NSURL URLWithString:@"x-apple.systempreferences:com.apple.preference.security?Privacy_Microphone"];
        [[NSWorkspace sharedWorkspace] openURL:url];
    });
}

// 检查麦克风权限状态
// 返回: 0=未决定, 1=已授权, 2=已拒绝
int CheckMicrophoneAccessC() {
    if (@available(macOS 10.14, *)) {
        AVAuthorizationStatus status = [AVCaptureDevice authorizationStatusForMediaType:AVMediaTypeAudio];
        switch (status) {
            case AVAuthorizationStatusNotDetermined:
                return 0;
            case AVAuthorizationStatusAuthorized:
                return 1;
            case AVAuthorizationStatusDenied:
            case AVAuthorizationStatusRestricted:
                return 2;
        }
    }
    return 1; // macOS 10.14 以下不需要权限
}

// 请求麦克风权限
void RequestMicrophoneAccessC(void (*callback)(bool granted)) {
    if (@available(macOS 10.14, *)) {
        [AVCaptureDevice requestAccessForMediaType:AVMediaTypeAudio completionHandler:^(BOOL granted) {
            if (callback != NULL) {
                callback(granted);
            }
        }];
    } else {
        if (callback != NULL) {
            callback(true);
        }
    }
}
*/
import "C"
import (
	"ai-assistant/pkg/logger"
	"errors"
	"unsafe"
)

// WindowHandle 窗口句柄类型（macOS 为 NSWindow 指针）
type WindowHandle uintptr

// 窗口级别常量
const (
	WindowLevelNormal   = 0 // 正常窗口级别
	WindowLevelFloating = 3 // 置顶窗口级别

	// NSWindow sharingType 常量
	nsWindowSharingNone     = 0
	nsWindowSharingReadOnly = 1
)

// 存储主窗口指针
var mainWindow unsafe.Pointer

// GetWindowHandle 获取当前进程主窗口句柄
func GetWindowHandle() (WindowHandle, error) {
	window := C.GetMainWindowC()
	if window == nil {
		return 0, errors.New("无法获取主窗口")
	}
	mainWindow = window
	return WindowHandle(uintptr(window)), nil
}

// ApplyGhostMode 应用幽灵模式（无边框、置顶、防录屏、不抢焦点）
func ApplyGhostMode(hwnd WindowHandle) error {
	window := unsafe.Pointer(uintptr(hwnd))

	// 无边框 + 透明背景
	C.SetWindowStyleMaskBorderlessC(window)

	// 设置圆角
	C.SetWindowCornerRadiusC(window, 12.0)

	// 置顶
	C.SetWindowLevelC(window, C.int(WindowLevelFloating))

	// 防录屏 macOS 14+ 才可以
	C.SetWindowSharingTypeC(window, C.int(nsWindowSharingNone))

	// 不抢焦点
	C.SetWindowNotActivatingC(window, C.bool(true))

	// 允许窗口在点击时自动接收鼠标事件（修复失焦后无法拖动的问题）
	C.SetWindowAcceptsMouseMovedC(window)

	logger.Println("[macOS] 幽灵模式已激活")
	return nil
}

// SetClickThrough 设置鼠标穿透
func SetClickThrough(hwnd WindowHandle, enabled bool) error {
	window := unsafe.Pointer(uintptr(hwnd))
	C.SetWindowIgnoresMouseEventsC(window, C.bool(enabled))
	return nil
}

// SetDisplayAffinity 设置防录屏状态
func SetDisplayAffinity(hwnd WindowHandle, hidden bool) error {
	window := unsafe.Pointer(uintptr(hwnd))
	if hidden {
		C.SetWindowSharingTypeC(window, C.int(nsWindowSharingNone))
	} else {
		C.SetWindowSharingTypeC(window, C.int(nsWindowSharingReadOnly))
	}
	return nil
}

// RestoreFocus 恢复焦点
func RestoreFocus(hwnd WindowHandle) error {
	window := unsafe.Pointer(uintptr(hwnd))
	C.SetWindowCanBecomeKeyC(window)
	return nil
}

// RemoveFocus 移除焦点
func RemoveFocus(hwnd WindowHandle) error {
	window := unsafe.Pointer(uintptr(hwnd))
	C.SetWindowNotActivatingC(window, C.bool(true))
	return nil
}

// CheckScreenCaptureAccess 检查截图权限 (macOS 10.15+)
func CheckScreenCaptureAccess() bool {
	return bool(C.CheckScreenCaptureAccessC())
}

// RequestScreenCaptureAccess 请求截图权限 (macOS 10.15+)
func RequestScreenCaptureAccess() bool {
	return bool(C.RequestScreenCaptureAccessC())
}

// OpenScreenCaptureSettings 打开系统设置的屏幕录制权限页面
func OpenScreenCaptureSettings() {
	C.OpenScreenCaptureSettingsC()
}

// SetWindowLevel 设置窗口层级 (用于临时取消/恢复置顶)
func SetWindowLevel(hwnd WindowHandle, level int) error {
	window := unsafe.Pointer(uintptr(hwnd))
	C.SetWindowLevelC(window, C.int(level))
	return nil
}

// CheckMicrophoneAccess 检查麦克风权限状态 (macOS 专用)
// 返回: 0=未决定, 1=已授权, 2=已拒绝
func CheckMicrophoneAccess() int {
	return int(C.CheckMicrophoneAccessC())
}

// RequestMicrophoneAccess 请求麦克风权限 (macOS 专用)
func RequestMicrophoneAccess() {
	C.RequestMicrophoneAccessC(nil)
}

// OpenMicrophoneSettings 打开系统设置的麦克风权限页面 (macOS 专用)
func OpenMicrophoneSettings() {
	C.OpenMicrophoneSettingsC()
}
