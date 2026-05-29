package tools

import (
	"ai-assistant/pkg/logger"
	"sync"
)

// Registry 工具注册表
type Registry struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

// NewRegistry 创建新的工具注册表
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register 注册工具
func (r *Registry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
	logger.Printf("[Registry] 已注册工具: %s", tool.Name())
}

// Get 获取工具
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	return tool, ok
}

// Has 检查工具是否存在
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.tools[name]
	return ok
}

// List 列出所有已注册的工具名称
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// Execute 执行指定名称的工具
// 如果工具不存在，返回包含错误的 ToolResult
func (r *Registry) Execute(ctx *ToolContext, toolID, toolName string) *ToolResult {
	r.mu.RLock()
	tool, ok := r.tools[toolName]
	r.mu.RUnlock()

	if !ok {
		logger.Printf("[Registry] 未知工具: %s", toolName)
		return &ToolResult{
			Text: "",
		}
	}

	logger.Printf("[Registry] 执行工具: %s (toolID=%s)", toolName, toolID)
	return tool.Execute(ctx, toolID)
}

// DefaultRegistry 默认全局注册表
var DefaultRegistry = NewRegistry()

// Register 向默认注册表注册工具
func Register(tool Tool) {
	DefaultRegistry.Register(tool)
}

// Execute 使用默认注册表执行工具
func Execute(ctx *ToolContext, toolID, toolName string) *ToolResult {
	return DefaultRegistry.Execute(ctx, toolID, toolName)
}
