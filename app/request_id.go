package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"regexp"
)

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

func newRequestID() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "request"
	}
	return hex.EncodeToString(value)
}

func validRequestID(value string) bool {
	return requestIDPattern.MatchString(value)
}

func (a *App) emitRequestEvent(ctx context.Context, taskID int64, event, requestID, field string, value any) {
	if ctx.Err() != nil || !a.taskManager.IsTaskRunning(taskID) {
		return
	}
	payload := map[string]any{"requestId": requestID}
	if field != "" {
		payload[field] = value
	}
	a.EmitEvent(event, payload)
}
