<script lang="ts">
    import Greeting from '$lib/components/Greeting.svelte';
    import MessageInput from '$lib/components/chat/MessageInput.svelte';
</script>

<div>
    <Greeting input="What task should I do for you?"/>
    <MessageInput isTaskMode={true}/>
</div>