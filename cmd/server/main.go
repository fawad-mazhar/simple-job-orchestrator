// main.go is the entry point of the job orchestration system
// It initializes the database, orchestrator, and HTTP server
// Also handles loading of task functions and job definitions from files

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/fawad-mazhar/simple-job-orchestrator/internal/api/routes"
	"github.com/fawad-mazhar/simple-job-orchestrator/internal/orchestrator"
	"github.com/fawad-mazhar/simple-job-orchestrator/internal/storage"
	"github.com/fawad-mazhar/simple-job-orchestrator/internal/task_functions"
	"github.com/fawad-mazhar/simple-job-orchestrator/pkg/models"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// Initialize BoltDB storage layer with a local file "jobs.db"
	// This database will store job definitions, executions, and queue state
	db, err := storage.NewBoltDB("jobs.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create a new orchestrator instance with 10 concurrent job slots
	// The orchestrator manages job execution and task scheduling
	orch, err := orchestrator.New(db, 10)
	if err != nil {
		log.Fatalf("Failed to initialize orchestrator: %v", err)
	}

	// Load all available task functions dynamically using reflection
	// These functions will be matched with task definitions in jobs
	taskFunctions, err := loadTaskFunctions()
	if err != nil {
		log.Fatalf("Failed to load task functions: %v", err)
	}

	// Load job definitions from JSON files and register them with the orchestrator
	// Also registers corresponding task functions for each task in the jobs
	if err := loadJobDefinitions(orch, taskFunctions); err != nil {
		log.Fatalf("Failed to load job definitions: %v", err)
	}

	// Set up the Chi router with standard middleware
	// Provides logging and panic recovery for the HTTP server
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Configure all API routes for the application
	// Routes are defined in the routes package
	routes.SetupRoutes(r, orch)

	// Start the HTTP server on port 8080
	// This provides the REST API for job management
	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// loadTaskFunctions discovers and loads task functions using reflection
// It examines the task_functions package for compatible method signatures
// Returns a map of function names to their implementations
func loadTaskFunctions() (map[string]orchestrator.TaskFunction, error) {
	taskFunctions := make(map[string]orchestrator.TaskFunction)

	// Get the type information for the TaskFunctions interface
	// This is used to find all available task function implementations
	pkgType := reflect.TypeOf((*task_functions.TaskFunctions)(nil)).Elem()

	// Iterate through all methods in the interface
	// Check each method for compatibility with the TaskFunction signature
	for i := 0; i < pkgType.NumMethod(); i++ {
		method := pkgType.Method(i)

		// Verify the method signature matches our TaskFunction type:
		// - Takes context.Context and map[string]interface{}
		// - Returns error
		if method.Type.NumIn() == 2 &&
			method.Type.In(0).String() == "context.Context" &&
			method.Type.In(1).String() == "map[string]interface {}" &&
			method.Type.NumOut() == 1 &&
			method.Type.Out(0).String() == "error" {

			// Convert method name to expected function name in job definitions
			// For example: "Process" becomes "processFunction"
			functionName := fmt.Sprintf("%sFunction", strings.ToLower(method.Name))

			// Get the actual function implementation
			fn := reflect.ValueOf(task_functions.GetTaskFunction(method.Name)).Interface().(func(context.Context, map[string]interface{}) error)

			// Store the function in our map
			taskFunctions[functionName] = fn
			fmt.Printf("Loaded task function: %s\n", functionName)
		}
	}

	// Ensure at least one task function was loaded
	if len(taskFunctions) == 0 {
		return nil, fmt.Errorf("no task functions found")
	}

	return taskFunctions, nil
}

// loadJobDefinitions reads and registers job definitions from JSON files
// It loads files from the job_definitions directory and validates them
// Also associates task functions with each task in the job definitions
func loadJobDefinitions(orch *orchestrator.Orchestrator, taskFunctions map[string]orchestrator.TaskFunction) error {
	// Read all files from the job definitions directory
	jobDefsDir := "job_definitions"
	files, err := os.ReadDir(jobDefsDir)
	if err != nil {
		return err
	}

	// Process each JSON file in the directory
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		// Read and parse the job definition file
		filePath := filepath.Join(jobDefsDir, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}

		// Unmarshal JSON into a JobDefinition struct
		var jobDef models.JobDefinition
		if err := json.Unmarshal(data, &jobDef); err != nil {
			return err
		}

		// Register the job definition with the orchestrator
		if err := orch.RegisterJobDefinition(&jobDef); err != nil {
			return err
		}

		// Register task functions for each task in the job
		// Ensures all required functions exist for the job's tasks
		for _, task := range jobDef.Tasks {
			fn, ok := taskFunctions[task.FunctionName]
			if !ok {
				return fmt.Errorf("no function found for task %s in job %s", task.ID, jobDef.ID)
			}
			orch.RegisterTaskFunction(task.ID, fn)
		}

		log.Printf("Loaded job definition: %s", jobDef.ID)
	}

	return nil
}
