// orchestrator.go defines the main orchestrator structure and its core operations
// It manages job execution, worker pools, and system state
// Acts as the central coordinator for the job processing system
package orchestrator

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/fawad-mazhar/simple-job-orchestrator/internal/storage"
	"github.com/fawad-mazhar/simple-job-orchestrator/pkg/models"
)

// Orchestrator manages the complete job execution system
// Controls worker pools, maintains job state, and coordinates task execution
// Provides thread-safe operation for concurrent job processing
type Orchestrator struct {
	db            storage.DB              // Persistent storage interface
	workerPool    chan struct{}           // Limits concurrent job executions
	ongoingJobs   sync.Map                // Tracks currently executing jobs
	taskFunctions map[string]TaskFunction // Maps task IDs to their implementations
	maxConcurrent int                     // Maximum number of concurrent jobs
	stop          chan struct{}           // Signal to stop processing
	done          chan struct{}           // Signal that processing has stopped
}

// New creates and initializes a new Orchestrator instance
// Sets up the worker pool and recovers any interrupted jobs
// Starts the job queue processing loop
func New(db storage.DB, maxConcurrent int) (*Orchestrator, error) {
	// Initialize orchestrator with configuration and channels
	// Creates worker pool and task function registry
	o := &Orchestrator{
		db:            db,
		workerPool:    make(chan struct{}, maxConcurrent),
		taskFunctions: make(map[string]TaskFunction),
		maxConcurrent: maxConcurrent,
		stop:          make(chan struct{}),
		done:          make(chan struct{}),
	}

	// Recover state from previous runs
	// Ensures jobs interrupted by shutdown are properly handled
	if err := o.recoverState(); err != nil {
		return nil, fmt.Errorf("failed to recover state: %v", err)
	}

	// Start the queue processing loop
	// Begins processing jobs in background
	go o.processQueue()

	return o, nil
}

// recoverState restores any running jobs from the last shutdown
// Prevents job loss during system restarts
// Re-queues previously running jobs for execution
func (o *Orchestrator) recoverState() error {
	// Get list of jobs that were running during last shutdown
	// These jobs need to be recovered and restarted
	runningJobs, err := o.db.GetRunningJobs()
	if err != nil {
		return err
	}

	// Restart each previously running job
	// Jobs are tracked and executed in new goroutines
	for _, jobID := range runningJobs {
		o.ongoingJobs.Store(jobID, struct{}{})
		go o.ExecuteJob(context.Background(), jobID)
	}

	return nil
}

// processQueue continuously processes jobs from the queue
// Manages worker allocation and job execution
// Runs until explicitly stopped
func (o *Orchestrator) processQueue() {
	defer close(o.done) // Signal when queue processing stops

	for {
		select {
		case <-o.stop:
			// Received shutdown signal
			// Stop processing new jobs
			return

		default:
			// Attempt to dequeue next job
			// If queue is empty, wait before retrying
			jobID, err := o.db.DequeueJob()
			if err != nil {
				if err.Error() == "queue is empty" {
					time.Sleep(time.Second)
					continue
				}
				log.Printf("Error dequeuing job: %v", err)
				continue
			}

			// Acquire worker slot from pool
			// Ensures we don't exceed max concurrent jobs
			o.workerPool <- struct{}{}

			// Execute job in new goroutine
			// Worker slot is released after completion
			go func(id string) {
				defer func() { <-o.workerPool }() // Release worker
				if err := o.ExecuteJob(context.Background(), id); err != nil {
					log.Printf("Error executing job %s: %v", id, err)
				}
			}(jobID)
		}
	}
}

// Close gracefully shuts down the orchestrator
// Stops queue processing and waits for completion
// Ensures clean shutdown of database connection
func (o *Orchestrator) Close() error {
	// Signal queue processor to stop
	o.stop <- struct{}{}

	// Wait for queue processor to finish
	<-o.done

	// Close database connection
	return o.db.Close()
}

// RegisterJobDefinition adds a new job definition to the system
// Stores the definition for future execution
// Enables jobs to be executed using this definition
func (o *Orchestrator) RegisterJobDefinition(jd *models.JobDefinition) error {
	return o.db.StoreJobDefinition(jd)
}

// GetSystemState retrieves the current state of the entire system
// Provides overview of active and queued jobs
// Used for monitoring and debugging
func (o *Orchestrator) GetSystemState() (*models.SystemState, error) {
	state := &models.SystemState{}

	// Collect state of all actively running jobs
	// Includes current status and progress
	o.ongoingJobs.Range(func(key, value interface{}) bool {
		jobID := key.(string)
		jobState, err := o.GetJobExecutionState(jobID)
		if err == nil {
			state.ActiveJobs = append(state.ActiveJobs, *jobState)
		}
		return true
	})

	// Get list of jobs waiting in queue
	// Shows pending work
	queuedJobs, err := o.db.GetQueuedJobs()
	if err != nil {
		return nil, err
	}
	state.QueuedJobs = queuedJobs

	// Get total count of queued jobs
	// Provides queue depth information
	queuedCount, err := o.db.GetQueuedJobCount()
	if err != nil {
		return nil, err
	}
	state.QueuedCount = queuedCount

	// Get total count of successfully executed jobs
	executedCount, err := o.db.GetExecutedJobsCount()
	if err != nil {
		return nil, err
	}
	state.ExecutedJobs = executedCount

	return state, nil
}
