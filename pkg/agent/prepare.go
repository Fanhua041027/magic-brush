package agent

import (
	"strings"
)

// PrepareContext assembles the system prompt from strategy + profile + skills + knowledge
func PrepareContext(
	query string,
	router RouterOutput,
	passages []RetrievedPassage,
	skillCards []SkillCard,
	profile *ProfileData,
) PreparedContext {
	strategy := router.Strategy
	if strategy == "" {
		strategy = DefaultStrategyForIntent(router.Intent)
	}
	instruction := GetStrategyInstruction(strategy)

	var parts []string

	// 1. Base role - AI acts as the INTERVIEWEE/CANDIDATE
	parts = append(parts, "你正在参加一场技术面试。面试官提出了一个问题，请根据以下上下文，以第一人称「我」的口吻给出回答。\n")

	// 2. Strategy instruction
	if instruction != "" {
		parts = append(parts, "[回答策略]\n"+instruction+"\n")
	}

	// 3. Profile context
	if profile != nil {
		if profile.ResumeSummary != "" {
			parts = append(parts, "[简历摘要]\n"+profile.ResumeSummary+"\n")
		}
		if profile.JDSummary != "" {
			parts = append(parts, "[岗位描述摘要]\n"+profile.JDSummary+"\n")
		}
	}

	// 4. Matched skill cards
	if len(router.MatchedSkillIDs) > 0 && len(skillCards) > 0 {
		var matched []SkillCard
		for _, sid := range router.MatchedSkillIDs {
			for _, s := range skillCards {
				if s.ID == sid {
					matched = append(matched, s)
				}
			}
		}
		if len(matched) > 0 {
			var skillsText strings.Builder
			for _, s := range matched {
				skillsText.WriteString("项目：")
				skillsText.WriteString(s.ProjectName)
				skillsText.WriteString("\n技术栈：")
				skillsText.WriteString(strings.Join(s.TechStack, ", "))
				skillsText.WriteString("\n你的角色：")
				skillsText.WriteString(s.Role)
				skillsText.WriteString("\n核心亮点：")
				skillsText.WriteString(strings.Join(s.Highlights, "; "))
				skillsText.WriteString("\n挑战：")
				skillsText.WriteString(strings.Join(s.Challenges, "; "))
				skillsText.WriteString("\n总结：")
				skillsText.WriteString(s.Summary)
				skillsText.WriteString("\n\n")
			}
			parts = append(parts, "[你的项目经历]\n"+skillsText.String()+"\n")
		}
	}

	// 5. Retrieved passages
	ragUsed := router.NeedsRAG && len(passages) > 0
	if ragUsed {
		var passagesText strings.Builder
		for _, p := range passages {
			passagesText.WriteString("来源「")
			passagesText.WriteString(p.Source)
			passagesText.WriteString("」：")
			passagesText.WriteString(p.Content)
			passagesText.WriteString("\n")
		}
		parts = append(parts, "[知识库参考]\n"+passagesText.String()+"\n")
	}

	// 6. Output instructions
	lang := "zh-CN"
	if profile != nil && profile.Language != "" {
		lang = profile.Language
	}
	if lang == "zh-CN" {
		parts = append(parts, "请用中文回答。回答要口语化、适合照着念，控制在 100 字以内。不要使用 markdown 格式。")
	} else {
		parts = append(parts, "Answer in English. Keep it conversational, under 100 words. No markdown formatting.")
	}

	systemPrompt := strings.Join(parts, "\n\n")

	return PreparedContext{
		SystemPrompt: systemPrompt,
		Strategy:     strategy,
		Intent:       router.Intent,
		RAGUsed:      ragUsed,
		PassagesUsed: len(passages),
	}
}
