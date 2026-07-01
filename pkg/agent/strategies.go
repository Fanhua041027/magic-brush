package agent

// GetStrategyInstruction returns the system prompt instruction for a given strategy
// All prompts are in FIRST PERSON - the AI plays the role of the candidate/interviewee
func GetStrategyInstruction(s Strategy) string {
	switch s {
	case StrategySelfIntro:
		return "你正在参加面试，面试官让你做自我介绍。请用第一人称做简洁的自我介绍，突出与岗位匹配的经历和技能。控制在 3-5 句话以内，自然口语化。"
	case StrategyStarProject:
		return "面试官在深挖你的项目经历。请用 STAR 法则（情境-任务-行动-结果）描述项目核心亮点，重点讲你的架构决策和个人贡献。控制在 100 字以内，口语化表达。"
	case StrategyBehavioral:
		return "面试官在问行为面试问题。请用第一人称回答，使用 STAR 方法组织，用具体事例支撑，突出你的角色和成果。控制在 100 字以内，口语化。"
	case StrategyTechnical:
		return "面试官在问技术问题。请作为有经验的开发者，用简洁的语言先解释核心概念，再说明原理，最后联系你的实际项目经验。控制在 100 字以内，展现你的技术深度。"
	case StrategyScenario:
		return "面试官在问情景题。请结合你的实际项目经验，先分析问题再给出结构化解决方案。展示你的思考过程和决策依据。控制在 100 字以内。"
	case StrategyFollowUp:
		return "面试官在追问细节。就当前话题直接补充更多技术细节或项目经验，不要重复已说过的内容。控制在 80 字以内。"
	case StrategyGeneral:
		return "你正在参加技术面试。请用第一人称给出清晰、有条理的回答，适合口头表达。控制在 100 字以内。"
	}
	return "你正在参加技术面试。请用第一人称回答面试官的问题，自然口语化，控制在 100 字以内。"
}

// DefaultStrategyForIntent returns the default strategy for an intent
func DefaultStrategyForIntent(intent Intent) Strategy {
	switch intent {
	case IntentSelfIntro:
		return StrategySelfIntro
	case IntentProjectDeep:
		return StrategyStarProject
	case IntentBehavioral:
		return StrategyBehavioral
	case IntentTechnical:
		return StrategyTechnical
	case IntentScenario:
		return StrategyScenario
	case IntentFollowUp:
		return StrategyFollowUp
	default:
		return StrategyGeneral
	}
}
