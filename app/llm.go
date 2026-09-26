package app

import "context"

func (a *App) TestConnection(apiKey, model string) string {
	if apiKey == "" {
		apiKey = a.configManager.Get().APIKey
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return a.llmService.TestConnection(ctx, apiKey, model)
}

func (a *App) GetModels(apiKey string) ([]string, error) {
	if apiKey == "" {
		apiKey = a.configManager.Get().APIKey
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return a.llmService.GetModels(ctx, apiKey)
}
