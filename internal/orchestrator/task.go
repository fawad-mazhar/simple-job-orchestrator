// task.go handles task execution logic within the orchestrator
// Provides task registration, execution, and retry mechanisms
// Manages individual task lifecycle within jobs
package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/fawad-mazhar/simple-job-orchestrator/pkg/models"
)

// TaskFunction defines the interface for executable tasks
// Takes a context for cancellation and a data map for task parameters
// Returns an error if the task fails to execute
type TaskFunction func(ctx context.Context, data map[string]interface{}) error

// RegisterTaskFunction associates a function with a task ID
// Allows the orchestrator to look up and execute task implementations
// Must be called before a task can be executed
func (o *Orchestrator) RegisterTaskFunction(taskID string, fn TaskFunction) {
	o.taskFunctions[taskID] = fn
}

// executeTask runs a single task with retry logic
// Handles task execution, retries, and error reporting
// Implements exponential backoff between retry attempts
func (o *Orchestrator) executeTask(ctx context.Context, task *models.Task, data map[string]interface{}) error {
	// Look up the task implementation
	// Ensures the task has been properly registered
	fn, ok := o.taskFunctions[task.ID]
	if !ok {
		return fmt.Errorf("no function registered for task ID: %s", task.ID)
	}

	// Execute the task with configured number of retries
	// Uses exponential backoff between attempts
	for retries := 0; retries <= task.MaxRetry; retries++ {
		// Attempt to execute the task
		// Pass context and data to task implementation
		err := fn(ctx, data)

		// If successful, return immediately
		// No need for further retry attempts
		if err == nil {
			return nil
		}

		// If we've exhausted all retries, return final error
		// Includes retry count in error message
		if retries == task.MaxRetry {
			return fmt.Errorf("task %s failed after %d retries: %v", task.ID, task.MaxRetry, err)
		}

		// Exponential backoff between retries
		// Wait time doubles after each failure: 1s, 2s, 4s, 8s, etc.
		time.Sleep(time.Duration(1<<retries) * time.Second)
	}

	// This should never be reached due to return in retry loop
	// Included for completeness and to satisfy compiler
	return fmt.Errorf("task %s failed after %d retries", task.ID, task.MaxRetry)
}
