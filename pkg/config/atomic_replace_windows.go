//go:build windows

package config

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32       = windows.NewLazySystemDLL("kernel32.dll")
	moveFileExProc = kernel32.NewProc("MoveFileExW")
)

func atomicReplaceFile(src, dst string) error {
	srcPtr, err := syscall.UTF16PtrFromString(src)
	if err != nil {
		return err
	}
	dstPtr, err := syscall.UTF16PtrFromString(dst)
	if err != nil {
		return err
	}
	ret, _, callErr := moveFileExProc.Call(
		uintptr(unsafe.Pointer(srcPtr)),
		uintptr(unsafe.Pointer(dstPtr)),
		windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH,
	)
	if ret == 0 {
		return fmt.Errorf("MoveFileExW: %w", callErr)
	}
	return nil
}
