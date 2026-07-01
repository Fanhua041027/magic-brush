package agent

import (
	"context"
	"strings"

	"ai-assistant/pkg/llm"
)

// InterviewAgent manages the interview pipeline
type InterviewAgent struct {
	llmProvider   llm.Provider
	knowledgeBase *KnowledgeBase
	profileMgr    *ProfileManager
}

// NewInterviewAgent creates a new interview agent
func NewInterviewAgent(provider llm.Provider, kb *KnowledgeBase, pm *ProfileManager) *InterviewAgent {
	return &InterviewAgent{
		llmProvider:   provider,
		knowledgeBase: kb,
		profileMgr:    pm,
	}
}

// SetProvider updates the LLM provider
func (ia *InterviewAgent) SetProvider(provider llm.Provider) {
	ia.llmProvider = provider
}

// ExecuteInterviewPipeline runs the full Interview Agent pipeline
func (ia *InterviewAgent) ExecuteInterviewPipeline(
	ctx context.Context,
	query string,
	history []llm.Message,
	snapshot *ProfileSnapshot,
	cb AgentCallbacks,
) error {
	// Convert history to map format for router
	historyMaps := make([]map[string]string, 0, len(history))
	for _, msg := range history {
		role := string(msg.Role)
		content := msg.Content
		if content == "" && len(msg.Parts) > 0 {
			content = msg.Parts[0].Text
		}
		if content != "" {
			historyMaps = append(historyMaps, map[string]string{
				"role":    role,
				"content": content,
			})
		}
	}

	skillCards := ia.profileMgr.GetSkillCards()

	// === Router stage ===
	if cb.OnEvent != nil {
		cb.OnEvent(PipelineEvent{Stage: "router", Status: "running", Detail: "分析问题意图..."})
	}

	router := NewRouter(func(messages []map[string]string) (string, error) {
		var llmMessages []llm.Message
		for _, m := range messages {
			llmMessages = append(llmMessages, llm.NewTextMessage(
				llm.Role(m["role"]),
				m["content"],
			))
		}
		result, err := ia.llmProvider.GenerateContent(ctx, "", llmMessages)
		if err != nil {
			return "", err
		}
		return result.Content, nil
	})

	output := router.RouteQuery(query, historyMaps, skillCards)

	if cb.OnEvent != nil {
		cb.OnEvent(PipelineEvent{
			Stage:  "router",
			Status: "done",
			Detail: "意图: " + string(output.Intent) + ", 策略: " + string(output.Strategy),
		})
	}

	// === Retrieve stage (parallel with router) ===
	var passages []RetrievedPassage
	if output.NeedsRAG {
		if cb.OnEvent != nil {
			cb.OnEvent(PipelineEvent{Stage: "retrieve", Status: "running", Detail: "检索知识库..."})
		}

		passages = ia.knowledgeBase.Search(output.Query, 3, 0.1)

		if cb.OnEvent != nil {
			cb.OnEvent(PipelineEvent{
				Stage:  "retrieve",
				Status: "done",
				Detail: "检索到 " + itoa(len(passages)) + " 条结果",
			})
		}
	}

	// === Prepare stage ===
	if cb.OnEvent != nil {
		cb.OnEvent(PipelineEvent{Stage: "prepare", Status: "running", Detail: "组装回答上下文..."})
	}

	var profile *ProfileData
	if snapshot != nil {
		profile = &snapshot.Profile
	}

	prepared := PrepareContext(output.Query, output, passages, skillCards, profile)

	if cb.OnEvent != nil {
		ragUsed := "否"
		if prepared.RAGUsed {
			ragUsed = "是"
		}
		cb.OnEvent(PipelineEvent{
			Stage:  "prepare",
			Status: "done",
			Detail: "策略: " + string(prepared.Strategy) + ", RAG: " + ragUsed,
		})
	}

	// === Agent stage ===
	if cb.OnEvent != nil {
		cb.OnEvent(PipelineEvent{Stage: "agent", Status: "running", Detail: "生成回答..."})
	}

	// Build messages
	var messages []llm.Message
	messages = append(messages, llm.NewSystemMessage(prepared.SystemPrompt))

	// Add history (non-system messages, last 8)
	nonSysCount := 0
	for i := len(history) - 1; i >= 0 && nonSysCount < 8; i-- {
		if history[i].Role != llm.RoleSystem {
			messages = append(messages, history[i])
			nonSysCount++
		}
	}

	messages = append(messages, llm.NewUserMessage(query))

	// Stream response
	_, err := ia.llmProvider.GenerateContentStream(ctx, messages, func(chunk llm.StreamChunk) {
		if chunk.Type == llm.ChunkContent && cb.OnStream != nil {
			cb.OnStream(chunk.Content)
		}
	})

	if err != nil {
		if strings.Contains(err.Error(), "canceled") || strings.Contains(err.Error(), "Canceled") {
			return nil
		}
		if cb.OnStreamError != nil {
			cb.OnStreamError(err.Error())
		}
		if cb.OnEvent != nil {
			cb.OnEvent(PipelineEvent{Stage: "agent", Status: "error", Detail: err.Error()})
		}
		return err
	}

	if cb.OnStreamEnd != nil {
		cb.OnStreamEnd()
	}
	if cb.OnEvent != nil {
		cb.OnEvent(PipelineEvent{Stage: "agent", Status: "done", Detail: "回答完成"})
	}

	return nil
}

// GenerateAnswer generates a direct non-streaming answer for the listen overlay.
// This is a simplified version that returns the full answer text.
func (ia *InterviewAgent) GenerateAnswer(ctx context.Context, query string, snapshot *ProfileSnapshot) (string, error) {
	historyMaps := make([]map[string]string, 0)
	skillCards := ia.profileMgr.GetSkillCards()

	// Router
	router := NewRouter(func(messages []map[string]string) (string, error) {
		var llmMessages []llm.Message
		for _, m := range messages {
			llmMessages = append(llmMessages, llm.NewTextMessage(llm.Role(m["role"]), m["content"]))
		}
		result, err := ia.llmProvider.GenerateContent(ctx, "", llmMessages)
		if err != nil {
			return "", err
		}
		return result.Content, nil
	})

	output := router.RouteQuery(query, historyMaps, skillCards)

	// Retrieve (if needed)
	var passages []RetrievedPassage
	if output.NeedsRAG {
		passages = ia.knowledgeBase.Search(output.Query, 3, 0.1)
	}

	// Prepare
	var profile *ProfileData
	if snapshot != nil {
		profile = &snapshot.Profile
	}

	prepared := PrepareContext(output.Query, output, passages, skillCards, profile)

	// Generate answer (non-streaming)
	var messages []llm.Message
	messages = append(messages, llm.NewSystemMessage(prepared.SystemPrompt))
	messages = append(messages, llm.NewUserMessage(query))

	result, err := ia.llmProvider.GenerateContent(ctx, "", messages)
	if err != nil {
		return "", err
	}

	return result.Content, nil
}
