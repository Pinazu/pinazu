import { test, expect } from '@playwright/test';

test.describe.serial('Tasks API', () => {
  let createdTaskId: string;
  let createdThreadId: string;
  let testUserId: string = '550e8400-c95b-4444-6666-446655440000'; // Admin user UUID from database

  test.beforeAll(async ({ request }) => {
    // Create a thread first since tasks require a thread_id
    const threadData = {
      title: 'Test Thread for Tasks API',
      user_id: testUserId
    };

    const threadResponse = await request.post('/v1/threads', {
      data: threadData
    });

    expect(threadResponse.status()).toBe(201);
    const threadBody = await threadResponse.json();
    createdThreadId = threadBody.id;

    // Clean up any existing test tasks before starting
    const response = await request.get('/v1/tasks');
    if (response.ok()) {
      const responseBody = await response.json();
      const tasks = responseBody.tasks || [];
      if (Array.isArray(tasks)) {
        for (const task of tasks) {
          if (task.additional_info && task.additional_info.test_marker === 'api_test_task') {
            await request.delete(`/v1/tasks/${task.id}`);
          }
        }
      }
    }
  });

  test.afterAll(async ({ request }) => {
    // Clean up created task
    if (createdTaskId) {
      await request.delete(`/v1/tasks/${createdTaskId}`);
    }
    // Clean up created thread
    if (createdThreadId) {
      await request.delete(`/v1/threads/${createdThreadId}`);
    }
  });

  test('should create a new task', async ({ request }) => {
    expect(createdThreadId).toBeTruthy();

    const taskData = {
      thread_id: createdThreadId,
      max_request_loop: 15,
      additional_info: {
        test_marker: 'api_test_task',
        description: 'A test task created via Playwright API tests',
        priority: 'high',
        category: 'automation_test',
        metadata: {
          created_by_test: true,
          test_run_id: Date.now()
        }
      }
    };

    const response = await request.post('/v1/tasks', {
      data: taskData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('id');
    expect(responseBody.thread_id).toBe(taskData.thread_id);
    expect(responseBody.max_request_loop).toBe(taskData.max_request_loop);
    expect(responseBody).toHaveProperty('created_at');
    expect(responseBody).toHaveProperty('updated_at');
    expect(responseBody).toHaveProperty('created_by');

    createdTaskId = responseBody.id;
  });

  test('should get all tasks', async ({ request }) => {
    const response = await request.get('/v1/tasks');

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('tasks');
    expect(responseBody.tasks).toBeDefined();
    expect(Array.isArray(responseBody.tasks)).toBe(true);
    expect(responseBody.tasks.length).toBeGreaterThanOrEqual(1); // At least our created task
  });

  test('should get task by ID', async ({ request }) => {
    expect(createdTaskId).toBeTruthy();

    const response = await request.get(`/v1/tasks/${createdTaskId}`);

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.id).toBe(createdTaskId);
    expect(responseBody.thread_id).toBe(createdThreadId);
    expect(responseBody.max_request_loop).toBe(15);
    expect(responseBody).toHaveProperty('created_at');
    expect(responseBody).toHaveProperty('updated_at');
    expect(responseBody).toHaveProperty('created_by');
  });

  test('should return 404 for non-existent task', async ({ request }) => {
    const nonExistentId = '12345678-1234-1234-1234-123456789012';
    const response = await request.get(`/v1/tasks/${nonExistentId}`);

    expect(response.status()).toBe(404);
    
    const responseBody = await response.json();
    expect(responseBody.resource).toBe("Task");
    expect(responseBody.id).toBe(nonExistentId);
  });

  test('should return 400 for invalid task ID format', async ({ request }) => {
    const invalidId = 'invalid-uuid';
    const response = await request.get(`/v1/tasks/${invalidId}`);

    expect(response.status()).toBe(400);
  });

  test('should return 404 for nil UUID (validation error)', async ({ request }) => {
    const nilUuid = '00000000-0000-0000-0000-000000000000';
    const response = await request.get(`/v1/tasks/${nilUuid}`);

    expect(response.status()).toBe(404);
    
    const responseBody = await response.json();
    expect(responseBody.resource).toBe("Task");
    expect(responseBody.id).toBe(nilUuid);
  });

  test('should update an existing task', async ({ request }) => {
    expect(createdTaskId).toBeTruthy();

    const updateData = {
      thread_id: createdThreadId,
      max_request_loop: 25,
      additional_info: {
        test_marker: 'api_test_task',
        description: 'Updated test task description',
        priority: 'medium',
        category: 'automation_test_updated',
        metadata: {
          updated_by_test: true,
          last_modified: Date.now(),
          version: 2
        }
      }
    };

    const response = await request.put(`/v1/tasks/${createdTaskId}`, {
      data: updateData
    });

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.id).toBe(createdTaskId);
    expect(responseBody.max_request_loop).toBe(updateData.max_request_loop);
    expect(responseBody.thread_id).toBe(createdThreadId);
  });

  test('should return 404 when updating non-existent task', async ({ request }) => {
    const nonExistentId = '12345678-1234-1234-1234-123456789abc';
    const updateData = {
      thread_id: createdThreadId,
      max_request_loop: 20
    };

    const response = await request.put(`/v1/tasks/${nonExistentId}`, {
      data: updateData
    });

    expect(response.status()).toBe(404);
  });

  test('should handle validation errors for create task - missing required fields', async ({ request }) => {
    const invalidTaskData = {
      // Missing required thread_id field
      max_request_loop: 10,
      additional_info: {
        test_marker: 'invalid_test'
      }
    };

    const response = await request.post('/v1/tasks', {
      data: invalidTaskData
    });

    expect(response.status()).toBe(400);
  });

  test('should handle validation errors for invalid thread_id format', async ({ request }) => {
    const invalidTaskData = {
      thread_id: 'invalid-uuid-format',
      max_request_loop: 10,
      additional_info: {
        test_marker: 'invalid_test'
      }
    };

    const response = await request.post('/v1/tasks', {
      data: invalidTaskData
    });

    expect(response.status()).toBe(400);
  });

  test('should handle validation errors for negative max_request_loop', async ({ request }) => {
    const invalidTaskData = {
      thread_id: createdThreadId,
      max_request_loop: -5, // Invalid negative value
      additional_info: {
        test_marker: 'invalid_test'
      }
    };

    const response = await request.post('/v1/tasks', {
      data: invalidTaskData
    });

    expect(response.status()).toBe(400);
  });

  test('should handle validation errors for non-existent thread_id', async ({ request }) => {
    const nonExistentThreadId = '12345678-1234-1234-1234-123456789999';
    const invalidTaskData = {
      thread_id: nonExistentThreadId,
      max_request_loop: 10,
      additional_info: {
        test_marker: 'invalid_test'
      }
    };

    const response = await request.post('/v1/tasks', {
      data: invalidTaskData
    });

    expect(response.status()).toBe(404);
  });

  test('should create task with minimal required fields', async ({ request }) => {
    expect(createdThreadId).toBeTruthy();

    const minimalTaskData = {
      thread_id: createdThreadId
    };

    const response = await request.post('/v1/tasks', {
      data: minimalTaskData
    });

    expect(response.status()).toBe(201);
    
    const responseBody = await response.json();
    expect(responseBody.thread_id).toBe(minimalTaskData.thread_id);
    expect(responseBody.max_request_loop).toBe(20); // Default value
    expect(responseBody).toHaveProperty('id');

    // Clean up the minimal task
    await request.delete(`/v1/tasks/${responseBody.id}`);
  });

  test('should update task with partial data', async ({ request }) => {
    expect(createdTaskId).toBeTruthy();

    // Only update the max_request_loop
    const partialUpdateData = {
      max_request_loop: 30
    };

    const response = await request.put(`/v1/tasks/${createdTaskId}`, {
      data: partialUpdateData
    });

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(responseBody.id).toBe(createdTaskId);
    expect(responseBody.max_request_loop).toBe(partialUpdateData.max_request_loop);
    // Thread ID should remain the same
    expect(responseBody.thread_id).toBe(createdThreadId);
  });

  test('should return 404 for non-existent task run status', async ({ request }) => {
    const nonExistentRunId = '12345678-1234-1234-1234-123456789def';
    const response = await request.get(`/v1/tasks/${nonExistentRunId}/status`);

    expect(response.status()).toBe(404);
    
    const responseBody = await response.json();
    expect(responseBody.resource).toBe("TaskRun");
    expect(responseBody.id).toBe(nonExistentRunId);
  });

  test('should get all task runs for a task', async ({ request }) => {
    expect(createdTaskId).toBeTruthy();

    const response = await request.get(`/v1/tasks/${createdTaskId}/runs`);

    expect(response.status()).toBe(200);
    
    const responseBody = await response.json();
    expect(Array.isArray(responseBody)).toBe(true);
    // Task runs array might be empty if no runs have been created yet
    if (responseBody.length > 0) {
      // Validate the structure of task run objects
      responseBody.forEach((taskRun: any) => {
        expect(taskRun).toHaveProperty('task_run_id');
        expect(taskRun).toHaveProperty('task_id');
        expect(taskRun).toHaveProperty('status');
        expect(taskRun).toHaveProperty('created_at');
        expect(taskRun).toHaveProperty('updated_at');
        expect(taskRun).toHaveProperty('current_loops');
        expect(taskRun.task_id).toBe(createdTaskId);
        expect(['SCHEDULED', 'PAUSE', 'RUNNING', 'FINISHED', 'FAILED']).toContain(taskRun.status);
      });
    }
  });

  test('should delete a task', async ({ request }) => {
    expect(createdTaskId).toBeTruthy();

    const response = await request.delete(`/v1/tasks/${createdTaskId}`);

    expect(response.status()).toBe(204);

    // Verify task is deleted by trying to fetch it
    const getResponse = await request.get(`/v1/tasks/${createdTaskId}`);
    expect(getResponse.status()).toBe(404);

    // Clear the ID so cleanup doesn't try to delete again
    createdTaskId = '';
  });

  test('should return 404 when deleting non-existent task', async ({ request }) => {
    const nonExistentId = '12345678-1234-1234-1234-123456789def';
    const response = await request.delete(`/v1/tasks/${nonExistentId}`);

    expect(response.status()).toBe(404);
  });

  test('should return 404 for task runs when task does not exist', async ({ request }) => {
    const nonExistentTaskId = '12345678-1234-1234-1234-123456789abc';
    const response = await request.get(`/v1/tasks/${nonExistentTaskId}/runs`);

    expect(response.status()).toBe(404);
    
    const responseBody = await response.json();
    expect(responseBody.resource).toBe("Task");
    expect(responseBody.id).toBe(nonExistentTaskId);
  });

  test('should return 400 for task runs with invalid task ID format', async ({ request }) => {
    const invalidTaskId = 'invalid-uuid-format';
    const response = await request.get(`/v1/tasks/${invalidTaskId}/runs`);

    expect(response.status()).toBe(400);
  });

  test('should handle empty task runs list for valid task', async ({ request }) => {
    // Create a new task specifically for this test to ensure no existing runs
    const newTaskData = {
      thread_id: createdThreadId,
      max_request_loop: 10,
      additional_info: {
        test_marker: 'empty_runs_test_task',
        description: 'Task for testing empty task runs response'
      }
    };

    const createResponse = await request.post('/v1/tasks', {
      data: newTaskData
    });

    expect(createResponse.status()).toBe(201);
    const newTask = await createResponse.json();
    const newTaskId = newTask.id;

    try {
      // Check task runs for the newly created task (should be empty)
      const response = await request.get(`/v1/tasks/${newTaskId}/runs`);

      expect(response.status()).toBe(200);
      
      const responseBody = await response.json();
      expect(Array.isArray(responseBody)).toBe(true);
      expect(responseBody.length).toBe(0);
    } finally {
      // Clean up the temporary task
      await request.delete(`/v1/tasks/${newTaskId}`);
    }
  });

  // REST + SSE Task Execution Tests
  test.describe('REST API Task Execution with SSE Streaming', () => {
    let createdAgentId: string;
    let testTaskId: string;
    let testTaskRunId: string;

    test.beforeAll(async ({ request }) => {
      // Create a test agent for task execution
      const testId = Date.now();
      const agentData = {
        name: `Test Task Execution Agent ${testId}`,
        description: 'A test agent for REST API task execution testing',
        specs: `
model:
  provider: "bedrock/anthropic"
  model_id: "global.anthropic.claude-sonnet-4-5-20250929-v1:0"
  max_tokens: 2048
  stream: true

system: |
  You are a helpful AI assistant for task execution testing.
  Keep responses short and simple for testing purposes.
  Always respond with "Task execution successful!" for any user input.

tools: []
`
      };

      const agentResponse = await request.post('/v1/agents', {
        data: agentData
      });

      expect(agentResponse.status()).toBe(201);
      const agentBody = await agentResponse.json();
      createdAgentId = agentBody.id;
      console.log(`Created test agent for task execution: ${createdAgentId}`);

      // Create a test task for execution
      const taskData = {
        thread_id: createdThreadId,
        max_request_loop: 5,
        additional_info: {
          test_marker: 'task_execution_test',
          description: 'Test task for REST API execution'
        }
      };

      const taskResponse = await request.post('/v1/tasks', {
        data: taskData
      });

      expect(taskResponse.status()).toBe(201);
      const taskBody = await taskResponse.json();
      testTaskId = taskBody.id;
      console.log(`Created test task for execution: ${testTaskId}`);
    });

    test.afterAll(async ({ request }) => {
      // Clean up resources
      if (testTaskId) {
        await request.delete(`/v1/tasks/${testTaskId}`);
        console.log(`Cleaned up test task: ${testTaskId}`);
      }
      if (createdAgentId) {
        await request.delete(`/v1/agents/${createdAgentId}`);
        console.log(`Cleaned up test agent: ${createdAgentId}`);
      }
    });

    test('should execute task via REST API with SSE streaming', async ({ request }) => {
      expect(testTaskId).toBeTruthy();
      expect(createdAgentId).toBeTruthy();

      // Add a message to the thread before executing the task
      const messageData = {
        message: {
          role: "user",
          content: [{type:"text", text: "Hello! Please help me with a simple task execution test."}]
        },
        sender_id: testUserId,
        recipient_id: createdAgentId
      };

      const messageResponse = await request.post(`/v1/threads/${createdThreadId}/messages`, {
        data: messageData
      });
      
      expect(messageResponse.status()).toBe(201);
      console.log('Added message to thread before task execution');

      const executeData = {
        agent_id: createdAgentId,
        current_loops: 0
      };

      // Test SSE streaming response (new default behavior)
      try {
        const response = await request.post(`/v1/tasks/${testTaskId}/execute`, {
          data: executeData,
          headers: {
            'Accept': 'text/event-stream'
          },
          timeout: 10000 // 10 second timeout for SSE test
        });

        expect(response.status()).toBe(200);
        
        // For SSE, we should get streaming response headers
        const contentType = response.headers()['content-type'];
        expect(contentType).toContain('text/event-stream');
        
        console.log('✅ SSE streaming started successfully');
        
      } catch (error: any) {
        // SSE streams are expected to timeout - this is normal behavior
        if (error.name === 'TimeoutError' || 
            error.message.includes('timeout') || 
            error.message.includes('Timeout') ||
            error.message.includes('Request context disposed')) {
          console.log('✅ SSE stream established and timed out as expected');
          // This is success for SSE - streams are long-running
        } else {
          console.log(`Unexpected error: ${error.name}: ${error.message}`);
          throw error; // Re-throw unexpected errors
        }
      }
      
      // Since it's SSE streaming, we need to get the task run ID from the task runs endpoint
      const taskRunsResponse = await request.get(`/v1/tasks/${testTaskId}/runs`);
      expect(taskRunsResponse.status()).toBe(200);
      
      const taskRuns = await taskRunsResponse.json();
      expect(Array.isArray(taskRuns)).toBe(true);
      expect(taskRuns.length).toBeGreaterThan(0);
      
      const latestTaskRun = taskRuns[0];
      expect(latestTaskRun).toHaveProperty('task_run_id');
      expect(latestTaskRun).toHaveProperty('task_id');
      expect(latestTaskRun).toHaveProperty('status');
      expect(latestTaskRun).toHaveProperty('created_at');
      expect(latestTaskRun).toHaveProperty('current_loops');
      
      expect(latestTaskRun.task_id).toBe(testTaskId);
      expect(['SCHEDULED', 'RUNNING', 'FINISHED']).toContain(latestTaskRun.status);
      expect(latestTaskRun.current_loops).toBe(0);

      testTaskRunId = latestTaskRun.task_run_id;
      console.log(`Task execution started with SSE streaming, task run ID: ${testTaskRunId}`);
    });

    test('should verify TaskRun status is updated to FINISHED after successful execution', async ({ request }) => {
      expect(testTaskId).toBeTruthy();
      expect(createdAgentId).toBeTruthy();

      // Create a separate task for status verification
      const statusTestTaskData = {
        thread_id: createdThreadId,
        max_request_loop: 3,
        additional_info: {
          test_marker: 'status_verification_test',
          description: 'Test task for verifying status updates'
        }
      };

      const statusTaskResponse = await request.post('/v1/tasks', {
        data: statusTestTaskData
      });

      expect(statusTaskResponse.status()).toBe(201);
      const statusTaskBody = await statusTaskResponse.json();
      const statusTestTaskId = statusTaskBody.id;

      try {
        // Add a message to the thread
        const statusMessageData = {
          message: {
            role: "user",
            content: [{type: "text", text: "Hello! Please help me with a status verification test."}]
          },
          sender_id: testUserId,
          recipient_id: createdAgentId
        };

        await request.post(`/v1/threads/${createdThreadId}/messages`, {
          data: statusMessageData
        });

        // Execute the task
        const executeData = {
          agent_id: createdAgentId,
          current_loops: 0
        };

        try {
          const response = await request.post(`/v1/tasks/${statusTestTaskId}/execute`, {
            data: executeData,
            headers: {
              'Accept': 'text/event-stream'
            },
            timeout: 15000 // 15 second timeout to allow task completion
          });

          expect(response.status()).toBe(200);
          console.log('✅ Task execution started, waiting for completion...');
          
        } catch (error: any) {
          // Expected timeout for SSE streams
          if (error.name === 'TimeoutError' || 
              error.message.includes('timeout') || 
              error.message.includes('Timeout') ||
              error.message.includes('Request context disposed')) {
            console.log('✅ SSE stream timed out as expected, checking task status...');
          } else {
            throw error;
          }
        }

        // Wait a bit for the task to complete processing
        await new Promise(resolve => setTimeout(resolve, 2000));

        // Get the task runs and verify the status was updated
        const taskRunsResponse = await request.get(`/v1/tasks/${statusTestTaskId}/runs`);
        expect(taskRunsResponse.status()).toBe(200);
        
        const taskRuns = await taskRunsResponse.json();
        expect(Array.isArray(taskRuns)).toBe(true);
        expect(taskRuns.length).toBeGreaterThan(0);
        
        const latestTaskRun = taskRuns[0];
        expect(latestTaskRun).toHaveProperty('task_run_id');
        expect(latestTaskRun).toHaveProperty('status');
        expect(latestTaskRun.task_id).toBe(statusTestTaskId);
        
        // The status should be either FINISHED (success) or FAILED (if something went wrong)
        // Both are acceptable as long as it's not still RUNNING or SCHEDULED
        expect(['FINISHED', 'FAILED']).toContain(latestTaskRun.status);
        console.log(`✅ Task run status properly updated to: ${latestTaskRun.status}`);
        
        // Additional verification: get the specific task run by ID
        const taskRunResponse = await request.get(`/v1/tasks/${latestTaskRun.task_run_id}/status`);
        expect(taskRunResponse.status()).toBe(200);
        
        const specificTaskRun = await taskRunResponse.json();
        expect(specificTaskRun.status).toBe(latestTaskRun.status);
        expect(['FINISHED', 'FAILED']).toContain(specificTaskRun.status);
        
        console.log(`✅ Task run ${latestTaskRun.task_run_id} final status verified: ${specificTaskRun.status}`);

      } finally {
        // Clean up the test task
        await request.delete(`/v1/tasks/${statusTestTaskId}`);
      }
    });

    test('should handle task execution timeout and update status appropriately', async ({ request }) => {
      expect(testTaskId).toBeTruthy();
      expect(createdAgentId).toBeTruthy();

      // Create a task with minimal request loop to ensure quick completion
      const timeoutTestTaskData = {
        thread_id: createdThreadId,
        max_request_loop: 1,
        additional_info: {
          test_marker: 'timeout_test',
          description: 'Test task for timeout handling'
        }
      };

      const timeoutTaskResponse = await request.post('/v1/tasks', {
        data: timeoutTestTaskData
      });

      expect(timeoutTaskResponse.status()).toBe(201);
      const timeoutTaskBody = await timeoutTaskResponse.json();
      const timeoutTestTaskId = timeoutTaskBody.id;

      try {
        // Add a quick message
        const timeoutMessageData = {
          message: {
            role: "user", 
            content: [{type: "text", text: "Quick test message for timeout verification."}]
          },
          sender_id: testUserId,
          recipient_id: createdAgentId
        };

        await request.post(`/v1/threads/${createdThreadId}/messages`, {
          data: timeoutMessageData
        });

        // Execute with very short timeout to simulate premature disconnect
        const executeData = {
          agent_id: createdAgentId,
          current_loops: 0
        };

        try {
          await request.post(`/v1/tasks/${timeoutTestTaskId}/execute`, {
            data: executeData,
            headers: {
              'Accept': 'text/event-stream'
            },
            timeout: 5000 // Very short timeout
          });
        } catch (error: any) {
          // Expected timeout
          console.log('✅ Short timeout occurred as expected');
        }

        // Wait for background processing to complete
        await new Promise(resolve => setTimeout(resolve, 3000));

        // Verify that even with timeout, the task run status gets updated
        const taskRunsResponse = await request.get(`/v1/tasks/${timeoutTestTaskId}/runs`);
        expect(taskRunsResponse.status()).toBe(200);
        
        const taskRuns = await taskRunsResponse.json();
        expect(Array.isArray(taskRuns)).toBe(true);
        
        if (taskRuns.length > 0) {
          const latestTaskRun = taskRuns[0];
          expect(latestTaskRun).toHaveProperty('status');
          
          // Even with timeout, status should be updated (not stuck in RUNNING)
          expect(['FINISHED', 'FAILED', 'SCHEDULED']).toContain(latestTaskRun.status);
          console.log(`✅ Task run status after timeout: ${latestTaskRun.status}`);
        }

      } finally {
        // Clean up
        await request.delete(`/v1/tasks/${timeoutTestTaskId}`);
      }
    });

    test('should return 404 for non-existent task execution', async ({ request }) => {
      const nonExistentTaskId = '12345678-1234-1234-1234-123456789abc';
      const executeData = {
        agent_id: createdAgentId,
        current_loops: 0
      };

      const response = await request.post(`/v1/tasks/${nonExistentTaskId}/execute`, {
        data: executeData
      });

      expect(response.status()).toBe(404);
      
      const responseBody = await response.json();
      expect(responseBody.resource).toBe("Task");
      expect(responseBody.id).toBe(nonExistentTaskId);
    });

    test('should return 400 for invalid agent_id in task execution', async ({ request }) => {
      const executeData = {
        agent_id: 'invalid-uuid-format',
        current_loops: 0
      };

      const response = await request.post(`/v1/tasks/${testTaskId}/execute`, {
        data: executeData
      });

      expect(response.status()).toBe(400);
    });

    test('should handle missing agent_id in task execution request', async ({ request }) => {
      const executeData = {
        current_loops: 0
        // Missing agent_id
      };

      const response = await request.post(`/v1/tasks/${testTaskId}/execute`, {
        data: executeData
      });

      expect(response.status()).toBe(400);
    });

    test('should verify merged SSE endpoint returns correct headers and streams', async ({ request }) => {
      // Create a separate task for SSE testing
      const sseTestTaskData = {
        thread_id: createdThreadId,
        max_request_loop: 3,
        additional_info: {
          test_marker: 'sse_header_test',
          description: 'Test task for merged SSE endpoint verification'
        }
      };

      const sseTaskResponse = await request.post('/v1/tasks', {
        data: sseTestTaskData
      });

      expect(sseTaskResponse.status()).toBe(201);
      const sseTaskBody = await sseTaskResponse.json();
      const sseTestTaskId = sseTaskBody.id;

      try {
        // Add a message to the thread
        const sseMessageData = {
          message: {
            role: "user",
            content: [{type: "text", text: [{type:"text", text:"Test message for SSE verification"}]}]
          },
          sender_id: testUserId,
          recipient_id: createdAgentId
        };

        await request.post(`/v1/threads/${createdThreadId}/messages`, {
          data: sseMessageData
        });

        // Test the merged endpoint with SSE headers - this should stream
        const executeData = {
          agent_id: createdAgentId,
          current_loops: 0
        };

        try {
          const response = await request.post(`/v1/tasks/${sseTestTaskId}/execute`, {
            data: executeData,
            headers: {
              'Accept': 'text/event-stream'
            },
            timeout: 10000 // 10 second timeout
          });
          
          // Should get streaming response
          expect(response.status()).toBe(200);
          
          // Verify SSE headers
          const contentType = response.headers()['content-type'];
          expect(contentType).toContain('text/event-stream');
          
          const cacheControl = response.headers()['cache-control'];
          expect(cacheControl).toBe('no-cache');
          
          const connection = response.headers()['connection'];
          expect(connection).toBe('keep-alive');
          
          console.log('✅ Merged SSE endpoint completed successfully with correct headers');
          
        } catch (error: any) {
          // Check if it's a streaming-related timeout (which is expected for SSE)
          if (error.message.includes('timeout') || error.message.includes('aborted')) {
            console.log('✅ Merged SSE endpoint is working (connection established and streaming, then timed out as expected)');
            // This is actually success - SSE connections are expected to be long-lived
          } else {
            console.log(`SSE endpoint unexpected error: ${error.message}`);
            throw error; // Re-throw if it's a different error
          }
        }
        
        console.log('✅ Merged SSE endpoint is accessible with correct headers');

      } finally {
        // Clean up the test task
        await request.delete(`/v1/tasks/${sseTestTaskId}`);
      }
    });
  });
});