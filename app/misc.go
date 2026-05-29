package app

import (
	"ai-assistant/pkg/domain"
	"encoding/base64"
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

	const prefix = "data:image/png;base64,"
	data := base64Data
	if len(data) > len(prefix) && data[:len(prefix)] == prefix {
		data = data[len(prefix):]
	}

	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return false, err
	}

	if err := os.WriteFile(filename, decoded, 0644); err != nil {
		return false, err
	}

	return true, nil
}
