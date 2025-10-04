# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Pinazu Flows is a Python library for orchestrating AI workflows using decorator-based definitions. It's part of the larger Pinazu Core microservices platform and provides a simple way to define and execute workflows with task dependencies, caching, and distributed execution support.

## Common Development Commands

### Development Setup
- `pip install -e .` - Install the library in development mode
- `python -m pytest test/` - Run all tests
- `python example/example_flow.py` - Run example sequential flow
- `python example/parallel_flow.py` - Run example parallel flow

### Building and Distribution
- `python -m build` - Build distribution packages (requires `build` package)
- `pip install .` - Install the library from source

## Library Architecture

### Core Components

#### Decorators (`src/pinazu_flows/decorators.py`)
- **@flow** - Marks a function as a workflow entry point. Only one @flow per Python file is allowed
- **@task** - Marks a function as a workflow task. Tasks can only be called within a @flow context
- Flow execution creates a FlowRunner instance with unique flow_run_id
- Task execution is managed through ThreadPoolExecutor for parallel processing

#### Runtime (`src/pinazu_flows/runtime.py`)
- **FlowRunner** - Core execution engine for flows, manages task scheduling and caching
- **TaskRegistry** - Global registry storing all decorated task functions
- **CacheManager** - S3-based caching system for task results using aiobotocore
- NATS integration for publishing flow/task status events

#### Models (`src/pinazu_flows/models.py`)
- **FlowStatus** - Enum for flow states (SCHEDULED, PENDING, RUNNING, SUCCESS, FAILED)
- **FlowRunStatusEvent/TaskRunStatusEvent** - Pydantic models for status messaging
- **CacheResult** - Model for cached task results with S3 cache keys
- **Event** - Base event model with headers, metadata, and tracing support

#### Logger (`src/pinazu_flows/logger.py`)
- **CustomLogger** - Colored console logging with JSON formatter option
- **CustomFormatter** - Color-coded log levels with timestamp formatting

### Key Dependencies
- **Python 3.10+** - Minimum required version
- **Pydantic 2.x** - Data validation and serialization
- **NATS** - Message broker for event publishing
- **aiobotocore/boto3** - AWS S3 integration for result caching
- **orjson** - Fast JSON serialization
- **python-json-logger** - Structured logging support

## Usage Patterns

### Basic Flow Definition
```python
from pinazu_flows import flow, task

@task
async def process_data(data: str) -> str:
    # Task logic here
    return processed_data

@flow
async def my_workflow(input_data: str) -> str:
    result = await process_data(input_data)
    return result
```

### Parallel Task Execution
Tasks can be executed in parallel using standard Python concurrency patterns like ThreadPoolExecutor or asyncio, as shown in `example/parallel_flow.py`.

### Flow Constraints
- Only one @flow decorator per Python file
- Tasks can only be called within a @flow context
- Each flow execution gets a unique flow_run_id (from FLOW_RUN_ID env var or auto-generated)
- Flow runners are stored in thread-safe registry during execution

## Environment Variables
- `FLOW_RUN_ID` - UUID for flow execution (auto-generated if not set)
- Standard AWS credentials for S3 caching functionality
- NATS connection settings for event publishing

## Testing
- Test files should be placed in `test/` directory
- Use pytest for running tests
- Examples in `example/` directory serve as integration tests
- Current test coverage is minimal (test_models.py is empty)

## Integration with Pinazu Core
This library publishes events to NATS topics:
- `v1.svc.worker.flow.status` - Flow status updates
- `v1.svc.worker.task.status` - Task status updates

Events follow the Pinazu Core event format with headers (user_id, thread_id, connection_id) and metadata (trace_id, timestamp).