import { writable, derived, get } from 'svelte/store';
import { apiClient } from '$lib/api/client';
import type { Agent, CreateAgentRequest, UpdateAgentRequest } from '$lib/types/api';

// Store state interfaces
interface AgentsState {
	agents: Agent[];
	loading: boolean;
	error: string | null;
	selectedAgent: Agent | null;
}

interface AgentOperationState {
	creating: boolean;
	updating: boolean;
	deleting: boolean;
}

// Initialize stores
const agentsState = writable<AgentsState>({
	agents: [],
	loading: false,
	error: null,
	selectedAgent: null
});

const operationState = writable<AgentOperationState>({
	creating: false,
	updating: false,
	deleting: false
});

// Store actions
const actions = {
	// Load all agents
	async loadAgents() {
		agentsState.update(state => ({ ...state, loading: true, error: null }));
		
		const response = await apiClient.getAgents();
		
		if (response.success) {
			agentsState.update(state => ({
				...state,
				agents: response.data.agents,
				loading: false
			}));
		} else {
			agentsState.update(state => ({
				...state,
				loading: false,
				error: response.error.message || 'Failed to load agents'
			}));
		}
	},

	// Load a specific agent
	async loadAgent(agentId: string) {
		agentsState.update(state => ({ ...state, loading: true, error: null }));
		
		const response = await apiClient.getAgent(agentId);
		
		if (response.success) {
			agentsState.update(state => ({
				...state,
				selectedAgent: response.data,
				loading: false
			}));
		} else {
			agentsState.update(state => ({
				...state,
				loading: false,
				error: response.error.message || 'Failed to load agent'
			}));
		}
	},

	// Create new agent
	async createAgent(agentData: CreateAgentRequest): Promise<Agent | null> {
		operationState.update(state => ({ ...state, creating: true }));
		agentsState.update(state => ({ ...state, error: null }));
		
		const response = await apiClient.createAgent(agentData);
		
		operationState.update(state => ({ ...state, creating: false }));
		
		if (response.success) {
			// Add new agent to the list
			agentsState.update(state => ({
				...state,
				agents: [...state.agents, response.data]
			}));
			return response.data;
		} else {
			agentsState.update(state => ({
				...state,
				error: response.error.message || 'Failed to create agent'
			}));
			return null;
		}
	},

	// Update existing agent
	async updateAgent(agentId: string, agentData: UpdateAgentRequest): Promise<Agent | null> {
		operationState.update(state => ({ ...state, updating: true }));
		agentsState.update(state => ({ ...state, error: null }));
		
		const response = await apiClient.updateAgent(agentId, agentData);
		
		operationState.update(state => ({ ...state, updating: false }));
		
		if (response.success) {
			// Update agent in the list
			agentsState.update(state => ({
				...state,
				agents: state.agents.map(agent => 
					agent.agent_id === agentId ? response.data : agent
				),
				selectedAgent: state.selectedAgent?.agent_id === agentId ? response.data : state.selectedAgent
			}));
			return response.data;
		} else {
			agentsState.update(state => ({
				...state,
				error: response.error.message || 'Failed to update agent'
			}));
			return null;
		}
	},

	// Delete agent
	async deleteAgent(agentId: string): Promise<boolean> {
		operationState.update(state => ({ ...state, deleting: true }));
		agentsState.update(state => ({ ...state, error: null }));
		
		const response = await apiClient.deleteAgent(agentId);
		
		operationState.update(state => ({ ...state, deleting: false }));
		
		if (response.success) {
			// Remove agent from the list
			agentsState.update(state => ({
				...state,
				agents: state.agents.filter(agent => agent.agent_id !== agentId),
				selectedAgent: state.selectedAgent?.agent_id === agentId ? null : state.selectedAgent
			}));
			return true;
		} else {
			agentsState.update(state => ({
				...state,
				error: response.error.message || 'Failed to delete agent'
			}));
			return false;
		}
	},

	// Select an agent
	selectAgent(agent: Agent | null) {
		agentsState.update(state => ({ ...state, selectedAgent: agent }));
	},

	// Clear error
	clearError() {
		agentsState.update(state => ({ ...state, error: null }));
	},

	// Reset store
	reset() {
		agentsState.set({
			agents: [],
			loading: false,
			error: null,
			selectedAgent: null
		});
		operationState.set({
			creating: false,
			updating: false,
			deleting: false
		});
	}
};

// Derived stores for easier component access
export const agents = derived(agentsState, state => state.agents);
export const agentsLoading = derived(agentsState, state => state.loading);
export const agentsError = derived(agentsState, state => state.error);
export const selectedAgent = derived(agentsState, state => state.selectedAgent);

export const agentOperations = derived(operationState, state => state);
export const isCreatingAgent = derived(operationState, state => state.creating);
export const isUpdatingAgent = derived(operationState, state => state.updating);
export const isDeletingAgent = derived(operationState, state => state.deleting);

// Agent store API
export const agentStore = {
	// Subscribable stores
	agents,
	agentsLoading,
	agentsError,
	selectedAgent,
	agentOperations,
	isCreatingAgent,
	isUpdatingAgent,
	isDeletingAgent,

	// Actions
	...actions
};