package resume

import (
	"ai-assistant/pkg/config"
	"ai-assistant/pkg/llm"
	"ai-assistant/pkg/logger"
	"ai-assistant/pkg/prompts"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/ledongthuc/pdf"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Service struct {
	mu           sync.RWMutex
	config       config.Config
	resumeBase64 string
	provider     llm.Provider
}

func NewService(cfg config.Config, cm *config.ConfigManager) *Service {
	s := &Service{
		config: cfg,
	}

	cm.Subscribe(func(newConfig config.Config, oldConfig config.Config) {
		s.mu.Lock()
		s.config = newConfig
		if newConfig.ResumePath != oldConfig.ResumePath {
			s.resumeBase64 = ""
		}
		s.mu.Unlock()
	})

	return s
}

func (s *Service) SelectResume(ctx context.Context) string {
	selection, err := runtime.OpenFileDialog(ctx, runtime.OpenDialogOptions{
		Title: "选择简历 (PDF)",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "PDF Files",
				Pattern:     "*.pdf",
			},
		},
	})

	if err != nil {
		logger.Printf("选择文件失败: %v\n", err)
		return ""
	}

	if selection == "" {
		return ""
	}

	return selection
}

func (s *Service) SetProvider(p llm.Provider) {
	s.mu.Lock()
	s.provider = p
	s.mu.Unlock()
}

func (s *Service) ClearResume() {
	s.mu.Lock()
	s.resumeBase64 = ""
	s.mu.Unlock()
	logger.Println("简历缓存已清除")
}

func (s *Service) GetResumeBase64() (string, error) {
	s.mu.RLock()
	cached := s.resumeBase64
	resumePath := s.config.ResumePath
	s.mu.RUnlock()

	if len(cached) > 0 {
		logger.Println("使用缓存的简历 Base64")
		return cached, nil
	}
	if resumePath == "" {
		return "", nil
	}

	fileInfo, err := os.Stat(resumePath)
	if err != nil {
		return "", err
	}

	const maxResumeSize = 5 * 1024 * 1024
	if fileInfo.Size() > maxResumeSize {
		return "", fmt.Errorf("简历文件大小超过 5MB 限制")
	}

	content, err := os.ReadFile(resumePath)
	if err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(content)
	s.mu.Lock()
	s.resumeBase64 = encoded
	s.mu.Unlock()
	return encoded, nil
}

func (s *Service) ParseResume(ctx context.Context) (string, error) {
	// 先获取缓存的 Base64（用于判断文件是否存在）
	resumeBase64, err := s.GetResumeBase64()
	if err != nil {
		return "", fmt.Errorf("读取简历失败: %v", err)
	}
	if resumeBase64 == "" {
		return "", fmt.Errorf("请先选择简历文件")
	}

	s.mu.RLock()
	resumePath := s.config.ResumePath
	cfg := s.config
	provider := s.provider
	s.mu.RUnlock()

	if provider == nil {
		return "", fmt.Errorf("LLM 服务未初始化")
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return "", fmt.Errorf("请先配置 API Key")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return "", fmt.Errorf("请先选择模型")
	}

	// 直接从 PDF 提取文本（不依赖 LLM 视觉能力）
	logger.Printf("从 PDF 提取文本: %s", resumePath)
	textContent, err := extractTextFromPDF(resumePath)
	if err != nil {
		return "", fmt.Errorf("PDF 文本提取失败: %v", err)
	}
	if strings.TrimSpace(textContent) == "" {
		return "", fmt.Errorf("PDF 中未提取到文本内容")
	}

	logger.Printf("PDF 文本提取成功 (%d 字符)，发送给 LLM 解析", len(textContent))

	messages := []llm.Message{
		llm.NewSystemMessage(prompts.ResumeParsePrompt),
		llm.NewUserMessage(fmt.Sprintf(
			"请将以下简历内容解析并整理为结构清晰的 Markdown 格式：\n\n---\n%s\n---",
			textContent,
		)),
	}

	result, err := provider.GenerateContent(ctx, cfg.Model, messages)
	if err != nil {
		logger.Printf("简历解析失败: %v", err)
		return "", err
	}

	content := strings.TrimSpace(result.Content)
	if content == "" {
		return "", fmt.Errorf("模型没有返回简历解析结果")
	}

	return content, nil
}

// extractTextFromPDF 使用 ledongthuc/pdf 从 PDF 文件中提取纯文本
func extractTextFromPDF(path string) (string, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", fmt.Errorf("打开 PDF 失败: %v", err)
	}
	defer f.Close()

	var builder strings.Builder
	totalPage := r.NumPage()

	for i := 1; i <= totalPage; i++ {
		p := r.Page(i)
		text, err := p.GetPlainText(nil)
		if err != nil {
			logger.Printf("提取第 %d 页文本失败: %v", i, err)
			continue
		}
		builder.WriteString(text)
		builder.WriteString("\n\n")
	}

	return builder.String(), nil
}
