"""Flow and Task decorators"""

import functools
import inspect
import uuid
import os
import threading
from typing import Callable
from pinazu.runtime import FlowRunner, TaskRegistry
import asyncio


class FlowError(Exception):
    """Custom exception for flow-related errors"""

    pass


class TaskError(Exception):
    """Custom exception for task-related errors"""

    pass


# Global registry to track flows per module
_flow_registry = {}

# Global registry to track active flow runners
_active_flows = {}
_flow_lock = threading.Lock()


def flow(func: Callable) -> Callable:
    """
    Decorator to mark a function as a flow.
    Each file can only contain one @flow.
    """
    # Get the module where this flow is defined
    frame = inspect.currentframe()
    module_name = "__main__"  # default

    if frame and frame.f_back:
        caller_globals = frame.f_back.f_globals
        module_name = caller_globals.get("__name__", "__main__")

    # Check if there's already a flow in this module
    if module_name in _flow_registry:
        existing_flow = _flow_registry[module_name]
        raise FlowError(
            "Only one @flow is allowed per file."
            f" Found existing flow: {existing_flow}"
        )

    # Register this flow
    _flow_registry[module_name] = func.__name__

    @functools.wraps(func)
    def wrapper(*args, **kwargs):
        # Create a flow runner instance
        flow_run_id = uuid.UUID(os.getenv("FLOW_RUN_ID", str(uuid.uuid4())))

        # Check environment variables for caching configuration
        is_cache = os.getenv("ENABLE_S3_CACHING", "false").lower() == "true"
        cache_bucket = os.getenv("CACHE_BUCKET", None)

        runner = FlowRunner(
            flow_run_id=flow_run_id,
            flow_name=func.__name__,
            enable_caching=is_cache,
            cache_bucket=cache_bucket,
        )

        # Store the runner in the function context
        func._flow_runner = runner  # type: ignore

        # Register this flow as active
        with _flow_lock:
            _active_flows[flow_run_id] = runner

        # Execute the flow
        try:
            return runner.run_flow(func, *args, **kwargs)
        finally:
            # Clean up active flows
            with _flow_lock:
                _active_flows.pop(flow_run_id, None)

    wrapper._is_flow = True  # type: ignore
    wrapper._flow_name = func.__name__  # type: ignore
    return wrapper


def task(func: Callable) -> Callable:
    """
    Decorator to mark a function as a task.
    Tasks can only be used inside flows.
    """

    @functools.wraps(func)
    def wrapper(*args, **kwargs):
        # First try to find flow context in active flows
        flow_context = None

        # Get the most recent active flow
        with _flow_lock:
            if _active_flows:
                # Get the most recently added flow
                # (assume it's the current one)
                flow_context = list(_active_flows.values())[-1]

        # If not found in active flows, try to find in call stack
        if not flow_context:
            frame = inspect.currentframe()

            # Walk up the call stack to find a flow context
            current_frame = frame
            while current_frame:
                if current_frame.f_back:
                    caller_locals = current_frame.f_back.f_locals
                    caller_globals = current_frame.f_back.f_globals

                    # Look for flow runner in locals or globals
                    for name, obj in {
                        **caller_locals,
                        **caller_globals,
                    }.items():
                        if hasattr(obj, "_flow_runner"):
                            flow_context = obj._flow_runner
                            break
                        elif isinstance(obj, FlowRunner):
                            flow_context = obj
                            break

                    if flow_context:
                        break
                current_frame = current_frame.f_back

        if not flow_context:
            err = f"Task '{func.__name__}' can only be called within a @flow context"  # noqa: E501
            raise TaskError(err)

        # Register and execute the task
        task_name = func.__name__

        loop = asyncio.get_event_loop()
        return loop.run_in_executor(
            flow_context.executor,
            lambda: asyncio.run(
                flow_context.run_task(task_name, func, *args, **kwargs)
            ),  # noqa: E501
        )
        # return (task_name, func, *args, **kwargs)

    wrapper._is_task = True  # type: ignore
    wrapper._task_name = func.__name__  # type: ignore

    # Register the task globally
    TaskRegistry.register_task(func.__name__, func)

    return wrapper
