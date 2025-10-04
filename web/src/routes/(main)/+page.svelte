<script lang="ts">
    import Greeting from '$lib/components/Greeting.svelte';
    import MessageInput from '$lib/components/chat/MessageInput.svelte';
    import SuggestedQuestions from '$lib/components/SuggestedQuestions.svelte';
</script>

<div class="max-h-screen flex flex-col justify-between">
    <Greeting input="How can I help you today?"/>
    <MessageInput isTaskMode={false}/>
    <SuggestedQuestions/>
</div>
