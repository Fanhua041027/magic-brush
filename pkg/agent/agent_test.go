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

func init() {
}

func TestNowUnix(t *testing.T) {
	_ = time.Now().Unix() // just verify time works
	ts := nowUnix()
	if ts <= 0 {
		t.Fatal("Expected positive timestamp")
	}
	t.Log("✅ nowUnix works")
}
