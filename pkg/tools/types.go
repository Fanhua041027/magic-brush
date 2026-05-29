package tools

import "context"

// ToolContext 工具执行上下文
type ToolContext struct {
	Ctx context.Context
}

// ToolResult 工具执行结果
type ToolResult struct {
	// 文本响应
	Text string
	// 图片数据 (可选)
	ImageData []byte
	// 图片 MIME 类型 (如 "image/jpeg")
	ImageMimeType string
	// 是否有图片
	HasImage bool
	// 错误信息
	Error error
}

// Tool 工具接口
type Tool interface {
	// Name 返回工具名称，用于匹配
	Name() string
	// Execute 执行工具，返回结果
	Execute(ctx *ToolContext, toolID string) *ToolResult
}
