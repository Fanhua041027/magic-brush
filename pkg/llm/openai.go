package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ai-assistant/pkg/config"
	"ai-assistant/pkg/logger"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type OpenAIAdapter struct {
	client *openai.Client
	config *config.Config
}

func NewOpenAIAdapter(cfg *config.Config) *OpenAIAdapter {
	model := cfg.Model
	if model == "" {
		model = openai.ChatModelGPT4o
	}

	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	opts := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
		option.WithBaseURL(baseURL),
	}

	client := openai.NewClient(opts...)

	return &OpenAIAdapter{
		client: &client,
		config: cfg,
	}
}

func (a *OpenAIAdapter) toOpenAIMessages(messages []Message) []openai.ChatCompletionMessageParamUnion {
	result := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))

	for _, msg := range messages {
		switch msg.Role {
		case RoleSystem:
			result = append(result, openai.SystemMessage(msg.Content))

		case RoleUser:
			if len(msg.Parts) > 0 {
				result = append(result, openai.UserMessage(a.toOpenAIParts(msg.Parts)))
			} else {
				result = append(result, openai.UserMessage(msg.Content))
			}

		case RoleAssistant:
			result = append(result, openai.AssistantMessage(msg.Content))
		}
	}

	return result
}

func (a *OpenAIAdapter) toOpenAIParts(parts []ContentPart) []openai.ChatCompletionContentPartUnionParam {
	result := make([]openai.ChatCompletionContentPartUnionParam, 0, len(parts))

	hasNonText := false
	for _, p := range parts {
		if p.Type != ContentText {
			hasNonText = true
			break
		}
	}

	// 检查模型是否支持图片输入
	supportsVision := a.supportsVision()

	// 如果不支持视觉，将所有非文本内容转为文本占位
	if !supportsVision && hasNonText {
		var textBuilder strings.Builder
		for _, part := range parts {
			if part.Type == ContentText {
				textBuilder.WriteString(part.Text)
				textBuilder.WriteString("\n")
			} else {
				textBuilder.WriteString("[用户上传了一张截图，请根据已提供的对话上下文回答问题]\n")
			}
		}
		result = append(result, openai.TextContentPart(textBuilder.String()))
		return result
	}

	for _, part := range parts {
		switch part.Type {
		case ContentText:
			result = append(result, openai.TextContentPart(part.Text))
		case ContentImage:
			if supportsVision {
				result = append(result, openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
					URL: part.Base64,
				}))
			} else {
				result = append(result, openai.TextContentPart("[图片]"))
			}
		case ContentPDF:
			if supportsVision {
				result = append(result, openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
					URL: part.Base64,
				}))
			} else {
				result = append(result, openai.TextContentPart("[PDF附件]"))
			}
		}
	}

	return result
}

// supportsVision 检查当前配置的模型是否支持图片/视觉输入
func (a *OpenAIAdapter) supportsVision() bool {
	model := strings.ToLower(a.config.Model)

	// 明确不支持视觉的模型列表
	noVisionModels := []string{
		"deepseek", "deepseek-chat",
		"deepseek-v3", "deepseek-r1",
		"gpt-3.5", "gpt-4-turbo",
	}

	for _, nv := range noVisionModels {
		if strings.Contains(model, nv) {
			return false
		}
	}

	// 明确支持视觉的模型列表
	visionModels := []string{
		"gpt-4o", "gpt-4.1", "gpt-4.5",
		"claude-3", "claude-3.5", "claude-4", "claude-fable", "claude-opus", "claude-sonnet", "claude-haiku",
		"gemini-2.0", "gemini-2.5",
		"qwen3.6", "qwen-vl", "qwen2-vl", "qwen2.5-vl", "qwen2.5-vl",
	}

	for _, vm := range visionModels {
		if strings.Contains(model, vm) {
			return true
		}
	}

	return false
}

func (a *OpenAIAdapter) GenerateContentStream(ctx context.Context, messages []Message, onChunk StreamCallback) (Message, error) {
	openaiMessages := a.toOpenAIMessages(messages)

	stream := a.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Model:    a.config.Model,
		Messages: openaiMessages,
	})

	defer stream.Close()

	var fullContent strings.Builder
	var fullThinking strings.Builder

	for stream.Next() {
		evt := stream.Current()

		if len(evt.Choices) > 0 {
			if thinkingRaw, ok := evt.Choices[0].Delta.JSON.ExtraFields["reasoning"]; ok {
				a.handleThinkingChunk(thinkingRaw.Raw(), &fullThinking, onChunk)
			} else if thinkingRaw, ok := evt.Choices[0].Delta.JSON.ExtraFields["reasoning_content"]; ok {
				a.handleThinkingChunk(thinkingRaw.Raw(), &fullThinking, onChunk)
			}

			content := evt.Choices[0].Delta.Content
			if content != "" {
				fullContent.WriteString(content)
				if onChunk != nil {
					onChunk(StreamChunk{Type: ChunkContent, Content: content})
				}
			}
		}
	}

	if err := stream.Err(); err != nil {
		return Message{}, a.parseError(err)
	}

	return Message{
		Role:     RoleAssistant,
		Content:  fullContent.String(),
		Thinking: fullThinking.String(),
	}, nil
}

func (a *OpenAIAdapter) handleThinkingChunk(rawJSON string, builder *strings.Builder, onChunk StreamCallback) {
	var decoded string
	if err := json.Unmarshal([]byte(rawJSON), &decoded); err != nil {
		logger.Printf("解析思考过程失败: %v", err)
		return
	}
	builder.WriteString(decoded)
	if onChunk != nil {
		onChunk(StreamChunk{Type: ChunkThinking, Content: decoded})
	}
}

func (a *OpenAIAdapter) parseError(err error) error {
	errStr := err.Error()

	startIndex := strings.Index(errStr, "{")
	if startIndex == -1 {
		return fmt.Errorf("请求失败，请稍后重试")
	}

	jsonPart := errStr[startIndex:]
	var response struct {
		StatusCode int    `json:"statusCode"`
		Code       string `json:"code"`
		Message    string `json:"message"`
		Type       string `json:"type"`
	}

	if parseErr := json.Unmarshal([]byte(jsonPart), &response); parseErr != nil {
		return fmt.Errorf("请求失败，请稍后重试")
	}

	if response.Message != "" {
		return fmt.Errorf("%s", response.Message)
	}

	return fmt.Errorf("请求失败，请稍后重试")
}

func (a *OpenAIAdapter) TestChat(ctx context.Context) error {
	_, err := a.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: a.config.Model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("hi"),
		},
		MaxTokens: openai.Int(17),
	})
	if err != nil {
		return a.parseError(err)
	}
	return nil
}

func (a *OpenAIAdapter) GenerateContent(ctx context.Context, model string, messages []Message) (Message, error) {
	if model == "" {
		model = a.config.Model
	}

	openaiMessages := a.toOpenAIMessages(messages)

	resp, err := a.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    model,
		Messages: openaiMessages,
	})

	if err != nil {
		return Message{}, a.parseError(err)
	}

	content := ""
	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.Content
	}

	return Message{
		Role:    RoleAssistant,
		Content: content,
	}, nil
}

func (a *OpenAIAdapter) GetModels(ctx context.Context) ([]string, error) {
	resp, err := a.client.Models.List(ctx)
	if err != nil {
		logger.Println("获取模型失败:", err.Error())
		return nil, a.parseError(err)
	}

	var models []string
	for _, m := range resp.Data {
		models = append(models, m.ID)
	}
	return models, nil
}
