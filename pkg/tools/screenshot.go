package tools

import (
	imageutil "ai-assistant/pkg/ImageUtil"
	"ai-assistant/pkg/logger"
	"fmt"

	"github.com/kbinani/screenshot"
)

// ScreenshotTool 截图工具
type ScreenshotTool struct{}

// NewScreenshotTool 创建截图工具
func NewScreenshotTool() *ScreenshotTool {
	return &ScreenshotTool{}
}

// Name 返回工具名称
func (t *ScreenshotTool) Name() string {
	return "get_exam_question"
}

// Execute 执行截图
func (t *ScreenshotTool) Execute(ctx *ToolContext, toolID string) *ToolResult {
	// 直接执行截图，不再依赖外部服务
	// Live API 专用：全屏截图 + 激进压缩
	bounds := screenshot.GetDisplayBounds(0)
	x, y, w, h := bounds.Min.X, bounds.Min.Y, bounds.Dx(), bounds.Dy()

	img, err := screenshot.Capture(x, y, w, h)
	if err != nil {
		logger.Printf("[ScreenshotTool] 截图失败: %v", err)
		return &ToolResult{
			Text:  "截图失败: " + err.Error(),
			Error: err,
		}
	}

	// Live API 截图参数：
	// - 最大尺寸 1280px（兼顾 2K/4K 屏幕清晰度与传输体积）
	// - 质量 85（保持清晰度，便于模型识别）
	// - 保留彩色（不灰度化，保留更多视觉信息）
	// - 适度锐化（增强文字边缘）
	imgBytes, err := imageutil.CompressForOCRWithMaxSize(img, 85, 0.5, false, 1280)
	if err != nil {
		logger.Printf("[ScreenshotTool] 图片处理失败: %v", err)
		return &ToolResult{
			Text:  "图片处理失败: " + err.Error(),
			Error: err,
		}
	}

	sizeKB := float64(len(imgBytes)) / 1024.0
	logger.Printf("[ScreenshotTool] 截图成功 (%d bytes, %.2f KB)", len(imgBytes), sizeKB)

	return &ToolResult{
		ImageData:     imgBytes,
		ImageMimeType: "image/jpeg",
		HasImage:      true,
	}
}

// CaptureFullScreen 全屏截图工具函数（供其他地方复用）
func CaptureFullScreen() ([]byte, error) {
	bounds := screenshot.GetDisplayBounds(0)
	x, y, w, h := bounds.Min.X, bounds.Min.Y, bounds.Dx(), bounds.Dy()

	img, err := screenshot.Capture(x, y, w, h)
	if err != nil {
		return nil, fmt.Errorf("截图失败: %v", err)
	}

	imgBytes, err := imageutil.CompressForOCRWithMaxSize(img, 85, 0.5, false, 1280)
	if err != nil {
		return nil, fmt.Errorf("图片处理失败: %v", err)
	}

	return imgBytes, nil
}

// init 自动注册截图工具到默认注册表
func init() {
	Register(NewScreenshotTool())
}
