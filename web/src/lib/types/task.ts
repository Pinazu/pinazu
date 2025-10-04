// Define the type for task mode
export type TaskMode = 'ask' | 'auto' | 'yolo';

// Define the type for task mode options
export interface TaskModeOption {
	id: TaskMode;
	label: string;
	description: string;
}

// Define the list of task modes with their descriptions
export const TASK_MODES: TaskModeOption[] = [
	{ id: 'ask', label: 'Ask', description: 'Ask before running the tool' },
	{ id: 'auto', label: 'Auto', description: 'Auto accept basic permission' },
	{ id: 'yolo', label: 'Yolo', description: 'Run autonomously background' }
];
