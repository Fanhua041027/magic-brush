package app

import (
	"ai-assistant/pkg/logger"
	"context"
	"fmt"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// DeepSeek API 配置
const (
	deepseekBaseURL = "https://api.deepseek.com"
	deepseekModel   = "deepseek-chat"
)

// ChatWithDeepSeek 使用 DeepSeek API 进行对话（非流式）
func (a *App) ChatWithDeepSeek(message string) string {
	apiKey := "sk-eeab470faa664cb8a3c954554354e711"

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(deepseekBaseURL),
	)

	ctx := context.Background()

	// 搜索知识库
	kbContext := ""
	if a.sidecar != nil && a.sidecar.IsRunning() {
		result, err := a.sidecar.Client().KBSearch(message, 3)
		if err == nil && len(result.Results) > 0 {
			kbContext = "\n\n【参考知识库】\n"
			for _, r := range result.Results {
				kbContext += fmt.Sprintf("- %s: %s\n", r.Header, r.Content[:min(200, len(r.Content))])
			}
		}
	}

	// 获取简历内容
	resumeContext := ""
	cfg := a.configManager.Get()
	if cfg.ResumeContent != "" {
		resumeContext = "\n\n【用户简历】\n" + cfg.ResumeContent
	}

	systemPrompt := "你是一个有用的AI助手。请用中文回答用户的问题。"
	if resumeContext != "" {
		systemPrompt += resumeContext
		systemPrompt += "\n\n请根据用户的简历内容回答问题，如果问题与简历相关，请结合简历信息进行回答。"
	}
	if kbContext != "" {
		systemPrompt += kbContext
	}

	resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: deepseekModel,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(message),
		},
	})

	if err != nil {
		logger.Printf("[Chat] DeepSeek API error: %v", err)
		return fmt.Sprintf("抱歉，请求失败: %v", err)
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content
	}

	return "抱歉，没有收到回复"
}

// ChatWithDeepSeekStream 使用 DeepSeek API 进行对话（流式输出）
func (a *App) ChatWithDeepSeekStream(message string) {
	apiKey := "sk-eeab470faa664cb8a3c954554354e711"

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(deepseekBaseURL),
	)

	ctx := context.Background()

	// 搜索知识库
	kbContext := ""
	if a.sidecar != nil && a.sidecar.IsRunning() {
		result, err := a.sidecar.Client().KBSearch(message, 3)
		if err == nil && len(result.Results) > 0 {
			kbContext = "\n\n【参考知识库】\n"
			for _, r := range result.Results {
				kbContext += fmt.Sprintf("- %s: %s\n", r.Header, r.Content[:min(200, len(r.Content))])
			}
		}
	}

	// 获取简历内容
	resumeContext := ""
	cfg := a.configManager.Get()
	if cfg.ResumeContent != "" {
		resumeContext = "\n\n【用户简历】\n" + cfg.ResumeContent
	}

	systemPrompt := "你是一个有用的AI助手。请用中文回答用户的问题。"
	if resumeContext != "" {
		systemPrompt += resumeContext
		systemPrompt += "\n\n请根据用户的简历内容回答问题，如果问题与简历相关，请结合简历信息进行回答。"
	}
	if kbContext != "" {
		systemPrompt += kbContext
	}

	stream := client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Model: deepseekModel,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(message),
		},
	})

	defer stream.Close()

	for stream.Next() {
		evt := stream.Current()
		if len(evt.Choices) > 0 {
			content := evt.Choices[0].Delta.Content
			if content != "" {
				a.EmitEvent("chat-stream-chunk", content)
			}
		}
	}

	if err := stream.Err(); err != nil {
		logger.Printf("[Chat] Stream error: %v", err)
		a.EmitEvent("chat-stream-error", err.Error())
		return
	}

	a.EmitEvent("chat-stream-done")
}

// ChatWithDeepSeekStreamWithContext 使用 DeepSeek API 进行带上下文的对话（流式输出）
func (a *App) ChatWithDeepSeekStreamWithContext(messages []map[string]string) {
	apiKey := "sk-eeab470faa664cb8a3c954554354e711"

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(deepseekBaseURL),
	)

	ctx := context.Background()

	// 转换消息格式
	openaiMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, msg := range messages {
		role := msg["role"]
		content := msg["content"]
		switch role {
		case "system":
			openaiMessages = append(openaiMessages, openai.SystemMessage(content))
		case "user":
			openaiMessages = append(openaiMessages, openai.UserMessage(content))
		case "assistant":
			openaiMessages = append(openaiMessages, openai.AssistantMessage(content))
		}
	}

	stream := client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Model:    deepseekModel,
		Messages: openaiMessages,
	})

	defer stream.Close()

	for stream.Next() {
		evt := stream.Current()
		if len(evt.Choices) > 0 {
			content := evt.Choices[0].Delta.Content
			if content != "" {
				a.EmitEvent("chat-stream-chunk", content)
			}
		}
	}

	if err := stream.Err(); err != nil {
		logger.Printf("[Chat] Stream error: %v", err)
		a.EmitEvent("chat-stream-error", err.Error())
		return
	}

	a.EmitEvent("chat-stream-done")
}

// ChatWithScreenshot 使用 DeepSeek API 进行截图追问对话（流式输出）
func (a *App) ChatWithScreenshot(message string, screenshotBase64 string, previousContext string) {
	apiKey := "sk-eeab470faa664cb8a3c954554354e711"

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(deepseekBaseURL),
	)

	ctx := context.Background()

	// 搜索知识库
	kbContext := ""
	if a.sidecar != nil && a.sidecar.IsRunning() {
		result, err := a.sidecar.Client().KBSearch(message, 3)
		if err == nil && len(result.Results) > 0 {
			kbContext = "\n\n【参考知识库】\n"
			for _, r := range result.Results {
				kbContext += fmt.Sprintf("- %s: %s\n", r.Header, r.Content[:min(200, len(r.Content))])
			}
		}
	}

	// 获取简历内容
	resumeContext := ""
	cfg := a.configManager.Get()
	if cfg.ResumeContent != "" {
		resumeContext = "\n\n【用户简历】\n" + cfg.ResumeContent
	}

	systemPrompt := "你是一个有用的AI助手。用户会发送截图和问题，请根据截图内容回答问题。请用中文回答。"
	if resumeContext != "" {
		systemPrompt += resumeContext
		systemPrompt += "\n\n请根据用户的简历内容回答问题，如果问题与简历相关，请结合简历信息进行回答。"
	}
	if kbContext != "" {
		systemPrompt += kbContext
	}
	if previousContext != "" {
		systemPrompt += "\n\n【之前的对话上下文】\n" + previousContext
	}

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
	}

	// 如果有截图，添加图片消息
	if screenshotBase64 != "" {
		messages = append(messages, openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
			openai.TextContentPart(message),
			openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
				URL: screenshotBase64,
			}),
		}))
	} else {
		messages = append(messages, openai.UserMessage(message))
	}

	stream := client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Model:    deepseekModel,
		Messages: messages,
	})

	defer stream.Close()

	for stream.Next() {
		evt := stream.Current()
		if len(evt.Choices) > 0 {
			content := evt.Choices[0].Delta.Content
			if content != "" {
				a.EmitEvent("chat-stream-chunk", content)
			}
		}
	}

	if err := stream.Err(); err != nil {
		logger.Printf("[Chat] Stream error: %v", err)
		a.EmitEvent("chat-stream-error", err.Error())
		return
	}

	a.EmitEvent("chat-stream-done")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
