// routes.go defines the API routing configuration for the job orchestration system
// Sets up HTTP endpoints and maps them to appropriate handlers
// Provides the RESTful API structure for job management
package routes

import (
	"github.com/fawad-mazhar/simple-job-orchestrator/internal/api/handlers"
	"github.com/fawad-mazhar/simple-job-orchestrator/internal/orchestrator"

	"github.com/go-chi/chi/v5"
)

// SetupRoutes configures all API routes for the application
// Takes a router instance and orchestrator reference
// Maps URLs to their corresponding handler functions
func SetupRoutes(r chi.Router, orch *orchestrator.Orchestrator) {
	// Create new handler instance with orchestrator reference
	// Handlers need orchestrator to perform job operations
	h := handlers.NewHandler(orch)

	// Register Job Definitions
	// POST /job-definitions
	// Used to create new job templates in the system
	r.Post("/job-definitions", h.HandleRegisterJobDefinition)

	// Execute Job
	// POST /jobs/{id}/execute
	// Triggers execution of a specific job definition
	r.Post("/jobs/{id}/execute", h.HandleExecuteJob)

	// Get Job State
	// GET /jobs/{id}/state
	// Retrieves current state of a job execution
	r.Get("/jobs/{id}/state", h.HandleGetJobState)

	// Get System State
	// GET /system/state
	// Retrieves overall system status
	r.Get("/system/state", h.HandleGetSystemState)
}

/* API Routes Overview:

1. Job Definition Management:
  - POST /job-definitions
  - Creates reusable job templates
  - Accepts: JSON job definition
  - Returns: Success confirmation

2. Job Execution:
  - POST /jobs/{id}/execute
  - Starts job execution
  - URL Param: job definition ID
  - Accepts: Optional JSON data
  - Returns: Execution ID

3. Job State Monitoring:
  - GET /jobs/{id}/state
  - Checks job execution progress
  - URL Param: execution ID
  - Returns: Current job state

4. System Monitoring:
  - GET /system/state
  - Checks overall system status
  - Returns: Active and queued jobs

Future Route Considerations:
- GET /job-definitions - List all job definitions
- DELETE /job-definitions/{id} - Remove job definition
- POST /jobs/{id}/cancel - Cancel running job
- GET /jobs - List all job executions
*/
