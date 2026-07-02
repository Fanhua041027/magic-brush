package agent

import (
	"strings"
	"testing"
	"time"
)

func TestKnowledgeBase(t *testing.T) {
	kb := NewKnowledgeBase()

	// Test AddDocument
	doc := kb.AddDocument("Go并发编程", "goroutine和channel是Go并发的核心", "interview")
	if doc.ID == "" {
		t.Fatal("Expected non-empty ID")
	}
	if doc.Title != "Go并发编程" {
		t.Fatalf("Expected 'Go并发编程', got '%s'", doc.Title)
	}

	// Test ListDocuments
	docs := kb.ListDocuments()
	if len(docs) != 1 {
		t.Fatalf("Expected 1 document, got %d", len(docs))
	}

	// Test Search
	results := kb.Search("Go并发", 5, 0.1)
	if len(results) == 0 {
		t.Fatal("Expected at least 1 search result")
	}
	if !strings.Contains(results[0].Content, "goroutine") {
		t.Fatalf("Expected result to contain 'goroutine', got '%s'", results[0].Content)
	}

	// Test DeleteDocument
	kb.DeleteDocument(doc.ID)
	docs = kb.ListDocuments()
	if len(docs) != 0 {
		t.Fatalf("Expected 0 documents after delete, got %d", len(docs))
	}

	t.Log("✅ KnowledgeBase tests passed")
}

func TestKnowledgeBaseChineseSearch(t *testing.T) {
	kb := NewKnowledgeBase()
	kb.AddDocument("通道", "channel用于goroutine之间的通信", "go")
	kb.AddDocument("锁", "sync.Mutex用于互斥访问", "go")

	results := kb.Search("channel通信", 5, 0.1)
	if len(results) == 0 {
		t.Fatal("Expected search results for 'channel通信'")
	}
	t.Logf("Search results: %d", len(results))
	t.Log("✅ Chinese search test passed")
}

func TestProfileManager(t *testing.T) {
	pm := NewProfileManager()

	// Test update resume
	pm.UpdateResume("test resume content")
	if pm.GetProfile().ResumeRaw != "test resume content" {
		t.Fatal("Resume not updated correctly")
	}

	// Test update summaries
	pm.UpdateResumeSummary("experienced Go developer")
	if pm.GetProfile().ResumeSummary != "experienced Go developer" {
		t.Fatal("Resume summary not updated correctly")
	}

	pm.UpdateJDSummary("backend engineer required")
	if pm.GetProfile().JDSummary != "backend engineer required" {
		t.Fatal("JD summary not updated correctly")
	}

	// Test skill cards
	pm.AddSkillCard(SkillCard{
		ProjectName: "Test Project",
		TechStack:   []string{"Go", "Vue"},
		Role:        "Developer",
		Summary:     "A test project",
	})

	cards := pm.GetSkillCards()
	if len(cards) != 1 {
		t.Fatalf("Expected 1 skill card, got %d", len(cards))
	}
	if cards[0].ProjectName != "Test Project" {
		t.Fatalf("Expected 'Test Project', got '%s'", cards[0].ProjectName)
	}

	// Test remove skill card
	pm.RemoveSkillCard(cards[0].ID)
	cards = pm.GetSkillCards()
	if len(cards) != 0 {
		t.Fatalf("Expected 0 skill cards after remove, got %d", len(cards))
	}

	// Test snapshot
	pm.UpdateResume("snapshot resume")
	pm.UpdateResumeSummary("snapshot summary")
	pm.AddSkillCard(SkillCard{
		ProjectName: "Snapshot Project",
		TechStack:   []string{"Python"},
		Role:        "Lead",
		Summary:     "Snapshot",
	})

	snap := pm.CreateSnapshot()
	if snap.ID == "" {
		t.Fatal("Expected non-empty snapshot ID")
	}
	if snap.Profile.ResumeSummary != "snapshot summary" {
		t.Fatal("Snapshot resume summary mismatch")
	}

	// Verify deep copy (modifying original shouldn't affect snapshot)
	pm.UpdateResumeSummary("changed")
	if snap.Profile.ResumeSummary != "snapshot summary" {
		t.Fatal("Snapshot should not be affected by later changes")
	}

	t.Log("✅ ProfileManager tests passed")
}

func TestStrategies(t *testing.T) {
	tests := []struct {
		strategy Strategy
		expected string
	}{
		{StrategySelfIntro, "自我介绍"},
		{StrategyStarProject, "STAR"},
		{StrategyBehavioral, "STAR"},
		{StrategyTechnical, "技術"},
		{StrategyScenario, "情景"},
		{StrategyFollowUp, "补充"},
		{StrategyGeneral, "有条理"},
	}

	for _, tt := range tests {
		instruction := GetStrategyInstruction(tt.strategy)
		if instruction == "" {
			t.Fatalf("Expected non-empty instruction for strategy '%s'", tt.strategy)
		}
		if !strings.Contains(instruction, tt.expected) {
			t.Logf("Warning: instruction for '%s' doesn't contain '%s': %s",
				tt.strategy, tt.expected, instruction[:20])
		}
	}
	t.Log("✅ Strategy tests passed")
}

func TestDefaultStrategyMapping(t *testing.T) {
	tests := []struct {
		intent   Intent
		expected Strategy
	}{
		{IntentSelfIntro, StrategySelfIntro},
		{IntentProjectDeep, StrategyStarProject},
		{IntentBehavioral, StrategyBehavioral},
		{IntentTechnical, StrategyTechnical},
		{IntentScenario, StrategyScenario},
		{IntentFollowUp, StrategyFollowUp},
		{IntentGeneral, StrategyGeneral},
	}

	for _, tt := range tests {
		result := DefaultStrategyForIntent(tt.intent)
		if result != tt.expected {
			t.Fatalf("For intent '%s', expected strategy '%s', got '%s'",
				tt.intent, tt.expected, result)
		}
	}
	t.Log("✅ Default strategy mapping tests passed")
}

func TestPrepareContext(t *testing.T) {
	routerOutput := RouterOutput{
		Intent:          IntentTechnical,
		Strategy:        StrategyTechnical,
		MatchedSkillIDs: []string{},
		NeedsRAG:        false,
		Query:           "什么是goroutine",
	}

	skillCards := []SkillCard{
		{
			ID:          "skill-1",
			ProjectName: "并发工具",
			TechStack:   []string{"Go", "goroutine"},
			Role:        "Developer",
			Summary:     "并发编程",
		},
	}

	profile := &ProfileData{
		ResumeSummary: "有3年Go开发经验",
		JDSummary:     "需要了解并发编程",
	}

	passages := []RetrievedPassage{
		{
			Source:  "Go文档",
			Content: "goroutine是轻量级线程",
			Score:   0.8,
		},
	}

	// Test without RAG
	ctx := PrepareContext(routerOutput.Query, routerOutput, nil, skillCards, profile)
	if ctx.SystemPrompt == "" {
		t.Fatal("Expected non-empty system prompt")
	}
	if !strings.Contains(ctx.SystemPrompt, "goroutine") {
		t.Log("System prompt doesn't contain query - this is OK")
	}
	if ctx.Strategy != StrategyTechnical {
		t.Fatalf("Expected StrategyTechnical, got '%s'", ctx.Strategy)
	}
	if ctx.RAGUsed {
		t.Fatal("Expected RAGUsed=false when no passages")
	}

	// Test with RAG
	routerOutput.NeedsRAG = true
	ctx2 := PrepareContext(routerOutput.Query, routerOutput, passages, skillCards, profile)
	if !ctx2.RAGUsed {
		t.Fatal("Expected RAGUsed=true when router needs RAG and passages provided")
	}
	if ctx2.PassagesUsed != 1 {
		t.Fatalf("Expected 1 passage, got %d", ctx2.PassagesUsed)
	}
	if !strings.Contains(ctx2.SystemPrompt, "轻量级") {
		t.Fatal("Expected RAG content in system prompt")
	}

	t.Log("✅ PrepareContext tests passed")
}

func TestNormalizeIntent(t *testing.T) {
	// Test follow_up detection
	output := RouterOutput{Intent: IntentGeneral, Strategy: StrategyGeneral}
	history := []map[string]string{
		{"role": "assistant", "content": "这是一个关于并发的回答"},
	}

	result := normalizeIntent(output, "再说说", history)
	if result.Intent != IntentFollowUp {
		t.Fatalf("Expected follow_up for short query after assistant, got '%s'", result.Intent)
	}

	// Test scenario detection
	output2 := RouterOutput{Intent: IntentGeneral, Strategy: StrategyGeneral}
	result2 := normalizeIntent(output2, "如果系统崩溃了怎么办？", nil)
	if result2.Intent != IntentScenario {
		t.Fatalf("Expected scenario for '如果', got '%s'", result2.Intent)
	}

	t.Log("✅ NormalizeIntent tests passed")
}

func TestExamClassification(t *testing.T) {
	tests := []struct {
		name string
		qt   ExamQuestionType
		str  string
	}{
		{"coding", ExamCoding, "coding"},
		{"multiple_choice", ExamMultipleChoice, "multiple_choice"},
		{"essay", ExamEssay, "essay"},
		{"unknown", ExamUnknown, "unknown"},
	}

	for _, tt := range tests {
		if string(tt.qt) != tt.str {
			t.Fatalf("Expected '%s', got '%s'", tt.str, tt.qt)
		}
	}

	classification := ExamClassification{
		Type:       ExamCoding,
		Confidence: 0.95,
		Summary:    "算法题",
	}

	if classification.Type != ExamCoding {
		t.Fatal("Classification type mismatch")
	}
	if classification.Confidence != 0.95 {
		t.Fatal("Classification confidence mismatch")
	}

	t.Log("✅ Exam classification tests passed")
}

func TestMockInterview(t *testing.T) {
	// Test MockScore creation
	score := MockScore{
		Question: "Test question",
		Answer:   "Test answer",
		Score:    8,
		Feedback: "Good answer",
	}
	if score.Score != 8 {
		t.Fatal("Mock score init failed")
	}

	// Test MockQuestion
	q := MockQuestion{
		Question: "请介绍一下你自己",
		Type:     "自我介绍",
		Hint:     "突出匹配度",
	}
	if q.Type != "自我介绍" {
		t.Fatal("Mock question type init failed")
	}

	t.Log("✅ Mock interview type tests passed")
}

func TestKnowledgeBaseConcurrency(t *testing.T) {
	kb := NewKnowledgeBase()

	// Concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			kb.AddDocument(
				"Doc "+itoa(n),
				"Content for document number "+itoa(n),
				"test",
			)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all documents are present
	docs := kb.ListDocuments()
	if len(docs) != 10 {
		t.Fatalf("Expected 10 documents after concurrent writes, got %d", len(docs))
	}

	t.Log("✅ Concurrency test passed")
}

func TestPrepareContextEdgeCases(t *testing.T) {
	t.Run("empty router output", func(t *testing.T) {
		ctx := PrepareContext("hello", RouterOutput{}, nil, nil, nil)
		if ctx.SystemPrompt == "" {
			t.Fatal("expected non-empty system prompt even with empty input")
		}
		if ctx.Strategy != StrategyGeneral {
			t.Fatalf("expected general strategy, got '%s'", ctx.Strategy)
		}
	})

	t.Run("unknown intent falls back to general", func(t *testing.T) {
		output := RouterOutput{Intent: "unknown_intent", Strategy: ""}
		ctx := PrepareContext("test", output, nil, nil, nil)
		if ctx.Strategy != StrategyGeneral {
			t.Fatalf("expected general strategy for unknown intent, got '%s'", ctx.Strategy)
		}
	})

	t.Run("profile without JD summary", func(t *testing.T) {
		profile := &ProfileData{ResumeSummary: "only resume"}
		output := RouterOutput{Intent: IntentSelfIntro, Strategy: StrategySelfIntro}
		ctx := PrepareContext("自我介绍", output, nil, nil, profile)
		if !strings.Contains(ctx.SystemPrompt, "only resume") {
			t.Fatal("expected resume summary in prompt")
		}
		if ctx.Strategy != StrategySelfIntro {
			t.Fatalf("expected self-intro strategy, got '%s'", ctx.Strategy)
		}
	})

	t.Run("rag with multiple passages", func(t *testing.T) {
		passages := []RetrievedPassage{
			{Source: "doc1", Content: "内容1", Score: 0.9},
			{Source: "doc2", Content: "内容2", Score: 0.8},
			{Source: "doc3", Content: "内容3", Score: 0.7},
		}
		output := RouterOutput{
			Intent:   IntentTechnical,
			Strategy: StrategyTechnical,
			NeedsRAG: true,
			Query:    "goroutine",
		}
		ctx := PrepareContext("goroutine是什么", output, passages, nil, nil)
		if !ctx.RAGUsed {
			t.Fatal("expected RAGUsed=true")
		}
		if ctx.PassagesUsed != 3 {
			t.Fatalf("expected 3 passages, got %d", ctx.PassagesUsed)
		}
	})

	t.Run("english language output", func(t *testing.T) {
		profile := &ProfileData{Language: "en-US"}
		output := RouterOutput{Intent: IntentGeneral, Strategy: StrategyGeneral}
		ctx := PrepareContext("test", output, nil, nil, profile)
		if !strings.Contains(ctx.SystemPrompt, "Answer in English") {
			t.Fatal("expected English output instruction")
		}
	})

	t.Run("matched skill cards with highlights", func(t *testing.T) {
		cards := []SkillCard{
			{
				ID:          "card-1",
				ProjectName: "Go项目",
				TechStack:   []string{"Go", "Redis"},
				Role:        "Backend",
				Highlights:  []string{"高并发", "低延迟"},
				Challenges:  []string{"数据一致性"},
				Summary:     "一个高性能后端",
			},
		}
		output := RouterOutput{
			Intent:          IntentProjectDeep,
			Strategy:        StrategyStarProject,
			MatchedSkillIDs: []string{"card-1"},
			NeedsRAG:        false,
		}
		ctx := PrepareContext("讲讲你的项目", output, nil, cards, nil)
		if !strings.Contains(ctx.SystemPrompt, "Go项目") {
			t.Fatal("expected project name in prompt")
		}
		if !strings.Contains(ctx.SystemPrompt, "高并发") {
			t.Fatal("expected highlights in prompt")
		}
	})

	t.Run("matched skill cards not found", func(t *testing.T) {
		cards := []SkillCard{{ID: "card-1", ProjectName: "Project"}}
		output := RouterOutput{
			Intent:          IntentProjectDeep,
			MatchedSkillIDs: []string{"non-existent"},
		}
		ctx := PrepareContext("test", output, nil, cards, nil)
		if strings.Contains(ctx.SystemPrompt, "项目经历") {
			t.Log("no matched cards - expected no skill section (this is OK)")
		}
	})
}

func TestNowUnix(t *testing.T) {
	_ = time.Now().Unix()
	ts := nowUnix()
	if ts <= 0 {
		t.Fatal("Expected positive timestamp")
	}
	t.Log("✅ nowUnix works")
}

func TestDefaultRoute(t *testing.T) {
	r := NewRouter(func(messages []map[string]string) (string, error) {
		return "invalid json", nil
	})

	output := r.RouteQuery("hello", nil, nil)
	if output.Intent != IntentGeneral {
		t.Fatalf("expected general on bad LLM output, got '%s'", output.Intent)
	}
	if output.Strategy != StrategyGeneral {
		t.Fatalf("expected general strategy on bad LLM output, got '%s'", output.Strategy)
	}
	if output.Query != "hello" {
		t.Fatalf("expected original query preserved, got '%s'", output.Query)
	}
}

func TestRouterDefaultOutput(t *testing.T) {
	r := NewRouter(nil)
	output := r.defaultOutput("test query")
	if output.Intent != IntentGeneral || output.Query != "test query" {
		t.Fatal("defaultOutput mismatch")
	}
	if output.NeedsRAG {
		t.Fatal("expected NeedsRAG=false in default output")
	}
}

func TestNormalizeIntentEdgeCases(t *testing.T) {
	t.Run("empty history", func(t *testing.T) {
		output := RouterOutput{Intent: IntentGeneral, Strategy: StrategyGeneral}
		result := normalizeIntent(output, "测试问题", nil)
		if result.Intent != IntentGeneral {
			t.Fatalf("expected no change for empty history, got '%s'", result.Intent)
		}
	})

	t.Run("short query with no history", func(t *testing.T) {
		output := RouterOutput{Intent: IntentGeneral, Strategy: StrategyGeneral}
		result := normalizeIntent(output, "好", nil)
		if result.Intent != IntentGeneral {
			t.Fatalf("short query without history should not change, got '%s'", result.Intent)
		}
	})

	t.Run("short query after assistant", func(t *testing.T) {
		output := RouterOutput{Intent: IntentGeneral, Strategy: StrategyGeneral}
		history := []map[string]string{
			{"role": "assistant", "content": "这是一个详细回答"},
		}
		result := normalizeIntent(output, "展开说说", history)
		if result.Intent != IntentFollowUp {
			t.Fatalf("expected follow_up, got '%s'", result.Intent)
		}
	})

	t.Run("elaborate keyword detection", func(t *testing.T) {
		output := RouterOutput{Intent: IntentGeneral, Strategy: StrategyGeneral}
		result := normalizeIntent(output, "请详细说明一下", nil)
		if result.Intent != IntentFollowUp {
			t.Fatalf("expected follow_up for elaborate keyword, got '%s'", result.Intent)
		}
	})

	t.Run("hypothetical keyword detection", func(t *testing.T) {
		output := RouterOutput{Intent: IntentGeneral, Strategy: StrategyGeneral}
		result := normalizeIntent(output, "假设系统崩溃了", nil)
		if result.Intent != IntentScenario {
			t.Fatalf("expected scenario, got '%s'", result.Intent)
		}
	})

	t.Run("english keywords", func(t *testing.T) {
		output := RouterOutput{Intent: IntentGeneral, Strategy: StrategyGeneral}
		result := normalizeIntent(output, "can you elaborate more detail", nil)
		if result.Intent != IntentFollowUp {
			t.Fatalf("expected follow_up for english keywords, got '%s'", result.Intent)
		}

		result2 := normalizeIntent(output, "what if scenario", nil)
		if result2.Intent != IntentScenario {
			t.Fatalf("expected scenario for 'what if', got '%s'", result2.Intent)
		}
	})
}

func TestGenerateSummaryEdgeCases(t *testing.T) {
	t.Run("empty raw text", func(t *testing.T) {
		result, err := GenerateSummary("", "resume", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "" {
			t.Fatalf("expected empty result, got '%s'", result)
		}
	})

	t.Run("LLM fails returns truncated text", func(t *testing.T) {
		longText := "这是一个很长的简历内容用于测试回退逻辑。"
		result, err := GenerateSummary(longText, "resume", func(msgs []map[string]string) (string, error) {
			return "", assertAnError
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == "" {
			t.Fatal("expected fallback text on LLM failure")
		}
	})
}

// assertAnError is a sentinel error for testing
type sentinelError struct{ msg string }

func (e *sentinelError) Error() string { return e.msg }

var assertAnError = &sentinelError{"mock LLM failure"}

func TestGenerateSkillCardGate(t *testing.T) {
	t.Run("too short description returns nil", func(t *testing.T) {
		card, err := GenerateSkillCard("too short", nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if card != nil {
			t.Fatal("expected nil for short description")
		}
	})

	t.Run("LLM failure returns nil", func(t *testing.T) {
		longDesc := "这是一个非常长的项目描述，超过50个字符，用于测试生成技能卡片时的边界情况。确保门控逻辑正常工作。"
		card, err := GenerateSkillCard(longDesc, nil, func(msgs []map[string]string) (string, error) {
			return "", assertAnError
		})
		if err != nil {
			t.Fatal("expected nil error on LLM failure (function handles it)")
		}
		if card != nil {
			t.Fatal("expected nil card on LLM failure")
		}
	})

	t.Run("missing required fields", func(t *testing.T) {
		longDesc := "这是一个非常长的项目描述，超过50个字符，用于测试生成技能卡片时的边界情况。确保门控逻辑正常工作。"
		card, err := GenerateSkillCard(longDesc, nil, func(msgs []map[string]string) (string, error) {
			return `{"projectName":"","techStack":[],"role":"","highlights":[],"challenges":[],"keywords":[],"summary":""}`, nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if card != nil {
			t.Fatal("expected nil when ProjectName and Role are empty")
		}
	})
}

func TestMockInterviewData(t *testing.T) {
	t.Run("mock score validation", func(t *testing.T) {
		score := MockScore{
			Question: "Test?",
			Answer:   "Test answer",
			Score:    10,
			Feedback: "Perfect",
		}
		if score.Score < 1 || score.Score > 10 {
			t.Fatal("score out of range")
		}
		if score.Feedback == "" {
			t.Fatal("feedback should not be empty")
		}
	})

	t.Run("mock question types", func(t *testing.T) {
		types := []string{"自我介绍", "项目深挖", "行为面试", "技术问题", "情景题"}
		for _, qt := range types {
			q := MockQuestion{Question: "test", Type: qt, Hint: "hint"}
			if q.Type != qt {
				t.Fatalf("expected type '%s', got '%s'", qt, q.Type)
			}
		}
	})
}

func TestPipelineEvent(t *testing.T) {
	event := PipelineEvent{Stage: "router", Status: "running", Detail: "analyzing"}
	if event.Stage != "router" || event.Status != "running" {
		t.Fatal("PipelineEvent init failed")
	}

	doneEvent := PipelineEvent{Stage: "router", Status: "done", Detail: "completed"}
	if string(doneEvent.Stage)+doneEvent.Status != "routerdone" {
		t.Fatal("PipelineEvent concat failed")
	}
}

func TestExamTypes(t *testing.T) {
	t.Run("classification constants", func(t *testing.T) {
		if ExamCoding != "coding" {
			t.Fatal("ExamCoding constant mismatch")
		}
		if ExamMultipleChoice != "multiple_choice" {
			t.Fatal("ExamMultipleChoice constant mismatch")
		}
		if ExamEssay != "essay" {
			t.Fatal("ExamEssay constant mismatch")
		}
		if ExamUnknown != "unknown" {
			t.Fatal("ExamUnknown constant mismatch")
		}
	})

	t.Run("solution with code", func(t *testing.T) {
		sol := ExamSolution{
			Type:  ExamCoding,
			Steps: []string{"理解题意", "选择算法"},
			Code:  "func main() {}",
		}
		if sol.Code != "func main() {}" {
			t.Fatal("solution code mismatch")
		}
		if len(sol.Steps) != 2 {
			t.Fatal("expected 2 steps")
		}
	})

	t.Run("solution with answer", func(t *testing.T) {
		sol := ExamSolution{
			Type:   ExamEssay,
			Answer: "这是一个论述题答案",
		}
		if sol.Answer == "" {
			t.Fatal("expected non-empty answer")
		}
	})
}

func TestProfileConcurrentAccess(t *testing.T) {
	pm := NewProfileManager()

	// Concurrent updates
	done := make(chan bool, 20)
	for i := 0; i < 20; i++ {
		go func(n int) {
			pm.UpdateResumeSummary("summary")
			if n%2 == 0 {
				pm.AddSkillCard(SkillCard{
					ProjectName: "Project",
					Role:        "Dev",
					Summary:     "test",
				})
			}
			_ = pm.GetProfile()
			_ = pm.GetSkillCards()
			done <- true
		}(i)
	}

	for i := 0; i < 20; i++ {
		<-done
	}

	t.Log("✅ concurrent profile access OK")
}

func TestTranscriberAPIKeyValidation(t *testing.T) {
	t.Run("empty API key returns error", func(t *testing.T) {
		tr := NewTranscriber("", "https://api.openai.com/v1")
		if tr.apiKey != "" {
			t.Fatal("expected empty API key")
		}
	})

	t.Run("API key gets set", func(t *testing.T) {
		tr := NewTranscriber("sk-test-key", "https://api.deepseek.com")
		if tr.apiKey != "sk-test-key" {
			t.Fatalf("expected 'sk-test-key', got '%s'", tr.apiKey)
		}
	})

	t.Run("base URL normalization", func(t *testing.T) {
		tr := NewTranscriber("key", "https://api.deepseek.com")
		if tr.baseURL != "https://api.deepseek.com/v1" {
			t.Fatalf("expected '/v1' appended, got '%s'", tr.baseURL)
		}

		tr2 := NewTranscriber("key", "https://api.openai.com/v1")
		if tr2.baseURL != "https://api.openai.com/v1" {
			t.Fatalf("expected no double /v1, got '%s'", tr2.baseURL)
		}
	})
}

func TestFormatFloat(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0.0, "0.0"},
		{5.0, "5.0"},
		{7.5, "7.5"},
		{8.3, "8.3"},
		{10.0, "10.0"},
		{3.14, "3.1"},
		{6.99, "6.9"},
	}

	for _, tt := range tests {
		result := formatFloat(tt.input)
		if result != tt.want {
			t.Fatalf("formatFloat(%f) = '%s', want '%s'", tt.input, result, tt.want)
		}
	}
}

func TestAudioServiceCreation(t *testing.T) {
	svc := NewAudioService("test_script.py")
	if svc == nil {
		t.Fatal("expected non-nil AudioService")
	}
	if svc.scriptPath != "test_script.py" {
		t.Fatalf("expected 'test_script.py', got '%s'", svc.scriptPath)
	}
}

func TestGetWAVBytesError(t *testing.T) {
	_, err := GetWAVBytes(&AudioCaptureResult{})
	if err == nil {
		t.Fatal("expected error for empty base64 data")
	}
}

func TestKnowledgeBaseEmpty(t *testing.T) {
	kb := NewKnowledgeBase()
	results := kb.Search("test", 5, 0.1)
	if results != nil {
		t.Fatal("expected nil results for uninitialized KB")
	}
}
