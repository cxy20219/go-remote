package server

import (
	"sync"
	"time"
)

// Task status constants
const (
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusTimeout   = "timeout"
	StatusFailed    = "failed"
)

// Task represents a command execution task
type Task struct {
	TaskID    string     `json:"task_id"`
	ClientID  string     `json:"client_id"`
	Command   string     `json:"command"`
	Timeout   int        `json:"timeout"` // timeout in seconds
	Status    string     `json:"status"`
	Result    *TaskResult `json:"result,omitempty"`
	CreateAt  time.Time  `json:"create_at"`
	StartAt   time.Time  `json:"start_at,omitempty"`
	EndAt     time.Time  `json:"end_at,omitempty"`
}

// TaskResult holds the execution result
type TaskResult struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	Error    string `json:"error,omitempty"`
}

// TaskStore manages tasks
type TaskStore struct {
	mu    sync.RWMutex
	tasks map[string]*Task // taskID -> Task
}

// NewTaskStore creates a new TaskStore
func NewTaskStore() *TaskStore {
	return &TaskStore{
		tasks: make(map[string]*Task),
	}
}

// Create creates a new task
func (s *TaskStore) Create(task *Task) {
	s.mu.Lock()
	defer s.mu.Unlock()
	task.CreateAt = time.Now()
	task.Status = StatusPending
	s.tasks[task.TaskID] = task
}

// Get retrieves a task by ID
func (s *TaskStore) Get(taskID string) *Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tasks[taskID]
}

// Update updates a task's status
func (s *TaskStore) Update(taskID string, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if task, ok := s.tasks[taskID]; ok {
		task.Status = status
		if status == StatusRunning {
			task.StartAt = time.Now()
		} else if status == StatusCompleted || status == StatusTimeout || status == StatusFailed {
			task.EndAt = time.Now()
		}
	}
}

// SetResult sets the result for a task
func (s *TaskStore) SetResult(taskID string, result *TaskResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if task, ok := s.tasks[taskID]; ok {
		task.Result = result
		task.Status = StatusCompleted
		task.EndAt = time.Now()
	}
}

// GetByClient returns all tasks for a specific client
func (s *TaskStore) GetByClient(clientID string) []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tasks := make([]*Task, 0)
	for _, task := range s.tasks {
		if task.ClientID == clientID {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

// List returns all tasks
func (s *TaskStore) List() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}
