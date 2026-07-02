package config

import (
	"ai-assistant/pkg/common"
	"ai-assistant/pkg/logger"
	"ai-assistant/pkg/shortcut"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type ConfigManager struct {
	config      Config
	mu          sync.RWMutex
	configPath  string
	oldConfig   Config // 这是老配置
	subscribers []func(NewConfig Config, oldConfig Config)
}

func NewConfigManager() *ConfigManager {
	cm := &ConfigManager{
		config:      NewDefaultConfig(),
		oldConfig:   NewDefaultConfig(),
		subscribers: make([]func(NewConfig Config, oldConfig Config), 0),
	}
	cm.configPath = cm.getConfigPath()
	return cm
}

// NewConfigManagerForTest 创建用于测试的 ConfigManager（使用临时路径）
func NewConfigManagerForTest(tempDir string) *ConfigManager {
	cm := &ConfigManager{
		config:      NewDefaultConfig(),
		oldConfig:   NewDefaultConfig(),
		subscribers: make([]func(NewConfig Config, oldConfig Config), 0),
	}
	cm.configPath = filepath.Join(tempDir, "config-test")
	return cm
}

func (cm *ConfigManager) getConfigPath() string {
	var appDir string

	sysConfigDir, err := os.UserConfigDir()
	if err != nil {
		// 如果获取系统目录失败（极少情况），回退到当前目录
		sysConfigDir = "."
	}
	// 拼接项目名称目录
	appDir = filepath.Join(sysConfigDir, common.AppName)

	if err := os.MkdirAll(appDir, 0755); err != nil {
	}
	fullPath := filepath.Join(appDir, "config")
	logger.Println("配置文件路径", fullPath)

	return fullPath
}

func (cm *ConfigManager) Load() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 先设置默认值
	cm.config = NewDefaultConfig()
	// 从文件加载（AES 加密存储）
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Printf("加载配置文件失败 (使用默认配置): %v", err)
		}
	} else {
		plain, err := decrypt(data)
		if err != nil {
			logger.Printf("解密配置文件失败 (使用默认配置): %v", err)
		} else if err := json.Unmarshal(plain, &cm.config); err != nil {
			logger.Printf("解析配置文件失败: %v", err)
		}
	}

	// 确保默认快捷键都存在（防止新增快捷键被旧配置覆盖）
	defaultShortcuts := NewDefaultConfig().Shortcuts
	for action, binding := range defaultShortcuts {
		if _, exists := cm.config.Shortcuts[action]; !exists {
			if cm.config.Shortcuts == nil {
				cm.config.Shortcuts = make(map[string]shortcut.KeyBinding)
			}
			cm.config.Shortcuts[action] = binding
			logger.Printf("添加快捷键: %s -> %s", action, binding.KeyName)
		}
	}

	logger.Println("配置已加载")
	return nil
}

func (cm *ConfigManager) Save() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	plain, err := json.MarshalIndent(cm.config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	data, err := encrypt(plain)
	if err != nil {
		return fmt.Errorf("加密配置失败: %w", err)
	}
	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	logger.Printf("配置已保存到: %s", cm.configPath)
	return nil
}

func (cm *ConfigManager) Get() Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}

// UpdateFromJSON 从前端 JSON 全量更新配置
func (cm *ConfigManager) UpdateFromJSON(jsonStr string) error {
	var newConfig Config
	if err := json.Unmarshal([]byte(jsonStr), &newConfig); err != nil {
		return fmt.Errorf("解析配置 JSON 失败: %w", err)
	}

	cm.mu.Lock()
	cm.oldConfig = cm.config //保存当前配置为之前的配置
	cm.config = newConfig
	configCopy := cm.config
	oldConfigCopy := cm.oldConfig
	subscribers := cm.subscribers
	cm.mu.Unlock()

	// 通知订阅者
	for _, sub := range subscribers {
		sub(configCopy, oldConfigCopy)
	}

	return cm.Save()
}

// Patch 部分更新配置字段（避免全量序列化/反序列化的开销）
// patchFn 接收当前配置指针，直接修改需要变更的字段
func (cm *ConfigManager) Patch(patchFn func(cfg *Config)) error {
	cm.mu.Lock()
	cm.oldConfig = cm.config
	patchFn(&cm.config)
	configCopy := cm.config
	oldConfigCopy := cm.oldConfig
	subscribers := cm.subscribers
	cm.mu.Unlock()

	// 通知订阅者
	for _, sub := range subscribers {
		sub(configCopy, oldConfigCopy)
	}

	return cm.Save()
}

func (cm *ConfigManager) Subscribe(callback func(NewConfig Config, oldConfig Config)) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.subscribers = append(cm.subscribers, callback)
}
