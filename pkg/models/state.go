// state.go defines system-wide state structures
// Used for monitoring and reporting overall system status
// Provides overview of all jobs and queue state
package models

// SystemState represents the current state of the entire system
// Used for system monitoring and status reporting
// Provides overview of active and queued jobs
type SystemState struct {
	ActiveJobs   []JobExecutionState `json:"activeJobs"`   // Currently executing jobs
	QueuedJobs   []string            `json:"queuedJobs"`   // Jobs waiting in queue
	QueuedCount  int                 `json:"queuedCount"`  // Total queue size
	ExecutedJobs int                 `json:"executedJobs"` // Count of successfully executed jobs
}
