package task

import (
	"ai-assistant/pkg/logger"
	"context"
	"sync"
	"sync/atomic"
)

// Task 表示一个可取消的任务
type Task struct {
	ID      int64
	Ctx     context.Context
	Cancel  context.CancelFunc
	Name    string
	Running bool
}

// TaskCoordinator 统一管理任务调度和取消
type TaskCoordinator struct {
	mu          sync.Mutex
	currentTask *Task
	taskCounter int64
}

// NewTaskCoordinator 创建任务协调器
func NewTaskCoordinator() *TaskCoordinator {
	return &TaskCoordinator{}
}

// StartTask 开始一个新任务，自动取消之前的任务
// 返回任务上下文和任务ID
func (tc *TaskCoordinator) StartTask(name string) (context.Context, int64) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// 取消之前的任务
	if tc.currentTask != nil && tc.currentTask.Cancel != nil {
		logger.Printf("取消之前的任务: %s (ID: %d)", tc.currentTask.Name, tc.currentTask.ID)
		tc.currentTask.Cancel()
	}

	// 创建新任务
	taskID := atomic.AddInt64(&tc.taskCounter, 1)
	ctx, cancel := context.WithCancel(context.Background())

	tc.currentTask = &Task{
		ID:      taskID,
		Ctx:     ctx,
		Cancel:  cancel,
		Name:    name,
		Running: true,
	}

	logger.Printf("开始新任务: %s (ID: %d)", name, taskID)
	return ctx, taskID
}

// CompleteTask 标记任务完成
func (tc *TaskCoordinator) CompleteTask(taskID int64) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.currentTask != nil && tc.currentTask.ID == taskID {
		tc.currentTask.Running = false
		tc.currentTask.Cancel = nil
		logger.Printf("任务完成: %s (ID: %d)", tc.currentTask.Name, taskID)
	}
}

// CancelCurrentTask 取消当前任务
func (tc *TaskCoordinator) CancelCurrentTask() bool {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.currentTask != nil && tc.currentTask.Cancel != nil {
		tc.currentTask.Cancel()
		tc.currentTask.Cancel = nil
		tc.currentTask.Running = false
		logger.Printf("已取消任务: %s (ID: %d)", tc.currentTask.Name, tc.currentTask.ID)
		return true
	}
	return false
}

// IsTaskRunning 检查指定任务是否还在运行
func (tc *TaskCoordinator) IsTaskRunning(taskID int64) bool {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	return tc.currentTask != nil && tc.currentTask.ID == taskID && tc.currentTask.Running
}

// HasRunningTask 检查是否有任务在运行
func (tc *TaskCoordinator) HasRunningTask() bool {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	return tc.currentTask != nil && tc.currentTask.Running
}

// GetCurrentTaskID 获取当前任务ID
func (tc *TaskCoordinator) GetCurrentTaskID() int64 {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.currentTask != nil {
		return tc.currentTask.ID
	}
	return 0
}
