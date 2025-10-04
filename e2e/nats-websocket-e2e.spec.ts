import { test, expect } from '@playwright/test';
import WebSocket from 'ws';
// @ts-ignore - NATS types not available
import { connect, NatsConnection } from 'nats';

// Test configuration constants
const WEBSOCKET_URL = 'ws://localhost:8080/v1/ws';
const NATS_URL = 'nats://localhost:4222';
const DEFAULT_TIMEOUT = 60000;
const SHORT_TIMEOUT = 10000;

test.describe.serial('NATS WebSocket E2E Integration', () => {
  let natsConnection: NatsConnection;
  let testUserId: string = '550e8400-c95b-4444-6666-446655440000'; // Admin user UUID

  // Agent configurations to test - all should produce same streaming results
  const agentConfigs = [
    {
      name: 'Anthropic Claude Sonnet 4',
      description: 'Anthropic Claude Sonnet 4 with thinking',
      specs: `
model:
  provider: "bedrock/anthropic"
  model_id: "apac.anthropic.claude-sonnet-4-20250514-v1:0"
  max_tokens: 2048
  stream: true
  thinking:
    enabled: true
    budget_token: 1024

system: |
  You are a helpful AI assistant for e2e testing.
  Keep responses very short and simple for testing purposes.
  MUST respond only with "Hello from agent!" when greeted. Think a bit for demonstrate the thinking process

tool_refs: []
`,
      expectedProvider: 'anthropic'
    },
    {
      name: 'Bedrock Amazon Nova',
      description: 'Bedrock Amazon Nova model',
      specs: `
model:
  provider: "bedrock"
  model_id: "apac.amazon.nova-lite-v1:0"
  max_tokens: 2048
  stream: true

system: |
  You are a helpful AI assistant for e2e testing.
  Keep responses very short and simple for testing purposes.
  MUST respond only with "Hello from agent!" when greeted.

tool_refs: []
`,
      expectedProvider: 'bedrock'
    },
    {
      name: 'Bedrock Anthropic Claude 3.7 Sonnet with Thinking',
      description: 'Bedrock Anthropic Claude 3.7 Sonnet with thinking enabled',
      specs: `
model:
  provider: "bedrock"
  model_id: "apac.anthropic.claude-3-7-sonnet-20250219-v1:0"
  max_tokens: 4000
  stream: true
  thinking:
    enabled: true
    budget_token: 1024

system: |
  You are a helpful AI assistant for e2e testing.
  Keep responses very short and simple for testing purposes.
  MUST respond only with "Hello from agent!" when greeted.

tool_refs: []
`,
      expectedProvider: 'bedrock'
    },
    {
      name: 'Google Gemini 2.5 Pro',
      description: 'Google Gemini 2.5 Pro model with thinking',
      specs: `
    model:
      provider: "google"
      model_id: "gemini-2.5-pro"
      max_tokens: 2048
      stream: true
      thinking:
        enabled: true
        budget_token: 1024

    system: |
      You are a helpful AI assistant for e2e testing.
      Keep responses very short and simple for testing purposes.
      MUST respond only with "Hello from agent!" when greeted. Think a bit for demonstrate the thinking process

    tool_refs: []
    `,
      expectedProvider: 'google'
    }
  ];

  let agentIds: { [key: string]: string } = {};
  let threadIds: { [key: string]: string } = {};

  test.beforeAll(async ({ request }) => {
    // Connect to NATS
    natsConnection = await connect({ servers: NATS_URL });
    console.log('Connected to NATS server');

    // Clean up any existing test data
    await cleanupTestData(request);

    // Create agents for each configuration
    for (const config of agentConfigs) {
      const agentData = {
        name: `Test ${config.name}`,
        description: config.description,
        specs: config.specs
      };

      const agentResponse = await request.post('/v1/agents', {
        data: agentData
      });

      expect(agentResponse.status()).toBe(201);
      const agentBody = await agentResponse.json();
      agentIds[config.name] = agentBody.id;
      console.log(`Created test agent (${config.name}) with ID: ${agentIds[config.name]}`);

      // Create a test thread for each agent
      const threadData = {
        title: `Test Thread for ${config.name}`,
        user_id: testUserId
      };

      const threadResponse = await request.post('/v1/threads', {
        data: threadData
      });

      expect(threadResponse.status()).toBe(201);
      const threadBody = await threadResponse.json();
      threadIds[config.name] = threadBody.id;
      console.log(`Created test thread for ${config.name} with ID: ${threadIds[config.name]}`);
    }
  });

  test.afterAll(async ({ request }) => {
    // Clean up created resources for all configurations
    for (const config of agentConfigs) {
      if (threadIds[config.name]) {
        await request.delete(`/v1/threads/${threadIds[config.name]}`);
        console.log(`Cleaned up thread for ${config.name}`);
      }
      if (agentIds[config.name]) {
        await request.delete(`/v1/agents/${agentIds[config.name]}`);
        console.log(`Cleaned up agent for ${config.name}`);
      }
    }

    // Close NATS connection
    if (natsConnection) {
      await natsConnection.close();
      console.log('NATS connection closed');
    }
  });

  // Helper function to clean up existing test data
  async function cleanupTestData(request: any) {
    // Clean up test agents
    const agentsResponse = await request.get('/v1/agents');
    if (agentsResponse.ok()) {
      const agentsBody = await agentsResponse.json();
      const agents = agentsBody.agents || [];
      if (Array.isArray(agents)) {
        for (const agent of agents) {
          if (agent.name && (agent.name.includes('Test Bedrock') || agent.name.includes('Test NATS WebSocket Agent') || agent.name.includes('Test Pure Bedrock') || agent.name.includes('Test Nova') || agent.name.includes('Test Google'))) {
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
          if (thread.title && (thread.title.includes('Test Thread') || thread.title.includes('Test NATS WebSocket Thread') || thread.title.includes('Test Bedrock') || thread.title.includes('Test Nova') || thread.title.includes('Test Google'))) {
            await request.delete(`/v1/threads/${thread.id}`);
          }
        }
      }
    }
  }

  // Helper function to create WebSocket connection
  async function createWebSocketConnection(url: string = WEBSOCKET_URL): Promise<WebSocket> {
    return new Promise<WebSocket>((resolve, reject) => {
      const ws = new WebSocket(url);
      const connectionTimeout = setTimeout(() => {
        ws.terminate();
        reject(new Error('WebSocket connection timeout'));
      }, DEFAULT_TIMEOUT);

      ws.on('open', () => {
        clearTimeout(connectionTimeout);
        resolve(ws);
      });

      ws.on('error', (error) => {
        clearTimeout(connectionTimeout);
        reject(error);
      });
    });
  }

  // Helper function to safely close WebSocket
  async function safeCloseConnection(ws: WebSocket): Promise<void> {
    return new Promise((resolve) => {
      if (ws.readyState === WebSocket.CLOSED) {
        resolve();
        return;
      }

      const closeTimeout = setTimeout(() => {
        ws.terminate();
        resolve();
      }, SHORT_TIMEOUT);

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

  for (const config of agentConfigs) {
    test(`should complete full NATS WebSocket flow with agent configuration: ${config.name}`, async () => {
      test.setTimeout(120000); // 2 minutes timeout for testing this configuration
      await testAgentsFunc(agentIds, config, threadIds, createWebSocketConnection, safeCloseConnection);
    });
  }
});

async function testAgentsFunc(agentIds: { [key: string]: string; }, config: { name: string; description: string; specs: string; expectedProvider: string; }, threadIds: { [key: string]: string; }, createWebSocketConnection: (url?: string) => Promise<WebSocket>, safeCloseConnection: (ws: WebSocket) => Promise<void>) {
  const agentId = agentIds[config.name];
  const threadId = threadIds[config.name];

  expect(agentId).toBeTruthy();
  expect(threadId).toBeTruthy();

  console.log(`\n🧪 Testing agent configuration: ${config.name}`);
  console.log(`Expected provider: ${config.expectedProvider}`);

  // WebSocket response tracking
  const receivedWebSocketMessages: string[] = [];
  const eventTypes: string[] = [];
  let streamingStarted = false;
  let streamingCompleted = false;
  let contentReceived = false;

  // Order validation tracking
  let orderValidationErrors: string[] = [];
  let taskStartReceived = false;

  // Create WebSocket connection and monitor streaming responses
  const ws = await createWebSocketConnection();

  try {
    // Set up WebSocket message handler to collect streaming responses
    ws.on('message', (data: WebSocket.Data) => {
      const message = data.toString();
      receivedWebSocketMessages.push(message);
      console.log(`⚙️ ${config.name} WebSocket received streaming message: ${message.substring(0, 1000)}...`);

      try {
        const parsedMessage = JSON.parse(message);
        const messageType = parsedMessage.message.type;
        eventTypes.push(messageType);

        // Validate message order - handle multiple content block sequences
        const validateMessageOrder = (type: string, events: string[]) => {
          const lastEvent = events[events.length - 2]; // Previous event

          switch (type) {
            case 'task_start':
              // Must be the very first event
              return events.length === 1;

            case 'message_start':
              // Must come after task_start or after message_stop (new stream)
              return lastEvent === 'task_start' || lastEvent === 'message_stop';

            case 'content_block_start':
              // Must come after message_start or content_block_stop
              return lastEvent === 'message_start' || lastEvent === 'content_block_stop';

            case 'content_block_delta':
              // Must come after content_block_start or another content_block_delta
              return lastEvent === 'content_block_start' || lastEvent === 'content_block_delta';

            case 'content_block_stop':
              // Must come after content_block_delta
              return lastEvent === 'content_block_delta';

            case 'message_delta':
              // Must come after content_block_stop (last content block finished)
              return lastEvent === 'content_block_stop';

            case 'message_stop':
              // Must come after message_delta
              return lastEvent === 'message_delta';

            case 'task_stop':
              // Must come after message_stop
              return lastEvent === 'message_stop';

            default:
              return false;
          }
        };

        if (eventTypes.length > 1 && !validateMessageOrder(messageType, eventTypes)) {
          const lastEvent = eventTypes[eventTypes.length - 2];
          orderValidationErrors.push(`Invalid order: ${messageType} cannot follow ${lastEvent}`);
        }

        // Validate streaming event types and format
        if (messageType === 'task_start') {
          taskStartReceived = true;
          console.log(`✓ ${config.name} Task Event: task_start received`);
          expect(parsedMessage.message.task_id).toBeDefined();
        } else if (messageType === 'message_start') {
          streamingStarted = true;
          console.log(`✓ ${config.name} Stream Event: message_start received`);
          expect(parsedMessage.message.message).toBeDefined();
          expect(parsedMessage.message.provider).toBeDefined();
        } else if (messageType === 'content_block_start') {
          console.log(`✓ ${config.name} Stream Event: content_block_start received`);
          expect(parsedMessage.message.content_block).toBeDefined();
        } else if (messageType === 'content_block_delta') {
          contentReceived = true;
          console.log(`✓ ${config.name} Stream Event: content_block_delta received with text:`, parsedMessage.message.delta?.text || 'no text');
          expect(parsedMessage.message.delta).toBeDefined();
        } else if (messageType === 'content_block_stop') {
          console.log(`✓ ${config.name} Stream Event: content_block_stop received`);
        } else if (messageType === 'message_delta') {
          console.log(`✓ ${config.name} Stream Event: message_delta received`);
          expect(parsedMessage.message.delta).toBeDefined();
        } else if (messageType === 'message_stop') {
          streamingCompleted = true;
          console.log(`✓ ${config.name} Stream Event: message_stop received`);
        } else if (messageType === 'task_stop') {
          console.log(`✓ ${config.name} Task Event: task_stop received`);
        }

        // Validate all streaming messages have required fields
        expect(parsedMessage.message.type).toBeDefined();

        // Only validate provider for AI streaming events, not task lifecycle events
        if (messageType !== 'task_start' && messageType !== 'task_stop') {
          expect(parsedMessage.message.provider).toBe(config.expectedProvider);
        }

      } catch (error) {
        console.error(`${config.name} message parsing error:`, error);
      }
    });

    // Send test message via WebSocket to initiate the full flow
    const testMessage = {
      agent_id: agentId,
      thread_id: threadId,
      messages: [
        {
          role: "user",
          content: [{
            type: "text",
            text: "Hello!"
          }]
        }
      ]
    };

    console.log(`🚀 Starting full NATS WebSocket flow test (${config.name})...`);
    console.log('Sending WebSocket message to initiate: WebSocket → Task → Agent → Streaming → WebSocket');
    ws.send(JSON.stringify(testMessage));

    // Wait for the complete flow to process (longer timeout for thinking models)
    const timeout = config.name.includes('Thinking') ? 18000 : 15000;
    await new Promise(resolve => setTimeout(resolve, timeout));

    // Verify Step 1: Complete WebSocket streaming flow
    expect(receivedWebSocketMessages.length).toBeGreaterThan(0);
    console.log(`✅ ${config.name} Step 1 Verified: WebSocket client received streaming responses`);
    console.log(`Total ${config.name} WebSocket messages received: ${receivedWebSocketMessages.length}`);

    // Verify task_start event is received first
    expect(taskStartReceived).toBe(true);
    console.log(`✅ ${config.name} Task Event: task_start event verified as first event`);
    expect(eventTypes[0]).toBe('task_start');
    console.log(`✅ ${config.name} Event Order: task_start is the first event in sequence`);
    expect(eventTypes[eventTypes.length - 1]).toBe('task_stop');
    console.log(`✅ ${config.name} Event Order: task_stop is the last event in sequence`);

    // Verify streaming event sequence and format
    expect(eventTypes).toContain('message_start');
    console.log(`✅ ${config.name} Streaming Format: message_start event verified`);

    // Verify message order is correct
    expect(orderValidationErrors).toEqual([]);
    if (orderValidationErrors.length === 0) {
      console.log(`✅ ${config.name} Message Order: All streaming events received in correct order`);
    } else {
      console.log(`❌ ${config.name} Message Order Errors:`, orderValidationErrors);
    }
    console.log(`📋 ${config.name} Event Sequence:`, eventTypes.join(' → '));

    if (taskStartReceived) {
      console.log(`✅ ${config.name} Task Flow: Task started correctly`);
    }

    if (streamingStarted) {
      console.log(`✅ ${config.name} Streaming Flow: Streaming started correctly`);
    }

    if (contentReceived) {
      console.log(`✅ ${config.name} Streaming Content: Content delta received`);
    }

    if (streamingCompleted) {
      console.log(`✅ ${config.name} Streaming Flow: Streaming completed correctly`);
    }

    // Verify unique event types received
    const uniqueEventTypes = [...new Set(eventTypes)];
    console.log(`✅ ${config.name} Event Types Received:`, uniqueEventTypes);
    expect(uniqueEventTypes.length).toBeGreaterThan(1);

    // Log complete flow success for this configuration
    console.log(`\n🎉 COMPLETE FLOW VERIFICATION SUCCESS (${config.name}):`);
    console.log(`   📊 Total streaming messages: ${receivedWebSocketMessages.length}`);
    console.log(`   📋 Event types: ${uniqueEventTypes.join(', ')}`);
    console.log(`   🏷️  Provider: ${config.expectedProvider}`);

  } finally {
    await safeCloseConnection(ws);
    // Small delay between configurations to avoid conflicts
    await new Promise(resolve => setTimeout(resolve, 1000));
  }
}

