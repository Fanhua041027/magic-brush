package agent

import (
	"encoding/json"
	"strings"
	"sync"
)

// ProfileManager manages resume/JD profiles, skill cards, and snapshots
type ProfileManager struct {
	mu        sync.RWMutex
	profile   ProfileData
	snapshot  *ProfileSnapshot
	cardIDSeq int
}

// NewProfileManager creates a new profile manager
func NewProfileManager() *ProfileManager {
	return &ProfileManager{}
}

// GetProfile returns the current profile
func (pm *ProfileManager) GetProfile() ProfileData {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.profile
}

// UpdateResume updates the raw resume text
func (pm *ProfileManager) UpdateResume(raw string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.profile.ResumeRaw = raw
}

// UpdateResumeSummary updates the resume summary
func (pm *ProfileManager) UpdateResumeSummary(summary string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.profile.ResumeSummary = summary
}

// UpdateJDSummary updates the job description summary
func (pm *ProfileManager) UpdateJDSummary(summary string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.profile.JDSummary = summary
}

// SetLanguage sets the profile language
func (pm *ProfileManager) SetLanguage(lang string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.profile.Language = lang
}

// GetSkillCards returns the current skill cards
func (pm *ProfileManager) GetSkillCards() []SkillCard {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	result := make([]SkillCard, len(pm.profile.SkillCards))
	copy(result, pm.profile.SkillCards)
	return result
}

// AddSkillCard adds a skill card
func (pm *ProfileManager) AddSkillCard(card SkillCard) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.cardIDSeq++
	card.ID = "skill-" + itoa(pm.cardIDSeq)
	pm.profile.SkillCards = append(pm.profile.SkillCards, card)
}

// RemoveSkillCard removes a skill card by ID
func (pm *ProfileManager) RemoveSkillCard(id string) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	for i, c := range pm.profile.SkillCards {
		if c.ID == id {
			pm.profile.SkillCards = append(pm.profile.SkillCards[:i], pm.profile.SkillCards[i+1:]...)
			return true
		}
	}
	return false
}

// ClearSkillCards removes all skill cards
func (pm *ProfileManager) ClearSkillCards() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.profile.SkillCards = nil
}

// CreateSnapshot creates a frozen snapshot of the current profile
func (pm *ProfileManager) CreateSnapshot() ProfileSnapshot {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	snapshot := ProfileSnapshot{
		ID: "snap-" + itoa(int(nowUnix())),
		Profile: ProfileData{
			ResumeRaw:     pm.profile.ResumeRaw,
			ResumeSummary: pm.profile.ResumeSummary,
			JDSummary:     pm.profile.JDSummary,
			Language:      pm.profile.Language,
		},
	}

	// Deep copy skill cards
	snapshot.Profile.SkillCards = make([]SkillCard, len(pm.profile.SkillCards))
	for i, c := range pm.profile.SkillCards {
		clone := c
		snapshot.Profile.SkillCards[i] = clone
	}

	pm.snapshot = &snapshot
	return snapshot
}

// GetSnapshot returns the current snapshot
func (pm *ProfileManager) GetSnapshot() *ProfileSnapshot {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	if pm.snapshot == nil {
		return nil
	}
	snap := *pm.snapshot
	return &snap
}

// GenerateSummary creates an AI summary from raw text
func GenerateSummary(rawText string, summaryType string, callLLM func(messages []map[string]string) (string, error)) (string, error) {
	if strings.TrimSpace(rawText) == "" {
		return "", nil
	}

	var prompt string
	if summaryType == "resume" {
		prompt = `请从以下简历中提取关键信息并生成 100 字以内的摘要：
- 教育背景
- 技术栈和技能
- 工作/项目经历重点

简历内容：
` + rawText + `

摘要（100 字以内）：`
	} else {
		prompt = `请从以下岗位描述中提取关键信息并生成 100 字以内的摘要：
- 岗位职责
- 技术要求
- 软性要求

岗位描述：
` + rawText + `

摘要（100 字以内）：`
	}

	messages := []map[string]string{
		{"role": "user", "content": prompt},
	}

	result, err := callLLM(messages)
	if err != nil {
		// Fallback: first 200 chars
		runes := []rune(rawText)
		if len(runes) > 200 {
			return string(runes[:200]), nil
		}
		return rawText, nil
	}

	return strings.TrimSpace(result), nil
}

// GenerateSkillCard creates a structured skill card from a project description
func GenerateSkillCard(projectDesc string, existingCards []SkillCard, callLLM func(messages []map[string]string) (string, error)) (*SkillCard, error) {
	// Gate 1: minimum description length
	if len([]rune(projectDesc)) < 50 {
		return nil, nil
	}

	prompt := `从以下项目描述中提取结构化的技能卡片信息。

项目描述：
` + projectDesc + `

请输出 JSON 格式：
{
  "projectName": "项目名称",
  "techStack": ["技术1", "技术2"],
  "role": "你的角色",
  "highlights": ["亮点1", "亮点2"],
  "challenges": ["挑战1", "挑战2"],
  "keywords": ["关键词1", "关键词2"],
  "summary": "项目总结（50字以内）"
}`

	messages := []map[string]string{
		{"role": "user", "content": prompt},
	}

	result, err := callLLM(messages)
	if err != nil {
		return nil, err
	}

	cleaned := strings.TrimSpace(result)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```JSON")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var cardData struct {
		ProjectName string   `json:"projectName"`
		TechStack   []string `json:"techStack"`
		Role        string   `json:"role"`
		Highlights  []string `json:"highlights"`
		Challenges  []string `json:"challenges"`
		Keywords    []string `json:"keywords"`
		Summary     string   `json:"summary"`
	}

	if err := json.Unmarshal([]byte(cleaned), &cardData); err != nil {
		return nil, err
	}

	// Gate 2: validate required fields
	if cardData.ProjectName == "" || cardData.Role == "" {
		return nil, nil
	}

	return &SkillCard{
		ProjectName: cardData.ProjectName,
		TechStack:   cardData.TechStack,
		Role:        cardData.Role,
		Highlights:  cardData.Highlights,
		Challenges:  cardData.Challenges,
		Keywords:    cardData.Keywords,
		Summary:     cardData.Summary,
	}, nil
}
