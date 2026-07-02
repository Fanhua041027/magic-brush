package task

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestNewTaskCoordinator(t *testing.T) {
	tc := NewTaskCoordinator()
	if tc == nil {
		t.Fatal("NewTaskCoordinator should not return nil")
	}
	if tc.HasRunningTask() {
		t.Fatal("new coordinator should have no running task")
	}
}

func TestStartTask(t *testing.T) {
	tc := NewTaskCoordinator()

	ctx, id := tc.StartTask("test")
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
	if id <= 0 {
		t.Fatal("expected positive task ID")
	}
	if !tc.HasRunningTask() {
		t.Fatal("expected running task after StartTask")
	}
}

func TestCompleteTask(t *testing.T) {
	tc := NewTaskCoordinator()

	_, id := tc.StartTask("test")
	if !tc.HasRunningTask() {
		t.Fatal("expected running task")
	}

	tc.CompleteTask(id)
	if tc.HasRunningTask() {
		t.Fatal("expected no running task after completion")
	}
}

func TestCancelTask(t *testing.T) {
	tc := NewTaskCoordinator()

	ctx, _ := tc.StartTask("test")
	if !tc.HasRunningTask() {
		t.Fatal("expected running task")
	}

	cancelled := tc.CancelCurrentTask()
	if !cancelled {
		t.Fatal("expected CancelCurrentTask to return true")
	}

	select {
	case <-ctx.Done():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("context should be cancelled")
	}

	if tc.HasRunningTask() {
		t.Fatal("expected no running task after cancellation")
	}

	if tc.CancelCurrentTask() {
		t.Fatal("expected CancelCurrentTask to return false when no task")
	}
}

func TestCancelCurrentTaskNoTask(t *testing.T) {
	tc := NewTaskCoordinator()
	if tc.CancelCurrentTask() {
		t.Fatal("expected false when no running task")
	}
}

func TestNewTaskCancelsOld(t *testing.T) {
	tc := NewTaskCoordinator()

	ctx1, _ := tc.StartTask("task1")
	if !tc.HasRunningTask() {
		t.Fatal("expected running task after start")
	}

	ctx2, id2 := tc.StartTask("task2")

	select {
	case <-ctx1.Done():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("first context should be cancelled when new task starts")
	}

	select {
	case <-ctx2.Done():
		t.Fatal("second context should NOT be cancelled")
	default:
	}

	if !tc.HasRunningTask() {
		t.Fatal("expected second task to be running")
	}

	tc.CompleteTask(id2)
	if tc.HasRunningTask() {
		t.Fatal("expected no running task after completion")
	}
}

func TestConcurrentStartTask(t *testing.T) {
	tc := NewTaskCoordinator()
	var counter int64

	for i := 0; i < 10; i++ {
		go func() {
			ctx, id := tc.StartTask("concurrent")
			if ctx == nil || id <= 0 {
				atomic.AddInt64(&counter, 1)
			}
		}()
	}

	time.Sleep(50 * time.Millisecond)

	if !tc.HasRunningTask() && atomic.LoadInt64(&counter) == 10 {
		t.Fatal("expected at least one task to be running")
	}
}

func TestTaskContextCancellation(t *testing.T) {
	tc := NewTaskCoordinator()

	_, id := tc.StartTask("long-task")

	if tc.HasRunningTask() != true {
		t.Fatal("expected running task")
	}

	tc.CompleteTask(id)

	if tc.HasRunningTask() {
		t.Fatal("expected no running task after CompleteTask")
	}
}

func TestRunningTaskChecks(t *testing.T) {
	tc := NewTaskCoordinator()

	if tc.HasRunningTask() {
		t.Fatal("expected no task initially")
	}

	_, id1 := tc.StartTask("task-a")
	if !tc.HasRunningTask() {
		t.Fatal("expected task running")
	}

	tc.CompleteTask(id1)
	if tc.HasRunningTask() {
		t.Fatal("expected no task after complete")
	}

	tc.StartTask("task-b")
	if !tc.HasRunningTask() {
		t.Fatal("expected task running after second start")
	}

	tc.CancelCurrentTask()
	if tc.HasRunningTask() {
		t.Fatal("expected no task after cancel")
	}
}
