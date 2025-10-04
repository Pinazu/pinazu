from __future__ import annotations
from pydantic import BaseModel, Field
from typing import Optional, Any
from enum import Enum
from datetime import datetime, timezone
from uuid import UUID


class NatTopics(str, Enum):
    """Enumeration for NATS topics"""

    FLOW_STATUS = "v1.svc.worker.flow.status"
    TASK_STATUS = "v1.svc.worker.task.status"


class FlowStatus(str, Enum):
    """Enumeration for flow status"""

    SCHEDULED = "SCHEDULED"
    PENDING = "PENDING"
    RUNNING = "RUNNING"
    SUCCESS = "SUCCESS"
    FAILED = "FAILED"


class EventHeaders(BaseModel):
    """
    UserID       uuid.UUID `json:"user_id"`
    ThreadID     *uuid.UUID `json:"thread_id,omitempty"`
    ConnectionID *uuid.UUID `json:"connection_id,omitempty"`
    """

    user_id: UUID
    thread_id: Optional[UUID] = None  # Optional thread ID
    connection_id: Optional[UUID] = None  # Optional connection ID


class EventMetadata(BaseModel):
    """
    Metadata for the event
    EventMetadata struct {
                TraceID   string    `json:"trace_id,omitempty"`
                Timestamp time.Time `json:"timestamp"`
        }"""

    trace_id: Optional[str] = None  # Optional trace ID for distributed tracing
    timestamp: datetime  # Timestamp of the event


class Event(BaseModel):
    """
    Base event model
    Event[T EventMessage] struct {
                H       *EventHeaders  `json:"header"`
                Msg     T              `json:"message"`
                M       *EventMetadata `json:"metadata"`
                Subject EventSubject   // Not serialized - used internally
        }
    """

    header: Optional[EventHeaders] = Field(
        default_factory=lambda: EventHeaders(
            user_id=UUID("00000000-0000-0000-0000-000000000000"),
            thread_id=None,
            connection_id=None,
        )
    )  # Event headers with user, thread, and connection IDs
    # The message payload, can be any Pydantic model
    message: FlowRunStatusEvent | TaskRunStatusEvent
    metadata: Optional[EventMetadata] = Field(
        default_factory=lambda: EventMetadata(
            timestamp=datetime.now(timezone.utc),
        )
    )


class FlowRunStatusEvent(BaseModel):
    """
    Model to represent the status of a flow run - matches Go struct
    FlowRunStatusEventMessage struct {
        FlowRunID      uuid.UUID `json:"flow_run_id"`
        Status         string    `json:"status"`
        EventTimestamp time.Time `json:"event_timestamp"`
        ErrorMessage   string    `json:"error_message,omitempty"`
    }
    """

    flow_run_id: UUID
    status: FlowStatus
    error_message: Optional[str] = None  # Error message if the flow failed
    traceback: Optional[str] = None  # Traceback for debugging
    event_timestamp: datetime


class TaskRunStatusEvent(BaseModel):
    """
    Model to represent the status of a task run - matches Go struct
    TaskRunStatusEventMessage struct {
        FlowRunID      uuid.UUID `json:"flow_run_id"`
        TaskName       string    `json:"task_name"`
        Status         string    `json:"status"`
        ResultCacheKey *string   `json:"result_cache_key,omitempty"`
        EventTimestamp time.Time `json:"event_timestamp"`
    }
    """

    flow_run_id: UUID
    task_name: str
    status: FlowStatus
    result_cache_key: Optional[str] = None  # S3 cache key for the task result
    error_message: Optional[str] = None  # Error message if the task failed
    traceback: Optional[str] = None  # Traceback for debugging
    event_timestamp: datetime


class CacheResult(BaseModel):
    """
    Model to represent a cached result
    {
        "flow_run_id": "uuid",
        "task_name": "string",
        "result": "any",  # The actual result of the task
        "cache_key": "string"  # S3 cache key where the result is stored
    }
    """

    flow_run_id: UUID
    task_name: str
    result: Any  # The actual result of the task
    cache_key: str  # S3 cache key where the result is stored
    timestamp: datetime = Field(
        default_factory=lambda: datetime.now(timezone.utc),
    )
