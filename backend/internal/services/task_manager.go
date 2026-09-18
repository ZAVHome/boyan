package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// Task представляет долгую фоновую операцию (рескан, импорт Calibre и т.д.).
type Task struct {
	ID             string     `json:"id"`
	Type           string     `json:"type"`
	Status         TaskStatus `json:"status"`
	Progress       int        `json:"progress"` // 0..100
	ProcessedCount int        `json:"processed_count"`
	TotalCount     int        `json:"total_count"`
	CurrentItem    string     `json:"current_item,omitempty"`
	Errors         []string   `json:"errors,omitempty"`
	Message        string     `json:"message,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	FinishedAt     *time.Time `json:"finished_at,omitempty"`

	cancel context.CancelFunc `json:"-"`
}

// TaskManager управляет жизненным циклом и опросом фоновых задач в памяти.
type TaskManager struct {
	mu    sync.RWMutex
	tasks map[string]*Task
	order []string
}

func NewTaskManager() *TaskManager {
	return &TaskManager{
		tasks: make(map[string]*Task),
		order: make([]string, 0),
	}
}

// CreateTask регистрирует новую задачу и возвращает ее контекст выполнения.
func (m *TaskManager) CreateTask(parentCtx context.Context, taskType string, total int) (*Task, context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ctx, cancel := context.WithCancel(parentCtx)
	now := time.Now().UTC()

	task := &Task{
		ID:             uuid.NewString(),
		Type:           taskType,
		Status:         TaskStatusPending,
		Progress:       0,
		ProcessedCount: 0,
		TotalCount:     total,
		Errors:         make([]string, 0),
		CreatedAt:      now,
		cancel:         cancel,
	}

	m.tasks[task.ID] = task
	m.order = append([]string{task.ID}, m.order...)

	// Ограничиваем историю 100 задачами
	if len(m.order) > 100 {
		oldID := m.order[len(m.order)-1]
		delete(m.tasks, oldID)
		m.order = m.order[:100]
	}

	return task, ctx
}

// StartTask переводит задачу в статус running.
func (m *TaskManager) StartTask(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if t, ok := m.tasks[id]; ok {
		t.Status = TaskStatusRunning
		now := time.Now().UTC()
		t.StartedAt = &now
	}
}

// UpdateProgress обновляет прогресс выполнения задачи.
func (m *TaskManager) UpdateProgress(id string, processed, total int, currentItem string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if t, ok := m.tasks[id]; ok {
		t.ProcessedCount = processed
		if total > 0 {
			t.TotalCount = total
		}
		if t.TotalCount > 0 {
			t.Progress = int((float64(t.ProcessedCount) / float64(t.TotalCount)) * 100)
			if t.Progress > 100 {
				t.Progress = 100
			}
		}
		t.CurrentItem = currentItem
	}
}

// AddError добавляет сообщение об ошибке к задаче.
func (m *TaskManager) AddError(id string, errStr string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if t, ok := m.tasks[id]; ok {
		t.Errors = append(t.Errors, errStr)
		if len(t.Errors) > 50 {
			t.Errors = t.Errors[len(t.Errors)-50:]
		}
	}
}

// CompleteTask завершает задачу с успехом.
func (m *TaskManager) CompleteTask(id string, msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if t, ok := m.tasks[id]; ok {
		t.Status = TaskStatusCompleted
		t.Progress = 100
		t.Message = msg
		t.CurrentItem = ""
		now := time.Now().UTC()
		t.FinishedAt = &now
	}
}

// FailTask завершает задачу с ошибкой.
func (m *TaskManager) FailTask(id string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if t, ok := m.tasks[id]; ok {
		t.Status = TaskStatusFailed
		t.Message = err.Error()
		t.CurrentItem = ""
		now := time.Now().UTC()
		t.FinishedAt = &now
	}
}

// CancelTask отменяет выполнение задачи.
func (m *TaskManager) CancelTask(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if t, ok := m.tasks[id]; ok {
		if t.Status == TaskStatusRunning || t.Status == TaskStatusPending {
			if t.cancel != nil {
				t.cancel()
			}
			t.Status = TaskStatusCancelled
			t.Message = "Task was cancelled by administrator"
			now := time.Now().UTC()
			t.FinishedAt = &now
			return true
		}
	}
	return false
}

func (t *Task) clone() *Task {
	if t == nil {
		return nil
	}
	cp := *t
	if t.Errors != nil {
		cp.Errors = make([]string, len(t.Errors))
		copy(cp.Errors, t.Errors)
	}
	return &cp
}

// GetTask возвращает копию состояния задачи.
func (m *TaskManager) GetTask(id string) (*Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	t, ok := m.tasks[id]
	if !ok {
		return nil, fmt.Errorf("task not found")
	}

	return t.clone(), nil
}

// ListTasks возвращает список недавних задач.
func (m *TaskManager) ListTasks(limit int) []*Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 || limit > len(m.order) {
		limit = len(m.order)
	}

	res := make([]*Task, 0, limit)
	for i := 0; i < limit; i++ {
		taskID := m.order[i]
		if t, ok := m.tasks[taskID]; ok {
			res = append(res, t.clone())
		}
	}
	return res
}
