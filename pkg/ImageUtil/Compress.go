package imageutil

import (
	"bytes"
	"image"
	"image/jpeg"

	"github.com/disintegration/imaging"
)

// ScreenshotMaxDimension 笔试模式截图固定最大边长（长边限制），统一压缩以节省 token
const ScreenshotMaxDimension = 1280

// CompressForOCR 接收原始图片，返回压缩后的 JPEG 字节流（分辨率固定压缩到 ScreenshotMaxDimension）
func CompressForOCR(originalImg image.Image, quality int, sharpen float64, Grayscale bool) ([]byte, error) {
	return CompressForOCRWithMaxSize(originalImg, quality, sharpen, Grayscale, ScreenshotMaxDimension)
}

// CompressForOCRWithMaxSize 接收原始图片，返回压缩后的 JPEG 字节流，支持自定义最大尺寸
func CompressForOCRWithMaxSize(originalImg image.Image, quality int, sharpen float64, Grayscale bool, maxDimension int) ([]byte, error) {
	// 1. 获取原始尺寸
	bounds := originalImg.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 2. 调整大小 (Resize)
	var processedImg image.Image = originalImg

	if width > maxDimension || height > maxDimension {
		if width > height {
			// 宽是长边
			processedImg = imaging.Resize(originalImg, maxDimension, 0, imaging.Lanczos)
		} else {
			// 高是长边
			processedImg = imaging.Resize(originalImg, 0, maxDimension, imaging.Lanczos)
		}
	}

	// 3. 灰度化 (Grayscale)
	// 这一步能去掉颜色干扰，并减小文件体积（对于某些编码格式）
	if Grayscale {
		processedImg = imaging.Grayscale(processedImg)

	}
	// 4. 锐化 (Sharpen) - 可选
	// 稍微锐化一点点有助于 OCR 识别文字边缘，但不要过度
	if sharpen > 0 {
		processedImg = imaging.Sharpen(processedImg, sharpen)
	}

	// 5. 编码为 JPEG 并输出到内存
	// JPEG 质量范围 1-100。如果传入 0 或更小，强制设为 1
	if quality < 1 {
		quality = 1
	}
	if quality > 90 {
		quality = 90
	}

	buf := new(bytes.Buffer)
	err := jpeg.Encode(buf, processedImg, &jpeg.Options{
		Quality: quality, // 质量 80 是体积和清晰度的最佳平衡点
	})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
