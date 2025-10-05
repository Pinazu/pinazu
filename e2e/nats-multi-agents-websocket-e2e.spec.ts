import { test, expect } from '@playwright/test';
import WebSocket from 'ws';
// @ts-ignore - NATS types not available
import { connect, NatsConnection } from 'nats';

// Test configuration constants
const WEBSOCKET_URL = 'ws://localhost:8080/v1/ws';
const NATS_URL = 'nats://localhost:4222';
const DEFAULT_TIMEOUT = 15000;
const SHORT_TIMEOUT = 5000;

test.describe.serial('NATS Tools Websocket E2E Integration', () => {
    let createdAgentId: string;
    let createdSubAgentId: string;
    let createdThreadId: string;
    let natsConnection: NatsConnection;
    let testUserId: string = '550e8400-c95b-4444-6666-446655440000'; // Admin user UUID

    test.beforeAll(async ({ request }) => {
        // Connect to NATS
        natsConnection = await connect({ servers: NATS_URL });
        console.log('Connected to NATS server');

        // Clean up any existing test data
        await cleanupTestData(request);

        // Create a sub agent for testing
        const subAgentData = {
            name: 'Test Sub Agent',
            description: 'A test sub agent for NATS WebSocket e2e testing',
            specs: `
model:
  provider: "bedrock/anthropic"
  model_id: "global.anthropic.claude-sonnet-4-5-20250929-v1:0"
  max_tokens: 200
  stream: true
  thinking:
    enabled: false

system: |
  You are a sub agent for testing. Only Answer 'Sub agent invoke successfully'
`
        }

        const subAgentResponse = await request.post('/v1/agents', {
            data: subAgentData
        });

        expect(subAgentResponse.status()).toBe(201);
        const subAgentBody = await subAgentResponse.json();
        createdSubAgentId = subAgentBody.id;
        console.log(`Created test sub agent with ID: ${createdSubAgentId}`);

        // Create a test agent for testing
        const agentData = {
            name: 'Test NATS WebSocket Agent',
            description: 'A test agent for NATS WebSocket e2e testing',
            specs: `
model:
  provider: "bedrock/anthropic"
  model_id: "global.anthropic.claude-sonnet-4-5-20250929-v1:0"
  max_tokens: 2048
  stream: true
  thinking:
    enabled: true
    budget_token: 1024

system: |
  You are a helpful AI assistant for tool e2e testing.
  You MUST use the available tools when asked to demonstrate them.
  Keep your final response short after using tools.

sub_agents:
  configs:
    shared_memory: true
  allows:
    - ${createdSubAgentId}
`
        };

        const agentResponse = await request.post('/v1/agents', {
            data: agentData
        });

        expect(agentResponse.status()).toBe(201);
        const agentBody = await agentResponse.json();
        createdAgentId = agentBody.id;
        console.log(`Created test agent with ID: ${createdAgentId}`);

        // Create a test thread
        const threadData = {
            title: 'Test NATS WebSocket Thread',
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

    test.afterAll(async ({ request }) => {
        // Clean up created resources
        if (createdThreadId) {
            await request.delete(`/v1/threads/${createdThreadId}`);
        }
        if (createdAgentId) {
            await request.delete(`/v1/agents/${createdAgentId}`);
        }
        if (createdSubAgentId) {
            await request.delete(`/v1/agents/${createdSubAgentId}`);
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
                    if (agent.name && (agent.name.includes('Test NATS WebSocket Agent') || agent.name.includes('Test NATS WebSocket Sub-Agent'))) {
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
                    if (thread.title && thread.title.includes('Test NATS WebSocket Thread')) {
                        await request.delete(`/v1/threads/${thread.id}`);
                    }
                }
            }
        }

        // Clean up test tools
        const toolsResponse = await request.get('/v1/tools');
        if (toolsResponse.ok()) {
            const tools = await toolsResponse.json();
            if (Array.isArray(tools)) {
                for (const tool of tools) {
                    if (tool.name && tool.name.includes('test_tool_')) {
                        await request.delete(`/v1/tools/${tool.id}`);
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

    test('should handle parallel tool calling (without batch_tool)', async () => {
        expect(createdAgentId).toBeTruthy();
        expect(createdThreadId).toBeTruthy();

        // WebSocket response tracking
        const receivedWebSocketMessages: string[] = [];
        const eventTypes: string[] = [];
        let streamingStarted = false;
        let streamingCompleted = false;
        let contentReceived = false;

        // Order validation tracking
        let orderValidationErrors: string[] = [];
        let taskStartReceived = false;
        let subTaskStartReceived = false;
        let subTaskStopReceived = false;
        let messageStartCount = 0;
        let messageStopCount = 0;

        // Create WebSocket connection and monitor streaming responses
        const ws = await createWebSocketConnection();

        try {
            // Set up WebSocket message handler to collect streaming responses
            ws.on('message', (data: WebSocket.Data) => {
                const message = data.toString();
                receivedWebSocketMessages.push(message);
                console.log(`⚙️ WebSocket received streaming message: ${message.substring(0, 100)}...`);

                try {
                    const parsedMessage = JSON.parse(message);
                    
                    // Handle error messages
                    if (parsedMessage.error) {
                        console.log(`❌ Error received: ${parsedMessage.error}`);
                        return;
                    }
                    
                    // Skip messages without message property
                    if (!parsedMessage.message || !parsedMessage.message.type) {
                        console.log(`⚠️ Skipping message without type:`, parsedMessage);
                        return;
                    }
                    
                    const messageType = parsedMessage.message.type;
                    eventTypes.push(messageType);

                    // Validate message order - handle multiple content block sequences
                    const validateMessageOrder = (type: string, events: string[]) => {
                        const lastEvent = events[events.length - 2]; // Previous event

                        switch (type) {
                            case 'task_start':
                                // Must be the very first event
                                return events.length === 1;

                            case 'sub_task_start':
                                // Must come after message_stop
                                return  lastEvent == 'message_stop'

                            case 'message_start':
                                // Must come after task_start or after message_stop (new stream) or after sub_task events (new sub task)
                                return lastEvent === 'task_start' || lastEvent === 'sub_task_start' || lastEvent === 'sub_task_stop' || lastEvent === 'message_stop';

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
                            
                            case 'sub_task_stop':
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
                        console.log('✓ Task Event: task_start received');
                        expect(parsedMessage.message.task_id).toBeDefined();
                    } else if (messageType == 'sub_task_start') {
                        subTaskStartReceived = true;
                        console.log('✓ Sub Task Event: sub_task_start received')
                        expect(parsedMessage.message.task_id).toBeDefined();
                    } else if (messageType === 'message_start') {
                        messageStartCount++;
                        streamingStarted = true;
                        console.log(`✓ Stream Event: message_start received (count: ${messageStartCount})`);
                        expect(parsedMessage.message.message).toBeDefined();
                        expect(parsedMessage.message.provider).toBeDefined();
                    } else if (messageType === 'content_block_start') {
                        console.log('✓ Stream Event: content_block_start received');
                        expect(parsedMessage.message.content_block).toBeDefined();
                    } else if (messageType === 'content_block_delta') {
                        contentReceived = true;
                        console.log('✓ Stream Event: content_block_delta received with text:', parsedMessage.message.delta?.text || 'no text');
                        expect(parsedMessage.message.delta).toBeDefined();
                    } else if (messageType === 'content_block_stop') {
                        console.log('✓ Stream Event: content_block_stop received');
                    } else if (messageType === 'message_delta') {
                        console.log('✓ Stream Event: message_delta received');
                        expect(parsedMessage.message.delta).toBeDefined();
                    } else if (messageType === 'message_stop') {
                        messageStopCount++;
                        streamingCompleted = true;
                        console.log(`✓ Stream Event: message_stop received (count: ${messageStopCount})`);
                    } else if (messageType === 'task_stop') {
                        console.log('✓ Task Event: task_stop received');
                    } else if (messageType == 'sub_task_stop') {
                        subTaskStopReceived = true;
                        console.log('✓ Sub Task Event: sub_task_stop received')
                    }

                    // Validate all streaming messages have required fields
                    expect(parsedMessage.message.type).toBeDefined();

                    // Only validate provider for AI streaming events, not task lifecycle events
                    if (messageType !== 'task_start' && messageType !== 'task_stop' && messageType !== 'sub_task_start' && messageType !== 'sub_task_stop') {
                        expect(parsedMessage.message.provider).toBe('anthropic');
                    }

                } catch (error) {
                    console.error(error);
                }
            });

            // Step 5: Send test message via WebSocket to initiate the full flow
            const testMessage = {
                agent_id: createdAgentId,
                thread_id: createdThreadId,
                messages: [
                    {
                        role: "user",
                        content: [{
                            type: "text",
                            text: "Demonstrate calling the sub agent."
                        }]
                    }
                ]
            };

            console.log('🚀 Starting full NATS WebSocket tool flow test...');
            console.log('Sending WebSocket message to initiate: WebSocket → Task → Agent → Tool Execution → Final Response → WebSocket');
            ws.send(JSON.stringify(testMessage));

            // Wait for the complete flow to process (longer timeout for tool execution)
            await new Promise(resolve => setTimeout(resolve, 20000));

            // Verify Step 1: Complete WebSocket streaming flow
            expect(receivedWebSocketMessages.length).toBeGreaterThan(0);
            console.log('✅ Step 1 Verified: WebSocket client received streaming responses');
            console.log(`Total WebSocket messages received: ${receivedWebSocketMessages.length}`);

            // Verify task_start event is received first
            expect(taskStartReceived).toBe(true);
            console.log('✅ Task Event: task_start event verified as first event');
            expect(eventTypes[0]).toBe('task_start');
            console.log('✅ Event Order: task_start is the first event in sequence');
            expect(eventTypes[eventTypes.length - 1]).toBe('task_stop');
            console.log('✅ Event Order: task_stop is the last event in sequence');

            // Verify sub-task events are received
            expect(subTaskStartReceived).toBe(true);
            console.log('✅ Sub Task Event: sub_task_start event verified');
            expect(subTaskStopReceived).toBe(true);
            console.log('✅ Sub Task Event: sub_task_stop event verified');

            // Verify sub-task events are in the event sequence
            expect(eventTypes).toContain('sub_task_start');
            expect(eventTypes).toContain('sub_task_stop');
            console.log('✅ Event Sequence: Both sub_task_start and sub_task_stop events are present');

            // Verify streaming event sequence and format
            expect(eventTypes).toContain('message_start');
            console.log('✅ Streaming Format: message_start event verified');

            // Verify message order is correct
            expect(orderValidationErrors).toEqual([]);
            if (orderValidationErrors.length === 0) {
                console.log('✅ Message Order: All streaming events received in correct order');
            } else {
                console.log('❌ Message Order Errors:', orderValidationErrors);
            }
            console.log('📋 Event Sequence:', eventTypes.join(' → '));

            if (taskStartReceived) {
                console.log('✅ Task Flow: Task started correctly');
            }

            if (streamingStarted) {
                console.log('✅ Streaming Flow: Streaming started correctly');
            }

            if (contentReceived) {
                console.log('✅ Streaming Content: Content delta received');
            }

            if (streamingCompleted) {
                console.log('✅ Streaming Flow: Streaming completed correctly');
            }

            // Verify tool execution - should have multiple message cycles (tool call + response)
            expect(messageStartCount).toBeGreaterThanOrEqual(2);
            expect(messageStopCount).toBeGreaterThanOrEqual(2);
            console.log(`✅ Tool Execution: Detected ${messageStartCount} message_start and ${messageStopCount} message_stop events`);
            
            // Verify message start/stop balance (should be close to equal)
            const messageDelta = Math.abs(messageStartCount - messageStopCount);
            expect(messageDelta).toBeLessThanOrEqual(1); // Should be balanced or off by 1
            console.log(`✅ Message Balance: Start/Stop delta is ${messageDelta} (expected ≤ 1)`);

            // Verify unique event types received
            const uniqueEventTypes = [...new Set(eventTypes)];
            console.log('✅ Event Types Received:', uniqueEventTypes);
            expect(uniqueEventTypes.length).toBeGreaterThan(1);

            // Log complete flow success
            console.log('\n🎉 COMPLETE TOOL FLOW VERIFICATION SUCCESS:');
            console.log(`   📊 Total streaming messages: ${receivedWebSocketMessages.length}`);
            console.log(`   📋 Event types: ${uniqueEventTypes.join(', ')}`);
            console.log(`   🔧 Tool execution cycles: ${messageStartCount} starts, ${messageStopCount} stops`);
            console.log(`   ⚖️ Message balance: ${messageDelta} delta (balanced execution)`);
            console.log(`   🎯 Sub-task events: sub_task_start=${subTaskStartReceived}, sub_task_stop=${subTaskStopReceived}`);

        } finally {
            await safeCloseConnection(ws);
        }
    });
});