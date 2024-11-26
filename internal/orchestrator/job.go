// job.go contains core job execution logic for the orchestrator
// It manages job lifecycle from enqueuing to completion
// Handles task execution, state management, and error handling
package orchestrator

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/fawad-mazhar/simple-job-orchestrator/pkg/models"
)

// EnqueueJob adds a new job to the execution queue
// It creates a new job execution instance and stores it in the database
// Returns the execution ID for tracking the job
func (o *Orchestrator) EnqueueJob(definitionID string, data map[string]interface{}) (string, error) {
	// Create a new job execution instance with unique ID and initial state
	// Uses timestamp-based ID for uniqueness and temporal tracking
	execution := &models.JobExecution{
		ID:           fmt.Sprintf("exec-%d", time.Now().UnixNano()),
		DefinitionID: definitionID,
		Status:       models.JobStatusQueued,
		StartTime:    time.Now(),
		Data:         data,
	}

	// Store the job execution in the database
	// This persists the initial state before queueing
	if err := o.db.StoreJobExecution(execution); err != nil {
		return "", err
	}

	// Add the job to the execution queue
	// Once queued, workers can pick it up for execution
	if err := o.db.EnqueueJob(execution.ID); err != nil {
		return "", err
	}

	return execution.ID, nil
}

// ExecuteJob runs a job and all its tasks in sequence
// Manages the complete lifecycle of a job execution
// Handles state transitions, task execution, and error cases
func (o *Orchestrator) ExecuteJob(ctx context.Context, executionID string) error {
	// Retrieve the job execution from the database
	je, err := o.db.GetJobExecution(executionID)
	if err != nil {
		return fmt.Errorf("failed to get job execution: %w", err)
	}

	// Retrieve the job definition associated with this execution
	jd, err := o.db.GetJobDefinition(je.DefinitionID)
	if err != nil {
		return fmt.Errorf("failed to get job definition: %w", err)
	}

	// Update the job status to running
	je.Status = models.JobStatusRunning
	if err := o.db.UpdateJobExecution(je); err != nil {
		return fmt.Errorf("failed to update job execution status to running: %w", err)
	}

	// Mark this job as ongoing in the orchestrator's internal state
	o.ongoingJobs.Store(executionID, struct{}{})
	// Ensure we remove the job from ongoing jobs when we're done
	defer o.ongoingJobs.Delete(executionID)

	// Initialize the task statuses if not already done
	if je.TaskStatuses == nil {
		je.TaskStatuses = make(map[string]models.TaskStatus)
	}

	// Create a WaitGroup to wait for all tasks to complete
	var wg sync.WaitGroup
	// Create a channel to communicate errors from goroutines
	errChan := make(chan error, len(jd.Tasks))

	// Create a map to track completed tasks
	completedTasks := make(map[string]bool)
	var completedTasksMutex sync.Mutex
	var jeUpdateMutex sync.Mutex

	// Function to check if all dependencies of a task are met
	dependenciesMet := func(taskID string) bool {
		completedTasksMutex.Lock()
		defer completedTasksMutex.Unlock()
		for _, dep := range jd.Graph[taskID] {
			if !completedTasks[dep] {
				return false
			}
		}
		return true
	}

	// Function to mark a task as completed
	markCompleted := func(taskID string) {
		completedTasksMutex.Lock()
		completedTasks[taskID] = true
		completedTasksMutex.Unlock()
	}

	// Iterate over all tasks in the job definition
	for taskID, task := range jd.Tasks {
		wg.Add(1)
		// Start a goroutine for each task
		go func(taskID string, task *models.Task) {
			defer wg.Done()

			// Wait for dependencies to be met before executing the task
			for !dependenciesMet(taskID) {
				select {
				case <-ctx.Done():
					// If the context is cancelled, report the error and return
					errChan <- ctx.Err()
					return
				case <-time.After(100 * time.Millisecond):
					// Check again after a short delay
				}
			}

			// Update task status to running
			jeUpdateMutex.Lock()
			je.TaskStatuses[taskID] = models.TaskStatusRunning
			o.db.UpdateJobExecution(je)
			jeUpdateMutex.Unlock()

			// Execute the task
			if err := o.executeTask(ctx, task, je.Data); err != nil {
				// If task execution fails, update status and report error
				jeUpdateMutex.Lock()
				je.TaskStatuses[taskID] = models.TaskStatusFailed
				o.db.UpdateJobExecution(je)
				jeUpdateMutex.Unlock()
				errChan <- fmt.Errorf("task %s failed: %w", taskID, err)
				return
			}

			// Task completed successfully
			jeUpdateMutex.Lock()
			je.TaskStatuses[taskID] = models.TaskStatusCompleted
			o.db.UpdateJobExecution(je)
			jeUpdateMutex.Unlock()
			markCompleted(taskID)
		}(taskID, task)
	}

	// Wait for all tasks to complete
	wg.Wait()
	// Close the error channel as no more errors will be sent
	close(errChan)

	// Check for any errors that occurred during task execution
	for err := range errChan {
		if err != nil {
			// If any task failed, mark the entire job as failed
			jeUpdateMutex.Lock()
			je.Status = models.JobStatusFailed
			o.db.UpdateJobExecution(je)
			jeUpdateMutex.Unlock()
			return err
		}
	}

	// All tasks completed successfully, mark job as completed
	jeUpdateMutex.Lock()
	je.Status = models.JobStatusCompleted
	je.EndTime = time.Now()
	if err := o.db.UpdateJobExecution(je); err != nil {
		jeUpdateMutex.Unlock()
		return fmt.Errorf("failed to update job execution status to completed: %w", err)
	}
	jeUpdateMutex.Unlock()

	// Increment the count of executed jobs
	if err := o.db.IncrementExecutedJobsCount(); err != nil {
		log.Printf("Failed to increment executed jobs count: %v", err)
	}

	return nil
}

// GetJobExecutionState retrieves the current state of a job execution
// Combines job execution state with task states for status reporting
// Returns a complete snapshot of job and task status
func (o *Orchestrator) GetJobExecutionState(executionID string) (*models.JobExecutionState, error) {
	// Get the current job execution state
	// Includes status, timing, and task states
	je, err := o.db.GetJobExecution(executionID)
	if err != nil {
		return nil, err
	}

	// Get the corresponding job definition
	// Used to include task metadata in state
	jd, err := o.db.GetJobDefinition(je.DefinitionID)
	if err != nil {
		return nil, err
	}

	// Create the state response structure
	// Combines execution state with job definition details
	state := &models.JobExecutionState{
		ID:           je.ID,
		DefinitionID: je.DefinitionID,
		Status:       je.Status,
		StartTime:    je.StartTime,
	}

	// Build task state list combining definition and execution state
	// Provides complete task execution progress
	for _, task := range jd.Tasks {
		taskState := models.TaskState{
			ID:     task.ID,
			Name:   task.Name,
			Status: je.TaskStatuses[task.ID],
		}
		state.Tasks = append(state.Tasks, taskState)
	}

	return state, nil
}
