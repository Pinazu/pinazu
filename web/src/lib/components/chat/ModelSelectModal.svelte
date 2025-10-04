<script lang="ts">
	import "../../../app.css"
	import { theme } from "$lib/stores/theme";
	import { onMount } from "svelte";
	import { MODEL_LIST, type Model } from "$lib/types/models";
	import XIcon from "$lib/icons/XIcon.svelte";
    import ImageIcon from "$lib/icons/ImageIcon.svelte";
    import VideoIcon from "$lib/icons/VideoIcon.svelte";
    import LightBulbIcon from "$lib/icons/LightBulbIcon.svelte";
    import AgenticIcon from "$lib/icons/AgenticIcon.svelte";
    import DocumentsIcon2 from "$lib/icons/DocumentsIcon2.svelte";
    import Tooltip from "../Tooltip.svelte";

	// Props
	interface Props {
		isOpen: boolean;
		currentModelName: string;
		onModelSelect: (modelName: string) => void;
		onClose: () => void;
	}

	let { isOpen, currentModelName, onModelSelect, onClose }: Props = $props();

	// Component state
	let overlayRef: HTMLDivElement | undefined = $state();

	// Provider configuration
	const PROVIDER_CONFIG: Record<string, { name: string; order: number }> = {
		'anthropic': { name: 'Anthropic', order: 1 },
		'openai': { name: 'OpenAI', order: 2 },
		'google': { name: 'Google', order: 3 },
		'bedrock': { name: 'AWS Bedrock', order: 4 }
	};

	// Utility functions
	function groupModelsByProvider() {
		return Object.entries(MODEL_LIST).reduce((groups, [modelName, modelData]) => {
			const provider = modelData.provider;
			if (!groups[provider]) {
				groups[provider] = [];
			}
			groups[provider].push({ name: modelName, data: modelData });
			return groups;
		}, {} as Record<string, Array<{ name: string, data: Model }>>);
	}

	function getSortedProviders() {
		const groupedModels = groupModelsByProvider();
		return Object.keys(groupedModels)
			.sort((a, b) => (PROVIDER_CONFIG[a]?.order || 999) - (PROVIDER_CONFIG[b]?.order || 999))
			.map(provider => ({
				key: provider,
				name: PROVIDER_CONFIG[provider]?.name || provider,
				models: groupedModels[provider].sort((a, b) => a.name.localeCompare(b.name))
			}));
	}

	// Computed data
	const sortedProviders = getSortedProviders();

	// Event handlers
	function handleOverlayClick(event: MouseEvent) {
		// Only close if clicking directly on the overlay (not on child elements)
		if (event.target === overlayRef) {
			onClose();
		}
	}

	function handleModelSelect(modelName: string) {
		onModelSelect(modelName);
	}

	function handleEscape(event: KeyboardEvent) {
		if (event.key === 'Escape' && isOpen) {
			onClose();
		}
	}

	// Setup keyboard listener
	onMount(() => {
		document.addEventListener('keydown', handleEscape);
		return () => document.removeEventListener('keydown', handleEscape);
	});
</script>

{#if isOpen}
	<!-- Modal Overlay with blur background -->
	<div 
		bind:this={overlayRef}
		onclick={handleOverlayClick}
		onkeydown={(e) => e.key === 'Enter' && e.target === overlayRef && onClose()}
		role="dialog"
		aria-modal="true"
		aria-labelledby="modal-title"
		tabindex="-1"
		class="
			fixed inset-0 z-[100]
			flex items-center justify-center 
			bg-black/20 backdrop-blur-[1px]
			transition-all duration-300
		"
	>
		<!-- Main Panel -->
		<div 
			class="
				relative 
				bg-[var(--bg-primary)]
				rounded-3xl shadow-2xl 
				border border-[var(--border-color)]
				max-w-3xl w-full mx-6
				max-h-[85vh] overflow-hidden
				transition-all duration-300
			"
		>
			<!-- Header -->
			<div class="
				flex items-center justify-between 
				p-4
				border-b border-[var(--border-color)]
			">
				<h2 
					id="modal-title"
					class="
						text-base font-medium 
						text-[var(--text-primary)]
						select-none
					"
				>
					Select Model
				</h2>
				<button
					type="button"
					onclick={onClose}
					class="
						bg-transparent hover:bg-[var(--bg-third)]/50
						flex justify-center items-center
						text-[var(--text-primary)]
						rounded-full
						hover:scale-105
						border-none cursor-pointer
						transition-all duration-200
					"
					style="width: 2.5rem; height: 2.5rem;"
				>
					<XIcon size="20" />
				</button>
			</div>

			<!-- Models Grid -->
			<div class="
				p-4 
				overflow-y-auto 
				max-h-[calc(85vh-96px)]
				scrollbar-thin
			">
				{#each sortedProviders as provider}
					<div class="mb-8">
						<!-- Provider Header -->
						<div class="mb-3 pb-2 border-b border-[var(--border-color)]/30">
							<h3 class="text-sm font-medium text-[var(--text-secondary)] uppercase tracking-wider select-none">
								{provider.name}
							</h3>
						</div>
						
						<!-- Models Grid for this provider -->
						<div class="
							grid 
							grid-cols-4
							gap-3
							auto-rows-max
						">
							{#each provider.models as { name: modelName, data: modelData }}
								{@render modelCard(modelName, modelData)}
							{/each}
						</div>
					</div>
				{/each}
			</div>
		</div>
	</div>
{/if}

{#snippet modelCard(modelName: string, modelData: Model)}
	{@const IconComponent = modelData.icon}
	{@const isSelected = currentModelName === modelName}
	
	<button
		type="button"
		onclick={() => handleModelSelect(modelName)}
		class="
			p-3 
			bg-[var(--bg-secondary)]/30
			hover:bg-[var(--bg-third)]/50
			border border-[var(--border-color)]/30
			rounded-2xl 
			cursor-pointer
			select-none
			transition-all duration-200 
			hover:scale-105
			focus:outline-none 
			text-[var(--text-primary)]
			w-full h-full
			{isSelected ? 'ring-2 ring-[var(--accent-color)] bg-[var(--accent-color)]/10' : ''}
		"
	>
		<!-- Content wrapper with relative positioning for indicator -->
		<div class="relative flex flex-col h-full min-h-[96px]">
			<!-- Selection indicator -->
			{#if isSelected}
				<div class="
					absolute -top-1 -right-1 
					w-4 h-4 
					bg-[var(--accent-color)] 
					rounded-full 
					flex items-center justify-center
				">
					<div class="w-1.5 h-1.5 bg-[var(--text-accent-color)] rounded-full"></div>
				</div>
			{/if}

			<!-- Model Icon -->
			<div class="flex justify-center mb-2 flex-shrink-0">
				<div class="flex items-center justify-center">
					<IconComponent color={$theme === 'dark' ? '#fff' : '#111'} size="24" />
				</div>
			</div>

			<!-- Model Name -->
			<div class="text-center flex-1 flex flex-col justify-center">
				<h3 class="
					font-medium 
					text-[var(--text-primary)] 
					text-[0.8rem] 
					mb-1
					line-clamp-1
				">
					{modelName}
				</h3>
				<p class="
					text-[0.7rem] 
					text-[var(--text-secondary)] 
					line-clamp-2
					leading-tight
				">
					{modelData.description}
				</p>
			</div>

			<!-- Model Capabilities -->
			<div class="flex gap-0.5 text-center justify-center mt-2">
				{#if modelData.capableOf.reasoning}
					<Tooltip text="Support Reasoning" delay={100} position="bottom">
						{#snippet children()}
							<LightBulbIcon size="14" className={$theme === 'dark' ? 'text-yellow-300' : 'text-yellow-700'} />
						{/snippet}
					</Tooltip>
				{/if}
				{#if modelData.capableOf.images}
					<Tooltip text="Support Images Input" delay={100} position="bottom">
						{#snippet children()}
							<ImageIcon size="14" className={$theme === 'dark' ? 'text-blue-300' : 'text-blue-700'} />
						{/snippet}
					</Tooltip>
				{/if}
				{#if modelData.capableOf.videos}
					<Tooltip text="Support Videos Input" delay={100} position="bottom">
						{#snippet children()}
							<VideoIcon size="14" className={$theme === 'dark' ? 'text-red-300' : 'text-red-700'} />
						{/snippet}
					</Tooltip>
				{/if}
				{#if modelData.capableOf.documents}
					<Tooltip text="Support Text Input" delay={100} position="bottom">
						{#snippet children()}
							<DocumentsIcon2 size="14" className={$theme === 'dark' ? 'text-green-300' : 'text-green-700'} />
						{/snippet}
					</Tooltip>
				{/if}
				{#if modelData.capableOf.agentic}
					<Tooltip text="Can use tools" delay={100} position="bottom">
						{#snippet children()}
							<AgenticIcon size="14" className={$theme === 'dark' ? 'text-purple-300' : 'text-purple-700'} />
						{/snippet}
					</Tooltip>
				{/if}
			</div>
		</div>
	</button>
{/snippet}

<style>
	.line-clamp-1 {
		display: -webkit-box;
		-webkit-line-clamp: 1;
		line-clamp: 1;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	.line-clamp-2 {
		display: -webkit-box;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
</style>