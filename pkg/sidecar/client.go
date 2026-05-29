package sidecar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"time"
)

const (
	defaultPort = 18765
	startTimeout = 120 * time.Second // 增加到 120 秒，因为首次需要下载模型
	healthInterval = 500 * time.Millisecond
)

type Manager struct {
	port   int
	cmd    *exec.Cmd
	client *Client
}

type Client struct {
	baseURL string
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
	ID               int    `json:"id"`
	Name             string `json:"name"`
	Channels         int    `json:"channels"`
	DefaultSamplerate int   `json:"default_samplerate"`
	HostAPI          string `json:"host_api"`
	IsDefault        bool   `json:"is_default"`
}

type STTDevicesResult struct {
	Devices           []STTDeviceInfo `json:"devices"`
	CurrentDeviceID   *int            `json:"current_device_id"`
	CurrentSampleRate int             `json:"current_sample_rate"`
	Error             string          `json:"error,omitempty"`
}

type STTSetDeviceResult struct {
	Status     string `json:"status"`
	DeviceID   *int   `json:"device_id"`
	SampleRate int    `json:"sample_rate"`
	Error      string `json:"error,omitempty"`
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
	Error   string        `json:"error,omitempty"`
}

type KBSearchItem struct {
	Source  string  `json:"source"`
	Header  string  `json:"header"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

func NewClient(port int) *Client {
	return &Client{
		baseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) Health() error {
	resp, err := c.http.Get(c.baseURL + "/api/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (c *Client) STTStart() error {
	resp, err := c.http.Post(c.baseURL+"/api/stt/start", "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("STT start failed (%d): %s", resp.StatusCode, string(body))
	}
	return nil
}

func (c *Client) STTStop() (*STTResult, error) {
	resp, err := c.http.Post(c.baseURL+"/api/stt/stop", "application/json", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result STTResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 && result.Error != "" {
		return nil, fmt.Errorf("STT error: %s", result.Error)
	}
	return &result, nil
}

func (c *Client) STTStatus() (*STTStatus, error) {
	resp, err := c.http.Get(c.baseURL + "/api/stt/status")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var status STTStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}
	return &status, nil
}

func (c *Client) STTDevices() (*STTDevicesResult, error) {
	resp, err := c.http.Get(c.baseURL + "/api/stt/devices")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result STTDevicesResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) STTSetDevice(deviceID int) (*STTSetDeviceResult, error) {
	body, _ := json.Marshal(map[string]int{"device_id": deviceID})
	resp, err := c.http.Post(c.baseURL+"/api/stt/device", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result STTSetDeviceResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 && result.Error != "" {
		return nil, fmt.Errorf("STT set device error: %s", result.Error)
	}
	return &result, nil
}

func (c *Client) KBLoad(path string) (*KBLoadResult, error) {
	body, _ := json.Marshal(map[string]string{"path": path})
	resp, err := c.http.Post(c.baseURL+"/api/kb/load", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result KBLoadResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) KBInfo() (*KBInfoResult, error) {
	resp, err := c.http.Get(c.baseURL + "/api/kb/info")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result KBInfoResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) KBSearch(query string, topK int) (*KBSearchResult, error) {
	body, _ := json.Marshal(map[string]interface{}{"query": query, "top_k": topK})
	resp, err := c.http.Post(c.baseURL+"/api/kb/search", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result KBSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
