package app

import (
	"ai-assistant/pkg/logger"
	"context"
	"fmt"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// DeepSeek API 配置（F7 面试助手强制使用）
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
求职意向：AI应用开发工程师（实习生）
核心能力：AI Agent 开发与编排（LangChain/LangGraph）、大模型应用（RAG、Prompt工程、模型微调）、Python/Java 全栈、K8s/Docker 部署
主修课程：人工智能、机器学习、深度学习、大模型(LLM)应用开发、LangChain/LangGraph框架、数据结构与算法
荣誉：省级以上竞赛奖项13+项、软件著作权4项、华为鸿蒙校园大使
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
1. 金融智能决策与投研多智能体系统（币星人）（负责人 | 2025.06-2025.12）
   基于 LangGraph 设计「总指挥+专家团」协作模式与 FSM 状态机，MoA 架构使研报生成效率提升 5 倍
   AWQ-4bit 量化将显存从 64GB 压缩至单卡可承载范围，单句推理延迟优化至 0.7s，精度损失<2%
   K8s 弹性伸缩确保单副本 50 QPS 无请求堆积，GPT-4o-mini 异步兜底保障服务 99.9% 可用性
   构建 RAG+代码解释器+知识图谱三元幻觉抑制引擎，多源冲突仲裁准确率达 92%
   Playwright 采集 4 个财经网站，NLP 提取实体关系，日均处理 5000 条原始文本

2. 工业级ERP系统Agent调度中间件（AI全栈 | 2026.02-2026.05）
   采用 FastAPI 构建独立 Agent 网关，定义基于 Pydantic 的强类型请求/响应 Schema 对接 Java 后端
   实现内存队列重试+超时熔断机制，构造 20+ 异常用例验证系统降级表现，无脏数据产生
   Agent 指令执行成功率从 72% 提升至 96%，在采购审批、库存预警、工单派发场景完成端到端集成
   开发 Streamlit 可视化控制台及 NLP-to-API 映射中间件，简单指令解析准确率达 90%
</项目经验>

<实习经历>
- 临汾市商巢科技 — 后端开发实习生（2025.06-2025.12）：搭建 GraphRAG 与金融知识图谱，关键资讯筛选准确率从 78% 提升至 92%；基于 LangGraph 构建 TradingAgents 多智能体系统；运维日均 2 万+条资讯的分布式采集管道
- 上海言楚实业 — AI全栈开发实习生（2026.02-2026.06）：基于 Python/FastAPI 构建高可用后端，设计多 Agent 协同架构与标准化通信协议；落地 Saga 模式补偿机制与深度容错策略，实现异常场景自动回滚与数据最终一致性
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

// ChatWithDeepSeek 使用 DeepSeek API 进行对话（非流式）—— 仅 F7 使用
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
		logger.Printf("[Chat] API error: %v", err)
		return fmt.Sprintf("抱歉，请求失败: %v", err)
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content
	}

	return "抱歉，没有收到回复"
}

// ChatWithDeepSeekStream 使用 DeepSeek API 进行对话（流式输出）—— 仅 F7 使用
func (a *App) ChatWithDeepSeekStream(message string) {
	apiKey := a.getAPIKey()
	if apiKey == "" {
		a.EmitEvent("chat-stream-error", "请先在设置中配置 API Key")
		return
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

// ChatWithDeepSeekStreamWithContext 使用 DeepSeek API 进行带上下文的对话（流式输出）—— 仅 F7 使用
func (a *App) ChatWithDeepSeekStreamWithContext(messages []map[string]string) {
	apiKey := a.getAPIKey()
	if apiKey == "" {
		a.EmitEvent("chat-stream-error", "请先在设置中配置 API Key")
		return
	}

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

// ChatWithScreenshot 使用用户配置的 API 进行截图追问对话（流式输出）—— F8 追问使用
func (a *App) ChatWithScreenshot(message string, screenshotBase64 string, previousContext string) {
	// 使用用户配置的 API Key 和模型（与 F8 截图相同）
	cfg := a.configManager.Get()
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = a.getAPIKey()
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
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

	// 如果有多模态模型则传图，否则文本说明
	userMessage := message
	if screenshotBase64 != "" {
		userMessage = message + "\n\n[注：用户已截图，截图内容已在之前的对话中提供]"
	}
	messages = append(messages, openai.UserMessage(userMessage))

	// 使用用户配置的模型
	modelToUse := cfg.Model
	if modelToUse == "" {
		modelToUse = deepseekModel
	}

	stream := client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Model:    modelToUse,
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
