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

	if err := os.MkdirAll(appDir, 0700); err != nil {
	}
	fullPath := filepath.Join(appDir, "config")
	logger.Println("配置文件路径", fullPath)

	return fullPath
}

func (cm *ConfigManager) Load() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	candidate := NewDefaultConfig()
	migrateLegacy := false
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Printf("加载配置文件失败 (使用默认配置): %v", err)
		}
	} else {
		legacyFormat := len(data) < len(currentCipherHeader) || string(data[:len(currentCipherHeader)]) != string(currentCipherHeader)
		plain, decryptErr := decrypt(data)
		if decryptErr != nil {
			logger.Printf("解密配置文件失败 (使用默认配置): %v", decryptErr)
		} else {
			merged, parseErr := mergeConfig(candidate, plain)
			if parseErr != nil {
				logger.Printf("解析配置文件失败 (使用默认配置): %v", parseErr)
			} else {
				candidate = merged
				migrateLegacy = legacyFormat
			}
		}
	}

	if err := candidate.Validate(); err != nil {
		logger.Printf("配置校验失败 (使用默认配置): %v", err)
		candidate = NewDefaultConfig()
	}
	cm.config = candidate
	cm.oldConfig = candidate

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

	if migrateLegacy {
		if err := cm.saveLocked(); err != nil {
			logger.Printf("迁移旧配置加密格式失败: %v", err)
		}
	}
	logger.Println("配置已加载")
	return nil
}

func (cm *ConfigManager) Save() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.saveLocked()
}

func (cm *ConfigManager) saveLocked() error {
	plain, err := json.MarshalIndent(cm.config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	data, err := encrypt(plain)
	if err != nil {
		return fmt.Errorf("加密配置失败: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(cm.configPath), ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("创建配置临时文件失败: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return fmt.Errorf("设置配置临时文件权限失败: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("写入配置临时文件失败: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("同步配置临时文件失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("关闭配置临时文件失败: %w", err)
	}
	if err := atomicReplaceFile(tmpPath, cm.configPath); err != nil {
		return fmt.Errorf("替换配置文件失败: %w", err)
	}
	if err := os.Chmod(cm.configPath, 0600); err != nil {
		return fmt.Errorf("设置配置文件权限失败: %w", err)
	}

	logger.Printf("配置已保存到: %s", cm.configPath)
	return nil
}

func mergeConfig(base Config, raw []byte) (Config, error) {
	defaults, err := json.Marshal(base)
	if err != nil {
		return Config{}, err
	}
	var merged map[string]json.RawMessage
	if err := json.Unmarshal(defaults, &merged); err != nil {
		return Config{}, err
	}
	var incoming map[string]json.RawMessage
	if err := json.Unmarshal(raw, &incoming); err != nil {
		return Config{}, err
	}
	for key, value := range incoming {
		merged[key] = value
	}
	combined, err := json.Marshal(merged)
	if err != nil {
		return Config{}, err
	}
	var out Config
	if err := json.Unmarshal(combined, &out); err != nil {
		return Config{}, err
	}
	return out, nil
}

func (cm *ConfigManager) Get() Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}

// UpdateFromJSON 从前端 JSON 全量更新配置
func (cm *ConfigManager) UpdateFromJSON(jsonStr string) error {
	cm.mu.Lock()
	oldConfig := cm.config
	newConfig, err := mergeConfig(oldConfig, []byte(jsonStr))
	if err != nil {
		cm.mu.Unlock()
		return fmt.Errorf("解析配置 JSON 失败: %w", err)
	}
	if err := newConfig.Validate(); err != nil {
		cm.mu.Unlock()
		return fmt.Errorf("配置校验失败: %w", err)
	}

	cm.oldConfig = oldConfig
	cm.config = newConfig
	if err := cm.saveLocked(); err != nil {
		cm.config = oldConfig
		cm.oldConfig = oldConfig
		cm.mu.Unlock()
		return err
	}
	configCopy := cm.config
	oldConfigCopy := cm.oldConfig
	subscribers := append([]func(Config, Config){}, cm.subscribers...)
	cm.mu.Unlock()

	for _, sub := range subscribers {
		sub(configCopy, oldConfigCopy)
	}
	return nil
}

func cloneConfig(src Config) (Config, error) {
	data, err := json.Marshal(src)
	if err != nil {
		return Config{}, err
	}
	var dst Config
	if err := json.Unmarshal(data, &dst); err != nil {
		return Config{}, err
	}
	return dst, nil
}

// Patch 部分更新配置字段（避免全量序列化/反序列化的开销）
// patchFn 接收当前配置副本，直接修改需要变更的字段
func (cm *ConfigManager) Patch(patchFn func(cfg *Config)) error {
	cm.mu.Lock()
	oldConfig, err := cloneConfig(cm.config)
	if err != nil {
		cm.mu.Unlock()
		return fmt.Errorf("复制配置失败: %w", err)
	}
	candidate, err := cloneConfig(oldConfig)
	if err != nil {
		cm.mu.Unlock()
		return fmt.Errorf("复制配置失败: %w", err)
	}
	patchFn(&candidate)
	if err := candidate.Validate(); err != nil {
		cm.mu.Unlock()
		return fmt.Errorf("配置校验失败: %w", err)
	}
	cm.oldConfig = oldConfig
	cm.config = candidate
	if err := cm.saveLocked(); err != nil {
		cm.config = oldConfig
		cm.oldConfig = oldConfig
		cm.mu.Unlock()
		return err
	}
	configCopy := cm.config
	oldConfigCopy := cm.oldConfig
	subscribers := append([]func(Config, Config){}, cm.subscribers...)
	cm.mu.Unlock()

	for _, sub := range subscribers {
		sub(configCopy, oldConfigCopy)
	}
	return nil
}

func (cm *ConfigManager) Subscribe(callback func(NewConfig Config, oldConfig Config)) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.subscribers = append(cm.subscribers, callback)
}
