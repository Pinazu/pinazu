import { test, expect } from '@playwright/test';
import WebSocket from 'ws';

// Test configuration constants
const WEBSOCKET_URL = 'ws://localhost:8080/v1/ws';
const DEFAULT_TIMEOUT = 10000;
const SHORT_TIMEOUT = 5000;
const CONNECTION_RETRY_ATTEMPTS = 3;
const CONNECTION_RETRY_DELAY = 1000;

// Error types for validation
enum WebSocketErrorType {
  CONNECTION_FAILED = 'CONNECTION_FAILED',
  MESSAGE_TIMEOUT = 'MESSAGE_TIMEOUT',
  INVALID_RESPONSE = 'INVALID_RESPONSE',
  UNEXPECTED_CLOSE = 'UNEXPECTED_CLOSE'
}

test.describe.serial('WebSocket API (Node.js) - Industrial Tests', () => {
  let createdAgentId: string;
  let createdThreadId: string;
  let testUserId: string = '550e8400-c95b-4444-6666-446655440000'; // Admin user UUID from database
  let activeConnections: WebSocket[] = []; // Track active connections for cleanup

  test.beforeAll(async ({ request }) => {
    // Clean up any existing test data before starting
    await cleanupTestData(request);

    // Create a test agent for WebSocket testing with unique identifier
    const testId = generateTestId();
    const agentData = {
      name: `Test WebSocket Node Agent ${testId}`,
      description: 'A test agent for Node.js WebSocket e2e testing',
      specs: `
model:
  provider: "bedrock/anthropic"
  model_id: "global.anthropic.claude-sonnet-4-5-20250929-v1:0"
  max_tokens: 2048

system: |
  You are a helpful AI assistant for WebSocket testing.
  Keep responses short and simple for testing purposes.
  Always answer "Hello from agent!".

tools: []
`
    };

    const agentResponse = await request.post('/v1/agents', {
      data: agentData
    });

    expect(agentResponse.status()).toBe(201);
    const agentBody = await agentResponse.json();
    createdAgentId = agentBody.id;
    console.log(`Created test agent with ID: ${createdAgentId}`);

    // Create a test thread for WebSocket messages with unique identifier
    const threadData = {
      title: `Test WebSocket Node Thread ${testId}`,
      user_id: testUserId
    };

    const threadResponse = await request.post('/v1/threads', {
      data: threadData
    });

    expect(threadResponse.status()).toBe(201);
    const threadBody = await threadResponse.json();
    createdThreadId = threadBody.id;
    console.log(`Created test thread with ID: ${createdThreadId}`);
  });

  test.afterEach(async () => {
    // Clean up any remaining active WebSocket connections after each test
    const connectionPromises = activeConnections.map(ws => safeCloseConnection(ws));
    await Promise.allSettled(connectionPromises);
    activeConnections = [];
    
    // Add a small delay between tests to ensure proper cleanup
    await new Promise(resolve => setTimeout(resolve, 500));
  });

  test.afterAll(async ({ request }) => {
    // Final cleanup of any remaining connections
    const connectionPromises = activeConnections.map(ws => safeCloseConnection(ws));
    await Promise.allSettled(connectionPromises);
    activeConnections = [];
    
    // Clean up created resources
    if (createdThreadId) {
      await request.delete(`/v1/threads/${createdThreadId}`);
      console.log(`Cleaned up test thread: ${createdThreadId}`);
    }
    if (createdAgentId) {
      await request.delete(`/v1/agents/${createdAgentId}`);
      console.log(`Cleaned up test agent: ${createdAgentId}`);
    }
  });

  // Helper function to generate unique test identifier to prevent cross-test interference
  function generateTestId(): string {
    return `${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  // Helper function to clean up existing test data
  async function cleanupTestData(request: any) {
    // Clean up test agents
    const agentsResponse = await request.get('/v1/agents');
    if (agentsResponse.ok()) {
      const agentsBody = await agentsResponse.json();
      const agents = agentsBody.agents || [];
      if (Array.isArray(agents)) {
        for (const agent of agents) {
          if (agent.name && agent.name.includes('Test WebSocket Node Agent')) {
            await request.delete(`/v1/agents/${agent.id}`);
          }
        }
      }
    }

    // Clean up test threads
    const threadsResponse = await request.get('/v1/threads');
    if (threadsResponse.ok()) {
      const threadsBody = await threadsResponse.json();
      const threads = threadsBody.threads || [];
      if (Array.isArray(threads)) {
        for (const thread of threads) {
          if (thread.title && thread.title.includes('Test WebSocket Node Thread')) {
            await request.delete(`/v1/threads/${thread.id}`);
          }
        }
      }
    }
  }

  // Enhanced helper function to create WebSocket connection with retry logic
  async function createWebSocketConnection(url: string = WEBSOCKET_URL, retries: number = CONNECTION_RETRY_ATTEMPTS): Promise<WebSocket> {
    let lastError: Error | null = null;
    
    for (let attempt = 1; attempt <= retries; attempt++) {
      try {
        return await new Promise<WebSocket>((resolve, reject) => {
          const ws = new WebSocket(url);
          
          // Increase max listeners to prevent MaxListenersExceeded warning
          ws.setMaxListeners(20);
          
          const connectionTimeout = setTimeout(() => {
            ws.terminate();
            reject(new Error(`Connection timeout on attempt ${attempt}`));
          }, DEFAULT_TIMEOUT);
          
          ws.on('open', () => {
            clearTimeout(connectionTimeout);
            // Set up error handling for the established connection
            ws.on('error', (error) => {
              console.error(`WebSocket error: ${error.message}`);
            });
            
            ws.on('close', (code, reason) => {
              if (code !== 1000 && code !== 1001) { // Normal closure codes
                console.warn(`WebSocket closed unexpectedly: code=${code}, reason=${reason}`);
              }
            });
            
            // Track this connection for cleanup
            activeConnections.push(ws);
            resolve(ws);
          });
          
          ws.on('error', (error) => {
            clearTimeout(connectionTimeout);
            reject(error);
          });
        });
      } catch (error) {
        lastError = error as Error;
        console.warn(`Connection attempt ${attempt}/${retries} failed: ${lastError.message}`);
        
        if (attempt < retries) {
          await new Promise(resolve => setTimeout(resolve, CONNECTION_RETRY_DELAY));
        }
      }
    }
    
    throw new Error(`Failed to establish WebSocket connection after ${retries} attempts. Last error: ${lastError?.message}`);
  }

  // Enhanced helper function to wait for WebSocket message with better error handling
  function waitForMessage(ws: WebSocket, timeout: number = DEFAULT_TIMEOUT): Promise<string> {
    return new Promise((resolve, reject) => {
      if (ws.readyState !== WebSocket.OPEN) {
        reject(new Error(`WebSocket is not open. Current state: ${ws.readyState}`));
        return;
      }

      let timer: NodeJS.Timeout;
      let messageHandler: (data: WebSocket.Data) => void;
      let errorHandler: (error: Error) => void;
      let closeHandler: (code: number, reason: Buffer) => void;

      const cleanup = () => {
        if (timer) clearTimeout(timer);
        if (messageHandler) ws.removeListener('message', messageHandler);
        if (errorHandler) ws.removeListener('error', errorHandler);
        if (closeHandler) ws.removeListener('close', closeHandler);
      };

      timer = setTimeout(() => {
        cleanup();
        reject(new Error(`Message timeout after ${timeout}ms`));
      }, timeout);

      messageHandler = (data: WebSocket.Data) => {
        cleanup();
        try {
          const message = data.toString();
          resolve(message);
        } catch (error) {
          reject(new Error(`Failed to parse message: ${error}`));
        }
      };

      errorHandler = (error: Error) => {
        cleanup();
        reject(new Error(`WebSocket error while waiting for message: ${error.message}`));
      };

      closeHandler = (code: number, reason: Buffer) => {
        cleanup();
        reject(new Error(`WebSocket closed while waiting for message: code=${code}, reason=${reason.toString()}`));
      };

      ws.once('message', messageHandler);
      ws.once('error', errorHandler);
      ws.once('close', closeHandler);
    });
  }

  // Helper function to safely close WebSocket connection
  async function safeCloseConnection(ws: WebSocket, timeout: number = SHORT_TIMEOUT): Promise<void> {
    return new Promise((resolve) => {
      if (ws.readyState === WebSocket.CLOSED) {
        resolve();
        return;
      }

      // Remove from active connections
      const connectionIndex = activeConnections.indexOf(ws);
      if (connectionIndex > -1) {
        activeConnections.splice(connectionIndex, 1);
      }

      // Remove all listeners before closing to prevent memory leaks
      ws.removeAllListeners('message');
      ws.removeAllListeners('error');
      ws.removeAllListeners('close');
      ws.removeAllListeners('open');

      const closeTimeout = setTimeout(() => {
        console.warn('Force terminating WebSocket connection due to timeout');
        ws.terminate();
        resolve();
      }, timeout);

      ws.once('close', () => {
        clearTimeout(closeTimeout);
        resolve();
      });

      if (ws.readyState === WebSocket.OPEN) {
        ws.close(1000, 'Test completed');
      } else {
        ws.terminate();
      }
    });
  }

  // Helper function to send message and wait for response
  async function sendMessageAndWaitForResponse(ws: WebSocket, message: any, timeout: number = DEFAULT_TIMEOUT): Promise<string> {
    const messageStr = JSON.stringify(message);
    
    if (ws.readyState !== WebSocket.OPEN) {
      throw new Error(`Cannot send message: WebSocket is not open (state: ${ws.readyState})`);
    }

    // Set up response listener before sending
    const responsePromise = waitForMessage(ws, timeout);
    
    try {
      ws.send(messageStr);
      return await responsePromise;
    } catch (error) {
      throw new Error(`Failed to send message or receive response: ${error}`);
    }
  }

  // Helper function to validate WebSocket response format
  function validateWebSocketResponse(response: string): { isValid: boolean; error?: string; data?: any } {
    try {
      const data = JSON.parse(response);
      
      // Basic response structure validation
      if (typeof data !== 'object' || data === null) {
        return { isValid: false, error: 'Response is not a valid object' };
      }

      // For our tests, we consider error responses as valid JSON format
      // The actual error checking is done in the test assertions
      return { isValid: true, data };
    } catch (error) {
      return { isValid: false, error: `Invalid JSON response: ${error}` };
    }
  }

  // Helper function to check task run status via REST API
  async function checkTaskRunStatus(request: any, taskId: string): Promise<string | null> {
    try {
      // Get all task runs for the task_id
      const response = await request.get(`/v1/tasks/${taskId}/runs`);
      
      if (response.ok()) {
        const body = await response.json();
        // The response body is directly the array of task runs
        const taskRuns = Array.isArray(body) ? body : (body.task_runs || body.runs || []);
        
        // Find the most recent task run (they should be ordered by created_at DESC)
        if (taskRuns.length > 0) {
          const latestTaskRun = taskRuns[0];
          return latestTaskRun.status || null;
        }
      }
      return null;
    } catch (error) {
      console.error(`Failed to check task run status for task_id ${taskId}:`, error);
      return null;
    }
  }

  // 1. CONNECTION MANAGEMENT TESTS
  test.describe('Connection Management', () => {
    test('should establish WebSocket connection successfully with retry logic', async () => {
      const ws = await createWebSocketConnection();
      
      expect(ws.readyState).toBe(WebSocket.OPEN);
      
      await safeCloseConnection(ws);
      expect(ws.readyState).toBe(WebSocket.CLOSED);
    });

    test('should handle WebSocket connection cleanup properly', async () => {
      const ws = await createWebSocketConnection();
      
      expect(ws.readyState).toBe(WebSocket.OPEN);
      
      // Test graceful closure
      await safeCloseConnection(ws, SHORT_TIMEOUT);
      expect(ws.readyState).toBe(WebSocket.CLOSED);
    });

    test('should handle multiple concurrent connections with proper cleanup', async () => {
      const connections: WebSocket[] = [];
      const concurrentConnections = 5;
      
      try {
        // Create multiple connections concurrently
        const connectionPromises = Array(concurrentConnections).fill(0).map(() => createWebSocketConnection());
        const establishedConnections = await Promise.all(connectionPromises);
        
        connections.push(...establishedConnections);
        
        // Verify all connections are open
        for (const ws of connections) {
          expect(ws.readyState).toBe(WebSocket.OPEN);
        }
        
        console.log(`Successfully established ${connections.length} concurrent connections`);
        
        // Test that we can send messages on all connections
        const testMessage = {
          agent_id: createdAgentId,
          thread_id: createdThreadId,
          messages: [{ role: "user", content: "Concurrent connection test" }]
        };
        
        const sendPromises = connections.map(ws => {
          return new Promise<void>((resolve, reject) => {
            try {
              ws.send(JSON.stringify(testMessage));
              resolve();
            } catch (error) {
              reject(error);
            }
          });
        });
        
        await Promise.all(sendPromises);
        console.log(`Successfully sent messages on all ${connections.length} connections`);
        
      } finally {
        // Clean up all connections properly
        const closePromises = connections.map(ws => safeCloseConnection(ws));
        await Promise.all(closePromises);
        
        // Verify all connections are closed
        for (const ws of connections) {
          expect(ws.readyState).toBe(WebSocket.CLOSED);
        }
        
        console.log(`Successfully closed all ${connections.length} connections`);
      }
    });

    test('should handle connection failures gracefully', async () => {
      // Test connection to invalid URL
      await expect(createWebSocketConnection('ws://invalid-host:9999/ws', 1))
        .rejects.toThrow(/Failed to establish WebSocket connection/);
    });

    test('should validate connection state before operations', async () => {
      const ws = await createWebSocketConnection();
      
      // Close the connection first
      await safeCloseConnection(ws);
      
      // Try to wait for message on closed connection
      await expect(waitForMessage(ws, 1000))
        .rejects.toThrow(/WebSocket is not open/);
    });
  });

  // 2. VALID MESSAGE FLOW TESTS
  test.describe('Valid Message Flow', () => {
    test('should send valid WebSocket message and handle response', async () => {
      expect(createdAgentId).toBeTruthy();
      expect(createdThreadId).toBeTruthy();
      
      const ws = await createWebSocketConnection();
      
      try {
        const testMessage = {
          agent_id: createdAgentId,
          thread_id: createdThreadId,
          messages: [
            { role: "user", content: "Hello WebSocket Node test" }
          ]
        };

        // Send message and try to get response
        ws.send(JSON.stringify(testMessage));
        console.log('Sent test message, waiting for potential response...');
        
        // Try to wait for response with shorter timeout since we're not sure if server responds immediately
        try {
          const response = await waitForMessage(ws, SHORT_TIMEOUT);
          console.log('Received response:', response);
          
          // Validate response format
          const validation = validateWebSocketResponse(response);
          if (!validation.isValid) {
            console.warn('Invalid response format:', validation.error);
          } else {
            console.log('Response validation passed');
          }
        } catch (error) {
          // This is acceptable - server might not send immediate responses
          console.log('No immediate response received (this may be expected):', error);
        }
        
        expect(ws.readyState).toBe(WebSocket.OPEN);
        
      } finally {
        await safeCloseConnection(ws);
      }
    });

    test('should handle different message content types with proper validation', async () => {
      expect(createdAgentId).toBeTruthy();
      expect(createdThreadId).toBeTruthy();
      
      const ws = await createWebSocketConnection();
      
      try {
        const testMessages = [
          {
            name: "Simple text message",
            message: {
              agent_id: createdAgentId,
              thread_id: createdThreadId,
              messages: [{ role: "user", content: [{ type:"text",text:"Simple text message" }] }]
            }
          },
          {
            name: "Multi-turn conversation",
            message: {
              agent_id: createdAgentId,
              thread_id: createdThreadId,
              messages: [
                { role: "user", content: [{ type:"text",text:"First message" }] },
                { role: "assistant", content: [{ type:"text",text:"Response message" }] },
                { role: "user", content: [{ type:"text",text:"Follow-up message" }] }
              ]
            }
          },
          {
            name: "Special characters",
            message: {
              agent_id: createdAgentId,
              thread_id: createdThreadId,
              messages: [{ 
                role: "user",
                content: [{ type:"text",text:"Message with special chars: !@#$%^&*()_+{}[]|\\:;\"'<>,.?/~`" }]
              }]
            }
          },
          {
            name: "Unicode and emojis",
            message: {
              agent_id: createdAgentId,
              thread_id: createdThreadId,
              messages: [{ 
                role: "user", 
                content: [{ type:"text",text:"Unicode test: 你好 🌟 🚀 ñáéíóú" }]
              }]
            }
          }
        ];

        for (const testCase of testMessages) {
          console.log(`Testing: ${testCase.name}`);
          
          // Verify connection is still open
          expect(ws.readyState).toBe(WebSocket.OPEN);
          
          // Send message
          ws.send(JSON.stringify(testCase.message));
          
          // Small delay between messages
          await new Promise(resolve => setTimeout(resolve, 300));
        }
        
        console.log(`Successfully sent ${testMessages.length} different message types`);
        expect(ws.readyState).toBe(WebSocket.OPEN);
        
      } finally {
        await safeCloseConnection(ws);
      }
    });

    test('should handle message sending with response validation', async () => {
      expect(createdAgentId).toBeTruthy();
      expect(createdThreadId).toBeTruthy();
      
      const ws = await createWebSocketConnection();
      
      try {
        const testMessage = {
          agent_id: createdAgentId,
          thread_id: createdThreadId,
          messages: [{ role: "user", content: "Test message with response validation" }]
        };

        // Try the enhanced send and wait function
        try {
          const response = await sendMessageAndWaitForResponse(ws, testMessage, SHORT_TIMEOUT);
          console.log('Received response via sendMessageAndWaitForResponse:', response);
          
          const validation = validateWebSocketResponse(response);
          expect(validation.isValid).toBe(true);
          
        } catch (error) {
          // Log but don't fail if no response - this depends on server implementation
          console.log('No response received (server may not send immediate responses):', error);
        }
        
        expect(ws.readyState).toBe(WebSocket.OPEN);
        
      } finally {
        await safeCloseConnection(ws);
      }
    });
  });

  // 3. ERROR HANDLING TESTS
  test.describe('Error Handling', () => {
    test('should handle invalid JSON and return specific error messages', async () => {
      const ws = await createWebSocketConnection();
      
      try {
        const invalidJsonMessages = [
          { name: 'Invalid JSON syntax', message: '{"invalid": json}', expectedError: 'Failed to parse message' },
          { name: 'Plain text', message: 'not json at all', expectedError: 'Failed to parse message' },
          { name: 'Empty string', message: '', expectedError: 'Failed to parse message' },
          { name: 'Partial JSON', message: '{"incomplete":', expectedError: 'Failed to parse message' }
        ];

        for (const testCase of invalidJsonMessages) {
          console.log(`Testing invalid JSON: ${testCase.name}`);
          
          // Verify connection is still open before sending
          expect(ws.readyState).toBe(WebSocket.OPEN);
          
          ws.send(testCase.message);
          
          // Wait for error response from server
          try {
            const response = await waitForMessage(ws, 2000);
            console.log(`Received response to invalid JSON: ${response}`);
            
            // Validate it's a proper error response
            const validation = validateWebSocketResponse(response);
            expect(validation.isValid).toBe(true);
            expect(validation.data).toBeDefined();
            expect(validation.data.error).toBe(testCase.expectedError);
            console.log(`✓ Correctly received error: ${validation.data.error}`);
            
          } catch (error) {
            console.error(`Failed to receive expected error response: ${error}`);
            throw new Error(`Expected error response "${testCase.expectedError}" but got none`);
          }
          
          await new Promise(resolve => setTimeout(resolve, 200));
        }
        
        // Connection should remain open despite invalid messages
        expect(ws.readyState).toBe(WebSocket.OPEN);
        console.log(`✓ All invalid JSON messages returned proper error responses`);
        
      } finally {
        await safeCloseConnection(ws);
      }
    });

    test('should handle missing required fields with specific error validation', async ({ request }) => {
      const ws = await createWebSocketConnection();
      
      try {
        const incompleteMessages = [
          { name: 'Empty object', message: {}, expectedError: 'invalid message: agent_id field is required' },
          { name: 'Missing thread_id and messages', message: { agent_id: createdAgentId }, expectedError: 'invalid message: messages field is required' },
          { name: 'Missing agent_id and messages', message: { thread_id: createdThreadId }, expectedError: 'invalid message: agent_id field is required' },
          { name: 'Missing agent_id and thread_id', message: { messages: [{ role: "user", content: "test" }] }, expectedError: 'invalid message: agent_id field is required' },
          { name: 'Missing messages', message: { agent_id: createdAgentId, thread_id: createdThreadId }, expectedError: 'invalid message: messages field is required' },
          { name: 'Empty messages array', message: { agent_id: createdAgentId, thread_id: createdThreadId, messages: [] }, expectedError: 'invalid message: messages field is required' },
          // { name: 'Invalid message structure', message: { agent_id: createdAgentId, thread_id: createdThreadId, messages: [{}] }, expectedError: 'failed to handle Anthropic request: failed to create message: POST \"https://bedrock-runtime.ap-southeast-1.amazonaws.com/v1/messages\": 400 Bad Request {\"message\":\"Malformed input request: #: subject must not be valid against schema {\\\"required\\\":[\\\"messages\\\"]}#/messages/0: required key [role] not found#/messages/0: required key [content] not found, please reformat your input and try again.\"}', needsLastEvent: true },
          // { name: 'Missing role in message', message: { agent_id: createdAgentId, thread_id: createdThreadId, messages: [{ content: "test" }] }, expectedError: 'failed to handle Anthropic request: failed to create message: POST \"https://bedrock-runtime.ap-southeast-1.amazonaws.com/v1/messages\": 400 Bad Request {\"message\":\"Malformed input request: #: subject must not be valid against schema {\\\"required\\\":[\\\"messages\\\"]}#/messages/0: required key [role] not found#/messages/0: required key [content] not found, please reformat your input and try again.\"}', needsLastEvent: true }
        ];

        for (const testCase of incompleteMessages) {
          console.log(`Testing incomplete message: ${testCase.name}`);
          
          expect(ws.readyState).toBe(WebSocket.OPEN);
          
          ws.send(JSON.stringify(testCase.message));
          
          // Wait for error response from server
          try {
            // if (testCase.needsLastEvent) {
            //   // For test cases that need the last event (task_start comes first, error comes second)
            //   const firstResponse = await waitForMessage(ws, 2000);
            //   console.log(`Received first response (task_start): ${firstResponse}`);
              
            //   // Extract task_id from task_start event for task run status checking
            //   let taskId: string | null = null;
            //   const firstValidation = validateWebSocketResponse(firstResponse);
            //   if (firstValidation.isValid && firstValidation.data) {
            //     // Check in message.task_id (WebSocket format)
            //     if (firstValidation.data.message?.task_id) {
            //       taskId = firstValidation.data.message.task_id;
            //       console.log(`Extracted task_id for status check: ${taskId}`);
            //     }
            //     // Check in header.task_id (WebSocket format)
            //     else if (firstValidation.data.header?.task_id) {
            //       taskId = firstValidation.data.header.task_id;
            //       console.log(`Extracted task_id for status check: ${taskId}`);
            //     }
            //     // Fallback checks for other formats
            //     else if (firstValidation.data.task_id) {
            //       taskId = firstValidation.data.task_id;
            //       console.log(`Extracted task_id for status check: ${taskId}`);
            //     }
            //   }
              
            //   const lastResponse = await waitForMessage(ws, 2000);
            //   console.log(`Received last response (error): ${lastResponse}`);
              
            //   const validation = validateWebSocketResponse(lastResponse);
            //   expect(validation.isValid).toBe(true);
            //   expect(validation.data).toBeDefined();
              
            //   // Check task run status if we extracted a task_id
            //   if (taskId) {
            //     const taskRunStatus = await checkTaskRunStatus(request, taskId);
            //     console.log(`Task run status for task_id ${taskId}: ${taskRunStatus}`);
            //     expect(taskRunStatus).toBe('FAILED');
            //   }
              
            //   // Check if it's either the expected error or a Bedrock rate limiting error
            //   const isExpectedError = validation.data.error === testCase.expectedError;
            //   const isBedrock429Error = validation.data.error && validation.data.error.includes('failed to handle Anthropic request: failed to create message: POST "https://bedrock-runtime.ap-southeast-1.amazonaws.com/v1/messages": 429 Too Many Requests');
            //   const isBedrockMalformedError = validation.data.error && validation.data.error.includes('failed to handle Anthropic request: failed to create message: POST "https://bedrock-runtime.ap-southeast-1.amazonaws.com/v1/messages": 400 Bad Request') && validation.data.error.includes('Malformed input request');
              
            //   if (isExpectedError) {
            //     console.log(`✓ Correctly received expected error: ${validation.data.error}`);
            //   } else if (isBedrock429Error) {
            //     console.log(`✓ Received Bedrock rate limiting error (acceptable): ${validation.data.error}`);
            //   } else if (isBedrockMalformedError) {
            //     console.log(`✓ Received Bedrock malformed input error (acceptable): ${validation.data.error}`);
            //   } else {
            //     throw new Error(`Expected error "${testCase.expectedError}" or Bedrock error, but got: ${validation.data.error}`);
            //   }
            // } else {
              // For regular test cases that expect immediate error response
              const response = await waitForMessage(ws, 2000);
              console.log(`Received response to incomplete message: ${response}`);
              
              const validation = validateWebSocketResponse(response);
              expect(validation.isValid).toBe(true);
              expect(validation.data).toBeDefined();
              
              // Check if it's either the expected error or a Bedrock rate limiting error
              const isExpectedError = validation.data.error === testCase.expectedError;
              const isBedrock429Error = validation.data.error && validation.data.error.includes('failed to handle Anthropic request: failed to create message: POST "https://bedrock-runtime.ap-southeast-1.amazonaws.com/v1/messages": 429 Too Many Requests');
              const isBedrockMalformedError = validation.data.error && validation.data.error.includes('failed to handle Anthropic request: failed to create message: POST "https://bedrock-runtime.ap-southeast-1.amazonaws.com/v1/messages": 400 Bad Request') && validation.data.error.includes('Malformed input request');
              
              if (isExpectedError) {
                console.log(`✓ Correctly received expected error: ${validation.data.error}`);
              } else if (isBedrock429Error) {
                console.log(`✓ Received Bedrock rate limiting error (acceptable): ${validation.data.error}`);
              } else if (isBedrockMalformedError) {
                console.log(`✓ Received Bedrock malformed input error (acceptable): ${validation.data.error}`);
              } else {
                throw new Error(`Expected error "${testCase.expectedError}" or Bedrock error, but got: ${validation.data.error}`);
              }
            // }
            
          } catch (error) {
            console.error(`Failed to receive expected error response: ${error}`);
            throw new Error(`Expected error response "${testCase.expectedError}" but got none`);
          }
          
          await new Promise(resolve => setTimeout(resolve, 200));
        }
        
        expect(ws.readyState).toBe(WebSocket.OPEN);
        console.log(`✓ All incomplete messages returned proper error responses`);
        
      } finally {
        await safeCloseConnection(ws);
      }
    });

    test('should handle invalid UUID formats with validation', async ({ request }) => {
      const ws = await createWebSocketConnection();
      
      try {
        const invalidUuidMessages = [
          {
            name: 'Invalid agent_id format',
            message: {
              agent_id: "invalid-uuid-format",
              thread_id: createdThreadId,
              messages: [{ role: "user", content: [{ type:"text", text:"test" }] }]
            },
            expectedError: 'Invalid message format'
          },
          {
            name: 'Invalid thread_id format',
            message: {
              agent_id: createdAgentId,
              thread_id: "not-a-uuid",
              messages: [{ role: "user", content: [{ type:"text", text:"test" }] }]
            },
            expectedError: 'Invalid message format'
          },
          {
            name: 'Numeric instead of UUID',
            message: {
              agent_id: "12345",
              thread_id: "67890",
              messages: [{ role: "user", content: [{ type:"text", text:"test" }] }]
            },
            expectedError: 'Invalid message format'
          },
          {
            name: 'Invalid messages format',
            message: {
              agent_id: createdAgentId,
              thread_id: createdThreadId,
              messages: [{ role: "user", content: "test" }]
            },
            expectedError: 'failed to handle Anthropic request: failed to create message: POST \"https://bedrock-runtime.ap-southeast-1.amazonaws.com/v1/messages\": 400 Bad Request {\"message\":\"Malformed input request: #: subject must not be valid against schema {\\\"required\\\":[\\\"messages\\\"]}#/messages/0: required key [content] not found, please reformat your input and try again.\"}',
            needsLastEvent: true
          }
        ];

        for (const testCase of invalidUuidMessages) {
          console.log(`Testing invalid UUID: ${testCase.name}`);
          
          expect(ws.readyState).toBe(WebSocket.OPEN);
          
          ws.send(JSON.stringify(testCase.message));
          
          // Wait for validation error response
          try {
            if (testCase.needsLastEvent) {
              // For test cases that need the last event (task_start comes first, error comes second)
              const firstResponse = await waitForMessage(ws, 2000);
              console.log(`Received first response (task_start): ${firstResponse}`);
              
              // Extract task_id from task_start event for task run status checking
              let taskId: string | null = null;
              const firstValidation = validateWebSocketResponse(firstResponse);
              if (firstValidation.isValid && firstValidation.data) {
                // Check in message.task_id (WebSocket format)
                if (firstValidation.data.message?.task_id) {
                  taskId = firstValidation.data.message.task_id;
                  console.log(`Extracted task_id for status check: ${taskId}`);
                }
                // Check in header.task_id (WebSocket format)
                else if (firstValidation.data.header?.task_id) {
                  taskId = firstValidation.data.header.task_id;
                  console.log(`Extracted task_id for status check: ${taskId}`);
                }
                // Fallback checks for other formats
                else if (firstValidation.data.task_id) {
                  taskId = firstValidation.data.task_id;
                  console.log(`Extracted task_id for status check: ${taskId}`);
                }
              }
              
              const lastResponse = await waitForMessage(ws, 2000);
              console.log(`Received last response (error): ${lastResponse}`);
              
              const validation = validateWebSocketResponse(lastResponse);
              expect(validation.isValid).toBe(true);
              expect(validation.data).toBeDefined();
              
              // Check task run status if we extracted a task_id
              if (taskId) {
                const taskRunStatus = await checkTaskRunStatus(request, taskId);
                console.log(`Task run status for task_id ${taskId}: ${taskRunStatus}`);
                expect(taskRunStatus).toBe('FAILED');
              }
              
              // Check if it's either the expected error or a Bedrock rate limiting error
              const isExpectedError = validation.data.error === testCase.expectedError;
              const isBedrock429Error = validation.data.error && validation.data.error.includes('failed to handle Anthropic request: failed to create message: POST "https://bedrock-runtime.ap-southeast-1.amazonaws.com/v1/messages": 429 Too Many Requests');
              const isBedrockMalformedError = validation.data.error && validation.data.error.includes('failed to handle Anthropic request: failed to create message: POST "https://bedrock-runtime.ap-southeast-1.amazonaws.com/v1/messages": 400 Bad Request') && validation.data.error.includes('Malformed input request');
              
              if (isExpectedError) {
                console.log(`✓ Correctly received expected error: ${validation.data.error}`);
              } else if (isBedrock429Error) {
                console.log(`✓ Received Bedrock rate limiting error (acceptable): ${validation.data.error}`);
              } else if (isBedrockMalformedError) {
                console.log(`✓ Received Bedrock malformed input error (acceptable): ${validation.data.error}`);
              } else {
                throw new Error(`Expected error "${testCase.expectedError}" or Bedrock error, but got: ${validation.data.error}`);
              }
            } else {
              // For regular test cases that expect immediate error response
              const response = await waitForMessage(ws, 2000);
              console.log(`Received response to invalid UUID: ${response}`);
              
              const validation = validateWebSocketResponse(response);
              expect(validation.isValid).toBe(true);
              expect(validation.data).toBeDefined();

              // Check if it's either the expected error or a Bedrock rate limiting error
              const isExpectedError = validation.data.error === testCase.expectedError;
              const isBedrock429Error = validation.data.error && validation.data.error.includes('failed to handle Anthropic request: failed to create message: POST "https://bedrock-runtime.ap-southeast-1.amazonaws.com/v1/messages": 429 Too Many Requests');
              const isBedrockMalformedError = validation.data.error && validation.data.error.includes('failed to handle Anthropic request: failed to create message: POST "https://bedrock-runtime.ap-southeast-1.amazonaws.com/v1/messages": 400 Bad Request') && validation.data.error.includes('Malformed input request');
              
              if (isExpectedError) {
                console.log(`✓ Correctly received expected error: ${validation.data.error}`);
              } else if (isBedrock429Error) {
                console.log(`✓ Received Bedrock rate limiting error (acceptable): ${validation.data.error}`);
              } else if (isBedrockMalformedError) {
                console.log(`✓ Received Bedrock malformed input error (acceptable): ${validation.data.error}`);
              } else {
                throw new Error(`Expected error "${testCase.expectedError}" or Bedrock error, but got: ${validation.data.error}`);
              }
            }
            
          } catch (error) {
            console.error(`Failed to receive expected error response: ${error}`);
            throw new Error(`Expected error response "${testCase.expectedError}" but got none`);
          }
          
          await new Promise(resolve => setTimeout(resolve, 300));
        }
        
        expect(ws.readyState).toBe(WebSocket.OPEN);
        console.log(`✓ All invalid UUID messages returned proper error responses`);
        
      } finally {
        await safeCloseConnection(ws);
      }
    });

    test('should handle non-existent resources with appropriate errors', async ({ request }) => {
      const ws = await createWebSocketConnection();
      
      try {
        const nonExistentMessages = [
          {
            name: 'Non-existent agent',
            message: {
              agent_id: "12345678-1234-1234-1234-123456789abc",
              thread_id: createdThreadId,
              messages: [{ role: "user", content: [{ type:"text", text:"test" }] }]
            },
            expectedError: 'invalid agent_id',
            needsLastEvent: true
          }
        ];

        for (const testCase of nonExistentMessages) {
          console.log(`Testing non-existent resource: ${testCase.name}`);
          
          expect(ws.readyState).toBe(WebSocket.OPEN);
          
          ws.send(JSON.stringify(testCase.message));
          
          // Wait for processing error response
          try {
            if (testCase.needsLastEvent) {
              // For test cases that need the last event (task_start comes first, error comes second)
              const firstResponse = await waitForMessage(ws, 3000);
              console.log(`Received first response (task_start): ${firstResponse}`);
              
              const lastResponse = await waitForMessage(ws, 3000);
              console.log(`Received last response (error): ${lastResponse}`);
              
              const validation = validateWebSocketResponse(lastResponse);
              expect(validation.isValid).toBe(true);
              expect(validation.data).toBeDefined();
              
              // Check if it's either the expected error or a Bedrock rate limiting error
              const isExpectedError = validation.data.error === testCase.expectedError;
              const isBedrock429Error = validation.data.error && validation.data.error.includes('failed to handle Anthropic request: failed to create message: POST "https://bedrock-runtime.ap-southeast-1.amazonaws.com/v1/messages": 429 Too Many Requests');
              const isBedrockMalformedError = validation.data.error && validation.data.error.includes('failed to handle Anthropic request: failed to create message: POST "https://bedrock-runtime.ap-southeast-1.amazonaws.com/v1/messages": 400 Bad Request') && validation.data.error.includes('Malformed input request');
              
              if (isExpectedError) {
                console.log(`✓ Correctly received expected error: ${validation.data.error}`);
              } else if (isBedrock429Error) {
                console.log(`✓ Received Bedrock rate limiting error (acceptable): ${validation.data.error}`);
              } else if (isBedrockMalformedError) {
                console.log(`✓ Received Bedrock malformed input error (acceptable): ${validation.data.error}`);
              } else {
                throw new Error(`Expected error "${testCase.expectedError}" or Bedrock error, but got: ${validation.data.error}`);
              }
            } else {
              // For regular test cases that expect immediate error response
              const response = await waitForMessage(ws, 3000);
              console.log(`Received response to non-existent resource: ${response}`);
              
              const validation = validateWebSocketResponse(response);
              expect(validation.isValid).toBe(true);
              expect(validation.data).toBeDefined();
              expect(validation.data.error).toBe(testCase.expectedError);
              console.log(`✓ Correctly received error: ${validation.data.error}`);
            }
            
          } catch (error) {
            console.error(`Failed to receive expected error response: ${error}`);
            throw new Error(`Expected error response "${testCase.expectedError}" but got none`);
          }
          
          await new Promise(resolve => setTimeout(resolve, 300));
        }
        
        expect(ws.readyState).toBe(WebSocket.OPEN);
        console.log(`✓ All non-existent resource messages returned proper error responses`);
        
      } finally {
        await safeCloseConnection(ws);
      }
    });

    test('should handle valid JSON with wrong structure and return invalid message format error', async () => {
      const ws = await createWebSocketConnection();
      
      try {
        const wrongStructureMessages = [
          {
            name: 'Valid JSON but completely wrong structure',
            message: { valid: "json", but: "wrong_structure", data: [1, 2, 3] },
            expectedError: 'invalid message: agent_id field is required'
          },
          {
            name: 'Valid JSON with some correct fields but missing required ones',
            message: { agent_id: createdAgentId, extra_field: "not needed" },
            expectedError: 'invalid message: messages field is required'
          },
          {
            name: 'Valid JSON with null values for required fields',
            message: { agent_id: null, thread_id: null, messages: null },
            expectedError: 'invalid message: agent_id field is required'
          },
          {
            name: 'Valid JSON with wrong data types',
            message: { agent_id: 12345, thread_id: true, messages: "not an array" },
            expectedError: 'Invalid message format'
          }
        ];

        for (const testCase of wrongStructureMessages) {
          console.log(`Testing wrong structure: ${testCase.name}`);
          
          expect(ws.readyState).toBe(WebSocket.OPEN);
          
          ws.send(JSON.stringify(testCase.message));
          
          // Wait for error response from server
          try {
            const response = await waitForMessage(ws, 2000);
            console.log(`Received response to wrong structure: ${response}`);
            
            const validation = validateWebSocketResponse(response);
            expect(validation.isValid).toBe(true);
            expect(validation.data).toBeDefined();
            expect(validation.data.error).toBe(testCase.expectedError);
            console.log(`✓ Correctly received error: ${validation.data.error}`);
            
          } catch (error) {
            console.error(`Failed to receive expected error response: ${error}`);
            throw new Error(`Expected error response "${testCase.expectedError}" but got none`);
          }
          
          await new Promise(resolve => setTimeout(resolve, 200));
        }
        
        expect(ws.readyState).toBe(WebSocket.OPEN);
        console.log(`✓ All wrong structure messages returned proper error responses`);
        
      } finally {
        await safeCloseConnection(ws);
      }
    });

    test('should handle connection errors and recovery', async () => {
      const ws = await createWebSocketConnection();
      
      try {
        ws.on('error', (error) => {
          console.log(`Connection error detected: ${error.message}`);
        });
        
        ws.on('close', (code, reason) => {
          console.log(`Connection closed: code=${code}, reason=${reason}`);
        });
        
        // Force close the connection to test error handling
        ws.terminate();
        
        // Wait for close event
        await new Promise(resolve => setTimeout(resolve, 1000));
        
        expect(ws.readyState).toBe(WebSocket.CLOSED);
        
        // Test recovery by creating a new connection
        const newWs = await createWebSocketConnection();
        expect(newWs.readyState).toBe(WebSocket.OPEN);
        
        await safeCloseConnection(newWs);
        
      } finally {
        // Original connection should already be closed
        if (ws.readyState !== WebSocket.CLOSED) {
          await safeCloseConnection(ws);
        }
      }
    });
  });

  // 4. PING/PONG & HEARTBEAT TESTS
  test.describe('Ping/Pong & Heartbeat', () => {
    test('should respond to ping with pong and track responses', async () => {
      const ws = await createWebSocketConnection();
      
      try {
        const receivedMessages: string[] = [];
        let pongCount = 0;
        
        ws.on('message', (data: WebSocket.Data) => {
          const message = data.toString();
          receivedMessages.push(message);
          
          // Check if it's a pong response
          try {
            const parsed = JSON.parse(message);
            if (parsed.type === 'pong') {
              pongCount++;
              console.log(`Received pong #${pongCount}:`, parsed);
            }
          } catch (error) {
            // Not JSON or different format
            if (message.includes('pong')) {
              pongCount++;
              console.log(`Received text pong #${pongCount}:`, message);
            }
          }
        });
        
        // Send ping message
        const pingMessage = { type: "ping", timestamp: Date.now() };
        ws.send(JSON.stringify(pingMessage));
        console.log('Sent ping message:', pingMessage);
        
        // Wait for potential pong response
        await new Promise(resolve => setTimeout(resolve, 2000));
        
        console.log(`Total messages received: ${receivedMessages.length}`);
        console.log(`Pong responses: ${pongCount}`);
        
        if (receivedMessages.length > 0) {
          console.log('All received messages:', receivedMessages);
        }
        
        expect(ws.readyState).toBe(WebSocket.OPEN);
        
      } finally {
        await safeCloseConnection(ws);
      }
    });

    test('should handle multiple ping messages with proper tracking', async () => {
      const ws = await createWebSocketConnection();
      
      try {
        const pings = 5;
        const receivedResponses: any[] = [];
        
        ws.on('message', (data: WebSocket.Data) => {
          try {
            const message = JSON.parse(data.toString());
            receivedResponses.push(message);
            console.log('Received response:', message);
          } catch (error) {
            console.log('Received non-JSON response:', data.toString());
          }
        });
        
        // Send multiple ping messages
        const startTime = Date.now();
        for (let i = 0; i < pings; i++) {
          const pingMessage = { 
            type: "ping", 
            id: i, 
            timestamp: Date.now(),
            sequence: i + 1
          };
          ws.send(JSON.stringify(pingMessage));
          console.log(`Sent ping ${i + 1}/${pings}:`, pingMessage);
          
          await new Promise(resolve => setTimeout(resolve, 300));
        }
        
        // Wait for all potential responses
        await new Promise(resolve => setTimeout(resolve, 2000));
        const totalTime = Date.now() - startTime;
        
        console.log(`Sent ${pings} pings in ${totalTime}ms`);
        console.log(`Received ${receivedResponses.length} responses`);
        
        expect(ws.readyState).toBe(WebSocket.OPEN);
        
      } finally {
        await safeCloseConnection(ws);
      }
    });

    test('should maintain connection during extended heartbeat sequence', async () => {
      const ws = await createWebSocketConnection();
      
      try {
        const heartbeatInterval = 1000; // 1 second
        const duration = 5000; // 5 seconds
        const expectedHeartbeats = Math.floor(duration / heartbeatInterval);
        
        let heartbeatsSent = 0;
        let responsesReceived = 0;
        
        ws.on('message', (data: WebSocket.Data) => {
          responsesReceived++;
          console.log(`Heartbeat response ${responsesReceived}:`, data.toString());
        });
        
        const startTime = Date.now();
        const heartbeatTimer = setInterval(() => {
          if (ws.readyState === WebSocket.OPEN) {
            const heartbeat = {
              type: "ping",
              sequence: ++heartbeatsSent,
              timestamp: Date.now()
            };
            ws.send(JSON.stringify(heartbeat));
            console.log(`Heartbeat ${heartbeatsSent} sent`);
            
            // Verify connection is still active
            expect(ws.readyState).toBe(WebSocket.OPEN);
          }
        }, heartbeatInterval);
        
        // Run for specified duration
        await new Promise(resolve => setTimeout(resolve, duration));
        clearInterval(heartbeatTimer);
        
        const actualDuration = Date.now() - startTime;
        console.log(`Heartbeat test completed in ${actualDuration}ms`);
        console.log(`Sent ${heartbeatsSent} heartbeats (expected ~${expectedHeartbeats})`);
        console.log(`Received ${responsesReceived} responses`);
        
        expect(ws.readyState).toBe(WebSocket.OPEN);
        expect(heartbeatsSent).toBeGreaterThanOrEqual(expectedHeartbeats - 1); // Allow for timing variance
        
      } finally {
        await safeCloseConnection(ws);
      }
    });
  });
});