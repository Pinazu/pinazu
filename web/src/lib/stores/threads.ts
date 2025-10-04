import { writable, derived } from 'svelte/store';
import { apiClient } from '$lib/api/client';
import type { 
	Thread, 
	CreateThreadRequest, 
	UpdateThreadRequest,
	Message,
	CreateMessageRequest,
	UpdateMessageRequest
} from '$lib/types/api';

// Store state interfaces
interface ThreadsState {
	threads: Thread[];
	loading: boolean;
	error: string | null;
	selectedThread: Thread | null;
	messages: Record<string, Message[]>; // threadId -> messages[]
	loadingMessages: Record<string, boolean>; // threadId -> loading state
}

interface ThreadOperationState {
	creating: boolean;
	updating: boolean;
	deleting: boolean;
	creatingMessage: boolean;
	updatingMessage: boolean;
	deletingMessage: boolean;
}

// Initialize stores
const threadsState = writable<ThreadsState>({
	threads: [],
	loading: false,
	error: null,
	selectedThread: null,
	messages: {},
	loadingMessages: {}
});

const operationState = writable<ThreadOperationState>({
	creating: false,
	updating: false,
	deleting: false,
	creatingMessage: false,
	updatingMessage: false,
	deletingMessage: false
});

// Store actions
const actions = {
	// Load all threads
	async loadThreads() {
		threadsState.update(state => ({ ...state, loading: true, error: null }));
		
		const response = await apiClient.getThreads();
		
		if (response.success) {
			threadsState.update(state => ({
				...state,
				threads: response.data.threads,
				loading: false
			}));
		} else {
			threadsState.update(state => ({
				...state,
				loading: false,
				error: response.error.message || 'Failed to load threads'
			}));
		}
	},

	// Load a specific thread
	async loadThread(threadId: string) {
		threadsState.update(state => ({ ...state, loading: true, error: null }));
		
		const response = await apiClient.getThread(threadId);
		
		if (response.success) {
			threadsState.update(state => ({
				...state,
				selectedThread: response.data,
				loading: false
			}));
		} else {
			threadsState.update(state => ({
				...state,
				loading: false,
				error: response.error.message || 'Failed to load thread'
			}));
		}
	},

	// Create new thread
	async createThread(threadData: CreateThreadRequest): Promise<Thread | null> {
		operationState.update(state => ({ ...state, creating: true }));
		threadsState.update(state => ({ ...state, error: null }));
		
		const response = await apiClient.createThread(threadData);
		
		operationState.update(state => ({ ...state, creating: false }));
		
		if (response.success) {
			// Add new thread to the list
			threadsState.update(state => ({
				...state,
				threads: [response.data, ...state.threads]
			}));
			return response.data;
		} else {
			threadsState.update(state => ({
				...state,
				error: response.error.message || 'Failed to create thread'
			}));
			return null;
		}
	},

	// Update existing thread
	async updateThread(threadId: string, threadData: UpdateThreadRequest): Promise<Thread | null> {
		operationState.update(state => ({ ...state, updating: true }));
		threadsState.update(state => ({ ...state, error: null }));
		
		const response = await apiClient.updateThread(threadId, threadData);
		
		operationState.update(state => ({ ...state, updating: false }));
		
		if (response.success) {
			// Update thread in the list
			threadsState.update(state => ({
				...state,
				threads: state.threads.map(thread => 
					thread.thread_id === threadId ? response.data : thread
				),
				selectedThread: state.selectedThread?.thread_id === threadId ? response.data : state.selectedThread
			}));
			return response.data;
		} else {
			threadsState.update(state => ({
				...state,
				error: response.error.message || 'Failed to update thread'
			}));
			return null;
		}
	},

	// Delete thread
	async deleteThread(threadId: string): Promise<boolean> {
		operationState.update(state => ({ ...state, deleting: true }));
		threadsState.update(state => ({ ...state, error: null }));
		
		const response = await apiClient.deleteThread(threadId);
		
		operationState.update(state => ({ ...state, deleting: false }));
		
		if (response.success) {
			// Remove thread from the list and clean up messages
			threadsState.update(state => {
				const { [threadId]: deletedMessages, ...remainingMessages } = state.messages;
				const { [threadId]: deletedLoading, ...remainingLoading } = state.loadingMessages;
				
				return {
					...state,
					threads: state.threads.filter(thread => thread.thread_id !== threadId),
					selectedThread: state.selectedThread?.thread_id === threadId ? null : state.selectedThread,
					messages: remainingMessages,
					loadingMessages: remainingLoading
				};
			});
			return true;
		} else {
			threadsState.update(state => ({
				...state,
				error: response.error.message || 'Failed to delete thread'
			}));
			return false;
		}
	},

	// Select a thread
	selectThread(thread: Thread | null) {
		threadsState.update(state => ({ ...state, selectedThread: thread }));
	},

	// Load messages for a thread
	async loadMessages(threadId: string) {
		threadsState.update(state => ({
			...state,
			loadingMessages: { ...state.loadingMessages, [threadId]: true },
			error: null
		}));
		
		const response = await apiClient.getMessages(threadId);
		
		if (response.success) {
			threadsState.update(state => ({
				...state,
				messages: { ...state.messages, [threadId]: response.data.messages },
				loadingMessages: { ...state.loadingMessages, [threadId]: false }
			}));
		} else {
			threadsState.update(state => ({
				...state,
				loadingMessages: { ...state.loadingMessages, [threadId]: false },
				error: response.error.message || 'Failed to load messages'
			}));
		}
	},

	// Create message in thread
	async createMessage(threadId: string, messageData: CreateMessageRequest): Promise<Message | null> {
		operationState.update(state => ({ ...state, creatingMessage: true }));
		threadsState.update(state => ({ ...state, error: null }));
		
		const response = await apiClient.createMessage(threadId, messageData);
		
		operationState.update(state => ({ ...state, creatingMessage: false }));
		
		if (response.success) {
			// Add message to the thread's messages
			threadsState.update(state => ({
				...state,
				messages: {
					...state.messages,
					[threadId]: [...(state.messages[threadId] || []), response.data]
				}
			}));
			return response.data;
		} else {
			threadsState.update(state => ({
				...state,
				error: response.error.message || 'Failed to create message'
			}));
			return null;
		}
	},

	// Update message
	async updateMessage(threadId: string, messageId: string, messageData: UpdateMessageRequest): Promise<Message | null> {
		operationState.update(state => ({ ...state, updatingMessage: true }));
		threadsState.update(state => ({ ...state, error: null }));
		
		const response = await apiClient.updateMessage(threadId, messageId, messageData);
		
		operationState.update(state => ({ ...state, updatingMessage: false }));
		
		if (response.success) {
			// Update message in the thread's messages
			threadsState.update(state => ({
				...state,
				messages: {
					...state.messages,
					[threadId]: (state.messages[threadId] || []).map(msg =>
						msg.message_id === messageId ? response.data : msg
					)
				}
			}));
			return response.data;
		} else {
			threadsState.update(state => ({
				...state,
				error: response.error.message || 'Failed to update message'
			}));
			return null;
		}
	},

	// Delete message
	async deleteMessage(threadId: string, messageId: string): Promise<boolean> {
		operationState.update(state => ({ ...state, deletingMessage: true }));
		threadsState.update(state => ({ ...state, error: null }));
		
		const response = await apiClient.deleteMessage(threadId, messageId);
		
		operationState.update(state => ({ ...state, deletingMessage: false }));
		
		if (response.success) {
			// Remove message from the thread's messages
			threadsState.update(state => ({
				...state,
				messages: {
					...state.messages,
					[threadId]: (state.messages[threadId] || []).filter(msg => msg.message_id !== messageId)
				}
			}));
			return true;
		} else {
			threadsState.update(state => ({
				...state,
				error: response.error.message || 'Failed to delete message'
			}));
			return false;
		}
	},

	// Get messages for a specific thread
	getThreadMessages(threadId: string): Message[] {
		const state = derived(threadsState, $state => $state);
		const currentState = get(state);
		return currentState.messages[threadId] || [];
	},

	// Clear error
	clearError() {
		threadsState.update(state => ({ ...state, error: null }));
	},

	// Reset store
	reset() {
		threadsState.set({
			threads: [],
			loading: false,
			error: null,
			selectedThread: null,
			messages: {},
			loadingMessages: {}
		});
		operationState.set({
			creating: false,
			updating: false,
			deleting: false,
			creatingMessage: false,
			updatingMessage: false,
			deletingMessage: false
		});
	}
};

// Import get function for derived stores
import { get } from 'svelte/store';

// Derived stores for easier component access
export const threads = derived(threadsState, state => state.threads);
export const threadsLoading = derived(threadsState, state => state.loading);
export const threadsError = derived(threadsState, state => state.error);
export const selectedThread = derived(threadsState, state => state.selectedThread);
export const threadMessages = derived(threadsState, state => state.messages);
export const loadingMessages = derived(threadsState, state => state.loadingMessages);

export const threadOperations = derived(operationState, state => state);
export const isCreatingThread = derived(operationState, state => state.creating);
export const isUpdatingThread = derived(operationState, state => state.updating);
export const isDeletingThread = derived(operationState, state => state.deleting);
export const isCreatingMessage = derived(operationState, state => state.creatingMessage);
export const isUpdatingMessage = derived(operationState, state => state.updatingMessage);
export const isDeletingMessage = derived(operationState, state => state.deletingMessage);

// Thread store API
export const threadStore = {
	// Subscribable stores
	threads,
	threadsLoading,
	threadsError,
	selectedThread,
	threadMessages,
	loadingMessages,
	threadOperations,
	isCreatingThread,
	isUpdatingThread,
	isDeletingThread,
	isCreatingMessage,
	isUpdatingMessage,
	isDeletingMessage,

	// Actions
	...actions
};