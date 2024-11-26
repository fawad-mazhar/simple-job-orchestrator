// task.go defines task-related structures and states
// Provides models for task definition and execution state
// Used for managing individual units of work within jobs
package models

// TaskStatus represents the possible states of a task
// Used to track progress of individual tasks
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "PENDING"   // Task waiting to execute
	TaskStatusRunning   TaskStatus = "RUNNING"   // Task is executing
	TaskStatusCompleted TaskStatus = "COMPLETED" // Task finished successfully
	TaskStatusFailed    TaskStatus = "FAILED"    // Task encountered an error
)

// Task defines a single unit of work
// Represents one step in a job
// Contains configuration for execution and retries
type Task struct {
	ID           string `json:"id"`           // Unique task identifier
	Name         string `json:"name"`         // Human-readable name
	MaxRetry     int    `json:"maxRetry"`     // Maximum retry attempts
	FunctionName string `json:"functionName"` // Name of function to execute
}

// TaskState represents the current state of a task
// Used for status reporting and monitoring
// Combined with other tasks to show job progress
type TaskState struct {
	ID     string     `json:"id"`     // Task identifier
	Name   string     `json:"name"`   // Task name
	Status TaskStatus `json:"status"` // Current status
}
