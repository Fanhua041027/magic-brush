package agent

import (
	"encoding/json"
	"strings"
)

// Router is responsible for intent classification using an LLM call
type Router struct {
	callLLM func(messages []map[string]string) (string, error)
}

// NewRouter creates a new Router
func NewRouter(callLLM func(messages []map[string]string) (string, error)) *Router {
	return &Router{callLLM: callLLM}
}

// RouteQuery classifies a user query using the LLM
func (r *Router) RouteQuery(query string, history []map[string]string, skillCards []SkillCard) RouterOutput {
	// Build skill card list
	var skillList string
	for _, s := range skillCards {
		skillList += "- [" + s.ID + "] " + s.ProjectName + " (" +
			strings.Join(s.TechStack, ", ") + "): " + s.Summary + "\n"
	}
	if skillList == "" {
		skillList = "（无技能卡片）"
	}

	systemPrompt := `你是一个面试问题分类器。分析用户问题并输出 JSON。

## 意图分类
- self_intro: 自我介绍类问题 ("请做个自我介绍")
- project_deep_dive: 项目深入 ("详细讲讲那个项目")
- behavioral: 行为面试 ("讲讲你遇到的一个挑战")
- technical: 技术问题 ("解释一下X是怎么工作的")
- scenario: 情景题 ("如果发生X你会怎么做")
- follow_up: 追问 ("能再展开说说吗")
- general: 以上都不匹配

## 回答策略
- self_introduction: 自我介绍
- star_project: 项目深入/STAR
- behavioral_question: 行为面试/STAR
- technical_explain: 技术解释
- scenario_reasoning: 情景推理
- follow_up_elaborate: 补充细节
- general_reply: 通用回复

## 技能卡片列表 (供匹配)
` + skillList + `

## 输出 JSON 格式
{
  "intent": "self_intro | project_deep_dive | behavioral | technical | scenario | follow_up | general",
  "strategy": "对应的策略名称",
  "matchedSkillIds": ["匹配的技能卡片ID数组，最多2个"],
  "needsRag": true/false,
  "searchSources": ["如果needsRag为true，填写搜索来源"],
  "query": "归一化后的查询文本"
}

注意：matchedSkillIds 从技能卡片列表中选择最相关的。如果不匹配任何卡片则返回空数组。
needsRag 为 true 仅当问题需要知识库/题库支持时。`

	messages := []map[string]string{
		{"role": "system", "content": systemPrompt},
	}

	// Add last few history entries
	start := 0
	if len(history) > 6 {
		start = len(history) - 6
	}
	for i := start; i < len(history); i++ {
		messages = append(messages, history[i])
	}

	messages = append(messages, map[string]string{"role": "user", "content": "请分类这个问题：" + query})

	result, err := r.callLLM(messages)
	if err != nil {
		return r.defaultOutput(query)
	}

	cleaned := strings.TrimSpace(result)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```JSON")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var output RouterOutput
	if err := json.Unmarshal([]byte(cleaned), &output); err != nil {
		return r.defaultOutput(query)
	}

	// Normalize intent based on rules
	output = normalizeIntent(output, query, history)
	return output
}

func (r *Router) defaultOutput(query string) RouterOutput {
	return RouterOutput{
		Intent:          IntentGeneral,
		Strategy:        StrategyGeneral,
		MatchedSkillIDs: []string{},
		NeedsRAG:        false,
		SearchSources:   []string{},
		Query:           query,
	}
}

// normalizeIntent applies deterministic rules to correct intent classification
func normalizeIntent(output RouterOutput, query string, history []map[string]string) RouterOutput {
	// Check last assistant message
	var lastAssistantMsg map[string]string
	for i := len(history) - 1; i >= 0; i-- {
		if history[i]["role"] == "assistant" {
			lastAssistantMsg = history[i]
			break
		}
	}

	// Rule 1: If last message was assistant and current is short → follow_up
	if lastAssistantMsg != nil && len([]rune(query)) < 15 {
		output.Intent = IntentFollowUp
		output.Strategy = StrategyFollowUp
		return output
	}

	// Rule 2: Contains expansion keywords → follow_up
	if strings.Contains(query, "展开") || strings.Contains(query, "详细") ||
		strings.Contains(query, "具体") || strings.Contains(query, "再说说") ||
		strings.Contains(query, "more detail") || strings.Contains(query, "elaborate") {
		output.Intent = IntentFollowUp
		output.Strategy = StrategyFollowUp
		return output
	}

	// Rule 3: Contains hypothetical keywords → scenario
	if strings.Contains(query, "假如") || strings.Contains(query, "如果") ||
		strings.Contains(query, "假设") || strings.Contains(query, "what if") ||
		strings.Contains(query, "scenario") || strings.Contains(query, "suppose") {
		output.Intent = IntentScenario
		output.Strategy = StrategyScenario
		return output
	}

	return output
}
