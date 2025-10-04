<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { 
		websocketManager, 
		websocketState, 
		websocketMessages,
		websocketConnected,
		websocketError,
		type WebSocketMessage 
	} from '$lib/stores/websocket';
	import { v4 as uuidv4 } from 'uuid';

	let message = '';
	let autoConnect = true;
	let sending = false;
	let lastSendTime = 0;
	let debounceDelay = 300; // 300ms minimum between sends
	let lastSentMessage = '';

	onMount(() => {
		if (autoConnect) {
			connect();
		}
	});

	onDestroy(() => {
		websocketManager.disconnect();
	});

	async function connect() {
		try {
			await websocketManager.connect();
		} catch (error) {
			console.error('Failed to connect:', error);
		}
	}

	function disconnect() {
		websocketManager.disconnect();
	}

	function sendMessage() {
		const now = Date.now();
		const messageToSend = message.trim();
		
		// Check if we're already sending, no message, not connected, too soon, or same message
		if (sending || !messageToSend || !$websocketConnected || 
		    (now - lastSendTime < debounceDelay) || messageToSend === lastSentMessage) {
			return;
		}
		
		sending = true;
		lastSendTime = now;
		lastSentMessage = messageToSend;
		
		// Clear message immediately to prevent duplicate content
		message = '';
		
		try {
			const success = websocketManager.send({
				agent_id: "550e8400-c95b-5555-6666-446655440000",
				thread_id: uuidv4(),
				messages: [
					{
						role: 'user',
						content: [
							{
								type: "text",
								text: messageToSend
							}
						]
					}
				]
			});
			
			// If send failed, restore the message
			if (!success) {
				message = messageToSend;
				lastSentMessage = '';
			}
		} finally {
			sending = false;
		}
	}

	function clearMessages() {
		websocketMessages.set([]);
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter' && !event.shiftKey && !sending) {
			event.preventDefault();
			event.stopPropagation();
			sendMessage();
		}
	}
</script>

<div class="websocket-demo p-4 max-w-2xl mx-auto">
	<div class="mb-4">
		<h2 class="text-xl font-semibold mb-2">WebSocket Connection</h2>
		
		<!-- Connection Status -->
		<div class="mb-4 p-3 rounded-lg {$websocketConnected ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}">
			{#if $websocketState.connecting}
				<span class="inline-block animate-spin mr-2">⟳</span>
				Connecting...
			{:else if $websocketConnected}
				<span class="mr-2">✓</span>
				Connected
				{#if $websocketState.latency !== null}
					<span class="ml-2 text-sm">
						(Latency: {$websocketState.latency}ms)
					</span>
				{/if}
				{#if $websocketState.lastPong}
					<span class="ml-2 text-sm">
						Last pong: {$websocketState.lastPong.toLocaleTimeString()}
					</span>
				{/if}
			{:else}
				<span class="mr-2">✗</span>
				Disconnected
			{/if}
		</div>

		<!-- Error Display -->
		{#if $websocketError}
			<div class="mb-4 p-3 bg-red-100 text-red-800 rounded-lg">
				Error: {$websocketError}
			</div>
		{/if}

		<!-- Controls -->
		<div class="mb-4 space-x-2">
			<button 
				class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 disabled:opacity-50"
				on:click={connect}
				disabled={$websocketConnected || $websocketState.connecting}
			>
				Connect
			</button>
			
			<button 
				class="px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600 disabled:opacity-50"
				on:click={disconnect}
				disabled={!$websocketConnected}
			>
				Disconnect
			</button>
			
			<button 
				class="px-4 py-2 bg-gray-500 text-white rounded hover:bg-gray-600"
				on:click={clearMessages}
			>
				Clear Messages
			</button>
		</div>
	</div>

	<!-- Message Input -->
	<div class="mb-4">
		<div class="flex space-x-2">
			<input
				bind:value={message}
				on:keydown={handleKeydown}
				placeholder="Type a message..."
				class="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
				disabled={!$websocketConnected || sending}
			>
			<button
				on:click={sendMessage}
				disabled={!$websocketConnected || !message.trim() || sending}
				class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 disabled:opacity-50"
			>
				{sending ? 'Sending...' : 'Send'}
			</button>
		</div>
	</div>

	<!-- Messages Display -->
	<div class="border border-gray-300 rounded-lg p-4 h-64 overflow-y-auto bg-gray-50">
		<h3 class="font-semibold mb-2">Messages ({$websocketMessages.length})</h3>
		
		{#each $websocketMessages as msg (msg.id)}
			<div class="mb-2 p-2 bg-white rounded border text-sm">
				<div class="font-mono text-xs text-gray-500 mb-1">
					{msg.timestamp.toLocaleTimeString()}
				</div>
				<div class="font-mono">
					{JSON.stringify(msg.data, null, 2)}
				</div>
			</div>
		{:else}
			<p class="text-gray-500 italic">No messages received yet...</p>
		{/each}
	</div>
</div>