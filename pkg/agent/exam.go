package agent

import (
	"context"
	"encoding/json"
	"strings"

	"ai-assistant/pkg/llm"
)

// ExamAgent handles screenshot-based exam question solving
type ExamAgent struct {
	llmProvider llm.Provider
}

// NewExamAgent creates a new exam agent
func NewExamAgent(provider llm.Provider) *ExamAgent {
	return &ExamAgent{llmProvider: provider}
}

// SetProvider updates the LLM provider
func (ea *ExamAgent) SetProvider(provider llm.Provider) {
	ea.llmProvider = provider
}

// ExamResult is the complete result from the exam pipeline
type ExamResult struct {
	Classification ExamClassification `json:"classification"`
	Solution       ExamSolution       `json:"solution"`
	OcrText        string             `json:"ocrText"`
}

// ExecuteExamPipeline runs the full exam pipeline:
// 1. OCR text extraction (from screenshot)
// 2. LLM classification
// 3. LLM solution generation
func (ea *ExamAgent) ExecuteExamPipeline(
	ctx context.Context,
	screenshotBase64 string,
	cb AgentCallbacks,
) (*ExamResult, error) {
	// Step 1: OCR text extraction
	if cb.OnEvent != nil {
		cb.OnEvent(PipelineEvent{Stage: "ocr", Status: "running", Detail: "正在识别截图文字..."})
	}

	// Use the LLM to extract text from the screenshot
	ocrText, err := ea.extractTextFromImage(ctx, screenshotBase64)
	if err != nil {
		if cb.OnEvent != nil {
			cb.OnEvent(PipelineEvent{Stage: "ocr", Status: "error", Detail: err.Error()})
		}
		ocrText = ""
	}

	if ocrText == "" {
		if cb.OnEvent != nil {
			cb.OnEvent(PipelineEvent{Stage: "ocr", Status: "error", Detail: "OCR 识别失败"})
		}
		return &ExamResult{
			Classification: ExamClassification{Type: ExamUnknown, Confidence: 0, Summary: "OCR 识别失败"},
			Solution: ExamSolution{
				Type:  ExamUnknown,
				Steps: []string{"❌ 无法识别截图内容。\n\n建议：确保截图包含清晰可读的题目文字后重试。"},
			},
			OcrText: "",
		}, nil
	}

	if cb.OnEvent != nil {
		cb.OnEvent(PipelineEvent{
			Stage:  "ocr",
			Status: "done",
			Detail: "OCR 提取到 " + itoa(len([]rune(ocrText))) + " 字符",
		})
	}

	// Step 2: Classification
	if cb.OnEvent != nil {
		cb.OnEvent(PipelineEvent{Stage: "classify", Status: "running", Detail: "正在分析题目类型..."})
	}

	classification, err := ea.classifyQuestion(ctx, ocrText)
	if err != nil {
		classification = ExamClassification{Type: ExamUnknown, Confidence: 0, Summary: "无法识别"}
	}

	if cb.OnEvent != nil {
		cb.OnEvent(PipelineEvent{
			Stage:  "classify",
			Status: "done",
			Detail: "题型: " + string(classification.Type),
		})
	}

	// Step 3: Solve
	if cb.OnEvent != nil {
		cb.OnEvent(PipelineEvent{Stage: "solve", Status: "running", Detail: "正在生成解答..."})
	}

	solution, err := ea.solveQuestion(ctx, ocrText, classification)
	if err != nil {
		solution = ExamSolution{
			Type:  classification.Type,
			Steps: []string{"❌ 生成解答失败，请重试。"},
		}
	}

	if cb.OnEvent != nil {
		cb.OnEvent(PipelineEvent{Stage: "solve", Status: "done", Detail: "解答完成"})
	}

	return &ExamResult{
		Classification: classification,
		Solution:       solution,
		OcrText:        ocrText,
	}, nil
}

func (ea *ExamAgent) extractTextFromImage(ctx context.Context, base64Data string) (string, error) {
	messages := []llm.Message{
		{
			Role: llm.RoleUser,
			Parts: []llm.ContentPart{
				llm.TextPart("请识别这张截图中的所有文字内容，直接输出识别结果，不要添加任何其他文字。"),
				llm.ImagePart(base64Data),
			},
		},
	}

	result, err := ea.llmProvider.GenerateContent(ctx, "", messages)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(result.Content), nil
}

func (ea *ExamAgent) classifyQuestion(ctx context.Context, text string) (ExamClassification, error) {
	systemPrompt := `你是一个笔试题目分类器。根据以下OCR提取的题目文字，判断题型。

题型：
- coding: 代码题（算法、数据结构、实现类）
- multiple_choice: 选择题（单选或多选，包括选项ABCD）
- essay: 简答/论述题（需要文字作答的开放式题目）
- unknown: 无法识别

输出 JSON 格式：
{
  "type": "coding | multiple_choice | essay | unknown",
  "confidence": 0.0-1.0,
  "summary": "题目简要描述，15字以内"
}`

	messages := []llm.Message{
		llm.NewSystemMessage(systemPrompt),
		llm.NewUserMessage("这是从截图OCR提取的题目文字：\n\n" + text + "\n\n请分析这是什么题型。"),
	}

	result, err := ea.llmProvider.GenerateContent(ctx, "", messages)
	if err != nil {
		return ExamClassification{Type: ExamUnknown, Confidence: 0, Summary: "无法识别"}, err
	}

	cleaned := strings.TrimSpace(result.Content)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```JSON")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var classification ExamClassification
	if err := json.Unmarshal([]byte(cleaned), &classification); err != nil {
		return ExamClassification{Type: ExamUnknown, Confidence: 0, Summary: "无法识别"}, err
	}

	return classification, nil
}

func (ea *ExamAgent) solveQuestion(ctx context.Context, text string, classification ExamClassification) (ExamSolution, error) {
	var solvingPrompt string
	switch classification.Type {
	case ExamCoding:
		solvingPrompt = `你是一个算法面试教练。请分析以下题目并提供：
1. 题目分析（理解题意）
2. 方法选择（选用哪种算法/数据结构及原因）
3. 带注释的代码实现
4. 时间和空间复杂度分析
5. 边界情况考量

用中文回答，代码用 markdown 代码块标注语言。`
	case ExamMultipleChoice:
		solvingPrompt = `你是一个笔试辅导老师。请分析题目并提供：
1. 逐项分析每个选项
2. 排除推理过程
3. 标明正确答案
4. 解释错误选项的原因

用中文回答。`
	case ExamEssay:
		solvingPrompt = `你是一个面试辅导老师。请分析题目并提供：
1. 结构化回答提纲
2. 核心论点
3. 支撑论据
4. 建议的段落结构

用中文回答。`
	default:
		solvingPrompt = `请分析以下题目内容，提供解答思路和答案。用中文回答。`
	}

	messages := []llm.Message{
		llm.NewSystemMessage(solvingPrompt),
		llm.NewUserMessage("以下是题目文字内容：\n\n" + text + "\n\n请根据题型给出解答。"),
	}

	result, err := ea.llmProvider.GenerateContent(ctx, "", messages)
	if err != nil {
		return ExamSolution{Type: classification.Type, Steps: []string{"❌ 生成解答失败"}}, err
	}

	return ExamSolution{
		Type:  classification.Type,
		Steps: []string{result.Content},
	}, nil
}
