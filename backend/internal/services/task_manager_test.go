package services

import (
	"context"
	"testing"
)

func TestTaskManager_Lifecycle(t *testing.T) {
	mgr := NewTaskManager()

	// 1. CreateTask
	task, ctx := mgr.CreateTask(context.Background(), "scan_library", 100)
	if task.ID == "" || task.Status != TaskStatusPending || task.TotalCount != 100 {
		t.Fatalf("unexpected task: %+v", task)
	}

	// 2. StartTask
	mgr.StartTask(task.ID)
	cur, err := mgr.GetTask(task.ID)
	if err != nil || cur.Status != TaskStatusRunning || cur.StartedAt == nil {
		t.Fatalf("expected running status, got: %+v, err: %v", cur, err)
	}

	// 3. UpdateProgress
	mgr.UpdateProgress(task.ID, 50, 100, "book50.fb2")
	cur, _ = mgr.GetTask(task.ID)
	if cur.Progress != 50 || cur.ProcessedCount != 50 || cur.CurrentItem != "book50.fb2" {
		t.Fatalf("progress update mismatch: %+v", cur)
	}

	// 4. AddError
	mgr.AddError(task.ID, "sample error")
	cur, _ = mgr.GetTask(task.ID)
	if len(cur.Errors) != 1 || cur.Errors[0] != "sample error" {
		t.Fatalf("expected 1 error, got: %+v", cur.Errors)
	}

	// 5. CompleteTask
	mgr.CompleteTask(task.ID, "Scan finished successfully")
	cur, _ = mgr.GetTask(task.ID)
	if cur.Status != TaskStatusCompleted || cur.Progress != 100 || cur.FinishedAt == nil {
		t.Fatalf("expected completed status, got: %+v", cur)
	}

	// 6. CancelTask on new task
	task2, ctx2 := mgr.CreateTask(context.Background(), "import_calibre", 10)
	mgr.StartTask(task2.ID)
	cancelled := mgr.CancelTask(task2.ID)
	if !cancelled {
		t.Errorf("expected task to be cancelled")
	}

	select {
	case <-ctx2.Done():
		// Context was cancelled properly
	default:
		t.Errorf("expected context to be cancelled")
	}

	cur2, _ := mgr.GetTask(task2.ID)
	if cur2.Status != TaskStatusCancelled {
		t.Errorf("expected cancelled status, got: %s", cur2.Status)
	}

	// 7. ListTasks
	tasks := mgr.ListTasks(10)
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks in history, got %d", len(tasks))
	}
	_ = ctx
}
