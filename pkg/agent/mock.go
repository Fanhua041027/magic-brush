package agent

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"ai-assistant/pkg/llm"
)

// MockInterview manages the mock interview flow
type MockInterview struct {
	mu           sync.Mutex
	llmProvider  llm.Provider
	profileMgr   *ProfileManager
	questions    []MockQuestion
	currentIdx   int
	scores       []MockScore
}

// NewMockInterview creates a new mock interview session
func NewMockInterview(provider llm.Provider, pm *ProfileManager) *MockInterview {
	return &MockInterview{
		llmProvider: provider,
		profileMgr:  pm,
	}
}

// SetProvider updates the LLM provider
func (mi *MockInterview) SetProvider(provider llm.Provider) {
	mi.llmProvider = provider
}

// Start begins a new mock interview, generating questions
func (mi *MockInterview) Start(ctx context.Context) ([]MockQuestion, error) {
	mi.mu.Lock()
	defer mi.mu.Unlock()

	mi.currentIdx = 0
	mi.scores = nil

	questions, err := mi.generateQuestions(ctx)
	if err != nil {
		// Use default questions as fallback
		questions = []MockQuestion{
			{Question: "请做一个简短的自我介绍。", Type: "自我介绍", Hint: "突出与岗位匹配的经历"},
			{Question: "详细讲讲你最有成就感的项目。", Type: "项目深挖", Hint: "用STAR法则"},
			{Question: "分享一次你遇到的重大挑战及如何解决的。", Type: "行为面试", Hint: "情境-行动-结果"},
			{Question: "解释一下你最有信心的技术领域中的一个核心概念。", Type: "技术问题", Hint: "由浅入深"},
			{Question: "如果团队内部出现意见分歧，你会怎么处理？", Type: "情景题", Hint: "结构化推理"},
		}
	}

	mi.questions = questions
	return questions, nil
}

// GetNextQuestion returns the next unanswered question
func (mi *MockInterview) GetNextQuestion() *MockQuestion {
	mi.mu.Lock()
	defer mi.mu.Unlock()
	if mi.currentIdx >= len(mi.questions) {
		return nil
	}
	q := mi.questions[mi.currentIdx]
	return &q
}

// SubmitAnswer submits an answer for scoring and moves to the next question
func (mi *MockInterview) SubmitAnswer(ctx context.Context, answer string) (*MockScore, *MockQuestion, bool, error) {
	mi.mu.Lock()

	if mi.currentIdx >= len(mi.questions) {
		mi.mu.Unlock()
		return nil, nil, true, nil
	}

	q := mi.questions[mi.currentIdx]
	mi.mu.Unlock()

	// Evaluate the answer
	score, err := mi.evaluateAnswer(ctx, q, answer)
	if err != nil {
		score = &MockScore{Score: 6, Feedback: "评估失败"}
	}
	score.Question = q.Question
	score.Answer = answer

	mi.mu.Lock()
	mi.scores = append(mi.scores, *score)
	mi.currentIdx++
	finished := mi.currentIdx >= len(mi.questions)

	var nextQ *MockQuestion
	if !finished {
		nq := mi.questions[mi.currentIdx]
		nextQ = &nq
	}
	mi.mu.Unlock()

	return score, nextQ, finished, nil
}

// GetSummary returns a summary of the mock interview session
func (mi *MockInterview) GetSummary() struct {
	Summary string      `json:"summary"`
	Scores  []MockScore `json:"scores"`
	Average float64     `json:"average"`
} {
	mi.mu.Lock()
	defer mi.mu.Unlock()

	if len(mi.scores) == 0 {
		return struct {
			Summary string      `json:"summary"`
			Scores  []MockScore `json:"scores"`
			Average float64     `json:"average"`
		}{
			Summary: "暂无数据",
			Scores:  []MockScore{},
			Average: 0,
		}
	}

	var total int
	var parts []string
	for i, s := range mi.scores {
		total += s.Score
		parts = append(parts, itoa(i+1)+". "+s.Question+"\n得分："+itoa(s.Score)+"/10\n反馈："+s.Feedback+"\n")
	}
	avg := float64(total) / float64(len(mi.scores))

	summary := "## 📋 模拟面试总结\n\n总题数：" + itoa(len(mi.scores)) + "\n平均得分：" + formatFloat(avg) + "/10\n\n各题详情：\n" + strings.Join(parts, "\n")

	scoresCopy := make([]MockScore, len(mi.scores))
	copy(scoresCopy, mi.scores)

	return struct {
		Summary string      `json:"summary"`
		Scores  []MockScore `json:"scores"`
		Average float64     `json:"average"`
	}{
		Summary: summary,
		Scores:  scoresCopy,
		Average: avg,
	}
}

// GetProgress returns current progress
func (mi *MockInterview) GetProgress() (current, total int) {
	mi.mu.Lock()
	defer mi.mu.Unlock()
	return mi.currentIdx, len(mi.questions)
}

func (mi *MockInterview) generateQuestions(ctx context.Context) ([]MockQuestion, error) {
	profile := mi.profileMgr.GetProfile()

	skillText := "未提供"
	if len(profile.SkillCards) > 0 {
		var techs []string
		for _, s := range profile.SkillCards {
			techs = append(techs, s.TechStack...)
		}
		skillText = strings.Join(techs, ", ")
	}

	jSummary := profile.JDSummary
	if jSummary == "" {
		jSummary = "未提供"
	}

	prompt := `你是一个面试官。根据以下信息生成5个面试问题（覆盖不同类别），用JSON数组返回。

岗位要求：` + jSummary + `
技能：` + skillText + `

输出 JSON 格式（严格数组）：
[
  {
    "question": "问题内容",
    "type": "自我介绍|项目深挖|行为面试|技术问题|情景题",
    "hint": "简要提示如何回答"
  }
]

生成5个问题，覆盖5种不同类型。`

	messages := []llm.Message{
		llm.NewUserMessage(prompt),
	}

	result, err := mi.llmProvider.GenerateContent(ctx, "", messages)
	if err != nil {
		return nil, err
	}

	cleaned := strings.TrimSpace(result.Content)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```JSON")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var questions []MockQuestion
	if err := json.Unmarshal([]byte(cleaned), &questions); err != nil {
		return nil, err
	}

	if len(questions) == 0 {
		return nil, nil
	}

	return questions, nil
}

func (mi *MockInterview) evaluateAnswer(ctx context.Context, q MockQuestion, answer string) (*MockScore, error) {
	prompt := `你是一个面试评估专家。评估以下回答：

问题（` + q.Type + `）：` + q.Question + `
回答：` + answer + `

请从1-10分评分并提供简要反馈。输出JSON：
{"score": 分数, "feedback": "2-3句反馈意见"}`

	messages := []llm.Message{
		llm.NewUserMessage(prompt),
	}

	result, err := mi.llmProvider.GenerateContent(ctx, "", messages)
	if err != nil {
		return &MockScore{Score: 6, Feedback: "评估失败，人工判断。"}, nil
	}

	cleaned := strings.TrimSpace(result.Content)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```JSON")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var score struct {
		Score    int    `json:"score"`
		Feedback string `json:"feedback"`
	}
	if err := json.Unmarshal([]byte(cleaned), &score); err != nil {
		return &MockScore{Score: 6, Feedback: "评估失败，人工判断。"}, nil
	}

	return &MockScore{Score: score.Score, Feedback: score.Feedback}, nil
}

// formatFloat formats a float64 to 1 decimal place
func formatFloat(f float64) string {
	i := int(f * 10)
	whole := i / 10
	dec := i % 10
	if dec < 0 {
		dec = -dec
	}
	return itoa(whole) + "." + itoa(dec)
}
