//go:build windows

package config

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"ai-assistant/pkg/common"
	"golang.org/x/sys/windows"
)

func currentEncryptionKey() ([]byte, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("获取配置目录失败: %w", err)
	}
	dir := filepath.Join(base, common.AppName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("创建密钥目录失败: %w", err)
	}
	path := filepath.Join(dir, ".config-key.dpapi")
	protected, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, fmt.Errorf("生成配置密钥失败: %w", err)
		}
		protected, err = protectForCurrentUser(key)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, protected, 0600); err != nil {
			return nil, fmt.Errorf("保存配置密钥失败: %w", err)
		}
		return key, nil
	}
	if err != nil {
		return nil, fmt.Errorf("读取配置密钥失败: %w", err)
	}
	key, err := unprotectForCurrentUser(protected)
	if err != nil {
		return nil, fmt.Errorf("解锁配置密钥失败: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("配置密钥长度无效")
	}
	return key, nil
}

func protectForCurrentUser(plain []byte) ([]byte, error) {
	in := windows.DataBlob{Size: uint32(len(plain)), Data: &plain[0]}
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, nil, 0, nil, 0, &out); err != nil {
		return nil, fmt.Errorf("DPAPI 加密配置密钥失败: %w", err)
	}
	defer windows.LocalFree(windows.Handle(uintptr(unsafe.Pointer(out.Data))))
	result := unsafe.Slice((*byte)(unsafe.Pointer(out.Data)), out.Size)
	return append([]byte(nil), result...), nil
}

func unprotectForCurrentUser(protected []byte) ([]byte, error) {
	if len(protected) == 0 {
		return nil, fmt.Errorf("DPAPI 密钥为空")
	}
	in := windows.DataBlob{Size: uint32(len(protected)), Data: &protected[0]}
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, nil, nil, 0, nil, 0, &out); err != nil {
		return nil, fmt.Errorf("DPAPI 解密配置密钥失败: %w", err)
	}
	defer windows.LocalFree(windows.Handle(uintptr(unsafe.Pointer(out.Data))))
	result := unsafe.Slice((*byte)(unsafe.Pointer(out.Data)), out.Size)
	return append([]byte(nil), result...), nil
}
