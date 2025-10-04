import AnthropicIcon from '$lib/icons/AnthropicIcon.svelte';
import AwsIcon from '../icons/AwsIcon.svelte';
import DeepseekIcon from '../icons/DeepseekIcon.svelte';
import GeminiIcon from '../icons/GeminiIcon.svelte';
import MetaIcons from '../icons/MetaIcons.svelte';
import WriterIcon from '../icons/WriterIcon.svelte';
import OpenaiIcon from '../icons/OpenaiIcon.svelte';

// Define the type for the model
export interface Model {
	id: string;
	description: string;
	provider: string;
	icon: any;
	capableOf: ModelCapability;
}

// Define the type for the model capability
export interface ModelCapability {
	reasoning: boolean;
	images: boolean;
	videos: boolean;
	documents: boolean;
	agentic: boolean;
}

// Define the list of models
export const MODEL_LIST: Record<string, Model> = {
	"Claude Opus 4.1": {
		id: "anthropic.claude-opus-4-1-20250805-v1:0",
		description: "Powerful, large model for complex challeges",
		provider: "anthropic",
		icon: AnthropicIcon,
		capableOf: {
			reasoning: true,
			images: true,
			videos: false,
			documents: true,
			agentic: true
		}
	},
	"Claude Sonnet 4": {
		id: "anthropic.claude-sonnet-4-20250514-v1:0",
		description: "Most capable model for complex tasks",
		provider: "anthropic",
		icon: AnthropicIcon,
		capableOf: {
			reasoning: true,
			images: true,
			videos: false,
			documents: true,
			agentic: true
		}
	},
	"Claude 3.5 Haiku": {
		id: "anthropic.claude-3-5-haiku-20241022-v1:0",
		description: "Fast and efficient for everyday tasks",
		provider: "anthropic",
		icon: AnthropicIcon,
		capableOf: {
			reasoning: true,
			images: false,
			videos: false,
			documents: true,
			agentic: true
		}
	},
	"Claude 3.7 Sonnet": {
		id: "anthropic.claude-3-7-sonnet-20250219-v1:0",
		description: "First hybrid model with the ability to solve complex problems with careful, step-by-step reasoning",
		provider: "anthropic",
		icon: AnthropicIcon,
		capableOf: {
			reasoning: true,
			images: true,
			videos: false,
			documents: true,
			agentic: true
		}
	},
	"Claude 3.5 Sonnet v2": {
		id: "anthropic.claude-3-5-sonnet-20241022-v2:0",
		description: "Improved version of the Sonnet model with better coding and agentic abilities",
		provider: "anthropic",
		icon: AnthropicIcon,
		capableOf: {
			reasoning: false,
			images: true,
			videos: false,
			documents: true,
			agentic: true
		}
	},
	"Claude 3.5 Sonnet": {
		id: "anthropic.claude-3-5-sonnet-20240620-v1:0",
		description: "Ground breaking multimodal model for agentic workflows",
		provider: "anthropic",
		icon: AnthropicIcon,
		capableOf: {
			reasoning: false,
			images: true,
			videos: false,
			documents: true,
			agentic: true
		}
	},
	"Claude 3 Haiku": {
		id: "anthropic.claude-3-haiku-20240307-v1:0",
		description: "Fastest and cheapest model for quick responses",
		provider: "anthropic",
		icon: AnthropicIcon,
		capableOf: {
			reasoning: false,
			images: true,
			videos: false,
			documents: true,
			agentic: true
		}
	},
	"Llama 4 Maverick": {
		id: "anthropic.claude-3-haiku-20240307-v1:0",
		description: "Industry-leading natively multimodal model for image and text understanding with fast responses at a low cost",
		provider: "bedrock",
		icon: MetaIcons,
		capableOf: {
			reasoning: false,
			images: true,
			videos: false,
			documents: true,
			agentic: true
		}
	},
	"Llama 4 Scout": {
		id: "anthropic.claude-3-haiku-20240307-v1:0",
		description: "Class-leading natively multimodal model with 10M context window for seamless long document analysis.",
		provider: "bedrock",
		icon: MetaIcons,
		capableOf: {
			reasoning: false,
			images: true,
			videos: false,
			documents: true,
			agentic: false
		}
	},
	"Nova Premier": {
		id: "amazon.nova-premier-v1:0",
		description: "Most capable model for complex tasks and teacher for model distillation",
		provider: "bedrock",
		icon: AwsIcon,
		capableOf: {
			reasoning: true,
			images: true,
			videos: true,
			documents: true,
			agentic: true
		}
	},
	"Nova Pro": {
		id: "amazon.nova-pro-v1:0",
		description: "Highly capable multimodal model with the best combination of accuracy, speed, and cost for a wide range of tasks",
		provider: "bedrock",
		icon: AwsIcon,
		capableOf: {
			reasoning: true,
			images: true,
			videos: true,
			documents: true,
			agentic: true
		}
	},
	"Nova Lite": {
		id: "amazon.nova-lite-v1:0",
		description: "Low-cost multimodal model that is lightning fast for processing image, video, and text inputs to generate text outputs",
		provider: "bedrock",
		icon: AwsIcon,
		capableOf: {
			reasoning: true,
			images: true,
			videos: true,
			documents: true,
			agentic: false
		}
	},
	"Nova Micro": {
		id: "amazon.nova-micro-v1:0",
		description: "Text-only model that delivers the lowest latency responses at a very low cost",
		provider: "bedrock",
		icon: AwsIcon,
		capableOf: {
			reasoning: true,
			images: false,
			videos: false,
			documents: true,
			agentic: false
		}
	},
	"DeepSeek R1": {
		id: "deepseek.r1-v1:0",
		description: "High-performance open source model that excels reasoning, coding, and multilingual understanding.",
		provider: "bedrock",
		icon: DeepseekIcon,
		capableOf: {
			reasoning: true,
			images: false,
			videos: false,
			documents: true,
			agentic: false
		}
	},
	"Palmyra X4": {
		id: "writer.palmyra-x4-v1:0",
		description: "Excels in processing and understanding complex tasks, making it ideal for workflow automation",
		provider: "bedrock",
		icon: WriterIcon,
		capableOf: {
			reasoning: false,
			images: false,
			videos: false,
			documents: true,
			agentic: true,
		}
	},
	"Palmyra X5": {
		id: "writer.palmyra-x5-v1:0",
		description: "A suite of enterprise-ready capabilities, including advanced reasoning, tool-calling",
		provider: "bedrock",
		icon: WriterIcon,
		capableOf: {
			reasoning: true,
			images: false,
			videos: false,
			documents: true,
			agentic: false,
		}
	},
	"Gemini Pro 2.5": {
		id: "gemini-2.5-pro",
		description: "A suite of enterprise-ready capabilities, including advanced reasoning, tool-calling",
		provider: "google",
		icon: GeminiIcon,
		capableOf: {
			reasoning: true,
			images: true,
			videos: true,
			documents: true,
			agentic: true,
		}
	},
	"Gemini Flash 2.5": {
		id: "gemini-2.5-flash",
		description: "A suite of enterprise-ready capabilities, including advanced reasoning, tool-calling",
		provider: "google",
		icon: GeminiIcon,
		capableOf: {
			reasoning: true,
			images: true,
			videos: true,
			documents: true,
			agentic: true,
		}
	},
	"Gemini Flash Lite 2.5": {
		id: "gemini-2.5-flash-lite",
		description: "A suite of enterprise-ready capabilities, including advanced reasoning, tool-calling",
		provider: "google",
		icon: GeminiIcon,
		capableOf: {
			reasoning: true,
			images: true,
			videos: true,
			documents: true,
			agentic: true,
		}
	},
	"Gemma 3n E4B": {
		id: "gemma-3n-e4b-it",
		description: "A suite of enterprise-ready capabilities, including advanced reasoning, tool-calling",
		provider: "google",
		icon: GeminiIcon,
		capableOf: {
			reasoning: false,
			images: false,
			videos: false,
			documents: true,
			agentic: true,
		}
	},
	"GPT 5": {
		id: "none",
		description: "Next-generation model with enhanced reasoning capabilities for the most challenging tasks",
		provider: "openai",
		icon: OpenaiIcon,
		capableOf: {
			reasoning: true,
			images: true,
			videos: false,
			documents: true,
			agentic: true,
		}
	},
	"GPT 5 mini": {
		id: "none",
		description: "Efficient version of GPT-5 with advanced reasoning at reduced cost",
		provider: "openai",
		icon: OpenaiIcon,
		capableOf: {
			reasoning: true,
			images: true,
			videos: false,
			documents: true,
			agentic: true,
		}
	},
	"GPT 5 nano": {
		id: "none",
		description: "Ultra-fast reasoning model optimized for quick responses and low latency",
		provider: "openai",
		icon: OpenaiIcon,
		capableOf: {
			reasoning: true,
			images: true,
			videos: false,
			documents: true,
			agentic: true,
		}
	},
	"GPT OSS 20b": {
		id: "none",
		description: "Most effective open source models with lowest latency",
		provider: "bedrock",
		icon: OpenaiIcon,
		capableOf: {
			reasoning: true,
			images: false,
			videos: false,
			documents: true,
			agentic: true,
		}
	},
	"GPT OSS 120b": {
		id: "none",
		description: "State of the art for open source models with balance in fast response, low cost and high quality",
		provider: "bedrock",
		icon: OpenaiIcon,
		capableOf: {
			reasoning: true,
			images: false,
			videos: false,
			documents: true,
			agentic: true,
		}
	},
}