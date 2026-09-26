package app

import (
	"ai-assistant/pkg/domain"
	"encoding/base64"
	"errors"
	"os"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) GetInitStatus() string {
	return a.stateManager.GetInitStatusString()
}

func (a *App) GetDomainCategories() []domain.Category {
	return domain.GetCategories()
}

func (a *App) SaveImageToFile(base64Data string) (bool, error) {
	const (
		prefix         = "data:image/png;base64,"
		maxEncodedSize = 12 << 20
		maxImageSize   = 8 << 20
	)
	if len(base64Data) <= len(prefix) || len(base64Data) > maxEncodedSize || base64Data[:len(prefix)] != prefix {
		return false, errors.New("图片内容无效或过大")
	}
	filename, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "保存图片",
		DefaultFilename: "q-solver-export.png",
		Filters: []runtime.FileFilter{
			{DisplayName: "PNG 图片", Pattern: "*.png"},
		},
	})
	if err != nil {
		return false, err
	}
	if filename == "" {
		return false, nil
	}

	data := base64Data[len(prefix):]

	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil || len(decoded) > maxImageSize {
		return false, errors.New("图片内容无效或过大")
	}

	if err := os.WriteFile(filename, decoded, 0644); err != nil {
		return false, err
	}

	return true, nil
}
