<script lang="ts">
	import type { Message } from '$lib/types/api';
	import { marked } from 'marked';

	// Props
	interface MessageProps {
		message: Message;
	}

	let { message }: MessageProps = $props();

	// Convert markdown to HTML
    async function renderMarkdown(content: string): Promise<string> {
		// Configure marked to handle line breaks properly
		marked.setOptions({ breaks: true, gfm: true });
		
		return await marked(content);
	}
</script>

<div class="mb-4 flex {message.sender_type === 'user' ? 'justify-end' : 'justify-start'}">
	<div class="
		px-3 py-2.5
		{message.sender_type === 'user' 
			? 'bg-[var(--accent-color)]/10 text-[var(--text-primary-color)] rounded-2xl max-w-[80%]' 
			: ''
		}
	">
		<!-- Message Content -->
		<div class="text-[0.95rem] prose prose-sm max-w-none">
			{#await renderMarkdown(message.message.content)}
				{message.message.content}
			{:then htmlContent}
				{@html htmlContent}
			{/await}
		</div>
	</div>
</div>