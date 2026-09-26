package sidecar

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"ai-assistant/pkg/logger"
)

func NewManager(port int) *Manager {
	return &Manager{port: port, client: NewClient(port)}
}

func (m *Manager) Start(model, device, language string, sensitivity float64, sttService string) error {
	return m.StartContext(context.Background(), model, device, language, sensitivity, sttService)
}

func (m *Manager) StartContext(parent context.Context, model, device, language string, sensitivity float64, sttService string) error {
	m.mu.Lock()
	if m.running || m.starting {
		m.mu.Unlock()
		return nil
	}
	scriptPath := findSidecarScript()
	if scriptPath == "" {
		m.mu.Unlock()
		return fmt.Errorf("sidecar/main.py not found")
	}
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		m.mu.Unlock()
		return fmt.Errorf("generate sidecar token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)
	m.client = NewClientWithToken(m.port, token)
	startCtx, cancel := context.WithTimeout(parent, startTimeout)
	m.startCancel, m.starting = cancel, true
	m.generation++
	generation := m.generation
	args := []string{scriptPath, "--port", fmt.Sprintf("%d", m.port), "--model", model, "--device", device, "--language", language, "--sensitivity", fmt.Sprintf("%.2f", sensitivity), "--stt", sttService}
	cmd := exec.Command(findPython(), args...)
	cmd.Dir = filepath.Dir(scriptPath)
	pythonPathEnv := filepath.Dir(scriptPath)
	if existing := os.Getenv("PYTHONPATH"); existing != "" {
		pythonPathEnv += string(os.PathListSeparator) + existing
	}
	cmd.Env = append(os.Environ(), "HF_ENDPOINT=https://hf-mirror.com", "PYTHONIOENCODING=utf-8", "PYTHONPATH="+pythonPathEnv, "MAGIC_BRUSH_SIDECAR_TOKEN="+token)
	m.mu.Unlock()

	if err := cmd.Start(); err != nil {
		m.mu.Lock()
		if m.generation == generation {
			m.starting = false
			m.startCancel = nil
		}
		m.mu.Unlock()
		cancel()
		return fmt.Errorf("start sidecar: %w", err)
	}
	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait() }()
	m.mu.Lock()
	if m.generation != generation || !m.starting {
		m.mu.Unlock()
		_ = cmd.Process.Kill()
		<-waitCh
		cancel()
		return context.Canceled
	}
	m.cmd, m.waitCh = cmd, waitCh
	m.mu.Unlock()

	ticker := time.NewTicker(healthInterval)
	defer ticker.Stop()
	for {
		select {
		case <-startCtx.Done():
			m.Stop()
			return fmt.Errorf("sidecar did not become ready within %v", startTimeout)
		case <-waitCh:
			m.mu.Lock()
			if m.generation == generation {
				m.cmd = nil
				m.waitCh = nil
				m.starting = false
			}
			m.mu.Unlock()
			return fmt.Errorf("sidecar exited before becoming ready")
		case <-ticker.C:
			probeCtx, probeCancel := context.WithTimeout(startCtx, time.Second)
			err := m.client.HealthContext(probeCtx)
			probeCancel()
			if err == nil {
				m.mu.Lock()
				if m.generation == generation {
					m.starting = false
					m.running = true
					m.startCancel = nil
				}
				m.mu.Unlock()
				logger.Printf("[Sidecar] Ready (PID %d, port %d)", cmd.Process.Pid, m.port)
				cancel()
				return nil
			}
		}
	}
}

func (m *Manager) Stop() {
	m.mu.Lock()
	if m.startCancel != nil {
		m.startCancel()
	}
	m.generation++
	cmd, waitCh := m.cmd, m.waitCh
	m.cmd, m.waitCh, m.starting, m.running, m.startCancel = nil, nil, false, false, nil
	m.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return
	}
	logger.Printf("[Sidecar] Stopping (PID %d)", cmd.Process.Pid)
	if m.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		_ = m.client.ShutdownContext(ctx)
		cancel()
	}
	_ = cmd.Process.Kill()
	if waitCh != nil {
		select {
		case <-waitCh:
		case <-time.After(5 * time.Second):
			logger.Println("[Sidecar] Timed out waiting for process exit")
		}
	}
}

func (m *Manager) Client() *Client {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.client
}
func (m *Manager) Port() int { return m.port }
func (m *Manager) IsRunning() bool {
	m.mu.Lock()
	running := m.running
	client := m.client
	m.mu.Unlock()
	if !running || client == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return client.HealthContext(ctx) == nil
}

func findSidecarScript() string {
	if configured := os.Getenv("MAGIC_BRUSH_SIDECAR"); configured != "" {
		if absolute, err := filepath.Abs(configured); err == nil {
			if _, err := os.Stat(absolute); err == nil {
				return absolute
			}
		}
	}

	candidates := []string{}
	if executable, err := os.Executable(); err == nil {
		base := filepath.Dir(executable)
		candidates = append(candidates,
			filepath.Join(base, "sidecar", "main.py"),
			filepath.Join(base, "resources", "sidecar", "main.py"),
			filepath.Join(base, "..", "sidecar", "main.py"),
		)
	}
	candidates = append(candidates,
		filepath.Join("sidecar", "main.py"),
		filepath.Join("..", "sidecar", "main.py"),
	)
	for _, candidate := range candidates {
		absolute, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if _, err := os.Stat(absolute); err == nil {
			return absolute
		}
	}
	return ""
}

func findPython() string {
	for _, name := range []string{"python", "python3"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	return "python"
}
