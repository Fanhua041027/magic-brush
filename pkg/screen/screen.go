package screen

import (
	imageutil "ai-assistant/pkg/ImageUtil"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image/png"

	"github.com/kbinani/screenshot"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type PreviewResult struct {
	ImgBytes []byte `json:"imgBytes"`
	Base64   string `json:"base64"`
	Size     string `json:"size"`
}

type Service struct {
	ctx context.Context
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Startup(ctx context.Context) {
	s.ctx = ctx
}

// CapturePreview 获取当前截图的预览（Base64）
func (s *Service) CapturePreview(quality int, sharpen float64, grayscale bool, noCompression bool, mode string) (PreviewResult, error) {
	var x, y, w, h int

	if mode == "fullscreen" {
		// 全屏模式：获取主屏幕尺寸
		bounds := screenshot.GetDisplayBounds(0)
		x, y, w, h = bounds.Min.X, bounds.Min.Y, bounds.Dx(), bounds.Dy()
	} else {
		// 窗口模式：获取当前窗口位置和大小
		if s.ctx == nil {
			return PreviewResult{}, fmt.Errorf("context not initialized")
		}
		x, y = runtime.WindowGetPosition(s.ctx)
		w, h = runtime.WindowGetSize(s.ctx)
	}

	// 截图
	img, err := screenshot.Capture(x, y, w, h)
	if err != nil {
		return PreviewResult{}, fmt.Errorf("截图失败: %v", err)
	}

	// 处理图片
	var imgBytes []byte
	var ImageBase64 string
	if noCompression {
		var buf bytes.Buffer
		err = png.Encode(&buf, img)
		if err != nil {
			return PreviewResult{}, fmt.Errorf("图片编码失败: %v", err)
		}
		imgBytes = buf.Bytes()
		ImageBase64 = fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(imgBytes))
	} else {
		imgBytes, err = imageutil.CompressForOCR(img, quality, sharpen, grayscale)
		if err != nil {
			return PreviewResult{}, fmt.Errorf("图片处理失败: %v", err)
		}
		ImageBase64 = fmt.Sprintf("data:image/jpeg;base64,%s", base64.StdEncoding.EncodeToString(imgBytes))
	}

	// 计算大小
	sizeKB := float64(len(ImageBase64)) / 1024.0
	sizeStr := fmt.Sprintf("%.2f KB", sizeKB)

	// 转 Base64
	return PreviewResult{
		ImgBytes: imgBytes,
		Base64:   ImageBase64,
		Size:     sizeStr,
	}, nil
}
