<script lang="ts">
    import ArrowDownIcon from "$lib/icons/ArrowDownIcon.svelte";
    import TickIcon from "$lib/icons/TickIcon.svelte";
    import { onMount } from "svelte";

    let selectedLanguage = $state('auto-detect');
    let showLanguageDropdown = $state(false);
    let languageContainer: HTMLDivElement | undefined = $state();

    const languageOptions = [
        { id: 'auto-detect', label: 'Auto-Detect', country: '' },
        { id: 'en-us', label: 'English', country: 'US' },
        { id: 'ar-sa', label: 'Arabic', country: 'SA' },
        { id: 'zh-cn', label: 'Chinese', country: 'CN' },
        { id: 'zh-tw', label: 'Chinese', country: 'TW' },
        { id: 'nl-nl', label: 'Dutch', country: 'NL' },
        { id: 'fi-fi', label: 'Finnish', country: 'FI' },
        { id: 'fr-fr', label: 'French', country: 'FR' },
        { id: 'de-de', label: 'German', country: 'DE' },
        { id: 'he-il', label: 'Hebrew', country: 'IL' },
        { id: 'hi-in', label: 'Hindi', country: 'IN' },
        { id: 'it-it', label: 'Italian', country: 'IT' },
        { id: 'ja-jp', label: 'Japanese', country: 'JP' },
        { id: 'pt-br', label: 'Portuguese', country: 'BR' },
        { id: 'ru-ru', label: 'Russian', country: 'RU' },
        { id: 'es-es', label: 'Spanish', country: 'ES' },
        { id: 'th-th', label: 'Thai', country: 'TH' },
        { id: 'vi-vn', label: 'Vietnamese', country: 'VN' },
    ] as const;

    onMount(() => {
        const handleClickOutside = (event: Event) => {
            const target = event.target as Node;
            const isOutside = !languageContainer?.contains(target);
            
            if (isOutside) showLanguageDropdown = false;
        };
        
        document.addEventListener('click', handleClickOutside);
        
        return () => {
            document.removeEventListener('click', handleClickOutside);
        };
    });

    function selectLanguage(languageId: string) {
        if (languageId === 'separator') return;
        selectedLanguage = languageId;
        showLanguageDropdown = false;
    }

    function getCurrentLanguageLabel() {
        const currentLanguage = languageOptions.find(option => option.id === selectedLanguage);
        if (!currentLanguage) return 'Auto-Detect';
        
        if (currentLanguage.country) {
            return `${currentLanguage.label} (${currentLanguage.country})`;
        }
        return currentLanguage.label;
    }
</script>

<div class="relative" bind:this={languageContainer}>
    <button 
        type="button"
        aria-label="Select Language"
        onclick={() => showLanguageDropdown = !showLanguageDropdown}
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
        <span class="font-medium text-[0.8rem]">{getCurrentLanguageLabel()}</span>
        <div class="flex items-center justify-center -mt-0.5">
            <ArrowDownIcon size=14/>
        </div>
    </button>
    
    {#if showLanguageDropdown}
        <div 
            class="
                fixed
                mt-1 bg-[var(--bg-primary)]
                border border-[var(--border-color)]
                rounded-2xl shadow-lg z-[150] min-w-48 max-w-64 p-2
                max-h-64 overflow-y-auto
            "
            style="
                top: {languageContainer?.getBoundingClientRect().bottom + 4}px;
                left: {languageContainer?.getBoundingClientRect().left}px;
            "
        >
            {#each languageOptions as option}
                <button
                    type="button"
                    onclick={() => selectLanguage(option.id)}
                    class="
                        flex justify-between items-center
                        w-full text-left px-3 py-2 
                        hover:bg-[var(--bg-third)]/50
                        text-[var(--text-primary)]
                        transition-colors duration-150
                        rounded-xl
                        cursor-pointer
                    "
                >
                    <div class="flex flex-col">
                        <div class="text-[0.85rem] font-medium">
                            {option.label}
                            {#if option.country}
                                <span class="text-[var(--text-secondary)] ml-1">({option.country})</span>
                            {/if}
                        </div>
                    </div>
                    {#if selectedLanguage === option.id}
                        <div class="flex justify-center items-center">
                            <TickIcon size=16/>
                        </div>
                    {/if}
                </button>
            {/each}
        </div>
    {/if}
</div>