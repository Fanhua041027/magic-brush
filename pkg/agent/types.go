// Package agent implements an AskCc-inspired multi-agent system:
// Profile Agent (resume/JD → skill cards → snapshot)
// Interview Agent (router || retrieve → prepare → agent)
// Exam Agent (classify → solve)
// Mock Interview (generate → answer → score)
package agent

// ============================================================
// Shared Types
// ============================================================

// Intent represents the classified intent of a user query
type Intent string

const (
	IntentSelfIntro     Intent = "self_intro"
	IntentProjectDeep   Intent = "project_deep_dive"
	IntentBehavioral    Intent = "behavioral"
	IntentTechnical     Intent = "technical"
	IntentScenario      Intent = "scenario"
	IntentFollowUp      Intent = "follow_up"
	IntentGeneral       Intent = "general"
)

// Strategy maps to a system prompt instruction for answering
type Strategy string

const (
	StrategySelfIntro       Strategy = "self_introduction"
	StrategyStarProject     Strategy = "star_project"
	StrategyBehavioral      Strategy = "behavioral_question"
	StrategyTechnical       Strategy = "technical_explain"
	StrategyScenario        Strategy = "scenario_reasoning"
	StrategyFollowUp        Strategy = "follow_up_elaborate"
	StrategyGeneral         Strategy = "general_reply"
)

// ============================================================
// Router Types
// ============================================================

// RouterOutput is the structured output from a single Router LLM call
type RouterOutput struct {
	Intent          Intent   `json:"intent"`
	Strategy        Strategy `json:"strategy"`
	MatchedSkillIDs []string `json:"matchedSkillIds"`
	NeedsRAG        bool     `json:"needsRag"`
	SearchSources   []string `json:"searchSources"`
	Query           string   `json:"query"`
}

// ============================================================
// Profile Types
// ============================================================

// SkillCard represents a structured project skill card
type SkillCard struct {
	ID          string   `json:"id"`
	ProjectName string   `json:"projectName"`
	TechStack   []string `json:"techStack"`
	Role        string   `json:"role"`
	Highlights  []string `json:"highlights"`
	Challenges  []string `json:"challenges"`
	Keywords    []string `json:"keywords"`
	Summary     string   `json:"summary"`
}

// ProfileData holds resume/JD information and skill cards
type ProfileData struct {
	ResumeRaw     string      `json:"resumeRaw"`
	JDSummary     string      `json:"jdSummary"`
	ResumeSummary string      `json:"resumeSummary"`
	SkillCards    []SkillCard `json:"skillCards"`
	Language      string      `json:"language"` // "zh-CN" or "en-US"
}

// ProfileSnapshot is an immutable snapshot created when interview starts
type ProfileSnapshot struct {
	ID        string      `json:"id"`
	CreatedAt int64       `json:"createdAt"`
	Profile   ProfileData `json:"profile"`
}

// ============================================================
// Retrieve Types
// ============================================================

// RetrievedPassage is a search result from the knowledge base
type RetrievedPassage struct {
	Source  string  `json:"source"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

// KBDocument is a knowledge base document
type KBDocument struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Source    string `json:"source"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

// ============================================================
// Prepare Types
// ============================================================

// PreparedContext is the assembled system prompt + message context
type PreparedContext struct {
	SystemPrompt string         `json:"systemPrompt"`
	Strategy     Strategy       `json:"strategy"`
	Intent       Intent         `json:"intent"`
	RAGUsed      bool           `json:"ragUsed"`
	PassagesUsed int            `json:"passagesUsed"`
}

// ============================================================
// Exam Types
// ============================================================

// ExamQuestionType represents the type of exam question
type ExamQuestionType string

const (
	ExamCoding        ExamQuestionType = "coding"
	ExamMultipleChoice ExamQuestionType = "multiple_choice"
	ExamEssay         ExamQuestionType = "essay"
	ExamUnknown       ExamQuestionType = "unknown"
)

// ExamClassification is the output of the classification stage
type ExamClassification struct {
	Type       ExamQuestionType `json:"type"`
	Confidence float64          `json:"confidence"`
	Summary    string           `json:"summary"`
}

// ExamSolution is the final solution from the exam agent
type ExamSolution struct {
	Type    ExamQuestionType `json:"type"`
	Steps   []string         `json:"steps"`
	Code    string           `json:"code,omitempty"`
	Answer  string           `json:"answer,omitempty"`
}

// ============================================================
// Mock Interview Types
// ============================================================

// MockQuestion is a generated interview question
type MockQuestion struct {
	Question  string `json:"question"`
	Type      string `json:"type"`
	Hint      string `json:"hint"`
}

// MockScore is the scoring result for a mock answer
type MockScore struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Score    int    `json:"score"`
	Feedback string `json:"feedback"`
}

// ============================================================
// Callbacks for streaming events
// ============================================================

// PipelineEvent represents a pipeline stage event
type PipelineEvent struct {
	Stage  string `json:"stage"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

// AgentCallbacks are the callbacks for agent pipeline events
type AgentCallbacks struct {
	OnEvent    func(event PipelineEvent)
	OnStream   func(chunk string)
	OnStreamEnd func()
	OnStreamError func(err string)
}
