// job.go defines the core job-related structures and states
// Provides models for both job definitions and executions
// Used throughout the system for job management
package models

import (
	"time"
)

// JobStatus represents the possible states of a job
// Used to track job progress through the system
type JobStatus string

const (
	JobStatusQueued    JobStatus = "QUEUED"    // Job is waiting in queue
	JobStatusRunning   JobStatus = "RUNNING"   // Job is currently executing
	JobStatusCompleted JobStatus = "COMPLETED" // Job finished successfully
	JobStatusFailed    JobStatus = "FAILED"    // Job encountered an error
)

// JobDefinition represents the template for a job
// Defines the sequence of tasks to be executed
// Used to create job executions
type JobDefinition struct {
	ID    string              `json:"id"`    // Unique identifier for the job definition
	Name  string              `json:"name"`  // Human-readable name
	Tasks map[string]*Task    `json:"tasks"` // Map of tasks
	Graph map[string][]string `json:"graph"` // Dependency graph that helps in parallel execution of tasks
}

// JobExecution represents a single run of a job
// Tracks the state and progress of job execution
// Maintains task status and execution metadata
type JobExecution struct {
	ID           string                 `json:"id"`                // Unique execution identifier
	DefinitionID string                 `json:"definitionId"`      // Reference to job definition
	Status       JobStatus              `json:"status"`            // Current execution status
	StartTime    time.Time              `json:"startTime"`         // When execution began
	EndTime      time.Time              `json:"endTime,omitempty"` // When execution finished
	Data         map[string]interface{} `json:"data"`              // Input data for tasks
	TaskStatuses map[string]TaskStatus  `json:"taskStatuses"`      // Status of each task
}

// JobExecutionState provides a snapshot of job execution
// Used for API responses and status reporting
// Combines execution status with task states
type JobExecutionState struct {
	ID           string      `json:"id"`           // Execution identifier
	DefinitionID string      `json:"definitionId"` // Reference to definition
	Status       JobStatus   `json:"status"`       // Current status
	StartTime    time.Time   `json:"startTime"`    // Execution start time
	Tasks        []TaskState `json:"tasks"`        // State of all tasks
}
