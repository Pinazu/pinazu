<script lang="ts">
	import MessageInput from '$lib/components/chat/MessageInput.svelte';
	import MessageList from '$lib/components/chat/MessageList.svelte';
	import { apiClient } from '$lib/api/client';
	import type { Thread, Message } from '$lib/types/api';
    import { page } from '$app/state';

	let threadId = $derived(page.params.thread_id);
	let thread: Thread | null = $state(null);
	let messages: Message[] = $state([]);

	// Async function to load thread data
	async function loadThreadData(threadId: string) {
		// Reset state when switching threads
		thread = null;
		messages = [];
		
		// Load thread details
		const threadResponse = await apiClient.getThread(threadId);
		if (!threadResponse.success) {
			console.log(threadResponse.error)
			return;
		}
		thread = threadResponse.data;

		// Load messages for this thread
		const messagesResponse = await apiClient.getMessages(threadId);
		if (!messagesResponse.success) {
			console.log(messagesResponse.error)
			return;
		}
		messages = messagesResponse.data.messages;
	}

	// Reactive effect that runs whenever the thread_id parameter changes
	$effect(() => {
		if (threadId) {
			loadThreadData(threadId);
		}
	});
</script>

<div class="flex flex-col justify-between h-screen">
	<div class="pb-4 px-4 flex flex-col justify-center items-center w-full transition-all duration-200 overflow-y-auto">
		<!-- Messages Area -->
		<MessageList {messages} />
	</div>

	<!-- Message Input -->
	<div class="">
		<MessageInput isTaskMode={false} isBottom={true}/>
	</div>
</div>