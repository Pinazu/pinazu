import { writable, derived, type Writable } from 'svelte/store';

export interface WebSocketMessage {
	id: string;
	type: string;
	data: any;
	timestamp: Date;
}

export interface WebSocketState {
	connected: boolean;
	connecting: boolean;
	error: string | null;
	lastMessage: WebSocketMessage | null;
	lastPong: Date | null;
	latency: number | null;
}

class WebSocketManager {
	private ws: WebSocket | null = null;
	private reconnectAttempts = 0;
	private maxReconnectAttempts = 5;
	private reconnectDelay = 1000;
	private url: string;
	private pingInterval: number | null = null;
	private pingTimeout: number | null = null;
	private pingIntervalMs = 30000; // Send ping every 30 seconds
	private pongTimeoutMs = 5000; // Wait 5 seconds for pong response
	private lastPingTime: Date | null = null;

	// Stores
	public state: Writable<WebSocketState> = writable({
		connected: false,
		connecting: false,
		error: null,
		lastMessage: null,
		lastPong: null,
		latency: null,
	});

	public messages: Writable<WebSocketMessage[]> = writable([]);

	constructor(url?: string) {
		this.url = url || this.getWebSocketUrl();
		
		// Auto-connect when running in browser
		if (typeof window !== 'undefined') {
			this.autoConnect();
		}
	}

	private autoConnect(): void {
		// Connect immediately when the store is initialized
		setTimeout(() => {
			this.connect().catch(error => {
				console.warn('Auto-connect failed:', error);
			});
		}, 100); // Small delay to ensure DOM is ready
	}

	private getWebSocketUrl(): string {
		const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
		const host = window.location.host;
		return `${protocol}//${host}/v1/ws`;
	}

	connect(): Promise<void> {
		return new Promise((resolve, reject) => {
			if (this.ws?.readyState === WebSocket.OPEN) {
				resolve();
				return;
			}

			this.state.update(state => ({ ...state, connecting: true, error: null }));

			try {
				this.ws = new WebSocket(this.url);

				this.ws.onopen = () => {
					this.state.update(state => ({
						...state,
						connected: true,
						connecting: false,
						error: null
					}));
					this.reconnectAttempts = 0;
					this.startPingPong();
					resolve();
				};

				this.ws.onmessage = (event) => {
					try {
						console.log(event)
						const data = JSON.parse(event.data);
						// Handle pong messages immediately and asynchronously
						if (data.type === 'pong') {
							// Use setTimeout to handle pong asynchronously
							setTimeout(() => this.handlePong(), 0);
							return;
						}
						
						// Handle regular messages
						const message: WebSocketMessage = {
							id: crypto.randomUUID(),
							type: 'message',
							data,
							timestamp: new Date()
						};

						this.state.update(state => ({ ...state, lastMessage: message }));
						this.messages.update(messages => [...messages, message]);
					} catch (error) {
						console.error('Failed to parse WebSocket message:', error);
					}
				};

				this.ws.onclose = (event) => {
					this.stopPingPong();
					this.state.update(state => ({
						...state,
						connected: false,
						connecting: false
					}));

					if (!event.wasClean && this.reconnectAttempts < this.maxReconnectAttempts) {
						this.scheduleReconnect();
					}
				};

				this.ws.onerror = (error) => {
					console.error('WebSocket error:', error);
					this.state.update(state => ({
						...state,
						error: 'WebSocket connection failed',
						connecting: false
					}));
					reject(error);
				};
			} catch (error) {
				this.state.update(state => ({
					...state,
					error: 'Failed to create WebSocket connection',
					connecting: false
				}));
				reject(error);
			}
		});
	}

	private scheduleReconnect(): void {
		this.reconnectAttempts++;
		const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);

		setTimeout(() => {
			if (this.ws?.readyState !== WebSocket.OPEN) {
				this.connect().catch(console.error);
			}
		}, delay);
	}

	send(data: any): boolean {
		if (this.ws?.readyState === WebSocket.OPEN) {
			try {
				this.ws.send(JSON.stringify(data));
				return true;
			} catch (error) {
				console.error('Failed to send WebSocket message:', error);
				return false;
			}
		}
		return false;
	}

	disconnect(): void {
		this.stopPingPong();
		if (this.ws) {
			this.ws.close(1000, 'Client disconnected');
			this.ws = null;
		}
		this.state.update(state => ({
			...state,
			connected: false,
			connecting: false,
			lastPong: null,
			latency: null
		}));
	}
	
	private startPingPong(): void {
		this.stopPingPong();
		
		this.pingInterval = setInterval(() => {
			if (this.ws?.readyState === WebSocket.OPEN) {
				this.sendPing();
			}
		}, this.pingIntervalMs);
	}
	
	private stopPingPong(): void {
		if (this.pingInterval) {
			clearInterval(this.pingInterval);
			this.pingInterval = null;
		}
		if (this.pingTimeout) {
			clearTimeout(this.pingTimeout);
			this.pingTimeout = null;
		}
	}
	
	private sendPing(): void {
		if (this.ws?.readyState === WebSocket.OPEN) {
			this.lastPingTime = new Date();
			
			// Clear any existing timeout first
			if (this.pingTimeout) {
				clearTimeout(this.pingTimeout);
			}
			
			// Set timeout for pong response
			this.pingTimeout = setTimeout(() => {
				console.warn('Ping timeout - no pong received');
				this.handlePingTimeout();
			}, this.pongTimeoutMs);
			
			try {
				// Send ping message
				this.ws.send(JSON.stringify({ type: 'ping', timestamp: this.lastPingTime.getTime() }));
			} catch (error) {
				console.error('Failed to send ping:', error);
				if (this.pingTimeout) {
					clearTimeout(this.pingTimeout);
					this.pingTimeout = null;
				}
			}
		}
	}
	
	private handlePong(): void {
		// Clear the timeout immediately when pong is received
		if (this.pingTimeout) {
			clearTimeout(this.pingTimeout);
			this.pingTimeout = null;
		}
		
		const now = new Date();
		const latency = this.lastPingTime ? now.getTime() - this.lastPingTime.getTime() : null;
		
		// Update state asynchronously to prevent blocking
		Promise.resolve().then(() => {
			this.state.update(state => ({
				...state,
				lastPong: now,
				latency,
				error: null // Clear any previous ping-related errors
			}));
		});
	}
	
	private handlePingTimeout(): void {
		console.error('WebSocket ping timeout - connection may be dead');
		
		// Clear the timeout reference
		this.pingTimeout = null;
		
		this.state.update(state => ({
			...state,
			error: 'Connection timeout - ping failed'
		}));
		
		// Force reconnection asynchronously to avoid blocking
		setTimeout(() => {
			if (this.ws && this.ws.readyState === WebSocket.OPEN) {
				this.ws.close(1000, 'Ping timeout');
			}
		}, 0);
	}

	// Derived store for connection status
	get connected() {
		return derived(this.state, state => state.connected);
	}

	// Derived store for errors
	get error() {
		return derived(this.state, state => state.error);
	}
}

// Create singleton instance
export const websocketManager = new WebSocketManager();

// Export stores for easy access
export const websocketState = websocketManager.state;
export const websocketMessages = websocketManager.messages;
export const websocketConnected = websocketManager.connected;
export const websocketError = websocketManager.error;