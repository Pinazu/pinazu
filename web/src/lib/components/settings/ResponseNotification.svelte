<script lang="ts">
    import ArrowDownIcon from "$lib/icons/ArrowDownIcon.svelte";
    import BellIcon from "$lib/icons/BellIcon.svelte";
    import { Toggle } from "flowbite-svelte";
    import { onMount } from "svelte";

    let showNotificationDropdown = $state(false);
    let notificationContainer: HTMLDivElement | undefined = $state();
    
    let pushEnabled = $state(true);
    let emailEnabled = $state(false);

    onMount(() => {
        const handleClickOutside = (event: Event) => {
            const target = event.target as Node;
            const isOutside = !notificationContainer?.contains(target);
            
            if (isOutside) showNotificationDropdown = false;
        };
        
        document.addEventListener('click', handleClickOutside);
        
        return () => {
            document.removeEventListener('click', handleClickOutside);
        };
    });

    function getCurrentNotificationLabel() {
        if (pushEnabled && emailEnabled) {
            return 'Push & Email';
        } else if (pushEnabled) {
            return 'Push';
        } else if (emailEnabled) {
            return 'Email';
        } else {
            return 'None';
        }
    }
</script>

<div class="relative" bind:this={notificationContainer}>
    <button 
        type="button"
        aria-label="Select Response Notifications"
        onclick={() => showNotificationDropdown = !showNotificationDropdown}
        class="
            flex items-center gap-1.5
            bg-transparent hover:bg-[var(--bg-third)]/50
            text-[0.9rem] text-[var(--text-secondary)]
            px-2.5 py-1.5 rounded-2xl
            hover:scale-105
            cursor-pointer border-none
            transition-all duration-200
        "
    >
        <span class="font-medium text-[0.8rem]">{getCurrentNotificationLabel()}</span>
        <div class="flex items-center justify-center -mt-0.5">
            <ArrowDownIcon size=14/>
        </div>
    </button>
    
    {#if showNotificationDropdown}
        <div 
            class="
                fixed
                mt-1 bg-[var(--bg-primary)]
                border border-[var(--border-color)]
                rounded-2xl shadow-lg z-[150] min-w-64 p-4
            "
            style="
                top: {notificationContainer?.getBoundingClientRect().bottom + 4}px;
                left: {notificationContainer?.getBoundingClientRect().left}px;
            "
        >   
            <div class="space-y-3">
                <div class="flex justify-between items-center">
                    <div class="flex flex-col">
                        <span class="text-[0.85rem] font-medium text-[var(--text-primary)]">Push Notifications</span>
                        <span class="text-[0.75rem] text-[var(--text-secondary)]/80">Browser notifications</span>
                    </div>
                    <Toggle bind:checked={pushEnabled}></Toggle>
                </div>
                
                <div class="border-t border-[var(--border-color)]/30 pt-3">
                    <div class="flex justify-between items-center">
                        <div class="flex flex-col">
                            <span class="text-[0.85rem] font-medium text-[var(--text-primary)]">Email Notifications</span>
                            <span class="text-[0.75rem] text-[var(--text-secondary)]/80">Send to your email address</span>
                        </div>
                        <Toggle bind:checked={emailEnabled}></Toggle>
                    </div>
                </div>
            </div>
        </div>
    {/if}
</div>