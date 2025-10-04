<script lang="ts">
    import "../../../app.css"
	import { gsap } from "gsap";
    import { theme } from "$lib/stores/theme";
	import { MODEL_LIST, type Model } from "$lib/types/models"
	import { TASK_MODES, type TaskMode } from "$lib/types/task"
	import { onMount } from "svelte";
	import { goto } from '$app/navigation';
	import { apiClient } from '$lib/api/client';
	import { currentUser } from '$lib/stores/users';
	import ModelSelectModal from "./ModelSelectModal.svelte";
    import PlusIcon from "$lib/icons/PlusIcon.svelte"
	import ToolsIcon from "$lib/icons/ToolsIcon.svelte"
    import RightArrowIcon from "$lib/icons/RightArrowIcon.svelte";
    import ArrowUpIcon from "$lib/icons/ArrowUpIcon.svelte";
    import AttachmentIcon from "$lib/icons/AttachmentIcon.svelte";
    import AddAppIcon from "$lib/icons/AddAppIcon.svelte";
    import GlobeIcon from "$lib/icons/GlobeIcon.svelte";
    import MoreIcon from "$lib/icons/MoreIcon.svelte";
    import ManageToolIcon from "$lib/icons/ManageToolIcon.svelte";
    import DrawingIcon from "$lib/icons/DrawingIcon.svelte";
    import XIcon from "$lib/icons/XIcon.svelte";
    import GoogleDriveIcon from "$lib/icons/GoogleDriveIcon.svelte";
    import OneDriveIcon from "$lib/icons/OneDriveIcon.svelte";
    import MicrophoneIcon from "$lib/icons/MicrophoneIcon.svelte";
    import DocumentsIcon from "$lib/icons/DocumentsIcon.svelte";
	import Tooltip from "$lib/components/Tooltip.svelte";
    import LightBulbIcon from "$lib/icons/LightBulbIcon.svelte";

	interface MessageInputProps {
		isTaskMode: boolean;
		isBottom?: boolean;
	}

	// Define props
	let { isTaskMode, isBottom = false }: MessageInputProps = $props();
	
	// State variables
	let currentTaskModeId: TaskMode = $state(TASK_MODES[0].id) // Default to 'ask'
	let currentModelName: string = $state(Object.keys(MODEL_LIST)[0]) // Default to first model
	let showUploadDropdown = $state(false)
	let showToolsDropdown = $state(false)
	let showModelsDropdown = $state(false)
	let showTaskModeDropdown = $state(false)
	let showAddFromAppPanel = $state(false)
	let showModelSelectModal = $state(false)
	let hoverTimeout: number | undefined

	// Tool Enable states
	let isEnableExtendedThinking = $state(false)
	let isEnableCreateImage = $state(false)
	let isEnableWebSearch = $state(false)
	
	// Hover states for ViewButtons
	let isHoveringExtendedThinking = $state(false)
	let isHoveringCreateImage = $state(false)
	let isHoveringWebSearch = $state(false)
	
	// Dropdown refs
	let uploadContainer: HTMLDivElement | undefined
	let toolsContainer: HTMLDivElement | undefined
	let modelsContainer: HTMLDivElement | undefined
	let taskModeContainer: HTMLDivElement | undefined
	let addFromAppContainer: HTMLDivElement | undefined
	
	// File upload refs and state
	let fileInput: HTMLInputElement | undefined
	let uploadedFiles: File[] = $state([])
	let isUploading = $state(false)
	
	// Animation refs
	let leftButton: HTMLButtonElement | undefined;
	let rightButton: HTMLButtonElement | undefined;
	let leftTitle: HTMLSpanElement | undefined;
	let rightTitle: HTMLSpanElement | undefined;
	
	
	// Form submission state
	let isSubmitting = $state(false);
	let messageText = $state('');
	
	onMount(() => {
		// Left button hover animations
		if (leftButton && rightTitle) {
			leftButton.addEventListener('mouseenter', () => {
				if (rightTitle) {
					// Hide right title
					gsap.to(rightTitle, {
						opacity: 0,
						duration: 0.15,
						ease: "power2.out"
					});
				}
			});
			
			leftButton.addEventListener('mouseleave', () => {
				if (rightTitle) {
					// Show right title
					gsap.to(rightTitle, {
						opacity: 1,
						duration: 0.15,
						ease: "power2.out"
					});
				}
			});
		}
		
		// Right button hover animations
		if (rightButton && leftTitle) {
			rightButton.addEventListener('mouseenter', () => {
				if (leftTitle) {
					// Hide left title
					gsap.to(leftTitle, {
						opacity: 0,
						duration: 0.15,
						ease: "power2.out"
					});
				}
			});
			
			rightButton.addEventListener('mouseleave', () => {
				if (leftTitle) {
					// Show left title
					gsap.to(leftTitle, {
						opacity: 1,
						duration: 0.15,
						ease: "power2.out"
					});
				}
			});
		}
		
		// Click outside to close dropdown
		const handleClickOutside = (event: Event) => {
			const target = event.target as Node;
			const isOutsideUpload = !uploadContainer?.contains(target);
			const isOutsideTools = !toolsContainer?.contains(target);
			const isOutsideModels = !modelsContainer?.contains(target);
			const isOutsideTaskMode = !taskModeContainer?.contains(target);
			const isOutsideAddFromApp = !addFromAppContainer?.contains(target);
			
			if (isOutsideUpload) showUploadDropdown = false;
			if (isOutsideTools) showToolsDropdown = false;
			if (isOutsideModels) showModelsDropdown = false;
			if (isOutsideTaskMode) showTaskModeDropdown = false;
			if (isOutsideAddFromApp) showAddFromAppPanel = false;
		};
		
		document.addEventListener('click', handleClickOutside);
		
		return () => {
			document.removeEventListener('click', handleClickOutside);
		};
	});

	// isToolEnabled check if any tool is enabled
	function isToolEnabled(): boolean {
		return isEnableExtendedThinking || isEnableWebSearch || isEnableCreateImage;
	}

	// handlePanelEnter and handlePanelLeave are used to show/hide the add from app panel
	// Hover timeout for the add from app panel to improve UX
	function handlePanelEnter() {
		if (hoverTimeout) {
			clearTimeout(hoverTimeout);
			hoverTimeout = undefined;
		}
		showAddFromAppPanel = true;
	}
	function handlePanelLeave() {
		hoverTimeout = setTimeout(() => {
			showAddFromAppPanel = false;
		}, 150); // 150ms delay before hiding
	}

	// Handle file upload
	function handleFileSelect(event: Event) {
		const target = event.target as HTMLInputElement;
		const files = target.files;
		
		if (files && files.length > 0) {
			isUploading = true;
			const fileArray = Array.from(files);
			
			// Validate file types and sizes
			const validFiles = fileArray.filter(file => {
				const maxSize = 10 * 1024 * 1024; // 10MB
				const allowedTypes = [
					'image/', 'text/', 'application/pdf', 
					'application/json', 'application/javascript',
					'application/typescript'
				];
				
				const isValidType = allowedTypes.some(type => file.type.startsWith(type));
				const isValidSize = file.size <= maxSize;
				
				return isValidType && isValidSize;
			});
			
			uploadedFiles = [...uploadedFiles, ...validFiles];
			isUploading = false;
			
			// Clear the input so the same file can be selected again
			if (target) target.value = '';
		}
	}

	// Remove a file from the uploaded files list
	function removeFile(index: number) {
		uploadedFiles = uploadedFiles.filter((_, i) => i !== index);
	}

	// Handle form submission - create thread and navigate
	async function handleSubmit() {
		if (!messageText.trim() || isSubmitting) return;
		
		const user = $currentUser;
		if (!user) {
			console.error('No current user found');
			return;
		}

		isSubmitting = true;

		try {
			// Create a new thread
			const threadResponse = await apiClient.createThread({
				thread_title: messageText.trim().slice(0, 50) + (messageText.trim().length > 50 ? '...' : ''),
				user_id: user.user_id
			});

			if (!threadResponse.success) {
				console.error('Failed to create thread:', threadResponse.error);
				return;
			}

			const thread = threadResponse.data;

			// Create the first message in the thread
			const messageResponse = await apiClient.createMessage(thread.thread_id, {
				message: messageText.trim(),
				sender_type: 'user',
				sender_id: user.user_id,
				recipient_id: 'system' // or agent_id if you have a specific agent
			});

			if (!messageResponse.success) {
				console.error('Failed to create message:', messageResponse.error);
				return;
			}

			// Clear the input
			messageText = '';

			// Navigate to the new thread
			await goto(`/chat/${thread.thread_id}`);

		} catch (error) {
			console.error('Error creating thread:', error);
		} finally {
			isSubmitting = false;
		}
	}
</script>

<div class="pb-4 px-4 flex flex-col justify-center w-full transition-all duration-200">
	<!-- Hidden file input -->
	<input 
		bind:this={fileInput}
		type="file" 
		multiple 
		class="hidden" 
		onchange={handleFileSelect}
		accept="image/*,.txt,.pdf,.json,.js,.ts,.tsx,.jsx,.md,.csv"
	/>

	<!-- Main Input Area -->
	<div class="relative">
		<div 
			class="
				flex flex-col w-full items-center 
				bg-[var(--bg-secondary)]/50
				p-2 rounded-[1.725rem] max-w-[46rem] shadow-sm mx-auto
				border
				border-[var(--border-color)]/50
				transition-all duration-200
			"
		>
			<div class="flex justify-start w-full">
				{#if uploadedFiles.length > 0}
					{@render uploadedFilesDisplay()}
				{/if}
			</div>
			{@render textArea()}
			<div class="flex justify-between w-full">
				<!-- Left side controls -->
				<div class="flex items-center gap-0.25">
					{@render uploadButton()}
					{@render toolsButton()}
					{#if isEnableExtendedThinking}
						{@render extendedThinkingViewButton()}
					{/if}
					{#if isEnableWebSearch}
						{@render webSearchViewButton()}
					{/if}
					{#if isEnableCreateImage}
						{@render createImageViewButton()}
					{/if}
				</div>

				<!-- Right side controls -->
				<div class="flex items-center gap-2">
					<div class="flex items-center gap-0.25">
						{#if !isTaskMode}
							{@render modelButton()}
						{/if}
						{@render dicationButton()}	
					</div>
					{#if !isTaskMode}
						{@render sendButton()}
					{:else}
						{@render taskModeButton()}
					{/if}
				</div>
			</div>
		</div>

	</div>
</div>

<!-- Text Area Input with Shaw's Auto-Resize Technique -->
{#snippet textArea()}
	<div 
		class="input-sizer w-full bg-transparent rounded-lg p-2 px-3.5 mb-3 ml-1" 
		data-value={messageText}
	>
		<textarea
			bind:value={messageText}
			placeholder="Type your message here..."
			rows="1"
			disabled={isSubmitting}
			class="
				w-full 
				bg-transparent 
				text-[0.95rem] text-[var(--text-primary)]
				focus:outline-none focus:ring-0 resize-none
				transition-all duration-200
				disabled:opacity-50"
			oninput={(e) => {
				const target = e.target as HTMLTextAreaElement;
				const parent = target.parentNode as HTMLElement;
				if (parent) {
					parent.setAttribute('data-value', target.value);
				}
			}}
			onkeydown={(e) => {
				if (e.key === 'Enter' && !e.shiftKey) {
					e.preventDefault();
					handleSubmit();
				}
			}}
		></textarea>
	</div>
{/snippet}

<!-- Upload Button -->
{#snippet uploadButton()}
    <div class="relative" bind:this={uploadContainer}>
        <Tooltip text={showUploadDropdown ? "Close" : "Upload Files"} position="left" delay={0}>
            {#snippet children()}
                <button 
                    type="button" 
                    aria-label="Upload Files"
                    onclick={() => showUploadDropdown = !showUploadDropdown}
                    class="
                        bg-transparent hover:bg-[var(--bg-third)]/50
                        flex mx-1 justify-center items-center
                        text-[var(--text-primary)]
                        rounded-full
                        hover:scale-105
                        border-none cursor-pointer select-none
                        disabled:opacity-50 disabled:cursor-not-allowed
                        transition-all duration-200
                    "
                    style="width: 2rem; height: 2rem;"
                >
                    <div class="transition-transform duration-200 {showUploadDropdown ? 'rotate-45' : 'rotate-0'}">
                        <PlusIcon />
                    </div>
                </button>
            {/snippet}
        </Tooltip>
        
        {#if showUploadDropdown}
			<div 
				class="
					absolute {isBottom ? 'bottom-full' : 'top-full'} left-0
					mt-1 bg-[var(--bg-primary)]
					border border-[var(--border-color)]
					rounded-3xl shadow-lg z-50 min-w-58 p-2
				"
			>
				{@render uploadFromComputerButton()}
				{@render uploadFromExternalAppButton()}
			</div>
        {/if}
    </div>
{/snippet}

<!-- Tools Option Button -->
{#snippet toolsButton()}
    <div class="relative dropdown-container" bind:this={toolsContainer}>
		<div class="flex gap-2">
			<button 
				type="button" 
				aria-label="Tool Options"
				onclick={() => showToolsDropdown = !showToolsDropdown}
				class="
					bg-transparent hover:bg-[var(--bg-third)]/50
					flex justify-center items-center
					text-[var(--text-secondary)]
					{isToolEnabled() ? 'rounded-full ' : 'rounded-3xl px-2.5 py-1 gap-2'}
					hover:scale-105
					border-none cursor-pointer select-none
					disabled:opacity-50 disabled:cursor-not-allowed
					transition-all duration-200
				"
				style="height: 2rem; {isToolEnabled() ? 'width: 2rem;' : ''}"
			>
				<ToolsIcon />
				{#if !isToolEnabled()}
					<span class="text-[0.9rem]">Tools</span>	
				{/if}
				
			</button>

			{#if isToolEnabled()}
				<div class="border-1.5 border-l border-[var(--border-color)] mr-2"></div>
			{/if}
		</div>
        
        {#if showToolsDropdown}
			<div 
				class="
					absolute {isBottom ? 'bottom-full' : 'top-full'} left-0
					mt-1 bg-[var(--bg-primary)]
					border border-[var(--border-color)]
					rounded-3xl shadow-lg z-50 min-w-58 p-2
				"
			>	
				{#if !isTaskMode}
					{@render extendedThinkingButton()}
					{@render createImageButton()}
					{@render webSearchToolButton()}
					<div class="border-t border-[var(--border-color)] my-2"></div>
				{/if}
				{@render addConnectorButton()}
				{@render manageConnectorsButton()}
			</div>
        {/if}
    </div>
{/snippet}

<!-- Model Selection Button -->
{#snippet modelButton()}
    <div class="relative" bind:this={modelsContainer}>
        <button 
            type="button"
            aria-label="Select AI Model"
            onclick={() => showModelsDropdown = !showModelsDropdown}
            class="
				bg-transparent hover:bg-[var(--bg-third)]/50
				text-[0.9rem] text-[var(--text-secondary)]
				px-2.5 py-1.5 rounded-full
				hover:scale-105
				cursor-pointer border-none
				select-none
				transition-all duration-200
			"
        >
            <span class="font-medium text-[0.9rem]">{currentModelName}</span>
        </button>
        
        {#if showModelsDropdown}
			<div 
				class="
					absolute {isBottom ? 'bottom-full' : 'top-full'} left-0
					mt-1 bg-[var(--bg-primary)]
					border border-[var(--border-color)]
					rounded-3xl shadow-lg z-50 min-w-68 p-2
				"
			>
				<!-- Display only the first 3 models initially -->
				{#each Object.entries(MODEL_LIST).slice(0, 3) as [modelName, modelData]}
					{@render modelOptionButton(modelName, modelData)}
				{/each}
				{@render showMoreModelsButton()}
			</div>
        {/if}
    </div>
{/snippet}

{#snippet dicationButton()}
	<Tooltip text="Dictation" position={isBottom ? "top" : "bottom"} delay={0}>
		{#snippet children()}
			<button 
				type="submit"
				aria-label="Voice input"
				class="
					bg-transparent hover:bg-[var(--bg-third)]/50
					flex justify-center items-center
					text-[var(--text-primary)]
					rounded-full
					hover:scale-105
					border-none cursor-pointer
					disabled:opacity-50 disabled:cursor-not-allowed
					transition-all duration-200
				"
				style="width: 2.16rem; height: 2.16rem"
			>
				<MicrophoneIcon size="19"/>
			</button>
		{/snippet}
	</Tooltip>
{/snippet}

<!-- Send Button -->
{#snippet sendButton()}
    <Tooltip text="Send messages" position={isBottom ? "top" : "bottom"} delay={0}>
		{#snippet children()}
			<button 
				type="button"
				onclick={handleSubmit}
				disabled={isSubmitting || !messageText.trim()}
				aria-label="Send message"
				class="
					bg-[var(--accent-color)]
					text-[var(--text-accent-color)]
					rounded-full 
					flex items-center justify-center flex-shrink-0
					hover:scale-107 cursor-pointer
					transition-all duration-200
					disabled:opacity-50 disabled:cursor-not-allowed
					disabled:hover:scale-100 disabled:hover:shadow-none
				"
				style="width: 2.25rem; height: 2.25rem"
			>
				{#if isSubmitting}
					<div class="animate-spin w-4 h-4 border-2 border-[var(--text-accent-color)] border-t-transparent rounded-full"></div>
				{:else}
					<ArrowUpIcon size="18" />
				{/if}
			</button>
		{/snippet}
	</Tooltip>
{/snippet}

<!-- Task Mode Submit and Select Button -->
{#snippet taskModeButton()}
	<div class="relative" bind:this={taskModeContainer}>
		<!-- Pill-shaped button with mode name and dropdown trigger -->
		<div class="
			flex bg-[var(--accent-color)] 
			rounded-3xl
			overflow-hidden
			hover:scale-105
			transition-all duration-200
			disabled:opacity-50 disabled:cursor-not-allowed
			relative
		">
			<!-- Left part: Mode name (submit button) -->
			<button
				bind:this={leftButton}
				type="button"
				onclick={handleSubmit}
				disabled={isSubmitting || !messageText.trim()}
				class="
					text-[var(--text-accent-color)]
					text-[0.9rem]
					px-3 py-1.25
					cursor-pointer
					select-none
					transition-all duration-200
					disabled:opacity-50 disabled:cursor-not-allowed
				"
				aria-label="Submit task with {currentTaskModeId} mode"
			>
				<span bind:this={leftTitle}>
					{TASK_MODES.find(mode => mode.id === currentTaskModeId)?.label || 'Ask'}
				</span>
			</button>
			
			<!-- Right part: Dropdown trigger with arrow icon -->
			<button
				bind:this={rightButton}
				type="button"
				onclick={() => showTaskModeDropdown = !showTaskModeDropdown}
				class="
					text-[var(--text-accent-color)]
					px-0.5 py-1.25 border-l border-[var(--text-accent-color)]/20
					cursor-pointer
					select-none
					transition-all duration-200
				"
			>
				<span bind:this={rightTitle}>
					<div class="transition-transform duration-200 {showTaskModeDropdown ? 'rotate-90' : 'rotate-0'}">
						<RightArrowIcon/>
					</div>
				</span>
			</button>
		</div>
		
		<!-- Dropdown Menu -->
		{#if showTaskModeDropdown}
			<div 
				class="
					absolute top-full right-0
					mt-1 bg-[var(--bg-primary)]
					border border-[var(--border-color)]
					rounded-3xl shadow-lg z-50 min-w-64 p-2
				"
			>		
				{#each TASK_MODES as mode}
					<button
						type="button"
						onclick={() => {
							currentTaskModeId = mode.id
							showTaskModeDropdown = false
						}}
						class="
							w-full text-left px-3 py-2 
							hover:bg-[var(--bg-third)]/50
							text-[var(--text-primary)]
							transition-colors duration-150
							rounded-2xl
							cursor-pointer
						"
					>
						<div class="text-[0.9rem] font-medium">{mode.label}</div>
						<div class="text-xs text-[var(--text-secondary)]">{mode.description}</div>
					</button>
				{/each}
			</div>
		{/if}
	</div>
{/snippet}

{#snippet modelOptionButton(modelName: string, modelData: Model)}
{@const IconComponent = modelData.icon}
<button
	type="button"
	onclick={() => {
		currentModelName = modelName
		showModelsDropdown = false
	}}
	class="
		w-full text-left px-3 py-2 
		hover:bg-[var(--bg-third)]/50
		text-[var(--text-primary)]
		transition-colors duration-150
		rounded-2xl
		cursor-pointer
		select-none
	"
>
	<div class="flex gap-2">
		<div class="flex flex-col justify-center pl-1">
			<IconComponent color={$theme === 'dark' ? '#fff' : '#111'}/>
		</div>
		<div class="text-[0.9rem] font-medium">{modelName}</div>
	</div>
	<div class="text-xs text-[var(--text-secondary)]">{modelData.description}</div>
</button>
{/snippet}

{#snippet showMoreModelsButton()}
<button
	type="button"
	onclick={() => {
		showModelsDropdown = false
		showModelSelectModal = true
	}}
	class="
		w-full text-left px-3 py-2 
		hover:bg-[var(--bg-third)]/50
		text-[var(--text-primary)]
		transition-colors duration-150
		rounded-2xl
		cursor-pointer
		select-none
	"
>
	<div class="flex justify-between items-center">
		<div class="flex flex-col">
			<div class="text-[0.9rem] font-medium">Show more</div>
			<div class="text-xs text-[var(--text-secondary)]">View all available models</div>
		</div>
		<MoreIcon size="18" className="text-[var(--text-secondary)] mr-4"/>
	</div>
</button>
{/snippet}

{#snippet uploadFromComputerButton()}
	<button
		type="button"
		onclick={() => {
			fileInput?.click();
			showUploadDropdown = false;
		}}
		class="
			flex
			w-full text-left px-2 py-1.75
			hover:bg-[var(--bg-third)]/50
			text-[var(--text-primary)]
			transition-colors duration-150
			rounded-2xl
			cursor-pointer
			select-none
			gap-2
		"
		disabled={isUploading}
	>
		<div class="flex flex-col justify-center pl-1">
			<AttachmentIcon/>
		</div>
		<div class="text-[0.9rem]">
			Upload from computer
		</div>
	</button>
{/snippet}

{#snippet uploadFromExternalAppButton()}
	<div class="relative" bind:this={addFromAppContainer}>
		<button
			type="button"
			onmouseenter={handlePanelEnter}
			onmouseleave={handlePanelLeave}
			class="
				flex justify-between
				w-full text-left px-2 py-1.75
				hover:bg-[var(--bg-third)]/50
				text-[var(--text-primary)]
				transition-colors duration-150
				rounded-2xl
				cursor-pointer
				select-none
			"
		>
			<div class="flex gap-2">
				<div class="flex flex-col justify-center pl-1">
					<AddAppIcon size=18/>
				</div>
				<div class="text-[0.9rem]">Add from app</div>
			</div>
			<div class="flex flex-col justify-center">
				<RightArrowIcon size=18/>
			</div>
		</button>
		
		{#if showAddFromAppPanel}
			<!-- Invisible bridge to connect button to panel -->
			<div 
				role="presentation"
				class="absolute top-0 left-full w-2 h-full z-40"
				onmouseenter={handlePanelEnter}
				onmouseleave={handlePanelLeave}
			></div>
			
			<div 
				role="menu"
				tabindex="-1"
				class="
					absolute -top-5 left-full ml-1
					bg-[var(--bg-primary)]
					border border-[var(--border-color)]
					rounded-3xl shadow-lg z-50 min-w-68 p-2
					transition-all duration-200
				"
				onmouseenter={handlePanelEnter}
				onmouseleave={handlePanelLeave}
			>
				{@render googleDriveOption()}
				{@render oneDriveOption()}
			</div>
		{/if}
	</div>
{/snippet}

{#snippet extendedThinkingButton()}
	<button
		type="button"
		onclick={() => {
			isEnableExtendedThinking = true
			isHoveringExtendedThinking = false
		}}
		class="
			flex
			w-full text-left px-2 py-1.75
			hover:bg-[var(--bg-third)]/50
			text-[var(--text-primary)]
			transition-colors duration-150
			rounded-2xl
			cursor-pointer
			select-none
			gap-2
		"
	>
		<div class="flex flex-col justify-center pl-1">
			<LightBulbIcon size=18/>
		</div>
		<div class="text-[0.9rem]">Extended thinking</div>
	</button>
{/snippet}

{#snippet extendedThinkingViewButton()}
	<button
		type="button"
		onclick={() => {
			isEnableExtendedThinking = false
		}}
		onmouseenter={() => isHoveringExtendedThinking = true}
		onmouseleave={() => isHoveringExtendedThinking = false}
		class="
			bg-transparent hover:bg-[var(--bg-third)]/50
			flex justify-center items-center
			{$theme === 'dark' ? 'text-sky-400/90' : 'text-blue-500'}
			px-2.5 py-1.5 rounded-3xl gap-2
			hover:scale-105
			border-none cursor-pointer select-none
			disabled:opacity-50 disabled:cursor-not-allowed
			transition-all duration-200
		"
	>
		<div class="flex flex-col justify-center">
			<LightBulbIcon/>
		</div>
		<div class="text-[0.9rem]">Think</div>
		{#if isHoveringExtendedThinking}
			<div class="flex flex-col justify-center">
				<XIcon/>
			</div>
		{/if}
	</button>
{/snippet}

{#snippet createImageButton()}
	<button
		type="button"
		onclick={() => {
			isEnableCreateImage = true
			isHoveringCreateImage = false
		}}
		class="
			flex
			w-full text-left px-2 py-1.75
			hover:bg-[var(--bg-third)]/50
			text-[var(--text-primary)]
			transition-colors duration-150
			rounded-2xl
			cursor-pointer
			select-none
			gap-2
		"
	>
		<div class="flex flex-col justify-center pl-1">
			<DrawingIcon/>
		</div>
		<div class="text-[0.9rem]">Create Image</div>
	</button>
{/snippet}

{#snippet createImageViewButton()}
	<button
		type="button"
		onclick={() => {
			isEnableCreateImage = false
		}}
		onmouseenter={() => isHoveringCreateImage = true}
		onmouseleave={() => isHoveringCreateImage = false}
		class="
			bg-transparent hover:bg-[var(--bg-third)]/50
			flex justify-center items-center
			{$theme === 'dark' ? 'text-sky-400/90' : 'text-blue-500'}
			px-2.5 py-1.5 rounded-3xl gap-2
			hover:scale-105
			border-none cursor-pointer select-none
			disabled:opacity-50 disabled:cursor-not-allowed
			transition-all duration-200
		"
	>
		<div class="flex flex-col justify-center">
			<DrawingIcon/>
		</div>
		<div class="text-[0.9rem]">Image</div>
		{#if isHoveringCreateImage}
			<div class="flex flex-col justify-center">
				<XIcon/>
			</div>
		{/if}
	</button>
{/snippet}

{#snippet webSearchToolButton()}
	<button
		type="button"
		onclick={() => {
			isEnableWebSearch = true
			isHoveringWebSearch = false
		}}
		class="
			flex
			w-full text-left px-2 py-1.75
			hover:bg-[var(--bg-third)]/50
			text-[var(--text-primary)]
			transition-colors duration-150
			rounded-2xl
			cursor-pointer
			select-none
			gap-2
		"
	>
		<div class="flex flex-col justify-center pl-1">
			<GlobeIcon size=18/>
		</div>
		<div class="text-[0.9rem]">Web Search</div>
	</button>
{/snippet}

{#snippet webSearchViewButton()}
	<button
		type="button"
		onclick={() => {
			isEnableWebSearch = false
		}}
		onmouseenter={() => isHoveringWebSearch = true}
		onmouseleave={() => isHoveringWebSearch = false}
		class="
			bg-transparent hover:bg-[var(--bg-third)]/50
			flex justify-center items-center
			{$theme === 'dark' ? 'text-sky-400/90' : 'text-blue-500'}
			px-2.5 py-1.5 rounded-3xl gap-2
			hover:scale-105
			border-none cursor-pointer select-none
			disabled:opacity-50 disabled:cursor-not-allowed
			transition-all duration-200
		"
	>
		<div class="flex flex-col justify-center">
			<GlobeIcon/>
		</div>
		<div class="text-[0.9rem]">Web</div>
		{#if isHoveringWebSearch}
			<div class="flex flex-col justify-center">
				<XIcon/>
			</div>
		{/if}
	</button>
{/snippet}

{#snippet addConnectorButton()}
	<button
		type="button"
		class="
			flex
			w-full text-left px-2 py-1.75
			hover:bg-[var(--bg-third)]/50
			text-[var(--text-primary)]
			transition-colors duration-150
			rounded-2xl
			cursor-pointer
			select-none
			gap-2
		"
	>
		<div class="flex flex-col justify-center pl-1">
			<PlusIcon/>
		</div>
		<div class="text-[0.9rem]">Add Connector</div>
	</button>
{/snippet}

{#snippet manageConnectorsButton()}
	<button
		type="button"
		class="
			flex
			w-full text-left px-2 py-1.75
			hover:bg-[var(--bg-third)]/50
			text-[var(--text-primary)]
			transition-colors duration-150
			rounded-2xl
			cursor-pointer
			select-none
			gap-2
		"
	>
		<div class="flex flex-col justify-center pl-1">
			<ManageToolIcon size=18/>
		</div>
		<div class="text-[0.9rem]">Manage Connectors</div>
	</button>
{/snippet}

{#snippet googleDriveOption()}
	<button
		type="button"
		role="menuitem"
		class="
			flex
			w-full text-left px-2 py-1.75
			hover:bg-[var(--bg-third)]/50
			text-[var(--text-primary)]
			transition-colors duration-150
			rounded-2xl
			cursor-pointer
			select-none
			gap-3
		"
	>
		<div class="flex flex-col justify-center">
			<GoogleDriveIcon/>
		</div>
		<div class="text-[0.9rem]">Google Drive</div>
	</button>
{/snippet}

{#snippet oneDriveOption()}
	<button
		type="button"
		role="menuitem"
		class="
			flex
			w-full text-left px-3 py-2 
			hover:bg-[var(--bg-third)]/50
			text-[var(--text-primary)]
			transition-colors duration-150
			rounded-2xl
			cursor-pointer
			select-none
			gap-3
		"
	>
		<div class="flex flex-col justify-center">
			<OneDriveIcon size=18/>
		</div>
		<div class="flex flex-col">
			<div class="text-[0.9rem]">Connect Microsoft OneDrive</div>
			<div class="text-xs text-[var(--text-secondary)]">Work/School - Includes Sharepoint</div>
		</div>
	</button>
{/snippet}

<!-- Uploaded Files Display -->
{#snippet uploadedFilesDisplay()}
	<div class="flex flex-wrap gap-2 px-2 mb-3">
		{#each uploadedFiles as file, index}
			<div class="
				flex items-center gap-2 
				bg-[var(--bg-third)]/30
				border border-[var(--border-color)]/30
				rounded-2xl px-3 py-1.5
				text-[0.85rem] text-[var(--text-secondary)]
			">
				<DocumentsIcon/>
				<span class="truncate max-w-32">
					{file.name}
				</span>
				<button
					type="button"
					onclick={() => removeFile(index)}
					class="
						text-[var(--text-secondary)] hover:text-[var(--text-primary)]
						transition-colors duration-150
						cursor-pointer
						ml-1
					"
				>
					<XIcon size="18"/>
				</button>
			</div>
		{/each}
	</div>
{/snippet}

<!-- Model Select Modal -->
<ModelSelectModal 
	isOpen={showModelSelectModal}
	currentModelName={currentModelName}
	onModelSelect={(modelName) => { currentModelName = modelName }}
	onClose={() => { showModelSelectModal = false }}
/>

<style>
	/* Shaw's Auto-Growing Input/Textarea Technique with 8-row limit */
	.input-sizer {
	display: inline-grid;
	vertical-align: top;
	align-items: stretch;
	position: relative;
	max-height: calc(8 * 1.4 * 0.95rem + 1rem); /* 8 rows * line-height * font-size + padding buffer */
	}

	.input-sizer::after,
	.input-sizer textarea {
	width: auto;
	min-width: 1em;
	grid-area: 1 / 1;
	font: inherit;
	padding: 0;
	margin: 0;
	resize: none;
	background: none;
	appearance: none;
	border: none;
	}

	.input-sizer textarea {
	overflow-y: auto;
	max-height: inherit; /* Inherit the max-height from the container */
	}

	.input-sizer::after {
	content: attr(data-value) ' ';
	visibility: hidden;
	white-space: pre-wrap;
	overflow: hidden; /* Prevent pseudo-element from showing scrollbar */
	}
</style>