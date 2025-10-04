<script lang="ts">
    import ArrowDownIcon from "$lib/icons/ArrowDownIcon.svelte";
    import TickIcon from "$lib/icons/TickIcon.svelte";
    import { themeSetting } from "$lib/stores/theme";
    import { onMount } from "svelte";

    let showThemeDropdown = $state(false);
    let themeContainer: HTMLDivElement | undefined = $state();

    const themeOptions = [
        { id: 'light', label: 'Light' },
        { id: 'dark', label: 'Dark' },
        { id: 'system', label: 'System' }
    ] as const;

    onMount(() => {
        const handleClickOutside = (event: Event) => {
            const target = event.target as Node;
            const isOutside = !themeContainer?.contains(target);
            
            if (isOutside) showThemeDropdown = false;
        };
        
        document.addEventListener('click', handleClickOutside);
        
        return () => {
            document.removeEventListener('click', handleClickOutside);
        };
    });

    function selectTheme(selectedTheme: 'light' | 'dark' | 'system') {
        themeSetting.set(selectedTheme);
        showThemeDropdown = false;
    }

    function getCurrentThemeLabel() {
        const currentTheme = $themeSetting;
        return themeOptions.find(option => option.id === currentTheme)?.label || 'System';
    }
</script>

<div class="relative" bind:this={themeContainer}>
    <button 
        type="button"
        aria-label="Select Theme"
        onclick={() => showThemeDropdown = !showThemeDropdown}
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
        <span class="font-medium text-[0.8rem]">{getCurrentThemeLabel()}</span>
        <div class="flex items-center justify-center -mt-0.5">
            <ArrowDownIcon size=14/>
        </div>
    </button>
    
    {#if showThemeDropdown}
        <div 
            class="
                fixed
                mt-1 bg-[var(--bg-primary)]
                border border-[var(--border-color)]
                rounded-2xl shadow-lg z-[150] min-w-32 p-2
            "
            style="
                top: {themeContainer?.getBoundingClientRect().bottom + 4}px;
                left: {themeContainer?.getBoundingClientRect().left}px;
            "
        >
            {#each themeOptions as option}
                <button
                    type="button"
                    onclick={() => selectTheme(option.id)}
                    class="
                        flex justify-between
                        w-full text-left px-3 py-2 
                        hover:bg-[var(--bg-third)]/50
                        text-[var(--text-primary)]
                        transition-colors duration-150
                        rounded-xl
                        cursor-pointer
                    "
                >
                    <div class="text-[0.9rem] font-medium">{option.label}</div>
                    {#if $themeSetting === option.id}
                        <div class="flex justify-center items-center">
                            <TickIcon size=16/>
                        </div>
                    {/if}
                </button>
            {/each}
        </div>
    {/if}
</div>