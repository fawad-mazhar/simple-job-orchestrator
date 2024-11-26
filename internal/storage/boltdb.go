// boltdb.go implements persistent storage using BoltDB
// Manages job definitions, executions, and the job queue
// Provides atomic operations for job state management
package storage

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fawad-mazhar/simple-job-orchestrator/pkg/models"

	"go.etcd.io/bbolt"
)

// Bucket names for different types of data in BoltDB
// Separates job definitions, executions, and queue data
const (
	jobDefinitionsBucket = "job_definitions"
	jobExecutionsBucket  = "job_executions"
	queueBucket          = "queue"
	statsBucket          = "stats"
)

// DB interface defines all storage operations
// Abstracts storage implementation details from the rest of the system
// Enables potential future support for different storage backends
type DB interface {
	StoreJobDefinition(jd *models.JobDefinition) error
	GetJobDefinition(id string) (*models.JobDefinition, error)
	GetRunningJobs() ([]string, error)
	StoreJobExecution(je *models.JobExecution) error
	GetJobExecution(id string) (*models.JobExecution, error)
	UpdateJobExecution(je *models.JobExecution) error
	GetQueuedJobs() ([]string, error)
	EnqueueJob(jobID string) error
	DequeueJob() (string, error)
	GetQueuedJobCount() (int, error)
	RemoveFromQueue(jobID string) error
	IncrementExecutedJobsCount() error
	GetExecutedJobsCount() (int, error)
	Close() error
}

// BoltDB implements the DB interface using BoltDB
// Provides persistent, transactional storage
// Handles all database operations
type BoltDB struct {
	db *bbolt.DB // Underlying BoltDB instance
}

// NewBoltDB creates and initializes a new BoltDB instance
// Creates required buckets if they don't exist
// Returns initialized database connection
func NewBoltDB(path string) (*BoltDB, error) {
	// Open BoltDB with a 1-second timeout
	// Creates the database file if it doesn't exist
	db, err := bbolt.Open(path, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("could not open db, %v", err)
	}

	// Create required buckets in a single transaction
	// Ensures database is properly initialized
	err = db.Update(func(tx *bbolt.Tx) error {
		buckets := []string{jobDefinitionsBucket, jobExecutionsBucket, queueBucket, statsBucket}
		for _, bucket := range buckets {
			_, err := tx.CreateBucketIfNotExists([]byte(bucket))
			if err != nil {
				return fmt.Errorf("could not create %s bucket: %v", bucket, err)
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("could not set up buckets, %v", err)
	}

	return &BoltDB{db: db}, nil
}

// StoreJobDefinition saves a job definition to the database
// Uses JSON serialization for storage
// Operates in a single transaction
func (b *BoltDB) StoreJobDefinition(jd *models.JobDefinition) error {
	return b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(jobDefinitionsBucket))
		buf, err := json.Marshal(jd)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(jd.ID), buf)
	})
}

// GetJobDefinition retrieves a job definition by ID
// Deserializes JSON data into JobDefinition struct
// Returns error if definition not found
func (b *BoltDB) GetJobDefinition(id string) (*models.JobDefinition, error) {
	var jd models.JobDefinition
	err := b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(jobDefinitionsBucket))
		v := bucket.Get([]byte(id))
		if v == nil {
			return fmt.Errorf("job definition not found")
		}
		return json.Unmarshal(v, &jd)
	})
	if err != nil {
		return nil, err
	}
	return &jd, nil
}

// GetRunningJobs returns IDs of all currently running jobs
// Scans job executions bucket for jobs in RUNNING state
// Used for state recovery after system restart
func (b *BoltDB) GetRunningJobs() ([]string, error) {
	var runningJobs []string
	err := b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(jobExecutionsBucket))
		return bucket.ForEach(func(k, v []byte) error {
			var je models.JobExecution
			if err := json.Unmarshal(v, &je); err != nil {
				return err
			}
			if je.Status == models.JobStatusRunning {
				runningJobs = append(runningJobs, je.ID)
			}
			return nil
		})
	})
	return runningJobs, err
}

// StoreJobExecution saves a job execution instance
// Handles both new executions and updates
// Uses JSON serialization
func (b *BoltDB) StoreJobExecution(je *models.JobExecution) error {
	return b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(jobExecutionsBucket))
		buf, err := json.Marshal(je)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(je.ID), buf)
	})
}

// GetJobExecution retrieves job execution details by ID
// Deserializes stored JSON into JobExecution struct
// Returns error if execution not found
func (b *BoltDB) GetJobExecution(id string) (*models.JobExecution, error) {
	var je models.JobExecution
	err := b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(jobExecutionsBucket))
		v := bucket.Get([]byte(id))
		if v == nil {
			return fmt.Errorf("job execution not found")
		}
		return json.Unmarshal(v, &je)
	})
	if err != nil {
		return nil, err
	}
	return &je, nil
}

// UpdateJobExecution updates an existing job execution
// Wraps StoreJobExecution as BoltDB uses same operation for create/update
func (b *BoltDB) UpdateJobExecution(je *models.JobExecution) error {
	return b.StoreJobExecution(je)
}

// GetQueuedJobs returns list of all jobs in the queue
// Used for system state reporting
// Returns job IDs in queue order
func (b *BoltDB) GetQueuedJobs() ([]string, error) {
	var queuedJobs []string
	err := b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(queueBucket))
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(k, v []byte) error {
			queuedJobs = append(queuedJobs, string(k))
			return nil
		})
	})
	return queuedJobs, err
}

// Close closes the database connection
// Should be called when shutting down the system
func (b *BoltDB) Close() error {
	return b.db.Close()
}

// EnqueueJob adds a job to the execution queue
// Uses job ID as key in queue bucket
// Simple implementation with no priority ordering
func (b *BoltDB) EnqueueJob(jobID string) error {
	return b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(queueBucket))
		if bucket == nil {
			return fmt.Errorf("queue bucket not found")
		}
		return bucket.Put([]byte(jobID), []byte{})
	})
}

// DequeueJob removes and returns the next job from the queue
// Uses FIFO ordering based on bucket iteration
// Returns error if queue is empty
func (b *BoltDB) DequeueJob() (string, error) {
	var jobID string
	err := b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(queueBucket))
		if bucket == nil {
			return fmt.Errorf("queue bucket not found")
		}
		cursor := bucket.Cursor()
		k, _ := cursor.First()
		if k == nil {
			return fmt.Errorf("queue is empty")
		}
		jobID = string(k)
		return bucket.Delete(k)
	})
	return jobID, err
}

// GetQueuedJobCount returns the number of jobs in queue
// Uses BoltDB bucket stats for efficient counting
func (b *BoltDB) GetQueuedJobCount() (int, error) {
	var count int
	err := b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(queueBucket))
		if bucket == nil {
			return nil
		}
		count = bucket.Stats().KeyN
		return nil
	})
	return count, err
}

// RemoveFromQueue removes a specific job from the queue
// Used when job execution completes or fails
func (b *BoltDB) RemoveFromQueue(jobID string) error {
	return b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(queueBucket))
		if bucket == nil {
			return fmt.Errorf("queue bucket not found")
		}
		return bucket.Delete([]byte(jobID))
	})
}

const executedJobsCountKey = "executed_jobs_count"

// Increment executed jobs count
// Used within orchestrator
func (b *BoltDB) IncrementExecutedJobsCount() error {
	return b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(statsBucket))
		if bucket == nil {
			return fmt.Errorf("stats bucket not found")
		}

		var count uint64 = 0
		existing := bucket.Get([]byte(executedJobsCountKey))
		if existing != nil {
			count = binary.BigEndian.Uint64(existing)
		}

		count++
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, count)
		return bucket.Put([]byte(executedJobsCountKey), buf)
	})
}

// Get executed jobs count
// Used by system stats
func (b *BoltDB) GetExecutedJobsCount() (int, error) {
	var count uint64
	err := b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(statsBucket))
		if bucket == nil {
			return nil // If bucket doesn't exist, count is 0
		}
		data := bucket.Get([]byte(executedJobsCountKey))
		if data == nil {
			return nil // If key doesn't exist, count is 0
		}
		count = binary.BigEndian.Uint64(data)
		return nil
	})
	return int(count), err
}
