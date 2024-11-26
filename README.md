# Simple Job Orchestrator
---

Fawad Mazhar <fawadmazhar@hotmail.com> 2024

---
A robust Go-based job orchestration system that manages the execution of task sequences with persistent state handling and a RESTful API interface.

## Disclaimer

This project is **educational and demonstrative** in nature, inspired by AWS Step Functions. While it implements core workflow orchestration concepts, there are important considerations:

- **Single Instance Design**: Unlike AWS Step Functions, this is designed to run as a single unit and is not distributed
- **Educational Purpose**: Ideal for learning workflow orchestration concepts and Go implementation patterns
- **Limited Production Use**: While functional, it lacks many production-ready features like:
  - Distributed execution
  - Advanced error handling
  - Comprehensive monitoring
  - Authentication/Authorization
  - Advanced workflow patterns
  - UI dashboard

#### Ideal Use Cases
- Learning workflow orchestration concepts
- Prototyping workflow ideas
- Small-scale internal tools
- Base for building more robust solutions

#### Not Recommended For
- Mission-critical workflows
- High-availability requirements
- Large-scale deployments
- Multi-region operations

If you need a production-ready workflow service, consider:
- AWS Step Functions
- Apache Airflow
- Temporal
- Cadence

## Features

- **Job Orchestration**: Define and execute sequences of tasks with dependencies
- **Persistent Storage**: State management using BoltDB for reliability
- **Concurrent Execution**: Configurable worker pool for parallel job processing
- **Retry Mechanism**: Built-in exponential backoff retry for failed tasks
- **RESTful API**: HTTP interface for job management and monitoring
- **State Recovery**: Automatic recovery of interrupted jobs after system restart


#### Main Components:
```bash
cmd/server/main.go
- Server initialization
- Job/task registration
- API setup

internal/
├── api/
│   ├── handlers/   - HTTP request handlers
│   └── routes/     - API endpoint definitions
├── orchestrator/   - Core job execution logic
└── storage/        - BoltDB persistence layer
└── task_functions/ - Actual task implementations
```

#### Data Flow
- Job Definitions loaded from JSON files
- Tasks registered with orchestrator
- Jobs can be triggered via API
- Orchestrator manages execution
- State persisted in BoltDB
- Status available via API endpoints


#### Example Job
```json
{
  "id": "example-job",
  "name": "Example Job with Multiple Tasks",
  "tasks": [
    {
      "id": "task1",
      "name": "First Task",
      "maxRetry": 3,
      "functionName": "task1Function"
    },
    {
      "id": "task2",
      "name": "Second Task",
      "maxRetry": 2,
      "functionName": "task2Function"
    }
  ],
  "graph": {
      "task1": [],
      "task2": ["task1"],
    }
}
```
The `graph` represents the dependencies and orchestration order of a task. In this example, `task1` can start immediately, `task2` can start after `task1` is completed.

## Getting Started
```bash
# Clone the repository
git clone https://github.com/fawad1985/go-job-orchestrator.git

# Change to project directory
cd go-job-orchestrator

# Install dependencies
go mod download

# Build the project
go build -o cmd/server/jobs cmd/server/main.go

# Running the server
./cmd/server/jobs
# The server will start on port 8080 by default.
```

## API Endpoints
<details>
  <summary>Register Job Definition</summary>
  
  ```bash
  POST /job-definitions
  Content-Type: application/json

  {
    "id": "sample-job",
    "name": "Sample Job with Parallel Tasks executions.",
    "tasks": {
      "task1": {"id": "task1", "name": "First Task", "maxRetry": 3, "functionName": "task1Function"},
      "task2": {"id": "task2", "name": "Second Task", "maxRetry": 2, "functionName": "task2Function"},
      "task3": {"id": "task3", "name": "Third Task", "maxRetry": 1, "functionName": "task3Function"},
      "task4": {"id": "task4", "name": "Fourth Task", "maxRetry": 2, "functionName": "task4Function"}
    },
    "graph": {
      "task1": [],
      "task2": ["task1"],
      "task3": ["task1"],
      "task4": ["task2", "task3"],
    }
  }
  ```
  The `graph` represents the task dependencies and orchestration order. In this example, `task1` can start immediately, `task2` and `task3` can start after `task1` is completed, and `task4` can only start after both `task2` and `task3` are completed.
</details>

<details>
  <summary>Execute Job</summary>
  
  ```bash
  POST /jobs/{job-definition-id}/execute
  Content-Type: application/json

  {
    "param1": "value1",
    "param2": "value2"
  }
  ```
</details>

<details>
  <summary>Get Job State</summary>
  
  ```bash
  GET /jobs/{execution-id}/state
  ```
</details>

<details>
  <summary>Get System State</summary>
  
  ```bash
  GET /system/state
  ```
</details>

## Configuration
The system can be configured through the following parameters:

- Maximum concurrent jobs: Set in cmd/server/main.go
- Database path: Set in cmd/server/main.go
- HTTP port: Set in cmd/server/main.go

## Error Handling
The system implements comprehensive error handling:

- Task retry with exponential backoff
- Persistent state tracking
- Error reporting via API
- Transaction-based state updates

## License

This project is licensed under the MIT License. See the LICENSE file for more details.