package sidecar

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	defaultPort      = 18765
	startTimeout     = 120 * time.Second
	healthInterval   = 500 * time.Millisecond
	maxResponseBytes = 4 << 20
)

type Manager struct {
	mu          sync.Mutex
	port        int
	cmd         *exec.Cmd
	waitCh      <-chan error
	client      *Client
	generation  uint64
	starting    bool
	running     bool
	startCancel context.CancelFunc
}

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

type STTStatus struct {
	Recording bool `json:"recording"`
}

type STTResult struct {
	Text  string `json:"text"`
	Error string `json:"error,omitempty"`
}

type STTDeviceInfo struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	Type              string `json:"type"`
	Channels          int    `json:"channels"`
	DefaultSamplerate int    `json:"default_samplerate"`
	HostAPI           string `json:"host_api"`
	IsDefault         bool   `json:"is_default"`
	Recommended       bool   `json:"recommended,omitempty"`
	Backend           string `json:"backend,omitempty"`
	CaptureMode       string `json:"capture_mode,omitempty"`
	IsLoopback        bool   `json:"is_loopback,omitempty"`
	SpeakerID         string `json:"speaker_id,omitempty"`
	Available         bool   `json:"available,omitempty"`
	UnavailableReason string `json:"unavailable_reason,omitempty"`
}

type AudioStatus struct {
	Ready            bool   `json:"ready"`
	Backend          string `json:"backend,omitempty"`
	CaptureMode      string `json:"capture_mode,omitempty"`
	IsLoopback       bool   `json:"is_loopback,omitempty"`
	SourceName       string `json:"source_name,omitempty"`
	NativeSampleRate int    `json:"native_sample_rate,omitempty"`
	OutputSampleRate int    `json:"output_sample_rate,omitempty"`
	Fallback         bool   `json:"fallback,omitempty"`
	FallbackReason   string `json:"fallback_reason,omitempty"`
	DroppedChunks    int    `json:"dropped_chunks,omitempty"`
}

type STTDevicesResult struct {
	Devices           []STTDeviceInfo `json:"devices"`
	CurrentDeviceID   *int            `json:"current_device_id"`
	CurrentSampleRate int             `json:"current_sample_rate"`
	Audio             AudioStatus     `json:"audio,omitempty"`
	Error             string          `json:"error,omitempty"`
}

type STTSetDeviceResult struct {
	Status     string `json:"status"`
	DeviceID   *int   `json:"device_id"`
	SampleRate int    `json:"sample_rate"`
	Error      string `json:"error,omitempty"`
}

type AudioLevelResult struct {
	Level float64 `json:"level"`
}

type KBLoadResult struct {
	Status       string `json:"status"`
	FileCount    int    `json:"file_count"`
	SectionCount int    `json:"section_count"`
	Error        string `json:"error,omitempty"`
}

type KBInfoResult struct {
	Ready        bool   `json:"ready"`
	FileCount    int    `json:"file_count"`
	SectionCount int    `json:"section_count"`
	KBPath       string `json:"kb_path"`
}

type KBSearchResult struct {
	Results []KBSearchItem `json:"results"`
	Error   string         `json:"error,omitempty"`
}

type KBSearchItem struct {
	Source  string  `json:"source"`
	Header  string  `json:"header"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

func NewClient(port int) *Client {
	return NewClientWithToken(port, "")
}

func NewClientWithToken(port int, token string) *Client {
	return &Client{
		baseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
		token:   token,
		http:    &http.Client{Timeout: 180 * time.Second},
	}
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, maxResponseBytes+1)
	payload, err := io.ReadAll(limited)
	if err != nil {
		return err
	}
	if len(payload) > maxResponseBytes {
		return errors.New("sidecar response exceeds size limit")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := "sidecar request failed"
		var envelope struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if json.Unmarshal(payload, &envelope) == nil {
			if envelope.Message != "" {
				message = envelope.Message
			} else if envelope.Error != "" {
				message = envelope.Error
			}
		}
		message = strings.TrimSpace(message)
		if len(message) > 300 {
			message = message[:300]
		}
		return fmt.Errorf("%s (status %d)", message, resp.StatusCode)
	}
	if out != nil && len(payload) > 0 {
		if err := json.Unmarshal(payload, out); err != nil {
			return fmt.Errorf("decode sidecar response: %w", err)
		}
	}
	return nil
}

func (c *Client) HealthContext(ctx context.Context) error {
	return c.do(ctx, http.MethodGet, "/api/health", nil, nil)
}
func (c *Client) Health() error { return c.HealthContext(context.Background()) }

func (c *Client) ShutdownContext(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/api/shutdown", struct{}{}, nil)
}

func (c *Client) STTStartContext(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/api/stt/start", struct{}{}, nil)
}
func (c *Client) STTStart() error { return c.STTStartContext(context.Background()) }

func (c *Client) STTStartStreamingContext(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/api/stt/start-streaming", struct{}{}, nil)
}
func (c *Client) STTStartStreaming() error { return c.STTStartStreamingContext(context.Background()) }

func (c *Client) STTStreamingResultsContext(ctx context.Context) ([]string, error) {
	var result struct {
		Results []string `json:"results"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/stt/streaming-results", nil, &result); err != nil {
		return nil, err
	}
	return result.Results, nil
}
func (c *Client) STTStreamingResults() ([]string, error) {
	return c.STTStreamingResultsContext(context.Background())
}

func (c *Client) STTStopContext(ctx context.Context) (*STTResult, error) {
	var result STTResult
	if err := c.do(ctx, http.MethodPost, "/api/stt/stop", struct{}{}, &result); err != nil {
		return nil, err
	}
	if result.Error != "" {
		return nil, errors.New(result.Error)
	}
	return &result, nil
}
func (c *Client) STTStop() (*STTResult, error) { return c.STTStopContext(context.Background()) }

func (c *Client) STTTranscribeContext(ctx context.Context, audioBase64 string, sampleRate int) (*STTResult, error) {
	var result STTResult
	body := map[string]any{"audio": audioBase64, "sample_rate": sampleRate}
	if err := c.do(ctx, http.MethodPost, "/api/stt/transcribe", body, &result); err != nil {
		return nil, err
	}
	if result.Error != "" {
		return nil, errors.New(result.Error)
	}
	return &result, nil
}
func (c *Client) STTTranscribe(audioBase64 string, sampleRate int) (*STTResult, error) {
	return c.STTTranscribeContext(context.Background(), audioBase64, sampleRate)
}

func (c *Client) STTStatusContext(ctx context.Context) (*STTStatus, error) {
	var result STTStatus
	if err := c.do(ctx, http.MethodGet, "/api/stt/status", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
func (c *Client) STTStatus() (*STTStatus, error) { return c.STTStatusContext(context.Background()) }

func (c *Client) STTDevicesContext(ctx context.Context) (*STTDevicesResult, error) {
	var result STTDevicesResult
	if err := c.do(ctx, http.MethodGet, "/api/stt/devices", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
func (c *Client) STTDevices() (*STTDevicesResult, error) {
	return c.STTDevicesContext(context.Background())
}

func (c *Client) STTSetDeviceContext(ctx context.Context, deviceID int) (*STTSetDeviceResult, error) {
	var result STTSetDeviceResult
	if err := c.do(ctx, http.MethodPost, "/api/stt/device", map[string]int{"device_id": deviceID}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
func (c *Client) STTSetDevice(deviceID int) (*STTSetDeviceResult, error) {
	return c.STTSetDeviceContext(context.Background(), deviceID)
}

func (c *Client) STTSetDeviceByNameContext(ctx context.Context, deviceName string) (*STTSetDeviceResult, error) {
	var result STTSetDeviceResult
	if err := c.do(ctx, http.MethodPost, "/api/stt/device", map[string]string{"device_name": deviceName}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
func (c *Client) STTSetDeviceByName(deviceName string) (*STTSetDeviceResult, error) {
	return c.STTSetDeviceByNameContext(context.Background(), deviceName)
}

func (c *Client) AudioLevelContext(ctx context.Context) (*AudioLevelResult, error) {
	var result AudioLevelResult
	if err := c.do(ctx, http.MethodGet, "/api/audio/level", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) KBLoadContext(ctx context.Context, path string) (*KBLoadResult, error) {
	var result KBLoadResult
	if err := c.do(ctx, http.MethodPost, "/api/kb/load", map[string]string{"path": path}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
func (c *Client) KBLoad(path string) (*KBLoadResult, error) {
	return c.KBLoadContext(context.Background(), path)
}

func (c *Client) KBInfoContext(ctx context.Context) (*KBInfoResult, error) {
	var result KBInfoResult
	if err := c.do(ctx, http.MethodGet, "/api/kb/info", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
func (c *Client) KBInfo() (*KBInfoResult, error) { return c.KBInfoContext(context.Background()) }

func (c *Client) KBSearchContext(ctx context.Context, query string, topK int) (*KBSearchResult, error) {
	var result KBSearchResult
	if err := c.do(ctx, http.MethodPost, "/api/kb/search", map[string]any{"query": query, "top_k": topK}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
func (c *Client) KBSearch(query string, topK int) (*KBSearchResult, error) {
	return c.KBSearchContext(context.Background(), query, topK)
}
