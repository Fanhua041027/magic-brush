package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

// Transcriber handles audio-to-text transcription
type Transcriber struct {
	apiKey  string
	baseURL string
}

// NewTranscriber creates a new transcription service
func NewTranscriber(apiKey, baseURL string) *Transcriber {
	// Use the base URL for the API endpoint
	// For OpenAI-compatible: {baseURL}/audio/transcriptions
	apiURL := strings.TrimRight(baseURL, "/")
	if !strings.Contains(apiURL, "/v1") {
		apiURL += "/v1"
	}

	return &Transcriber{
		apiKey:  apiKey,
		baseURL: apiURL,
	}
}

// TranscribeWAV sends a WAV file to the OpenAI-compatible Whisper endpoint
// Returns the transcribed text
func (t *Transcriber) TranscribeWAV(wavData []byte) (string, error) {
	if t.apiKey == "" {
		return "", fmt.Errorf("API Key not configured for transcription")
	}

	endpoint := t.baseURL + "/audio/transcriptions"

	// Build multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add the audio file
	part, err := writer.CreateFormFile("file", "capture.wav")
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(wavData)); err != nil {
		return "", fmt.Errorf("failed to copy audio data: %w", err)
	}

	// Add model parameter
	writer.WriteField("model", "whisper-1")
	writer.WriteField("language", "zh")
	writer.WriteField("response_format", "json")

	writer.Close()

	// Send request
	req, err := http.NewRequest("POST", endpoint, &buf)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+t.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("transcription request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read transcription response: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("transcription API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse transcription: %w", err)
	}

	return strings.TrimSpace(result.Text), nil
}

// IsWhisperAvailable checks if the transcription endpoint is available
func (t *Transcriber) IsWhisperAvailable() bool {
	if t.apiKey == "" {
		return false
	}

	// Try to access the models endpoint to verify
	endpoint := t.baseURL + "/models"
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+t.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}
