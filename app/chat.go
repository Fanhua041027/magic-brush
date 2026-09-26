package app

import (
	"ai-assistant/pkg/logger"
	"fmt"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// AI 面试助手使用设置页中配置的通用模型与 API 地址。
func (a *App) interviewClient() (openai.Client, string, error) {
	if a.authService == nil || a.authService.Current() == nil {
		return openai.Client{}, "", fmt.Errorf("请先登录后使用 AI 服务")
	}
	cfg := a.configManager.Get()
	if cfg.APIKey == "" {
		return openai.Client{}, "", fmt.Errorf("请先在设置中配置 API Key")
	}
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	model := cfg.Model
	if model == "" {
		return openai.Client{}, "", fmt.Errorf("请先在设置中配置模型")
	}
	return openai.NewClient(option.WithAPIKey(cfg.APIKey), option.WithBaseURL(baseURL)), model, nil
}

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

<回答要求>
- **第一人称**：从我（朱晋辉）的角度说话，语气像在校生，自信且谦逊
- **结构化**：先说结论/观点，再跟一个关键数据或案例支撑，适当展开
- **口语化**：适合面试时口头表达，不要书面语
- **不熟悉的**：坦诚说"这块我了解不多，我的理解是…"
- **不说废话**：不要总结、不要铺垫、不要说"首先/其次/总的来说"
- **字数控制**：普通问题回答 ≤ 300 字，复杂问题 ≤ 600 字
</回答要求>

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
<回答规则>
- **先结论后展开**：直接亮观点，再用具体数据或项目案例支撑，适当展开细节
- **STAR 法则**：用情境-任务-行动-结果组织项目描述，每个环节 1-2 句话
- **技术题**：先讲核心原理，再结合你的实践经验说明
- **行为题**：用一个具体事例 + 量化结果说明
- **代码题**：先讲思路，再写代码，关键行加注释
- **遇到追问**：只补充新信息，不重复前面说过的

<格式>
- 口语化表达，适合面试时直接说出来
- 重要概念自然加重语气
- 代码用代码块（标明语言）
</回答规则>`

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
	if err := validateChatText(message); err != nil {
		return err.Error()
	}
	client, model, err := a.interviewClient()
	if err != nil {
		return err.Error()
	}

	ctx, taskID := a.taskManager.StartTask("chat")
	defer a.taskManager.CompleteTask(taskID)

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
		Model: model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(message),
		},
	})

	if err != nil {
		logger.Printf("[Chat] API request failed")
		return safeProviderError()
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content
	}

	return "抱歉，没有收到回复"
}

// ChatWithDeepSeekStream 使用 DeepSeek API 进行对话（流式输出）—— 仅 F7 使用
func (a *App) ChatWithDeepSeekStream(requestID, message string) {
	if !validRequestID(requestID) {
		return
	}
	if err := validateChatText(message); err != nil {
		a.EmitEvent("chat-stream-error", map[string]any{"requestId": requestID, "error": err.Error()})
		return
	}
	client, model, err := a.interviewClient()
	if err != nil {
		a.EmitEvent("chat-stream-error", map[string]any{"requestId": requestID, "error": err.Error()})
		return
	}

	ctx, taskID := a.taskManager.StartRequest("chat", requestID)
	defer a.taskManager.CompleteTask(taskID)

	// 搜索知识库
	kbContext := ""
	if a.sidecar != nil && a.sidecar.IsRunning() {
		result, err := a.sidecar.Client().KBSearchContext(ctx, message, 3)
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
		Model: model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(message),
		},
	})

	defer stream.Close()

	receivedContent := false
	for stream.Next() {
		evt := stream.Current()
		if len(evt.Choices) > 0 {
			content := evt.Choices[0].Delta.Content
			if content != "" {
				receivedContent = true
				a.emitRequestEvent(ctx, taskID, "chat-stream-chunk", requestID, "chunk", content)
			}
		}
	}

	if err := stream.Err(); err != nil {
		logger.Printf("[Chat] Stream request failed")
		a.emitRequestEvent(ctx, taskID, "chat-stream-error", requestID, "error", safeProviderError())
		return
	}

	if !receivedContent {
		a.emitRequestEvent(ctx, taskID, "chat-stream-error", requestID, "error", "模型没有返回内容")
		return
	}
	a.emitRequestEvent(ctx, taskID, "chat-stream-done", requestID, "", nil)
}

// ChatWithDeepSeekStreamWithContext 使用 DeepSeek API 进行带上下文的对话（流式输出）—— 仅 F7 使用
func (a *App) ChatWithDeepSeekStreamWithContext(requestID string, messages []map[string]string) {
	if !validRequestID(requestID) {
		return
	}
	if err := validateChatContext(messages); err != nil {
		a.EmitEvent("chat-stream-error", map[string]any{"requestId": requestID, "error": err.Error()})
		return
	}
	client, model, err := a.interviewClient()
	if err != nil {
		a.EmitEvent("chat-stream-error", map[string]any{"requestId": requestID, "error": err.Error()})
		return
	}

	ctx, taskID := a.taskManager.StartRequest("chat", requestID)
	defer a.taskManager.CompleteTask(taskID)

	// 转换消息格式
	openaiMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages)+1)
	openaiMessages = append(openaiMessages, openai.SystemMessage(interviewPersonaPrompt+interviewBehaviorRules))
	for _, msg := range messages {
		role := msg["role"]
		content := msg["content"]
		switch role {
		case "user":
			openaiMessages = append(openaiMessages, openai.UserMessage(content))
		case "assistant":
			openaiMessages = append(openaiMessages, openai.AssistantMessage(content))
		}
	}

	stream := client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Model:    model,
		Messages: openaiMessages,
	})

	defer stream.Close()

	receivedContent := false
	for stream.Next() {
		evt := stream.Current()
		if len(evt.Choices) > 0 {
			content := evt.Choices[0].Delta.Content
			if content != "" {
				receivedContent = true
				a.emitRequestEvent(ctx, taskID, "chat-stream-chunk", requestID, "chunk", content)
			}
		}
	}

	if err := stream.Err(); err != nil {
		logger.Printf("[Chat] Stream request failed")
		a.emitRequestEvent(ctx, taskID, "chat-stream-error", requestID, "error", safeProviderError())
		return
	}

	if !receivedContent {
		a.emitRequestEvent(ctx, taskID, "chat-stream-error", requestID, "error", "模型没有返回内容")
		return
	}
	a.emitRequestEvent(ctx, taskID, "chat-stream-done", requestID, "", nil)
}

// ChatWithScreenshot 进行截图追问对话（流式输出）—— F8 追问使用
// 优先使用截图专用配置（ScreenshotAPIKey/ScreenshotBaseURL/ScreenshotModel），
// 无截图专用配置时回退到通用配置
func (a *App) ChatWithScreenshot(requestID, message string, screenshotBase64 string, previousContext string) {
	if !validRequestID(requestID) {
		return
	}
	if a.authService == nil || a.authService.Current() == nil {
		a.EmitEvent("chat-stream-error", map[string]any{"requestId": requestID, "error": "请先登录后使用 AI 服务"})
		return
	}
	if err := validateScreenshotInput(message, screenshotBase64, previousContext); err != nil {
		a.EmitEvent("chat-stream-error", map[string]any{"requestId": requestID, "error": err.Error()})
		return
	}
	cfg := a.configManager.Get()
	apiKey := cfg.ScreenshotAPIKey
	if apiKey == "" {
		apiKey = cfg.APIKey
	}
	if apiKey == "" {
		apiKey = a.getAPIKey()
	}

	baseURL := cfg.ScreenshotBaseURL
	if baseURL == "" {
		baseURL = cfg.BaseURL
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	modelToUse := cfg.ScreenshotModel
	if modelToUse == "" {
		modelToUse = cfg.Model
	}
	if modelToUse == "" {
		modelToUse = "gpt-4o-mini"
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	)

	ctx, taskID := a.taskManager.StartRequest("chat", requestID)
	defer a.taskManager.CompleteTask(taskID)

	// 搜索知识库
	kbContext := ""
	if a.sidecar != nil && a.sidecar.IsRunning() {
		result, err := a.sidecar.Client().KBSearchContext(ctx, message, 3)
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

	// 构建用户消息；截图追问必须使用多模态格式，不能只发送文字占位符。
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
		Model:    modelToUse,
		Messages: messages,
	})

	defer stream.Close()

	receivedContent := false
	for stream.Next() {
		evt := stream.Current()
		if len(evt.Choices) > 0 {
			content := evt.Choices[0].Delta.Content
			if content != "" {
				receivedContent = true
				a.emitRequestEvent(ctx, taskID, "chat-stream-chunk", requestID, "chunk", content)
			}
		}
	}

	if err := stream.Err(); err != nil {
		logger.Printf("[Chat] Stream request failed")
		a.emitRequestEvent(ctx, taskID, "chat-stream-error", requestID, "error", safeProviderError())
		return
	}

	if !receivedContent {
		a.emitRequestEvent(ctx, taskID, "chat-stream-error", requestID, "error", "模型没有返回内容")
		return
	}
	a.emitRequestEvent(ctx, taskID, "chat-stream-done", requestID, "", nil)
}

// ChatWithScreenshotSync 使用用户配置的 API 进行截图追问对话（非流式，支持图片）
func (a *App) ChatWithScreenshotSync(message string, screenshotBase64 string, previousContext string) string {
	if a.authService == nil || a.authService.Current() == nil {
		return "请先登录后使用 AI 服务"
	}
	if err := validateScreenshotInput(message, screenshotBase64, previousContext); err != nil {
		return err.Error()
	}
	cfg := a.configManager.Get()
	apiKey := cfg.ScreenshotAPIKey
	if apiKey == "" {
		apiKey = cfg.APIKey
	}
	if apiKey == "" {
		apiKey = a.getAPIKey()
	}

	baseURL := cfg.ScreenshotBaseURL
	if baseURL == "" {
		baseURL = cfg.BaseURL
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	)

	ctx, taskID := a.taskManager.StartTask("chat")
	defer a.taskManager.CompleteTask(taskID)

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

	modelToUse := cfg.ScreenshotModel
	if modelToUse == "" {
		modelToUse = cfg.Model
	}
	if modelToUse == "" {
		modelToUse = "gpt-4o-mini"
	}

	resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    modelToUse,
		Messages: messages,
	})

	if err != nil {
		logger.Printf("[Chat] Screenshot request failed")
		return safeProviderError()
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
