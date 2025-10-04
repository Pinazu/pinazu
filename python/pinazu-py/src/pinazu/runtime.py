"""Runtime logic for flows and tasks"""

import os
import nats
import logging
import hashlib
import orjson as json
from concurrent.futures import ThreadPoolExecutor
from typing import Any, Callable, Dict, Optional, Union
from aiobotocore.session import get_session
from botocore.exceptions import ClientError
from pinazu.models import (
    FlowRunStatusEvent,
    TaskRunStatusEvent,
    FlowStatus,
    NatTopics,
    Event,
    CacheResult,
)
from pinazu.logger import CustomLogger
from logging import Logger
from datetime import datetime, timezone
from uuid import UUID


# Configure logging


class TaskRegistry:
    """Registry to store task functions"""

    _tasks: Dict[str, Callable] = {}

    @classmethod
    def register_task(cls, name: str, func: Callable):
        cls._tasks[name] = func

    @classmethod
    def get_task(cls, name: str) -> Optional[Callable]:
        return cls._tasks.get(name)


class CacheManager:
    """Manages S3 caching for task results"""

    def __init__(
        self,
        bucket_name: Optional[str] = None,
        logger: Optional[Logger] = None,
    ):
        bucket = bucket_name or os.getenv("CACHE_BUCKET", "flow-cache-bucket")
        self.bucket_name = bucket
        self.session = get_session()
        self.logger = logger or CustomLogger(
            log_level=logging.DEBUG,
            name="CacheManager",
        )

    @staticmethod
    def _generate_cache_key(
        flow_run_id: UUID, task_name: str, args: tuple, kwargs: dict
    ) -> str:
        """Generate cache key based on task inputs"""
        input_data = {"task_name": task_name, "args": args, "kwargs": kwargs}
        input_str = json.dumps(
            input_data,
            default=str,
            option=json.OPT_SORT_KEYS,
        )
        input_hash = hashlib.md5(input_str).hexdigest()
        flow_run_id_str = flow_run_id.__str__()
        # Use task name and input hash for consistent caching across flows
        cache_key = f"result_cache/flow-{flow_run_id_str}/tasks/{task_name}/{input_hash}.json"  # noqa: E501
        return cache_key

    async def get_cached_result(
        self, flow_run_id: UUID, task_name: str, *args, **kwargs
    ) -> Optional[CacheResult]:
        """Get cached result from S3"""
        self.logger.debug(
            f"CacheManager: Flow Run Id {flow_run_id}, Task Name: {task_name}, Args: {args}, Kwargs: {kwargs}"  # noqa: E501
        )  # noqa: E501
        cache_key = CacheManager._generate_cache_key(
            flow_run_id,
            task_name,
            args,
            kwargs,
        )
        async with self.session.create_client("s3") as s3_client:
            try:
                response = await s3_client.get_object(
                    Bucket=self.bucket_name, Key=cache_key
                )
                data = await response["Body"].read()
                self.logger.info(f"Cache hit for task {task_name}: {data}")
                return CacheResult.model_validate_json(data)
            except ClientError as e:
                if e.response.get("Error", {}).get("Code") == "NoSuchKey":
                    self.logger.info(f"Cache miss for task {task_name}")
                else:
                    self.logger.error(f"Error retrieving cache: {e}")
            except Exception as e:
                self.logger.error(f"Unexpected error retrieving cache: {e}")

        return None

    async def store_result(
        self, flow_run_id: UUID, task_name: str, result: Any, *args, **kwargs
    ) -> str:
        """Store task result in S3"""
        cache_key = self._generate_cache_key(
            flow_run_id=flow_run_id,
            task_name=task_name,
            args=args,
            kwargs=kwargs,
        )

        try:
            cache_data = CacheResult(
                flow_run_id=flow_run_id,
                task_name=task_name,
                cache_key=cache_key,
                result=result,
            )
            async with self.session.create_client("s3") as s3_client:
                await s3_client.put_object(
                    Bucket=self.bucket_name,
                    Key=cache_key,
                    Body=cache_data.model_dump_json().encode("utf-8"),
                    ContentType="application/json",
                )
                s3_path = f"s3://{self.bucket_name}/{cache_key}"
                info = f"Stored result for task {task_name} at {s3_path}"
                self.logger.info(info)
            return s3_path
        except Exception as e:
            self.logger.error(f"Error storing cache: {e}")
            return ""


class InMemoryCacheManager:
    """
    In-memory cache manager for task results
    (fallback when S3 caching is disabled)
    """

    def __init__(self, logger: Optional[Logger] = None):
        self._cache: Dict[str, CacheResult] = {}
        self.logger = logger or CustomLogger(
            log_level=logging.DEBUG,
            name="InMemoryCacheManager",
        )

    @staticmethod
    def _generate_cache_key(
        flow_run_id: UUID, task_name: str, args: tuple, kwargs: dict
    ) -> str:
        """Generate cache key based on task inputs"""
        input_data = {"task_name": task_name, "args": args, "kwargs": kwargs}
        input_str = json.dumps(
            input_data,
            default=str,
            option=json.OPT_SORT_KEYS,
        )
        input_hash = hashlib.md5(input_str).hexdigest()
        flow_run_id_str = flow_run_id.__str__()
        cache_key = f"memory_cache/flow-{flow_run_id_str}/tasks/{task_name}/{input_hash}"  # noqa: E501
        return cache_key

    async def get_cached_result(
        self, flow_run_id: UUID, task_name: str, *args, **kwargs
    ) -> Optional[CacheResult]:
        """Get cached result from memory"""
        cache_key = self._generate_cache_key(
            flow_run_id=flow_run_id,
            task_name=task_name,
            args=args,
            kwargs=kwargs,
        )

        if cache_key in self._cache:
            cached_result = self._cache[cache_key]
            self.logger.info(f"Memory cache hit for task {task_name}")
            return cached_result
        else:
            self.logger.info(f"Memory cache miss for task {task_name}")
            return None

    async def store_result(
        self, flow_run_id: UUID, task_name: str, result: Any, *args, **kwargs
    ) -> str:
        """Store task result in memory"""
        cache_key = self._generate_cache_key(
            flow_run_id=flow_run_id,
            task_name=task_name,
            args=args,
            kwargs=kwargs,
        )

        try:
            cache_data = CacheResult(
                flow_run_id=flow_run_id,
                task_name=task_name,
                cache_key=cache_key,
                result=result,
            )
            self._cache[cache_key] = cache_data
            self.logger.info(
                f"Stored result for task {task_name} in memory cache"  # noqa: E501
            )
            return cache_key
        except Exception as e:
            self.logger.error(f"Error storing memory cache: {e}")
            return ""

    def clear_cache(self):
        """Clear all cached results"""
        self._cache.clear()
        self.logger.info("Memory cache cleared")


class LogManager:
    """Manages logging for both local and remote modes"""

    def __init__(
        self,
        nats_url: Optional[str] = None,
        logger: Optional[Logger] = None,
    ):
        self.nats_url = nats_url or os.getenv("NATS_URL")
        self.logger = logger or CustomLogger(
            log_level=logging.DEBUG,
            name="LogManager",
        )
        self.is_remote = self.nats_url is not None
        self._nat_client = None

    @property
    async def nats_client(self):
        """Connect to NATS server"""
        if self._nat_client:
            return self._nat_client
        if self.nats_url is not None:
            try:
                self._nat_client = await nats.connect([self.nats_url])
                self.logger.info(f"Connected to NATS at {self.nats_url}")
            except Exception as e:
                self.logger.error(f"Failed to connect to NATS: {e}")
                raise
            return self._nat_client
        raise RuntimeError("NATS URL is not set")

    async def log_task_status(
        self,
        flow_run_id: UUID,
        task_name: str,
        status: FlowStatus,
        result_cache_key: Optional[str] = None,
    ):
        """Log task status"""
        log_data = TaskRunStatusEvent(
            flow_run_id=flow_run_id,
            task_name=task_name,
            status=status,
            result_cache_key=result_cache_key,
            event_timestamp=datetime.now(timezone.utc),
        )

        if self.is_remote:
            await self._send_remote_log(log_data, NatTopics.TASK_STATUS)
        else:
            self.logger.info(log_data.model_dump_json())

    async def log_flow_status(self, flow_run_id: UUID, status: FlowStatus):
        """Log flow status"""
        log_data = FlowRunStatusEvent(
            flow_run_id=flow_run_id,
            status=status,
            event_timestamp=datetime.now(timezone.utc),
        )

        if self.is_remote:
            await self._send_remote_log(log_data, NatTopics.FLOW_STATUS)
        else:
            self.logger.info(log_data.model_dump_json())

    async def _send_remote_log(
        self,
        log_data: Union[FlowRunStatusEvent, TaskRunStatusEvent],
        log_type: NatTopics,
    ):
        """Send log to NATS URL"""
        try:
            subject = log_type.value

            event = Event(message=log_data)
            nats_client = await self.nats_client
            event_json = event.model_dump_json().encode("utf-8")

            # Add validation to ensure we're not sending empty messages
            if b'"message":{}' in event_json:
                self.logger.error(f"Empty message detected: {event_json}")
                err = "Generated empty message - this indicates a serialization issue"  # noqa: E501
                raise ValueError(err)

            await nats_client.publish(subject, event_json)

            debug = f"Sent remote log for {log_type}: {event_json.decode()}"
            self.logger.debug(debug)
        except Exception as e:
            self.logger.error(f"Failed to send remote log: {e}")


class FlowRunner:
    """Manages flow execution"""

    def __init__(
        self,
        flow_run_id: UUID,
        flow_name: str,
        num_threads: int = 4,
        cache_bucket: Optional[str] = None,
        enable_caching: bool = True,
    ):
        self.flow_run_id = flow_run_id
        self.flow_name = flow_name
        self.enable_caching = enable_caching
        self.logger = CustomLogger(
            log_level=logging.getLevelNamesMapping()[
                os.getenv("LOG_LEVEL", "INFO").upper()
            ],  # noqa: E501
            name=f"FlowRunner-{flow_name}",
        )
        # Hybrid caching: S3 when enabled, in-memory as fallback
        if enable_caching:
            self.cache_manager = CacheManager(
                bucket_name=cache_bucket, logger=self.logger
            )
            self.cache_type = "s3"
        else:
            self.cache_manager = InMemoryCacheManager(logger=self.logger)
            self.cache_type = "memory"
        self.log_manager = LogManager(logger=self.logger)
        self.tasks_status: Dict[str, FlowStatus] = {}
        self.failed = False
        self.executor = ThreadPoolExecutor(max_workers=num_threads)

        # Start logging thread for local mode

    async def run_flow(self, flow_func: Callable, *args, **kwargs) -> Any:
        """Execute the flow function"""
        try:
            await self.log_manager.log_flow_status(
                flow_run_id=self.flow_run_id,
                status=FlowStatus.RUNNING,
            )

            # Execute the flow function
            result = await flow_func(*args, **kwargs)

            if not self.failed:
                await self.log_manager.log_flow_status(
                    self.flow_run_id, FlowStatus.SUCCESS
                )

            return result
        except Exception as e:
            self.failed = True
            await self.log_manager.log_flow_status(
                flow_run_id=self.flow_run_id,
                status=FlowStatus.FAILED,
            )
            self.logger.error(f"Flow {self.flow_name} failed: {e}")
            raise
        finally:
            self.executor.shutdown(wait=True)
            # Clear in-memory cache when flow completes
            if self.cache_type == "memory" and hasattr(
                self.cache_manager, "clear_cache"
            ):
                self.cache_manager.clear_cache()

    async def run_task(
        self, task_name: str, task_func: Callable, *args, **kwargs
    ) -> Any:
        """Execute a task with caching"""
        if self.failed:
            raise RuntimeError("Flow is in failed state")

        self.tasks_status[task_name] = FlowStatus.RUNNING
        try:
            print(
                f"FlowRunner: Flow Run Id {self.flow_run_id}"
                f", Task Name: {task_name} (using {self.cache_type} cache)"
            )

            # Check cache first (always enabled - either S3 or in-memory)
            cached_result = await self.cache_manager.get_cached_result(
                self.flow_run_id, task_name, *args, **kwargs
            )

            if cached_result is not None:
                self.tasks_status[task_name] = FlowStatus.SUCCESS
                await self.log_manager.log_task_status(
                    flow_run_id=self.flow_run_id,
                    task_name=task_name,
                    status=FlowStatus.SUCCESS,
                    result_cache_key=cached_result.cache_key,
                )
                return cached_result.result

            # Execute task
            result = await task_func(*args, **kwargs)

            # Store result in cache (always enabled - either S3 or in-memory)
            result_cache_key = await self.cache_manager.store_result(
                self.flow_run_id,
                task_name,
                result,
                *args,
                **kwargs,
            )

            self.tasks_status[task_name] = FlowStatus.SUCCESS
            await self.log_manager.log_task_status(
                flow_run_id=self.flow_run_id,
                task_name=task_name,
                status=FlowStatus.SUCCESS,
                result_cache_key=result_cache_key,
            )

            return result

        except Exception as e:
            self.failed = True
            self.tasks_status[task_name] = FlowStatus.FAILED
            await self.log_manager.log_task_status(
                self.flow_run_id, task_name, FlowStatus.FAILED, None
            )
            self.logger.error(f"Task {task_name} failed: {e}")
            raise

    # def run_tasks_parallel(self, task_funcs: List[tuple]) -> List[Any]:
    #     """Run multiple tasks in parallel"""
    #     if self.failed:
    #         raise RuntimeError("Flow is in failed state")

    #     futures = []
    #     for task_name, task_func, args, kwargs in task_funcs:
    #         future = self.executor.submit(
    #             self.run_task, task_name, task_func, *args, **kwargs
    #         )
    #         futures.append(future)

    #     results = []
    #     for future in as_completed(futures):
    #         try:
    #             result = future.result()
    #             results.append(result)
    #         except Exception as e:
    #             self.failed = True
    #             logger.error(f"Parallel task execution failed: {e}")
    #             # Cancel remaining futures
    #             for f in futures:
    #                 f.cancel()
    #             raise

    #     return results
