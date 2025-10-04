<script lang="ts">
	import type { Message } from '$lib/types/api';
	import MessageComponent from './Message.svelte';

	// Props
	interface MessageListProps {
		messages: Message[];
		streamingMessage?: {
			message: Message;
			content: string;
		} | null;
	}

	let { messages = [] }: MessageListProps = $props();

	// Auto-scroll to bottom when new messages arrive
	let messagesContainer: HTMLDivElement | undefined;
	let shouldAutoScroll = $state(true);

	// Scroll to bottom function
	function scrollToBottom() {
		if (messagesContainer && shouldAutoScroll) {
			messagesContainer.scrollTop = messagesContainer.scrollHeight;
		}
	}

	// Effect to scroll when messages change
	$effect(() => {
		if (messages.length > 0) {
			// Small delay to ensure DOM is updated
			setTimeout(scrollToBottom, 50);
		}
	});

	// Handle scroll to detect if user has scrolled up
	function handleScroll() {
		if (messagesContainer) {
			const { scrollTop, scrollHeight, clientHeight } = messagesContainer;
			// If user is within 100px of bottom, enable auto-scroll
			shouldAutoScroll = scrollTop + clientHeight >= scrollHeight - 100;
		}
	}
</script>

<div 
	bind:this={messagesContainer}
	class="flex flex-col h-full max-w-[46rem] w-full"
	onscroll={handleScroll}
>
	{#if messages.length === 0}
		<div class="flex flex-col items-center justify-center h-full">	
			<div class="text-center text-[var(--text-secondary)]">
				<div class="text-lg font-medium mb-2">No messages yet</div>
				<div class="text-sm">Start the conversation by sending a message!</div>
			</div>
		</div>
	{:else}
		<!-- Render existing messages -->
		{#each messages as message (message.message_id)}
			<MessageComponent {message} />
		{/each}
	{/if}
</div>