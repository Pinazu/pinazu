// API Client for Pinazu Core API
import type {
	Agent, AgentList, CreateAgentRequest, UpdateAgentRequest,
	User, UserList, CreateUserRequest, UpdateUserRequest,
	Thread, ThreadList, CreateThreadRequest, UpdateThreadRequest,
	Message, MessageList, CreateMessageRequest, UpdateMessageRequest,
	Permission, PermissionList, CreatePermissionRequest, UpdatePermissionRequest,
	Role, RoleList, CreateRoleRequest, UpdateRoleRequest,
	ApiResponse, ApiError
} from '$lib/types/api';

export class ApiClient {
	private baseUrl: string;
	private defaultHeaders: HeadersInit;

	constructor(baseUrl?: string) {
		this.baseUrl = baseUrl || this.getDefaultBaseUrl();
		this.defaultHeaders = {
			'Content-Type': 'application/json',
		};
	}

	private getDefaultBaseUrl(): string {
		// Use same protocol as current page
		const protocol = typeof window !== 'undefined' ? window.location.protocol : 'http:';
		const host = typeof window !== 'undefined' ? window.location.host : 'localhost:8080';
		
		// If running in dev mode, always use localhost:8080
		if (typeof window !== 'undefined' && window.location.port === '5173') {
			return 'http://localhost:8080';
		}
		
		return `${protocol}//${host}`;
	}

	private async request<T>(
		endpoint: string, 
		options: RequestInit = {}
	): Promise<ApiResponse<T>> {
		try {
			const url = `${this.baseUrl}/v1${endpoint}`;
			const response = await fetch(url, {
				headers: { ...this.defaultHeaders, ...options.headers },
				...options,
			});

			const data = await response.json();

			if (!response.ok) {
				return {
					success: false,
					error: {
						error: `HTTP ${response.status}`,
						message: data.message || response.statusText,
						details: data
					}
				};
			}

			return {
				success: true,
				data
			};
		} catch (error) {
			return {
				success: false,
				error: {
					error: 'Network Error',
					message: error instanceof Error ? error.message : 'Unknown error',
					details: error
				}
			};
		}
	}

	// Agent endpoints
	async getAgents(): Promise<ApiResponse<AgentList>> {
		return this.request<AgentList>('/agents');
	}

	async getAgent(agentId: string): Promise<ApiResponse<Agent>> {
		return this.request<Agent>(`/agents/${agentId}`);
	}

	async createAgent(agent: CreateAgentRequest): Promise<ApiResponse<Agent>> {
		return this.request<Agent>('/agents', {
			method: 'POST',
			body: JSON.stringify(agent)
		});
	}

	async updateAgent(agentId: string, agent: UpdateAgentRequest): Promise<ApiResponse<Agent>> {
		return this.request<Agent>(`/agents/${agentId}`, {
			method: 'PUT',
			body: JSON.stringify(agent)
		});
	}

	async deleteAgent(agentId: string): Promise<ApiResponse<void>> {
		return this.request<void>(`/agents/${agentId}`, {
			method: 'DELETE'
		});
	}

	// User endpoints
	async getUsers(): Promise<ApiResponse<UserList>> {
		return this.request<UserList>('/users');
	}

	async getUser(userId: string): Promise<ApiResponse<User>> {
		return this.request<User>(`/users/${userId}`);
	}

	async createUser(user: CreateUserRequest): Promise<ApiResponse<User>> {
		return this.request<User>('/users', {
			method: 'POST',
			body: JSON.stringify(user)
		});
	}

	async updateUser(userId: string, user: UpdateUserRequest): Promise<ApiResponse<User>> {
		return this.request<User>(`/users/${userId}`, {
			method: 'PUT',
			body: JSON.stringify(user)
		});
	}

	async deleteUser(userId: string): Promise<ApiResponse<void>> {
		return this.request<void>(`/users/${userId}`, {
			method: 'DELETE'
		});
	}

	// Thread endpoints
	async getThreads(): Promise<ApiResponse<ThreadList>> {
		return this.request<ThreadList>('/threads');
	}

	async getThread(threadId: string): Promise<ApiResponse<Thread>> {
		return this.request<Thread>(`/threads/${threadId}`);
	}

	async createThread(thread: CreateThreadRequest): Promise<ApiResponse<Thread>> {
		return this.request<Thread>('/threads', {
			method: 'POST',
			body: JSON.stringify(thread)
		});
	}

	async updateThread(threadId: string, thread: UpdateThreadRequest): Promise<ApiResponse<Thread>> {
		return this.request<Thread>(`/threads/${threadId}`, {
			method: 'PUT',
			body: JSON.stringify(thread)
		});
	}

	async deleteThread(threadId: string): Promise<ApiResponse<void>> {
		return this.request<void>(`/threads/${threadId}`, {
			method: 'DELETE'
		});
	}

	// Message endpoints
	async getMessages(threadId: string): Promise<ApiResponse<MessageList>> {
		return this.request<MessageList>(`/threads/${threadId}/messages`);
	}

	async getMessage(threadId: string, messageId: string): Promise<ApiResponse<Message>> {
		return this.request<Message>(`/threads/${threadId}/messages/${messageId}`);
	}

	async createMessage(threadId: string, message: CreateMessageRequest): Promise<ApiResponse<Message>> {
		return this.request<Message>(`/threads/${threadId}/messages`, {
			method: 'POST',
			body: JSON.stringify(message)
		});
	}

	async updateMessage(threadId: string, messageId: string, message: UpdateMessageRequest): Promise<ApiResponse<Message>> {
		return this.request<Message>(`/threads/${threadId}/messages/${messageId}`, {
			method: 'PUT',
			body: JSON.stringify(message)
		});
	}

	async deleteMessage(threadId: string, messageId: string): Promise<ApiResponse<void>> {
		return this.request<void>(`/threads/${threadId}/messages/${messageId}`, {
			method: 'DELETE'
		});
	}

	// Permission endpoints
	async getPermissions(): Promise<ApiResponse<PermissionList>> {
		return this.request<PermissionList>('/permissions');
	}

	async getPermission(permissionId: string): Promise<ApiResponse<Permission>> {
		return this.request<Permission>(`/permissions/${permissionId}`);
	}

	async createPermission(permission: CreatePermissionRequest): Promise<ApiResponse<Permission>> {
		return this.request<Permission>('/permissions', {
			method: 'POST',
			body: JSON.stringify(permission)
		});
	}

	async updatePermission(permissionId: string, permission: UpdatePermissionRequest): Promise<ApiResponse<Permission>> {
		return this.request<Permission>(`/permissions/${permissionId}`, {
			method: 'PUT',
			body: JSON.stringify(permission)
		});
	}

	async deletePermission(permissionId: string): Promise<ApiResponse<void>> {
		return this.request<void>(`/permissions/${permissionId}`, {
			method: 'DELETE'
		});
	}

	// Role endpoints
	async getRoles(): Promise<ApiResponse<RoleList>> {
		return this.request<RoleList>('/roles');
	}

	async getRole(roleId: string): Promise<ApiResponse<Role>> {
		return this.request<Role>(`/roles/${roleId}`);
	}

	async createRole(role: CreateRoleRequest): Promise<ApiResponse<Role>> {
		return this.request<Role>('/roles', {
			method: 'POST',
			body: JSON.stringify(role)
		});
	}

	async updateRole(roleId: string, role: UpdateRoleRequest): Promise<ApiResponse<Role>> {
		return this.request<Role>(`/roles/${roleId}`, {
			method: 'PUT',
			body: JSON.stringify(role)
		});
	}

	async deleteRole(roleId: string): Promise<ApiResponse<void>> {
		return this.request<void>(`/roles/${roleId}`, {
			method: 'DELETE'
		});
	}
}

// Singleton instance
export const apiClient = new ApiClient();