# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Pinazu Core is a microservices-based platform for orchestrating and managing AI Generative Application workflows. It provides a modular architecture with separate services for agents, API gateway, workflow orchestration, task management, and tool integration.

## Common Development Commands

### Building and Running
- `make prepare` - Install development tools (sqlc)
- `make build-run` - Build and run the application in one command (make build + make run)
- `make run` - Run the application in hot-reload mode
- `make build` - Complete build pipeline (frontend + database + API + core)
  - `make build-fe` - Frontend build only (SvelteKit)
  - `make build-api` - API and event contract generation
  - `make build-db` - Database code generation (SQLC)
  - `make build-core` - Go binary compilation only
- `make clean` - Clean build artifacts
- `make d-up` - Start development environment with Docker Compose
- `make d-down` - Stop development environment and delete all the databases

### Testing
- `make test` - Run all tests units + e2e integration. This will auto reset the docker environments and auto clean up the docker environements after test.

### Integration Testing
- `cd e2e && npm install` - Install integration test dependencies
- `cd e2e && npm run test:integration` - Run all Playwright integration tests

### Frontend Development (SvelteKit)
- `cd web && npm install` - Install frontend dependencies
- `cd web && npm run dev` - Start development server with hot reload
- `cd web && npm run build` - Build for production
- `cd web && npm run test` - Run end-to-end tests with Playwright (alias for test:e2e)

### Python Workflows Library
- Located in `python/pinazu-py/` directory
- Python 3.10+ required (as specified in pyproject.toml)
- Uses decorator-based workflow definitions (@flow and @task decorators)
- Example flows in `python/pinazu-py/example/`
- Development: `pip install -e .` in the python/pinazu-py directory
- Testing: `python -m pytest test/` in the python/pinazu-py directory

### Running Services
- `pinazu serve all` - Start all services (default configuration)
- `pinazu serve all -c configs/config.yaml` - Start with custom configuration
- `pinazu serve <service>` - Start individual services:
  - `pinazu serve agent` - AI agent invoke service
  - `pinazu serve api` - HTTP REST API gateway and WebSocket manager
  - `pinazu serve flows` - Workflow orchestration service
  - `pinazu serve tasks` - Agent Task lifecycle management service
  - `pinazu serve tools` - Tool execution orchestration service
  - `pinazu serve worker` - Python workflow execution engine
- `pinazu version` - Display application version information

### Database Operations
- `sqlc generate` - Generate Go code from SQL queries (run after modifying SQL files)
- SQL queries in `sql/queries/` are converted to Go code via SQLC
- Database models are automatically generated in `internal/db/`
- Migration files are in `sql/migrations/`
- Always run after modifying SQL queries
- Database migrations are handled automatically on service startup

### Code Generate Pipeline Overview
The `make build-api` command executes these steps in order:
1. **OpenAPI Generation**: `go run scripts/build-openapi.go`
2. **API Code Generation**: `go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=oapi-codegen.yaml api/openapi.yaml`
3. **Event Generation**: `go generate ./...`

#### OpenAPI Schema Generation
- **Individual Commands**:
  - `go run scripts/build-openapi.go` - Generate unified `api/openapi.yaml` from base.yaml, paths/, and schemas/
  - `go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=oapi-codegen.yaml api/openapi.yaml` - Generate Go API stubs in `internal/api/api.gen.go`
  - Always run after modifying schemas, paths or base.yaml OpenAPI files

#### Event Contract Generation
- **Individual Command**: `go generate ./...` - Generate event code from `api/events/` YAML files
- **Generated File**: `internal/service/events.go` (runs `go run ../../api/generate_event.go`)
- Event schemas in `api/events/` are converted to Go code for event-driven microservice communication
- Event contracts define type-safe message structures for NATS-based inter-service messaging
- Always run after modifying event YAML files

## Architecture Overview

### Core Services

#### API Gateway Service (`internal/api/`)
- **Primary Role**: HTTP REST API gateway and WebSocket endpoint manager
- **Pub/Sub Pattern**:
  - **Consumes**: `v1.svc.api.ws.response.*` (streaming AI responses), `v1.svc.api.ws.task.lifecycle.*` (task lifecycle events)
  - **Publishes**: `v1.svc.task.execute` (task execution requests from WebSocket clients)
- **Descriptions**:
  - The HTTP REST API code using Chi framework + auto-generated OpenAPI stubs.
- **Special Features**:
  - Automatic database migration on startup
  - Thread-safe WebSocket connection management with UUID tracking
  - Real-time bidirectional communication via WebSocket at `/v1/ws`
  - Server-Sent Events (SSE) Middleware with auto-flush for endpoint
  - Comprehensive CRUD operations for all entities
- **Key Handlers**: None (pure HTTP/WebSocket gateway)
- **Dependencies**:
  - PostgreSQL (database operations, migrations, SQLC-generated queries)
  - NATS (pub/sub messaging for inter-service communication)
  - Chi HTTP router (REST API routing and middleware)
  - WebSocket library (`github.com/coder/websocket` for WebSocket handling)
  - Service framework (service lifecycle management and configuration)
- **Workflows**:
  - **WebSocket Connection**: Client connects → Generate UUID → Store connection → Subscribe to user-specific NATS subjects → Handle bidirectional messaging
  - **WebSocket Message Processing**: Client sends message → Parse JSON → Validate format → Publish to `v1.svc.task.execute` → Return response to client
  - **NATS Message Handling**: Subscribe to `v1.svc.api.ws.response.*` && `v1.svc.api.ws.task.lifecycle.*` → Route messages to specific WebSocket connections → Forward to client
  - **SSE Endpoint**: Client connects → Set SSE headers → Auto-flush responses → Maintain persistent connection for real-time updates
  - **HTTP REST API**: Client request → Chi router → OpenAPI-generated handlers → Database operations via SQLC → JSON response
  - **Service Startup**: Load config → Connect to PostgreSQL && NATS → Run database migrations → Start HTTP server → Register WebSocket handlers

#### Agent Service (`internal/agents/`)
- **Primary Role**: AI model invocation and multi-provider response handling
- **Pub/Sub Pattern**:
  - **Consumes**: `v1.svc.agent.invoke` (agent invocation requests)
  - **Publishes**: `v1.svc.api.ws.response.*` (streaming AI responses to WebSocket clients), `v1.svc.task.finish` (task completion events), `v1.svc.tool.dispatch` (tool execution requests)
- **Descriptions**:
  - Direct AI model invocation service that handles requests by calling AI provider APIs and streaming responses back to clients with provider-specific message parsing.
- **Special Features**:
  - Multi-provider support with provider-specific message parsing (Anthropic, OpenAI, Google Gemini, AWS Bedrock)
  - Real-time streaming response handling with WebSocket integration
  - Dynamic agent configuration loading from database YAML specs
  - Provider-specific error handling and message transformation
  - AWS credential management with assume role support for Bedrock
  - Tool preparation with automatic tool fetching and dispatch coordination
- **Key Handlers**: `invokeEventCallback` (main agent invocation handler)
- **Dependencies**:
  - Multiple AI provider SDKs (Anthropic SDK, OpenAI SDK, Google Gemini SDK, AWS Bedrock SDK)
  - PostgreSQL (agent specs storage via SQLC queries)
  - NATS (pub/sub messaging for inter-service communication)
  - Service framework (service lifecycle management and configuration)
  - AWS SDK (Bedrock client and STS credential management)
  - YAML library (agent specification parsing)
- **Workflows**:
  - **Agent Invocation**: Receive `v1.svc.agent.invoke` → Parse event → Load agent specs from PostgreSQL → Route to provider-specific handler
  - **Provider-specific Processing**: Validate message format → Convert to provider format → Initialize AI client → Handle streaming or non-streaming responses
  - **Streaming Response**: Create API request → Stream response → Convert each event to WebSocket format → Publish to `v1.svc.api.ws.response.*` → Real-time delivery to WebSocket clients
  - **Task Completion**: Process final response → Determine stop reason → Publish `v1.svc.task.finish` (end_turn) or `v1.svc.tool.dispatch` (tool_use)
  - **Error Handling**: Catch errors at any step → Create error events → Publish error to both WebSocket && task completion channels
  - **Service Startup**: Load config → Initialize AI provider clients (Anthropic, OpenAI, Gemini, Bedrock) → Connect to PostgreSQL && NATS → Register event handlers
- **AI Providers Supported**:
  - Anthropic Claude (via Bedrock and direct API)
  - OpenAI GPT models
  - Google Gemini
  - AWS Bedrock Foundation Models

#### Flows Service (`internal/flows/`)
- **Primary Role**: Workflow orchestration and state management
- **Pub/Sub Pattern**:
  - **Consumes**: `v1.svc.flowrun.execute` (flow execution requests)
  - **Publishes**: `v1.svc.worker.flow.execute` (to Worker service)
  - **JetStream Consumers**: Flow status updates via `FLOWS_STATUS` stream
- **Special Features**:
  - Dual messaging pattern: NATS for requests, JetStream for status updates
  - Automatic flow run state tracking with database persistence
  - Request-response pattern for flow execution with immediate database record creation
  - JetStream stream management for reliable status processing

#### Tasks Service (`internal/tasks/`)
- **Primary Role**: Task lifecycle management and conversation thread coordination
- **Pub/Sub Pattern**:
  - **Consumes**: `v1.svc.task.execute` (task execution requests), `v1.svc.task.finish` (task completion events), `v1.svc.task.cancel` (task cancellation requests)
  - **Publishes**: `v1.svc.agent.invoke` (agent invocation requests), `v1.svc.api.ws.task.lifecycle.*` (task lifecycle events to WebSocket clients), `v1.svc.tool.gather` (tool result collection requests)
- **Descriptions**:
  - Task execution coordinator that manages conversation threads, task runs, message persistence, and coordinates agent-tool workflows with loop iteration controls.
- **Special Features**:
  - Concurrent database operations using goroutines and channels
  - Automatic thread creation for new conversations
  - Task loop management with max iteration limits (default: 20 loops)
  - Message history management and retrieval for agent context
  - Task lifecycle events (task_start, task_stop) for real-time client updates
  - Advanced error handling with task failure status tracking
- **Key Handlers**: `executeEventCallback` (main task execution handler), `finishEventCallback` (task completion handler), `cancelEventCallback` (task cancellation handler), `errorEventCallback` (error handling for failed tasks)
- **Dependencies**:
  - PostgreSQL (extensive SQLC queries for tasks, task runs, threads, messages)
  - NATS (pub/sub messaging for inter-service communication)
  - Service framework (service lifecycle management and configuration)
  - Anthropic SDK (message parsing in task completion handling)
- **Workflows**:
  - **Task Execute**: Receive `v1.svc.task.execute` → Parse event → Ensure thread exists (create if missing) → Insert user messages to database → Get sender-recipient message history → Manage task runs (create task/task runs, update loops, check max limits) → Publish task_start event → Publish `v1.svc.agent.invoke` with complete message history
  - **Task Finish**: Receive `v1.svc.task.finish` → Parse event → Create agent response message in database → Publish task_stop lifecycle event → Publish `v1.svc.tool.gather` to check for tool usage (Tools Service determines if task continues with tool execution or completes)
  - **Task Cancel**: Currently only logs cancellation events without full implementation
  - **Thread Management**: Check if ThreadID exists → Create new thread with user/connection-based title → Update request headers with new thread ID
  - **Task Loop Control**: Check current loops vs max loops → Update task run status (RUNNING or PAUSE) → Increment loop counter → Handle max loop reached scenarios
  - **Message Operations**: Insert user messages sequentially → Retrieve sender-recipient message history → Return complete conversation context
  - **Error Handling**: Catch task errors → Update task run status to FAILED → Log error details
  - **Service Startup**: Load config → Connect to PostgreSQL && NATS → Register event handlers → Start goroutine for graceful shutdown

#### Tools Service (`internal/tools/`)
- **Primary Role**: Tool execution orchestration and multi-type tool support
- **Pub/Sub Pattern**:
  - **Consumes**: `v1.svc.tool.dispatch`, `v1.svc.tool.gather`
  - **Publishes**: `v1.svc.tool.standalone.execute`, `v1.svc.flowrun.execute`, `v1.svc.task.execute` (for agent handoffs)
- **Three-Tier Tool Architecture**:
  - **Standalone Tools**: HTTP API-based tools with OpenAPI schema validation
  - **Workflow Tools**: Python-based tools executed via S3/MinIO code storage
  - **MCP Tools**: Model Context Protocol tools (schema ready, implementation pending)
- **Special Features**:
  - Recursive tool processing with `batch_tool` support
  - Multi-provider tool parsing (currently Anthropic-focused)
  - Agent-to-agent handoff capability via `invoke_tool_agent`
  - Hierarchical tool execution with parent-child relationships
  - MCP protocol support (stdio, gRPC, SSE protocols defined)

#### Worker Service (`internal/worker/`)
- **Primary Role**: Python workflow execution engine with process management
- **Pub/Sub Pattern**:
  - **JetStream Consumer**: `WORKER_FLOWS` stream for `v1.svc.worker.flow.execute`
  - **Publishes**: `v1.svc.worker.flow.status`, `v1.svc.worker.task.status`
- **Special Features**:
  - JetStream-based reliable message processing with delivery tracking
  - S3/MinIO integration for remote code execution
  - Local and remote Python code execution with process isolation
  - Comprehensive process lifecycle management with cleanup
  - Retry logic with configurable max delivery attempts

### Key Components

#### Service Infrastructure (`internal/service/`)
- **Common Service Framework**: Standardized service creation with NATS, PostgreSQL, and OpenTelemetry
- **Event System**: Type-safe event messaging with generic Event[T] wrapper and validation
- **JetStream Integration**: Stream and consumer management for reliable message processing
- **Performance Optimizations**:
  - pgx/v5 connection pooling for high-performance database access
  - Concurrent processing with goroutines and channels
  - NATS message batching for efficiency
  - Automatic resource cleanup and connection management
- **Features**:
  - Automatic service discovery and health monitoring
  - Distributed tracing with OpenTelemetry
  - Graceful shutdown handling with context cancellation
  - Thread-safe message handling with worker goroutines
  - Service statistics and monitoring endpoints
  - Custom colored logging with ANSI color support

#### Database Layer (`internal/db/`)
- **SQLC Integration**: Auto-generated type-safe database access layer from SQL queries
- **Custom Type Mappings**: JsonRaw for flexible JSON storage, UUID support, specialized enums
- **Migration Management**: Automatic database schema migrations on service startup
- **Performance Features**:
  - Pgx/v5 connection pooling for high-performance concurrent access
  - Optimized indexes for message history and workflow queries
  - JSONB columns for flexible data storage with efficient querying
  - UUID-based entity relationships for distributed system compatibility
- **Database Schema**: 755-line initialization migration with comprehensive entity definitions

#### Command Line Interface (`cli/`)
- **Service Management**: Start individual or all services with configuration support
- **Built with**: urfave/cli/v3 for robust command handling
- **Features**:
  - Service-specific configuration loading
  - Development and production environment support
  - Integrated testing commands

#### Frontend (`web/`)
- **Framework**: SvelteKit with TypeScript
- **Styling**: TailwindCSS v4.0 + Flowbite components
- **Advanced Features**:
  - Real-time WebSocket communication with connection management
  - Agent management interface with settings components
  - Workflow execution monitoring
  - User authentication flows (login, register, pending)
  - Admin and main application route groups
  - Voice language selection and response notifications
  - File upload capabilities with FilePond integration
  - GSAP animations for enhanced UX
  - Responsive design with dark/light theme support

#### Python Workflow Library (`python/pinazu-py/`)
- **Decorator-Based**: @flow and @task decorators for workflow definition
- **Runtime Engine**: FlowRunner with task registry and execution management
- **Advanced Architecture**:
  - Global registries for flows per module and active flow runners
  - Concurrent execution with ThreadPoolExecutor
  - Task context discovery through call stack inspection
  - One flow per file limitation (architectural constraint)
- **Features**:
  - Parallel and sequential task execution
  - Flow state management with parameter passing
  - Error handling and retry mechanisms
  - Result caching and persistence
  - NATS integration with WebSocket support

#### WebSocket Handler (`internal/api/websocket/`)
- **Real-time Communication**: Bidirectional WebSocket communication at `/v1/ws`
- **Connection Management**: Thread-safe UUID-based connection tracking
- **Message Routing**: User-specific message routing with NATS integration
- **Features**:
  - Automatic connection cleanup on disconnect
  - Message validation and error handling
  - Integration with task lifecycle events

### External Dependencies

#### NATS with JetStream
- **Core Messaging**: Pub/sub communication between all microservices
- **JetStream Streams**: Reliable message processing for critical workflows
- **Key Streams**:
  - `WORKER_FLOWS`: Flow execution events (persistent, work queue)
  - `FLOWS_STATUS`: Flow and task status updates (persistent, work queue)
- **Message Patterns**:
  - Regular NATS: Real-time events (task execution, agent invocation)
  - JetStream: Reliable processing (flow execution, status updates)
- **Delivery Guarantees**: At-least-once delivery with acknowledgment handling

#### PostgreSQL 17
- **Primary Database**: All persistent state and configuration storage
- **Key Features**:
  - UUID-based primary keys for distributed system compatibility
  - JSONB columns for flexible agent specs and message storage
  - Optimized indexes for message history and workflow queries
  - Automatic migrations on service startup

#### MinIO (S3-Compatible Storage)
- **Code Storage**: Remote Python workflow code execution
- **File Caching**: Workflow artifacts and intermediate results
- **Integration**: Automatic code download for Worker service execution

#### Docker Compose Environment
- **Services**: NATS (with JetStream), PostgreSQL 17, Grafana Alloy, MinIO
- **Networking**: Service discovery and communication
- **Development**: Hot-reload and debugging support

### Supported AI Providers
The agents service supports multiple AI providers:
- Anthropic Claude (via `internal/agents/anthropic.go`)
- OpenAI (via `internal/agents/openai.go`)
- Google Gemini (via `internal/agents/gemini.go`)
- AWS Bedrock (via `internal/agents/bedrock.go`)

## Database Schema
The application uses PostgreSQL with the following key entities:
- **Agents** - AI agent configurations with YAML specs
- **Users** - User management and authentication
- **Flows** - Workflow definitions with parameter schemas
- **Flow Runs** - Workflow execution instances with status tracking
- **Flow Task Runs** - Individual task executions within workflows
- **Threads** - Conversation threads
- **Messages** - Chat messages and interactions
- **Tool Runs** - Tool execution records and results
- **Tools** - Tool definitions and configurations
- **Worker Heartbeats** - Worker service health monitoring
- **Roles & Permissions** - Authorization and access control

## Configuration
- Default configuration: `configs/config.yaml`
- Environment variables can override configuration values
- Services support both YAML configuration files and CLI flags
- Database connection details, NATS URL, HTTP port, and tracing configuration are configurable

## Testing Patterns
- Unit tests are co-located with source files (`*_test.go`)
- Integration tests may require external dependencies (PostgreSQL, NATS)
- Test scripts available in `scripts/` directory
- Use `scripts/test-agents-handler-service.sh` for manual agent testing

## Development Workflow

### Standard Code Changes
1. Modify code or SQL queries
2. Run `sqlc generate` if SQL files were changed
3. Run `make test` to ensure tests pass
5. Individual services can be tested with `pinazu serve <service>`

### Adding New Event Contracts
1. **Define Event Schema**: Create or modify YAML files in `api/events/` directory:
   - Define event name, type (consumer or request_response), and NATS subject
   - Specify message fields with Go types and optional validation
   - Add custom validation logic if needed
2. **Generate Event Code**: Run `make build-api` to generate Go event structures
3. **Implement Event Handlers**: Use generated event types in service implementations
4. **Register Handlers**: Register event handlers with the service using generated subject constants
5. **Test**: Verify event flow between microservices

### Adding New API Endpoints
1. **Update OpenAPI Specification**: Modify files in `api/` directory (DO NOT edit `openapi.yaml` directly):
   - Add new endpoints to `api/paths/` YAML files
   - Add new schemas to `api/schemas/` YAML files
   - Update `api/base.yaml` for general API configuration if needed
   - The `api/openapi.yaml` file is auto-generated - do not modify it manually
2. **Generate API Code**: Run `make build-api` to:
   - First generate `api/openapi.yaml` from base.yaml, paths/, and schemas/ using `scripts/build-openapi.go`
   - Then generate Go stubs in `internal/api/api.gen.go` from the generated openapi.yaml
3. **Add Database Queries**: If new database operations are needed:
   - Add SQL queries to appropriate files in `sql/queries/`
   - Run `sqlc generate` to generate Go database code
4. **Implement Handlers**: Manually implement the endpoint logic in `internal/api/impl.go`
   - The generated `api.gen.go` provides the interface stubs
   - Your implementation in `impl.go` should satisfy these interfaces
   - Use the generated SQLC code for database operations
5. **Update Tests**: Add tests for new endpoints
6. **Test**: Run `make test` and verify functionality with `make d-up`

## Real-Time Communication

### WebSocket Implementation

#### WebSocket Endpoint
- **URL**: `/v1/ws`
- **Library**: Uses `github.com/coder/websocket` for robust WebSocket handling
- **Protocol**: Text-based JSON message format

### Server-Sent Events (SSE) Infrastructure
- **Custom Middleware**: `internal/api/middleware/sse_flush.go` for streaming optimization
- **Auto-flush Capability**: Automatic buffer flushing for real-time data delivery
- **Nginx Compatibility**: Headers to disable Nginx buffering for proper streaming
- **Connection Management**: HTTP connection hijacking support for persistent streams
- **Use Cases**: Real-time agent responses, workflow status updates, tool execution streaming

### Message Flow
1. Client connects to WebSocket endpoint (`/v1/ws`)
2. Connection ID is generated and stored in thread-safe map
3. Client sends JSON message with `agent_id`, `thread_id`, and `messages`
4. WebSocket handler validates and publishes to NATS `TaskExecuteEventSubject`
5. Agent responses are routed back via `WebsocketResponseEventSubject`
6. Responses are delivered to original client connection

### Message Format
```json
{
  "agent_id": "uuid",
  "thread_id": "uuid", 
  "messages": [
    {"role": "user", "content": "message"}
  ]
}
```

### Key Files
- **Handler**: `internal/api/websocket/handler.go`
- **Tests**: `internal/api/websocket/handler_test.go`
- **Events**: `internal/api/websocket/events.go` (WebsocketResponseEventMessage)
- **Router**: WebSocket endpoint registered in API service

### Connection Management
- Each WebSocket connection gets unique UUID identifier
- Connections stored in thread-safe `SyncMap` for concurrent access
- Automatic cleanup on client disconnect prevents memory leaks
- Connection health monitored via ping/pong mechanism

## Go Version
- **Go Version**: 1.24 (as specified in go.mod)

## Development Environment
Docker Compose provides the following services:
- **NATS Server** - Message broker with JetStream enabled (ports 4222, 6222, 8222)
- **PostgreSQL 17** - Database server (port 5432)
- **Grafana Alloy** - OpenTelemetry collector (port 4317)
- **MinIO** - S3-compatible storage (ports 9000, 9001)

## Event Contract Generation System

### Event Schema Structure
Event contracts are defined in YAML files located in `api/events/` directory:
- **agents.yaml** - Agent invocation events
- **flows.yaml** - Workflow orchestration events  
- **tasks.yaml** - Task execution events
- **tools.yaml** - Tool dispatch and gathering events
- **websocket.yaml** - WebSocket response events

### Event Schema Format
```yaml
events:
  - name: EventName
    type: consumer | request_response
    description: "Event description"
    subject: "v1.svc.domain.operation"
    messageFields:
      - name: FieldName
        type: Go type (e.g., uuid.UUID, string, []db.JsonRaw)
        import: "import/path" # optional
        description: "Field description" # optional
        optional: true # optional
    customValidation: |
      // Custom Go validation code
    responseFields: [] # For request_response type only
```

### Generated Event Code
- **Generated File**: `internal/service/events.go` (do not edit manually)
- **Generated Content**:
  - Event subject constants (e.g., `TaskExecuteEventSubject`)
  - Type-safe event message structs
  - Interface implementation methods (`Subject()`, `Validate()`)
  - Request-response structs for bidirectional events
- **Command**: Run `make build-api` to regenerate event contracts

### Event Interface Implementation
All generated events implement the `EventMessage` interface from `internal/service/types.go`:
```go
type EventMessage interface {
    Validate() error
    Subject() EventSubject
}
```

## OpenAPI Code Generation
- **Source Files** (user editable):
  - `api/base.yaml` - Base OpenAPI configuration
  - `api/paths/*.yaml` - Endpoint definitions
  - `api/schemas/*.yaml` - Data models
- **Generated Files** (do not edit manually):
  - `api/openapi.yaml` - Complete OpenAPI spec (generated from base.yaml + paths/ + schemas/)
  - `internal/api/api.gen.go` - Go interface stubs and types
- **Manual Implementation**: `internal/api/impl.go` (your endpoint handlers)
- **Code Generation Config**: `oapi-codegen.yaml`
- **Command**: Run `make build-api` to regenerate both `openapi.yaml` and Go code

**Important**: 
1. Only modify files in `api/base.yaml`, `api/paths/`, and `api/schemas/` - never edit `api/openapi.yaml` directly
2. Only modify event schema files in `api/events/` - never edit `internal/service/events.go` directly
3. The `make build-api` command generates both OpenAPI stubs and event contracts
4. You must manually implement the actual endpoint logic in `impl.go` to satisfy the generated interfaces

## Service Startup and Orchestration

### Service Dependencies
- **Database Migration**: API Gateway runs migrations before other services start
- **NATS Connection**: All services require NATS connectivity for inter-service communication
- **JetStream Setup**: Flows and Worker services create required streams and consumers
- **Configuration**: Services load configuration from YAML files with environment overrides

### Graceful Shutdown
- **Context Cancellation**: Services listen for context cancellation signals
- **Resource Cleanup**: Proper cleanup of NATS subscriptions, database connections, HTTP servers
- **Message Draining**: NATS message draining to prevent message loss
- **Tracing Shutdown**: OpenTelemetry tracer cleanup with timeout

### Common Development Workflows

#### Adding New Event Types
1. Define event schema in `api/events/{service}.yaml`
2. Run `make build-api` to generate Go event types
3. Implement event handlers in appropriate service
4. Register event handlers with service using generated subject constants
5. Test event flow end-to-end

#### Adding New Tool Types
1. Add tool definition to database via API or migration
2. Implement tool execution logic in Tools service
3. Add tool dispatch handling for specific provider message formats
4. Test tool integration with agent workflows

#### Implementing New AI Providers
1. Add provider-specific client initialization in Agent service
2. Implement message parsing for provider's message format
3. Add provider-specific streaming response handling
4. Update agent specifications to support new provider
5. Test end-to-end agent invocation and response flow