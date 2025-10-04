import { writable, derived, get } from 'svelte/store';
import { apiClient } from '$lib/api/client';
import type { User, CreateUserRequest, UpdateUserRequest } from '$lib/types/api';

// Store state interfaces
interface UsersState {
	users: User[];
	loading: boolean;
	error: string | null;
	currentUser: User | null;
	selectedUser: User | null;
}

interface UserOperationState {
	creating: boolean;
	updating: boolean;
	deleting: boolean;
	authenticating: boolean;
}

// Initialize stores
const usersState = writable<UsersState>({
	users: [],
	loading: false,
	error: null,
	currentUser: null,
	selectedUser: null
});

const operationState = writable<UserOperationState>({
	creating: false,
	updating: false,
	deleting: false,
	authenticating: false
});

// Store actions
const actions = {
	// Load all users
	async loadUsers() {
		usersState.update(state => ({ ...state, loading: true, error: null }));
		
		const response = await apiClient.getUsers();
		
		if (response.success) {
			usersState.update(state => ({
				...state,
				users: response.data.users,
				loading: false
			}));
		} else {
			usersState.update(state => ({
				...state,
				loading: false,
				error: response.error.message || 'Failed to load users'
			}));
		}
	},

	// Load a specific user
	async loadUser(userId: string) {
		usersState.update(state => ({ ...state, loading: true, error: null }));
		
		const response = await apiClient.getUser(userId);
		
		if (response.success) {
			usersState.update(state => ({
				...state,
				selectedUser: response.data,
				loading: false
			}));
		} else {
			usersState.update(state => ({
				...state,
				loading: false,
				error: response.error.message || 'Failed to load user'
			}));
		}
	},

	// Create new user
	async createUser(userData: CreateUserRequest): Promise<User | null> {
		operationState.update(state => ({ ...state, creating: true }));
		usersState.update(state => ({ ...state, error: null }));
		
		const response = await apiClient.createUser(userData);
		
		operationState.update(state => ({ ...state, creating: false }));
		
		if (response.success) {
			// Add new user to the list
			usersState.update(state => ({
				...state,
				users: [...state.users, response.data]
			}));
			return response.data;
		} else {
			usersState.update(state => ({
				...state,
				error: response.error.message || 'Failed to create user'
			}));
			return null;
		}
	},

	// Update existing user
	async updateUser(userId: string, userData: UpdateUserRequest): Promise<User | null> {
		operationState.update(state => ({ ...state, updating: true }));
		usersState.update(state => ({ ...state, error: null }));
		
		const response = await apiClient.updateUser(userId, userData);
		
		operationState.update(state => ({ ...state, updating: false }));
		
		if (response.success) {
			// Update user in the list
			usersState.update(state => ({
				...state,
				users: state.users.map(user => 
					user.user_id === userId ? response.data : user
				),
				selectedUser: state.selectedUser?.user_id === userId ? response.data : state.selectedUser,
				currentUser: state.currentUser?.user_id === userId ? response.data : state.currentUser
			}));
			return response.data;
		} else {
			usersState.update(state => ({
				...state,
				error: response.error.message || 'Failed to update user'
			}));
			return null;
		}
	},

	// Delete user
	async deleteUser(userId: string): Promise<boolean> {
		operationState.update(state => ({ ...state, deleting: true }));
		usersState.update(state => ({ ...state, error: null }));
		
		const response = await apiClient.deleteUser(userId);
		
		operationState.update(state => ({ ...state, deleting: false }));
		
		if (response.success) {
			// Remove user from the list
			usersState.update(state => ({
				...state,
				users: state.users.filter(user => user.user_id !== userId),
				selectedUser: state.selectedUser?.user_id === userId ? null : state.selectedUser,
				currentUser: state.currentUser?.user_id === userId ? null : state.currentUser
			}));
			return true;
		} else {
			usersState.update(state => ({
				...state,
				error: response.error.message || 'Failed to delete user'
			}));
			return false;
		}
	},

	// Set current user (for authentication)
	setCurrentUser(user: User | null) {
		usersState.update(state => ({ ...state, currentUser: user }));
	},

	// Select a user
	selectUser(user: User | null) {
		usersState.update(state => ({ ...state, selectedUser: user }));
	},

	// Update user online status
	async updateOnlineStatus(userId: string, isOnline: boolean): Promise<boolean> {
		const response = await apiClient.updateUser(userId, { is_online: isOnline });
		
		if (response.success) {
			usersState.update(state => ({
				...state,
				users: state.users.map(user => 
					user.user_id === userId ? response.data : user
				),
				currentUser: state.currentUser?.user_id === userId ? response.data : state.currentUser
			}));
			return true;
		}
		return false;
	},

	// Authentication helpers
	async authenticate(email: string, password: string): Promise<User | null> {
		operationState.update(state => ({ ...state, authenticating: true }));
		usersState.update(state => ({ ...state, error: null }));
		
		// Since the API doesn't have explicit auth endpoint in the OpenAPI spec,
		// this would need to be implemented based on actual auth system
		// For now, we'll simulate by finding a user with matching email
		const response = await apiClient.getUsers();
		
		operationState.update(state => ({ ...state, authenticating: false }));
		
		if (response.success) {
			const user = response.data.users.find(u => u.email === email);
			if (user) {
				usersState.update(state => ({ ...state, currentUser: user }));
				return user;
			} else {
				usersState.update(state => ({
					...state,
					error: 'Invalid credentials'
				}));
			}
		} else {
			usersState.update(state => ({
				...state,
				error: response.error.message || 'Authentication failed'
			}));
		}
		
		return null;
	},

	// Logout
	logout() {
		usersState.update(state => ({ ...state, currentUser: null }));
	},

	// Clear error
	clearError() {
		usersState.update(state => ({ ...state, error: null }));
	},

	// Reset store
	reset() {
		usersState.set({
			users: [],
			loading: false,
			error: null,
			currentUser: null,
			selectedUser: null
		});
		operationState.set({
			creating: false,
			updating: false,
			deleting: false,
			authenticating: false
		});
	}
};

// Derived stores for easier component access
export const users = derived(usersState, state => state.users);
export const usersLoading = derived(usersState, state => state.loading);
export const usersError = derived(usersState, state => state.error);
export const currentUser = derived(usersState, state => state.currentUser);
export const selectedUser = derived(usersState, state => state.selectedUser);
export const isAuthenticated = derived(usersState, state => state.currentUser !== null);

export const userOperations = derived(operationState, state => state);
export const isCreatingUser = derived(operationState, state => state.creating);
export const isUpdatingUser = derived(operationState, state => state.updating);
export const isDeletingUser = derived(operationState, state => state.deleting);
export const isAuthenticating = derived(operationState, state => state.authenticating);

// User store API
export const userStore = {
	// Subscribable stores
	users,
	usersLoading,
	usersError,
	currentUser,
	selectedUser,
	isAuthenticated,
	userOperations,
	isCreatingUser,
	isUpdatingUser,
	isDeletingUser,
	isAuthenticating,

	// Actions
	...actions
};