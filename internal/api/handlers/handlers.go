// handlers.go implements HTTP request handlers for the job orchestration API
// Manages job registration, execution, and status reporting
// Provides RESTful interface to the orchestrator
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/fawad-mazhar/simple-job-orchestrator/internal/orchestrator"
	"github.com/fawad-mazhar/simple-job-orchestrator/pkg/models"

	"github.com/go-chi/chi/v5"
)

// Handler contains dependencies for HTTP request handling
// Encapsulates the orchestrator for job management operations
type Handler struct {
	orch *orchestrator.Orchestrator // Reference to the orchestrator instance
}

// NewHandler creates a new Handler instance
// Initializes with reference to orchestrator for job operations
// Used by routing setup to create handler instance
func NewHandler(orch *orchestrator.Orchestrator) *Handler {
	return &Handler{orch: orch}
}

// HandleRegisterJobDefinition processes requests to register new job definitions
// POST /job-definitions
// Expects JSON body containing job definition
func (h *Handler) HandleRegisterJobDefinition(w http.ResponseWriter, r *http.Request) {
	// Parse the incoming job definition from request body
	// Unmarshal JSON into JobDefinition struct
	var jd models.JobDefinition
	if err := json.NewDecoder(r.Body).Decode(&jd); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Register the job definition with the orchestrator
	// Returns error if registration fails
	if err := h.orch.RegisterJobDefinition(&jd); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success response
	// HTTP 201 Created with confirmation message
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Job definition registered successfully",
	})
}

// HandleExecuteJob processes requests to execute a job
// POST /jobs/{id}/execute
// Takes optional JSON body with execution data
func (h *Handler) HandleExecuteJob(w http.ResponseWriter, r *http.Request) {
	// Extract job definition ID from URL parameters
	// Uses Chi router's URL parameter extraction
	definitionID := chi.URLParam(r, "id")

	// Parse optional execution data from request body
	// If no data provided, initialize empty map
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		data = make(map[string]interface{})
	}

	// Enqueue the job for execution
	// Returns execution ID for tracking
	executionID, err := h.orch.EnqueueJob(definitionID, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return execution ID in response
	// HTTP 202 Accepted as job is queued, not completed
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"executionID": executionID,
	})
}

// HandleGetJobState processes requests to get job execution state
// GET /jobs/{id}/state
// Returns current state of job execution
func (h *Handler) HandleGetJobState(w http.ResponseWriter, r *http.Request) {
	// Extract execution ID from URL parameters
	// Uses Chi router's URL parameter extraction
	executionID := chi.URLParam(r, "id")

	// Get current state of job execution
	// Returns error if job not found
	state, err := h.orch.GetJobExecutionState(executionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Return job state in response
	// Automatically serialized to JSON
	json.NewEncoder(w).Encode(state)
}

// HandleGetSystemState processes requests to get overall system state
// GET /system/state
// Returns state of all jobs and queue information
func (h *Handler) HandleGetSystemState(w http.ResponseWriter, r *http.Request) {
	// Get current state of entire system
	// Includes active and queued jobs
	state, err := h.orch.GetSystemState()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return system state in response
	// Automatically serialized to JSON
	json.NewEncoder(w).Encode(state)
}
