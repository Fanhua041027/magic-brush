package app

// ── STT (Speech-to-Text) bindings ─────────────────────────────────────

func (a *App) GetSTTStatus() map[string]bool {
	if a.sidecar == nil || !a.sidecar.IsRunning() {
		return map[string]bool{"recording": false, "ready": false}
	}
	status, err := a.sidecar.Client().STTStatus()
	if err != nil {
		return map[string]bool{"recording": false, "ready": false}
	}
	return map[string]bool{"recording": status.Recording, "ready": true}
}

func (a *App) ToggleSTT() map[string]string {
	if a.sidecar == nil || !a.sidecar.IsRunning() {
		return map[string]string{"error": "Sidecar not running"}
	}

	status, err := a.sidecar.Client().STTStatus()
	if err != nil {
		return map[string]string{"error": err.Error()}
	}

	if status.Recording {
		result, err := a.sidecar.Client().STTStop()
		if err != nil {
			return map[string]string{"error": err.Error()}
		}
		return map[string]string{"text": result.Text, "status": "transcribed"}
	}

	if err := a.sidecar.Client().STTStart(); err != nil {
		return map[string]string{"error": err.Error()}
	}
	return map[string]string{"status": "recording"}
}

// ── KB (Knowledge Base) bindings ──────────────────────────────────────

func (a *App) LoadKB(path string) map[string]interface{} {
	if a.sidecar == nil || !a.sidecar.IsRunning() {
		return map[string]interface{}{"error": "Sidecar not running"}
	}
	result, err := a.sidecar.Client().KBLoad(path)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{
		"status":        result.Status,
		"file_count":    result.FileCount,
		"section_count": result.SectionCount,
	}
}

func (a *App) SearchKB(query string) map[string]interface{} {
	if a.sidecar == nil || !a.sidecar.IsRunning() {
		return map[string]interface{}{"error": "Sidecar not running"}
	}
	result, err := a.sidecar.Client().KBSearch(query, 5)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	items := make([]map[string]interface{}, 0, len(result.Results))
	for _, r := range result.Results {
		items = append(items, map[string]interface{}{
			"source":  r.Source,
			"header":  r.Header,
			"content": r.Content,
			"score":   r.Score,
		})
	}
	return map[string]interface{}{"results": items}
}

func (a *App) GetKBStatus() map[string]interface{} {
	if a.sidecar == nil || !a.sidecar.IsRunning() {
		return map[string]interface{}{"ready": false}
	}

	cfg := a.configManager.Get()
	return map[string]interface{}{
		"ready":     cfg.KBPath != "",
		"kb_path":   cfg.KBPath,
	}
}
