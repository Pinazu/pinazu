"""Example flow demonstrating usage of @flow and @task decorators"""

import asyncio
import time

from pinazu import flow, task


@task
async def add_numbers(a: int, b: int) -> int:
    """Add two numbers"""
    time.sleep(1)  # Simulate work
    return a + b


@task
async def multiply_numbers(a: int, b: int) -> int:
    """Multiply two numbers"""
    time.sleep(1)  # Simulate work
    return a * b


@task
async def format_result(result: int) -> str:
    """Format result as string"""
    return f"Final result: {result}"


@flow
async def math_pipeline(x: int, y: int) -> str:
    """A simple math pipeline flow"""
    # Sequential execution
    sum_result = await add_numbers(x, y)
    product_result = await multiply_numbers(sum_result, 2)

    # Format final result
    formatted = await format_result(product_result)

    return formatted


# This script will be uploaded through execute_flow.py.
# It will then be executed through worker
if __name__ == "__main__":
    # Run the flow
    result = asyncio.run(math_pipeline(5, 3))
    print(f"Pipeline result: {result}")
