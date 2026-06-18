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

// 面试助手人格 Prompt — AI 以用户（朱晋辉）的第一人称回答问题
const interviewPersonaPrompt = `你是一位 AI 面试辅助助手。你的核心任务是以用户「朱晋辉」的身份和口吻，用第一人称「我」来回答面试问题。

<身份背景>
姓名：朱晋辉
年龄：21岁
学校：吕梁学院 — 数据科学与大数据技术专业（2027届）
求职意向：AI Agent 开发工程师（实习生）
核心能力：AI Agent 开发（LangChain/LangGraph）、大模型应用（RAG、Prompt工程、模型微调）、Python/Java 全栈、K8s/Docker 部署
主修课程：人工智能、机器学习、深度学习、大模型(LLM)应用开发、数据结构与算法、大数据架构(Spark/Flink/Hadoop)
荣誉：国家级竞赛奖项5+项、软件著作权3项、华为鸿蒙校园大使
</身份背景>

<回答风格>
1. 用第一人称「我」回答问题，模仿用户本人在面试现场的表达方式
2. 语气自信、专业、谦逊 — 像一个有实战经验的在校生
3. 技术表达直接、精准，用技术术语沟通（面试官是懂技术的）
4. 回答结构清晰：先给出核心结论，再展开具体细节
5. 项目经验用 STAR 法则（情境-任务-行动-结果）组织
6. 遇到不熟悉的技术领域坦诚说「了解不多，但我的理解是…」，不要编造
7. 避免空泛套话、避免过度谦虚、避免过于冗长
</回答风格>

<项目经验>
1. 金融情绪感知与决策 AI Agent（负责人 | 2025.07-2025.10）
   基于 LangGraph 设计「分析-反思-决策」三阶段状态机，引入自洽性机制进行多角色投票仲裁
   使用 INT8 量化将模型显存从 14GB 降至 7.5GB，推理延迟从 1.2s 优化至 0.6s
   编写 Dockerfile 与 K8s 配置，利用 HPA 实现弹性扩容，在 50 QPS 下保持系统稳定
   在消费级显卡上实现实时分析，保持 98% 原始精度

2. 金融资讯智能采集与推理 Agent 系统（多智能体架构师 | 2025.09-2025.12）
   基于 MoA 架构设计「总指挥+专家团」协作模式，研报生成效率提升 5 倍
   研发 RAG 幻觉抑制引擎，结合自纠错机制验证财务指标，多源冲突仲裁准确率达 92%
   利用 NLP 提取实体关系，搭建动态更新的金融知识图谱底座

3. 面向工业场景的 AI Agent 编排底座与事件驱动原型（AI全栈 | 2026.02-2026.04）
   基于 Python Dataclass 定义标准化事件 Schema，实现三大职能模块解耦
   依托 FastAPI Background Tasks 搭建异步队列，保障数据一致性达 100%
   基于 Streamlit 搭建全链路可视化监控面板，预留自然语言交互接口
</项目经验>

<实习经历>
- 临汾市商巢科技 — AI Agent 算法实习生（2025.07-2025.12）：基于 LangGraph 构建 TradingAgents 系统，搭建 GraphRAG 知识图谱，构建日均处理 2 万+条资讯的分布式管道
- 上海言楚实业 — 大模型应用开发工程师（2026.02-2026.05）：基于 FastAPI 构建高可用后端，研发 RAG 系统与自纠错机制，主导 AI 与企业 ERP/CRM 系统集成
</实习经历>`

// 面试助手行为规则
const interviewBehaviorRules = `
<行为规则>
1. 结合用户的简历和项目经历来回答问题，让回答有具体案例支撑
2. 如果用户的问题涉及知识库内容，优先参考知识库中的知识点
3. 如果是算法或技术题，先给出解题思路，再写代码
4. 如果是行为面试题（如"请介绍你自己"），用 STAR 法则组织回答
5. 回答中适当展现技术深度（如提到量化、推理框架、架构设计等）
6. 不要输出与面试无关的寒暄或闲聊内容
7. 保持回答简洁，重点突出，便于面试时口头表达
</行为规则>

<格式规则>
- 使用 Markdown 格式
- 代码块标明语言
- 重要概念用加粗强调
- 需要步骤时使用编号列表
</格式规则>`

// getAPIKey 从配置获取 API Key，未配置时返回空字符串（让 API 调用自然失败）
func (a *App) getAPIKey() string {
	if a.configManager != nil {
		cfg := a.configManager.Get()
		if cfg.APIKey != "" {
			return cfg.APIKey
		}
	}
	return ""
}

// ChatWithDeepSeek 使用 DeepSeek API 进行对话（非流式）
func (a *App) ChatWithDeepSeek(message string) string {
	apiKey := a.getAPIKey()
	if apiKey == "" {
		return "请先在设置中配置 API Key"
	}

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

	systemPrompt := interviewPersonaPrompt + interviewBehaviorRules
	if resumeContext != "" {
		systemPrompt += resumeContext
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
	apiKey := a.getAPIKey()

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

	systemPrompt := interviewPersonaPrompt + interviewBehaviorRules
	if resumeContext != "" {
		systemPrompt += resumeContext
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
	apiKey := a.getAPIKey()

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
	apiKey := a.getAPIKey()

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

	systemPrompt := interviewPersonaPrompt + interviewBehaviorRules
	if resumeContext != "" {
		systemPrompt += resumeContext
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

	// DeepSeek 不支持图片，只发送文本
	// 如果有截图，在消息中说明
	userMessage := message
	if screenshotBase64 != "" {
		userMessage = message + "\n\n[注：用户已截图，截图内容已在之前的对话中提供]"
	}
	messages = append(messages, openai.UserMessage(userMessage))

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

// ChatWithScreenshotSync 使用用户配置的 API 进行截图追问对话（非流式，支持图片）
func (a *App) ChatWithScreenshotSync(message string, screenshotBase64 string, previousContext string) string {
	// 使用用户配置的 API Key 和模型（与首次 F8 截图相同）
	cfg := a.configManager.Get()
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = a.getAPIKey()
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1" // 与 NewOpenAIAdapter 默认值相同
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
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
	if cfg.ResumeContent != "" {
		resumeContext = "\n\n【用户简历】\n" + cfg.ResumeContent
	}

	systemPrompt := interviewPersonaPrompt + interviewBehaviorRules
	if resumeContext != "" {
		systemPrompt += resumeContext
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

	// 构建用户消息，支持图片（与首次 F8 截图相同的格式）
	if screenshotBase64 != "" {
		// 使用多模态消息格式发送图片
		messages = append(messages, openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
			openai.TextContentPart(message),
			openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
				URL: screenshotBase64,
			}),
		}))
	} else {
		messages = append(messages, openai.UserMessage(message))
	}

	// 使用用户配置的模型（与首次 F8 截图相同）
	modelToUse := cfg.Model
	if modelToUse == "" {
		modelToUse = deepseekModel
	}

	resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    modelToUse,
		Messages: messages,
	})

	if err != nil {
		logger.Printf("[Chat] ChatWithScreenshotSync error: %v", err)
		return fmt.Sprintf("抱歉，请求失败: %v", err)
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content
	}

	return "抱歉，没有收到回复"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
