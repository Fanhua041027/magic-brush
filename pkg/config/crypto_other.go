//go:build !windows

package config

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"

	"ai-assistant/pkg/common"
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
	path := filepath.Join(dir, ".config-key")
	key, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		key = make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, fmt.Errorf("生成配置密钥失败: %w", err)
		}
		if err := os.WriteFile(path, key, 0600); err != nil {
			return nil, fmt.Errorf("保存配置密钥失败: %w", err)
		}
		return key, nil
	}
	if err != nil {
		return nil, fmt.Errorf("读取配置密钥失败: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("配置密钥长度无效")
	}
	return key, nil
}
