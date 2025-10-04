// API Types based on OpenAPI specification

// Core entities from OpenAPI schema
export interface Agent {
	agent_id: string;
	agent_name: string;
	agent_description?: string;
	agent_specs?: string;
	created_by: string;
	created_at: string;
	updated_at: string;
}

export interface CreateAgentRequest {
	agent_name: string;
	agent_description?: string;
	agent_specs?: string;
}

export interface UpdateAgentRequest {
	agent_name?: string;
	agent_description?: string;
	agent_specs?: string;
}

export interface User {
	user_id: string;
	username: string;
	email: string;
	additional_info?: any;
	password_hash: string;
	provider?: string;
	is_online?: boolean;
	last_login?: string;
	created_at: string;
	updated_at: string;
}

export interface CreateUserRequest {
	username: string;
	email: string;
	additional_info?: any;
	password_hash: string;
	provider?: string;
}

export interface UpdateUserRequest {
	username?: string;
	email?: string;
	additional_info?: any;
	provider?: string;
	is_online?: boolean;
}

export interface Thread {
	thread_id: string;
	thread_title: string;
	user_id: string;
	created_at: string;
	updated_at: string;
}

export interface CreateThreadRequest {
	thread_title: string;
	user_id: string;
}

export interface UpdateThreadRequest {
	thread_title: string;
}

export interface Message {
	message_id: string;
	thread_id: string;
	message: any;
	sender_type: 'user' | 'agent';
	result_type?: string;
	stop_reason?: string;
	sender_id: string;
	recipient_id: string;
	citations?: any;
	created_at: string;
	updated_at: string;
}

export interface CreateMessageRequest {
	message: any;
	sender_type: 'user' | 'agent';
	result_type?: string;
	stop_reason?: string;
	sender_id: string;
	recipient_id: string;
	citations?: any;
}

export interface UpdateMessageRequest {
	message: any;
}

export interface Permission {
	permission_id: string;
	permission_name: string;
	permission_description?: string;
	permission_content: any;
	created_at: string;
	updated_at: string;
}

export interface CreatePermissionRequest {
	permission_name: string;
	permission_description?: string;
	permission_content: any;
}

export interface UpdatePermissionRequest {
	permission_name?: string;
	permission_description?: string;
	permission_content?: any;
}

export interface Role {
	role_id: string;
	role_name: string;
	role_description?: string;
	is_system_role?: boolean;
	created_at: string;
	updated_at: string;
}

export interface CreateRoleRequest {
	role_name: string;
	role_description?: string;
	is_system_role?: boolean;
}

export interface UpdateRoleRequest {
	role_name?: string;
	role_description?: string;
	is_system_role?: boolean;
}

// List types with pagination
export interface PaginationMeta {
	total: number;
	page: number;
	per_page: number;
	total_pages: number;
}

export interface AgentList {
	agents: Agent[];
	meta?: PaginationMeta;
}

export interface UserList {
	users: User[];
	meta?: PaginationMeta;
}

export interface ThreadList {
	threads: Thread[];
	meta?: PaginationMeta;
}

export interface MessageList {
	messages: Message[];
	meta?: PaginationMeta;
}

export interface PermissionList {
	permissions: Permission[];
	meta?: PaginationMeta;
}

export interface RoleList {
	roles: Role[];
	meta?: PaginationMeta;
}

// API Error response
export interface ApiError {
	error: string;
	message?: string;
	details?: any;
}

// API Response wrapper
export type ApiResponse<T> = {
	success: true;
	data: T;
} | {
	success: false;
	error: ApiError;
};