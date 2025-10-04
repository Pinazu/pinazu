<script lang="ts">
    import MessageInput from "$lib/components/chat/MessageInput.svelte";
    import MessageList from "$lib/components/chat/MessageList.svelte";
</script>

<div class="flex flex-col justify-between h-screen">
	<div class="pb-4 px-4 flex flex-col justify-center items-center w-full transition-all duration-200 overflow-y-auto">
		<MessageList messages={[]} />
	</div>
    <div class="">
        <MessageInput isTaskMode={false} isBottom={true}/>
    </div>
</div>