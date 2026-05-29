package sidecar

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"ai-assistant/pkg/logger"
)

func NewManager(port int) *Manager {
	return &Manager{
		port:   port,
		client: NewClient(port),
	}
}

func (m *Manager) Start(model, device string) error {
	// Check if already running
	if err := m.client.Health(); err == nil {
		logger.Printf("[Sidecar] Already running on port %d", m.port)
		return nil
	}

	scriptPath := findSidecarScript()
	if scriptPath == "" {
		return fmt.Errorf("sidecar/main.py not found")
	}

	args := []string{scriptPath, "--port", fmt.Sprintf("%d", m.port), "--model", model, "--device", device}
	m.cmd = exec.Command("python", args...)
	m.cmd.Env = append(os.Environ(), "HF_ENDPOINT=https://hf-mirror.com")

	if err := m.cmd.Start(); err != nil {
		m.cmd = nil
		return fmt.Errorf("start sidecar: %w", err)
	}

	// Wait for health
	deadline := time.Now().Add(startTimeout)
	for time.Now().Before(deadline) {
		if err := m.client.Health(); err == nil {
			logger.Printf("[Sidecar] Ready (PID %d, port %d)", m.cmd.Process.Pid, m.port)
			return nil
		}
		time.Sleep(healthInterval)
	}

	// Timeout — kill and return error
	m.Stop()
	return fmt.Errorf("sidecar did not become ready within %v", startTimeout)
}

func (m *Manager) Stop() {
	if m.cmd != nil && m.cmd.Process != nil {
		logger.Printf("[Sidecar] Stopping (PID %d)", m.cmd.Process.Pid)
		m.cmd.Process.Kill()
		m.cmd.Wait()
		m.cmd = nil
	}
}

func (m *Manager) Client() *Client {
	return m.client
}

func (m *Manager) IsRunning() bool {
	if m.cmd == nil || m.cmd.Process == nil {
		return false
	}
	return m.client.Health() == nil
}

func findSidecarScript() string {
	// Try common locations relative to the executable
	paths := []string{
		"sidecar/main.py",
		"../sidecar/main.py",
		"./sidecar/main.py",
	}
	for _, path := range paths {
		cmd := exec.Command("python", "-c", "import os; print(os.path.isfile('"+path+"'))")
		if out, err := cmd.Output(); err == nil && string(out) == "True\n" {
			return path
		}
	}
	// Fallback: assume CWD is project root
	return "sidecar/main.py"
}
