"""
Example to upload the python script into S3/Minio
and execute the script flow through pinazu client
"""

import os
import time
import minio
import pinazu


s3_client = minio.Minio(
    endpoint="localhost:9000",
    access_key="minioadmin",
    secret_key="minioadmin",
    secure=False,
)

client = pinazu.Client()

# Create bucket if it doesn't exist
bucket_name = "test-bucket"
if not s3_client.bucket_exists(bucket_name):
    s3_client.make_bucket(bucket_name)

# Create cache bucket else it will get error for cache.
# If you memory cache type. NO NEED to create a cache bucket
cache_bucket_name = "flow-cache-bucket"
if not s3_client.bucket_exists(cache_bucket_name):
    s3_client.make_bucket(cache_bucket_name)

try:
    # Get the file path + s3 key based on the file path
    file_path = os.path.join(os.path.dirname(os.path.abspath(__file__)))
    flow_file_path = os.path.join(file_path, "basic_flow.py")
    flow_s3_key = os.path.basename(flow_file_path)

    # Upload the files to MinIO/S3 using the MinIO client
    s3_client.fput_object(
        bucket_name=bucket_name,
        object_name=flow_s3_key,
        file_path=flow_file_path,
    )

    flow_s3_location = f"s3://{bucket_name}/{flow_s3_key}"
    print(f"Uploaded {flow_file_path} to {flow_s3_location}\n")

    # Create a new flow with the code location in S3 bucket
    flow = client.create_flow(
        name="basic_flow_example",
        engine="process",
        parameters_schema={
            "x": {"type": "integer"},
            "y": {"type": "integer"},
        },
        code_location=flow_s3_location,
        entrypoint="python",
    )

    # Print out the new flow info
    print(f"[BASIC FLOW] info: {flow}\n")

    # Execute flows
    flow_run = client.execute_flow(
        flow_id=flow.id,
        parameters={"x": 1, "y": 2},  # Pass the parameters
    )

    # Print out the information of the flow run above
    print(f"[BASIC FLOW] run info: {flow_run}\n")

    # A continue pull status for checking the realtime status update
    while True:
        done = True
        # Retrieve the status of the flow
        if flow_run.status != "SUCCESS":
            flow_run = client.get_flow_run(flow_run.flow_run_id)
            print(f"[BASIC FLOW] run status: {flow_run.status}\n")
            done = False

        time.sleep(0.5)
        if done:
            break
finally:
    # Clean up the flow
    if flow:
        client.delete_flow(flow.id)
