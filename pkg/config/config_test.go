package config

import (
	"encoding/json"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// newTestConfigManager 创建使用临时路径的 ConfigManager（避免污染真实配置）
func newTestConfigManager() *ConfigManager {
	dir, _ := os.MkdirTemp("", "config-test-*")
	cm := NewConfigManagerForTest(dir)
	return cm
}

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	if cfg.BaseURL != "https://api.openai.com/v1" {
		t.Fatalf("expected default base URL, got '%s'", cfg.BaseURL)
	}
	if cfg.DomainId != "general-assistant" {
		t.Fatalf("expected default domain 'general-assistant', got '%s'", cfg.DomainId)
	}
	if cfg.Opacity != 1.0 {
		t.Fatalf("expected opacity 1.0, got %f", cfg.Opacity)
	}
	if cfg.ScreenshotMode != "fullscreen" {
		t.Fatalf("expected 'fullscreen', got '%s'", cfg.ScreenshotMode)
	}
	if cfg.CompressionQuality != 92 {
		t.Fatalf("expected 92, got %d", cfg.CompressionQuality)
	}
	if cfg.STTService != "qwen_cloud" {
		t.Fatalf("expected 'qwen_cloud', got '%s'", cfg.STTService)
	}
	if cfg.Shortcuts == nil {
		t.Fatal("expected shortcuts to be initialized")
	}
}

func TestConfigDefaults(t *testing.T) {
	t.Run("empty config has defaults", func(t *testing.T) {
		cfg := NewDefaultConfig()
		if cfg.KeepContext {
			t.Fatal("expected KeepContext to be false")
		}
		if cfg.Theme != "light" {
			t.Fatalf("expected theme 'light', got '%s'", cfg.Theme)
		}
		if cfg.STTLanguage != "zh" {
			t.Fatalf("expected STTLanguage 'zh', got '%s'", cfg.STTLanguage)
		}
		if cfg.STTSensitivity != 0.5 {
			t.Fatalf("expected STTSensitivity 0.5, got %f", cfg.STTSensitivity)
		}
	})

	t.Run("validate valid config", func(t *testing.T) {
		cfg := NewDefaultConfig()
		if err := cfg.Validate(); err != nil {
			t.Fatalf("unexpected validation error: %v", err)
		}
	})

	t.Run("validate invalid screenshot mode", func(t *testing.T) {
		cfg := NewDefaultConfig()
		cfg.ScreenshotMode = "invalid"
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected validation error for invalid screenshot mode")
		}
	})

	t.Run("validate invalid opacity", func(t *testing.T) {
		cfg := NewDefaultConfig()
		cfg.Opacity = 1.5
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected validation error for opacity > 1")
		}
		cfg.Opacity = -0.1
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected validation error for opacity < 0")
		}
	})

	t.Run("validate invalid compression", func(t *testing.T) {
		cfg := NewDefaultConfig()
		cfg.CompressionQuality = 0
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected validation error for compression < 1")
		}
		cfg.CompressionQuality = 101
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected validation error for compression > 100")
		}
	})

	t.Run("reject unsafe service URL and oversized content", func(t *testing.T) {
		cfg := NewDefaultConfig()
		cfg.BaseURL = "http://example.com/v1"
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected insecure remote URL to be rejected")
		}
		cfg.BaseURL = "http://127.0.0.1:8080/v1"
		if err := cfg.Validate(); err != nil {
			t.Fatalf("localhost URL rejected: %v", err)
		}
		cfg.BaseURL = "https://example.com/v1"
		cfg.ResumeContent = strings.Repeat("x", (1<<20)+1)
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected oversized resume content to be rejected")
		}
	})
}

func TestConfigToJSON(t *testing.T) {
	cfg := NewDefaultConfig()
	json := cfg.ToJSON()
	if json == "" {
		t.Fatal("expected non-empty JSON")
	}
	if len(json) < 10 {
		t.Fatal("JSON too short")
	}
}

func TestPublicConfigDoesNotExposeSecrets(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.APIKey = "main-secret"
	cfg.ScreenshotAPIKey = "vision-secret"
	encoded, err := json.Marshal(cfg.Public())
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	if strings.Contains(text, "main-secret") || strings.Contains(text, "vision-secret") || strings.Contains(text, `"apiKey"`) || strings.Contains(text, `"screenshotApiKey"`) {
		t.Fatalf("public config leaked a credential: %s", text)
	}
	if !strings.Contains(text, `"apiKeyConfigured":true`) || !strings.Contains(text, `"screenshotApiKeyConfigured":true`) {
		t.Fatalf("public config omitted credential status: %s", text)
	}
}

func TestUpdateFromJSONPreservesOmittedSecret(t *testing.T) {
	cm := newTestConfigManager()
	if err := cm.Patch(func(cfg *Config) { cfg.APIKey = "secret" }); err != nil {
		t.Fatal(err)
	}
	if err := cm.UpdateFromJSON(`{"model":"new-model"}`); err != nil {
		t.Fatal(err)
	}
	if got := cm.Get(); got.APIKey != "secret" || got.Model != "new-model" {
		t.Fatalf("unexpected merged config: key=%q model=%q", got.APIKey, got.Model)
	}
	if err := cm.UpdateFromJSON(`{"apiKey":""}`); err != nil {
		t.Fatal(err)
	}
	if got := cm.Get().APIKey; got != "" {
		t.Fatalf("explicit secret clear failed: %q", got)
	}
}

func TestSaveUsesPrivatePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose owner-only mode bits through os.FileMode")
	}
	cm := newTestConfigManager()
	if err := cm.Save(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(cm.configPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("config permissions = %o, want 600", info.Mode().Perm())
	}
}

func TestShortcutDefaults(t *testing.T) {
	shortcuts := getDefaultShortcuts()

	if _, ok := shortcuts["screenshot"]; !ok {
		t.Fatal("expected screenshot shortcut")
	}
	if _, ok := shortcuts["send"]; !ok {
		t.Fatal("expected send shortcut")
	}
	if _, ok := shortcuts["chat"]; !ok {
		t.Fatal("expected chat shortcut")
	}
	if _, ok := shortcuts["standalone"]; !ok {
		t.Fatal("expected standalone shortcut")
	}
	if _, ok := shortcuts["standalone2"]; !ok {
		t.Fatal("expected standalone2 shortcut")
	}

	if shortcuts["screenshot"].KeyName != "F8" {
		t.Fatalf("expected F8, got '%s'", shortcuts["screenshot"].KeyName)
	}
	if shortcuts["chat"].KeyName != "F7" {
		t.Fatalf("expected F7, got '%s'", shortcuts["chat"].KeyName)
	}
	if shortcuts["standalone"].KeyName != "F1" {
		t.Fatalf("expected F1, got '%s'", shortcuts["standalone"].KeyName)
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{Field: "apiKey", Message: "API Key is required"}
	if err.Error() != "apiKey: API Key is required" {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
}

func TestConfigManager(t *testing.T) {
	t.Run("default config on fresh manager", func(t *testing.T) {
		cm := newTestConfigManager()
		cfg := cm.Get()
		if cfg.APIKey != "" {
			t.Fatal("expected empty API key initially")
		}
	})

	t.Run("patch updates config", func(t *testing.T) {
		cm := newTestConfigManager()
		err := cm.Patch(func(cfg *Config) {
			cfg.APIKey = "test-key"
			cfg.Model = "test-model"
		})
		if err != nil {
			t.Fatalf("patch failed: %v", err)
		}
		cfg := cm.Get()
		if cfg.APIKey != "test-key" {
			t.Fatalf("expected 'test-key', got '%s'", cfg.APIKey)
		}
		if cfg.Model != "test-model" {
			t.Fatalf("expected 'test-model', got '%s'", cfg.Model)
		}
	})

	t.Run("subscribe receives updates", func(t *testing.T) {
		cm := newTestConfigManager()
		var mu sync.Mutex
		received := ""
		cm.Subscribe(func(newCfg, oldCfg Config) {
			mu.Lock()
			received = newCfg.APIKey
			mu.Unlock()
		})
		cm.Patch(func(cfg *Config) {
			cfg.APIKey = "new-key"
		})
		time.Sleep(50 * time.Millisecond)
		mu.Lock()
		if received != "new-key" {
			t.Fatalf("expected 'new-key', got '%s'", received)
		}
		mu.Unlock()
	})

	t.Run("multiple patches accumulate", func(t *testing.T) {
		cm := newTestConfigManager()
		cm.Patch(func(cfg *Config) {
			cfg.APIKey = "key1"
			cfg.Model = "model1"
		})
		cm.Patch(func(cfg *Config) {
			cfg.Model = "model2"
		})
		cfg := cm.Get()
		if cfg.APIKey != "key1" {
			t.Fatalf("expected 'key1', got '%s'", cfg.APIKey)
		}
		if cfg.Model != "model2" {
			t.Fatalf("expected 'model2', got '%s'", cfg.Model)
		}
	})
}

func TestEncryptUsesCurrentFormat(t *testing.T) {
	plaintext := []byte(`{"apiKey":"secret"}`)
	ciphertext, err := encrypt(plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if len(ciphertext) <= len(currentCipherHeader) || string(ciphertext[:len(currentCipherHeader)]) != string(currentCipherHeader) {
		t.Fatal("ciphertext missing current header")
	}
	decoded, err := decrypt(ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != string(plaintext) {
		t.Fatalf("round trip mismatch: %q", decoded)
	}
}

func TestLegacyCiphertextCanBeRead(t *testing.T) {
	plaintext := []byte(`{"model":"legacy"}`)
	ciphertext, err := sealWithKey(legacyAESKey, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decrypt(ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != string(plaintext) {
		t.Fatalf("legacy round trip mismatch: %q", decoded)
	}
}
