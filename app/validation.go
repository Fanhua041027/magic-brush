package app

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const (
	maxChatMessageBytes  = 32 << 10
	maxChatContextBytes  = 256 << 10
	maxChatMessages      = 100
	maxScreenshotBytes   = 8 << 20
	maxAudioBase64Bytes  = 8 << 20
	maxDecodedAudioBytes = 5 << 20
)

var errInvalidChatInput = errors.New("输入内容为空、无效或过大")
var errAuthenticationRequired = errors.New("请先登录后使用此功能")

func (a *App) requireAuthenticated() error {
	if a.authService == nil || a.authService.Current() == nil {
		return errAuthenticationRequired
	}
	return nil
}

func validateChatText(text string) error {
	if strings.TrimSpace(text) == "" || !utf8.ValidString(text) || len(text) > maxChatMessageBytes {
		return errInvalidChatInput
	}
	return nil
}

func validateChatContext(messages []map[string]string) error {
	if len(messages) == 0 || len(messages) > maxChatMessages {
		return errInvalidChatInput
	}
	total := 0
	for _, message := range messages {
		role, content := message["role"], message["content"]
		if role != "user" && role != "assistant" {
			return errors.New("上下文角色无效")
		}
		if !utf8.ValidString(content) || len(content) > maxChatMessageBytes {
			return errInvalidChatInput
		}
		total += len(content)
		if total > maxChatContextBytes {
			return errInvalidChatInput
		}
	}
	return nil
}

func validateScreenshotInput(message, screenshot, previousContext string) error {
	if err := validateChatText(message); err != nil {
		return err
	}
	if !utf8.ValidString(previousContext) || len(previousContext) > maxChatContextBytes {
		return errInvalidChatInput
	}
	if len(screenshot) > maxScreenshotBytes || (screenshot != "" && !strings.HasPrefix(screenshot, "data:image/")) {
		return errors.New("截图内容无效或过大")
	}
	return nil
}

func safeProviderError() string { return "AI 服务请求失败，请稍后重试" }
