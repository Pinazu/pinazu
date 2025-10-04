-- +goose Up
-- =============================================
-- WORKFLOW ORCHESTRATION TABLES (MERGED)
-- =============================================

-- Update flows table to match PDF specification
-- TASKS TABLE REMOVED: Code will auto-generate tasks, flow_task_runs with 
-- composite PK (flow_run_id, task_name) is sufficient for tracking execution

-- Create flow_runs table for flow run instances
CREATE TABLE IF NOT EXISTS flow_runs (
    flow_run_id UUID PRIMARY KEY,
    flow_id UUID NOT NULL REFERENCES flows(id) ON DELETE CASCADE,
    parameters JSONB,
    status VARCHAR(50) NOT NULL DEFAULT 'SCHEDULED' CHECK (status IN ('SCHEDULED', 'PENDING', 'RUNNING', 'SUCCESS', 'FAILED')),
    engine VARCHAR(50) NOT NULL DEFAULT 'process',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    finished_at TIMESTAMP WITH TIME ZONE,
    task_statuses JSONB DEFAULT '{}', -- Object mapping task names to their statuses
    success_task_results JSONB DEFAULT '{}', -- Object mapping task names to their result cache keys
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3
);

-- Add indexes for flow_runs table
CREATE INDEX IF NOT EXISTS idx_flow_runs_flow_id ON flow_runs (flow_id);
CREATE INDEX IF NOT EXISTS idx_flow_runs_status ON flow_runs (status);
CREATE INDEX IF NOT EXISTS idx_flow_runs_created_at ON flow_runs (created_at);
CREATE INDEX IF NOT EXISTS idx_flow_runs_updated_at ON flow_runs (updated_at);

-- Add trigger for flow_runs updated_at
DROP TRIGGER IF EXISTS set_timestamp_flow_runs ON flow_runs;
CREATE TRIGGER set_timestamp_flow_runs BEFORE UPDATE ON flow_runs FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();

-- Create flow_task_runs table for individual task executions
-- Using composite primary key (flow_run_id, task_name) since task names are unique within a flow
CREATE TABLE IF NOT EXISTS flow_task_runs (
    flow_run_id UUID NOT NULL REFERENCES flow_runs(flow_run_id) ON DELETE CASCADE,
    task_name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'RUNNING', 'SUCCESS', 'FAILED')),
    result JSONB,
    result_cache_key TEXT, -- S3 cache key: s3://<cache_bucket>/result_cache/<flow_run_id>/<task_name>.json
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    finished_at TIMESTAMP WITH TIME ZONE,
    duration_seconds FLOAT,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    
    PRIMARY KEY (flow_run_id, task_name)
);

-- Add indexes for flow_task_runs table
-- Note: No need for separate flow_run_id index since it's part of the primary key
CREATE INDEX IF NOT EXISTS idx_flow_task_runs_status ON flow_task_runs (status);
CREATE INDEX IF NOT EXISTS idx_flow_task_runs_created_at ON flow_task_runs (created_at);
CREATE INDEX IF NOT EXISTS idx_flow_task_runs_task_name ON flow_task_runs (task_name);

-- Add trigger for flow_task_runs updated_at
DROP TRIGGER IF EXISTS set_timestamp_flow_task_runs ON flow_task_runs;
CREATE TRIGGER set_timestamp_flow_task_runs BEFORE UPDATE ON flow_task_runs FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();

-- Create flow_run_events table for event tracking and audit trail
CREATE TABLE IF NOT EXISTS flow_run_events (
    event_id UUID PRIMARY KEY,
    flow_run_id UUID REFERENCES flow_runs(flow_run_id) ON DELETE CASCADE,
    task_name VARCHAR(255), -- NULL for flow-level events
    event_type VARCHAR(100) NOT NULL, -- FlowRunRequest, FlowRunResponse, FlowRunExecuteEvent, FlowRunStatusEvent, TaskRunStatusEvent
    event_data JSONB NOT NULL,
    event_timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    source VARCHAR(100), -- orchestrator, worker, scheduler, etc.
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Add indexes for flow_run_events table
CREATE INDEX IF NOT EXISTS idx_flow_run_events_flow_run_id ON flow_run_events (flow_run_id);
CREATE INDEX IF NOT EXISTS idx_flow_run_events_event_type ON flow_run_events (event_type);
CREATE INDEX IF NOT EXISTS idx_flow_run_events_event_timestamp ON flow_run_events (event_timestamp);
CREATE INDEX IF NOT EXISTS idx_flow_run_events_task_name ON flow_run_events (task_name);

-- Create worker_heartbeats table for worker management
CREATE TABLE IF NOT EXISTS worker_heartbeats (
    worker_id VARCHAR(255) PRIMARY KEY,
    worker_name VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE', 'FAILED')),
    last_heartbeat TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    worker_info JSONB, -- Additional worker metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Add indexes for worker_heartbeats table
CREATE INDEX IF NOT EXISTS idx_worker_heartbeats_status ON worker_heartbeats (status);
CREATE INDEX IF NOT EXISTS idx_worker_heartbeats_last_heartbeat ON worker_heartbeats (last_heartbeat);

-- Add trigger for worker_heartbeats updated_at
DROP TRIGGER IF EXISTS set_timestamp_worker_heartbeats ON worker_heartbeats;
CREATE TRIGGER set_timestamp_worker_heartbeats BEFORE UPDATE ON worker_heartbeats FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();


-- +goose Down
DROP TRIGGER IF EXISTS set_timestamp_flows ON flows;
DROP TRIGGER IF EXISTS set_timestamp_flow_runs ON flow_runs;
DROP TRIGGER IF EXISTS set_timestamp_flow_task_runs ON flow_task_runs;
DROP TRIGGER IF EXISTS set_timestamp_worker_heartbeats ON worker_heartbeats;

-- Drop indexes
DROP INDEX IF EXISTS idx_flow_runs_flow_id;
DROP INDEX IF EXISTS idx_flow_runs_status;
DROP INDEX IF EXISTS idx_flow_runs_created_at;
DROP INDEX IF EXISTS idx_flow_runs_updated_at;
DROP INDEX IF EXISTS idx_flow_task_runs_flow_run_id;
DROP INDEX IF EXISTS idx_flow_task_runs_task_name;
DROP INDEX IF EXISTS idx_flow_task_runs_status;
DROP INDEX IF EXISTS idx_flow_task_runs_created_at;
DROP INDEX IF EXISTS idx_flow_run_events_flow_run_id;
DROP INDEX IF EXISTS idx_flow_run_events_event_type;
DROP INDEX IF EXISTS idx_flow_run_events_event_timestamp;
DROP INDEX IF EXISTS idx_flow_run_events_task_name;
DROP INDEX IF EXISTS idx_worker_heartbeats_status;
DROP INDEX IF EXISTS idx_worker_heartbeats_last_heartbeat;

DROP TABLE IF EXISTS flow_run_events CASCADE;
DROP TABLE IF EXISTS flow_task_runs CASCADE;
DROP TABLE IF EXISTS flow_runs CASCADE;
DROP TABLE IF EXISTS worker_heartbeats CASCADE;