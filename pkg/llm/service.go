package llm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"ai-assistant/pkg/config"
	"ai-assistant/pkg/logger"
)

// Service manages the active OpenAI-compatible provider.
type Service struct {
	mu       sync.RWMutex
	config   config.Config
	provider Provider
}

func NewService(cfg config.Config, cm *config.ConfigManager) *Service {
	s := &Service{
		config: cfg,
	}
	s.updateProviderLocked()

	cm.Subscribe(func(newConfig config.Config, oldConfig config.Config) {
		s.mu.Lock()
		s.config = newConfig
		s.updateProviderLocked()
		s.mu.Unlock()
		logger.Println("LLM Provider 已更新")
	})

	return s
}

func (s *Service) updateProviderLocked() {
	s.provider = NewOpenAIAdapter(&s.config)
}

func (s *Service) GetProvider() Provider {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.provider
}

func (s *Service) TestConnection(ctx context.Context, apiKey, model string) string {
	if apiKey == "" {
		return "API Key 不能为空"
	}
	if model == "" {
		return "请选择模型"
	}

	s.mu.RLock()
	tempConfig := s.config
	s.mu.RUnlock()

	tempConfig.APIKey = apiKey
	tempConfig.Model = model

	tempProvider := NewOpenAIAdapter(&tempConfig)

	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if err := tempProvider.TestChat(timeoutCtx); err != nil {
		return err.Error()
	}

	return ""
}

func (s *Service) GetModels(ctx context.Context, apiKey string) ([]string, error) {
	s.mu.RLock()
	currentAPIKey := s.config.APIKey
	currentProvider := s.provider
	cfg := s.config
	s.mu.RUnlock()

	if apiKey == "" {
		apiKey = currentAPIKey
	}

	if apiKey != currentAPIKey {
		tempConfig := cfg
		tempConfig.APIKey = apiKey
		tempProvider := NewOpenAIAdapter(&tempConfig)
		return tempProvider.GetModels(ctx)
	}

	if currentProvider == nil {
		return nil, fmt.Errorf("provider not initialized")
	}
	return currentProvider.GetModels(ctx)
}
